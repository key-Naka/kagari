package main

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"html"
	"net"
	"net/mail"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gofiber/fiber/v2"
	"github.com/microcosm-cc/bluemonday"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const (
	visitorMessageNicknameLimit = 80
	visitorMessageEmailLimit    = 254
	visitorMessageContentLimit  = 1000
)

var (
	errVisitorMessageNotFound = errors.New("visitor message not found")
	visitorMessageTextPolicy  = bluemonday.StrictPolicy()
)

type visitorMessage struct {
	ID        uint      `gorm:"primaryKey"`
	Nickname  string    `gorm:"size:80;not null"`
	Email     string    `gorm:"size:254;not null"`
	Content   string    `gorm:"type:text;not null"`
	CreatedAt time.Time `gorm:"not null"`
}

type visitorMessageRepository interface {
	ListNewestFirst(context.Context) ([]visitorMessage, error)
	Save(context.Context, visitorMessage) (visitorMessage, error)
	Delete(context.Context, uint) error
}

type gormVisitorMessageRepository struct{ db *gorm.DB }

func (repository gormVisitorMessageRepository) ListNewestFirst(ctx context.Context) ([]visitorMessage, error) {
	var messages []visitorMessage
	if err := repository.db.WithContext(ctx).Order("created_at DESC").Find(&messages).Error; err != nil {
		return nil, fmt.Errorf("list visitor messages: %w", err)
	}
	return messages, nil
}

func (repository gormVisitorMessageRepository) Save(ctx context.Context, message visitorMessage) (visitorMessage, error) {
	if err := repository.db.WithContext(ctx).Create(&message).Error; err != nil {
		return visitorMessage{}, fmt.Errorf("create visitor message: %w", err)
	}
	return message, nil
}

func (repository gormVisitorMessageRepository) Delete(ctx context.Context, id uint) error {
	result := repository.db.WithContext(ctx).Delete(&visitorMessage{}, id)
	if result.Error != nil {
		return fmt.Errorf("delete visitor message: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errVisitorMessageNotFound
	}
	return nil
}

type visitorMessageRateLimiter interface {
	Allow(context.Context, string) (retryAfter time.Duration, allowed bool, err error)
}

type redisVisitorMessageRateLimiter struct {
	client *redis.Client
	limit  int64
	window time.Duration
}

var visitorMessageRateLimitScript = redis.NewScript(`
local count = redis.call('INCR', KEYS[1])
if count == 1 then
  redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
local ttl = redis.call('PTTL', KEYS[1])
return {count, ttl}
`)

func (limiter redisVisitorMessageRateLimiter) Allow(ctx context.Context, source string) (time.Duration, bool, error) {
	if limiter.client == nil || limiter.limit <= 0 || limiter.window <= 0 {
		return 0, false, errors.New("visitor message rate limit is not configured")
	}
	digest := sha256.Sum256([]byte(source))
	key := fmt.Sprintf("visitor-message-rate-limit:v1:%x", digest)
	values, err := visitorMessageRateLimitScript.Run(ctx, limiter.client, []string{key}, limiter.window.Milliseconds()).Slice()
	if err != nil {
		return 0, false, fmt.Errorf("apply visitor message rate limit: %w", err)
	}
	if len(values) != 2 {
		return 0, false, errors.New("visitor message rate limit returned an invalid result")
	}
	count, countOK := values[0].(int64)
	retryMilliseconds, retryOK := values[1].(int64)
	if !countOK || !retryOK {
		return 0, false, errors.New("visitor message rate limit returned invalid counters")
	}
	retryAfter := time.Duration(retryMilliseconds) * time.Millisecond
	if retryAfter <= 0 {
		retryAfter = limiter.window
	}
	return retryAfter, count <= limiter.limit, nil
}

type visitorMessageRequest struct {
	Nickname string `json:"nickname"`
	Email    string `json:"email"`
	Content  string `json:"content"`
}

type publicVisitorMessageResponse struct {
	ID        uint   `json:"id"`
	Nickname  string `json:"nickname"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
}

type adminVisitorMessageResponse struct {
	ID        uint   `json:"id"`
	Nickname  string `json:"nickname"`
	Email     string `json:"email"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
}

func (service appServices) publicVisitorMessages(c *fiber.Ctx) error {
	messages, err := service.visitorMessages.ListNewestFirst(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "list visitor messages"})
	}
	response := make([]publicVisitorMessageResponse, 0, len(messages))
	for _, message := range messages {
		response = append(response, publicVisitorMessage(message))
	}
	return c.JSON(response)
}

func (service appServices) createVisitorMessage(c *fiber.Ctx) error {
	message, err := parseVisitorMessageRequest(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if service.messageLimiter != nil {
		if retryAfter, allowed, err := service.messageLimiter.Allow(c.Context(), visitorMessageRateLimitSource(c)); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "visitor message rate limit unavailable"})
		} else if !allowed {
			retryAfterSeconds := (retryAfter + time.Second - 1) / time.Second
			c.Set(fiber.HeaderRetryAfter, strconv.FormatInt(int64(retryAfterSeconds), 10))
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "visitor message rate limit exceeded"})
		}
	}
	message, err = service.visitorMessages.Save(c.Context(), message)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "create visitor message"})
	}
	return c.Status(fiber.StatusCreated).JSON(publicVisitorMessage(message))
}

func (service appServices) adminVisitorMessages(c *fiber.Ctx) error {
	messages, err := service.visitorMessages.ListNewestFirst(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "list visitor messages"})
	}
	response := make([]adminVisitorMessageResponse, 0, len(messages))
	for _, message := range messages {
		response = append(response, adminVisitorMessage(message))
	}
	return c.JSON(response)
}

func (service appServices) deleteVisitorMessage(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 0)
	if err != nil || id == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "visitor message not found"})
	}
	if err := service.visitorMessages.Delete(c.Context(), uint(id)); errors.Is(err, errVisitorMessageNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "visitor message not found"})
	} else if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "delete visitor message"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func parseVisitorMessageRequest(c *fiber.Ctx) (visitorMessage, error) {
	var request visitorMessageRequest
	if err := c.BodyParser(&request); err != nil {
		return visitorMessage{}, errors.New("visitor message must be a JSON object")
	}
	nickname := sanitizedVisitorMessageText(request.Nickname)
	emailAddress := strings.ToLower(sanitizedVisitorMessageText(request.Email))
	content := sanitizedVisitorMessageText(request.Content)
	if utf8.RuneCountInString(nickname) > visitorMessageNicknameLimit {
		return visitorMessage{}, errors.New("visitor message nickname is too long")
	}
	if utf8.RuneCountInString(emailAddress) > visitorMessageEmailLimit {
		return visitorMessage{}, errors.New("visitor message email is too long")
	}
	if emailAddress != "" {
		parsed, err := mail.ParseAddress(emailAddress)
		if err != nil || parsed.Address != emailAddress {
			return visitorMessage{}, errors.New("visitor message email is invalid")
		}
	}
	contentLength := utf8.RuneCountInString(content)
	if contentLength == 0 {
		return visitorMessage{}, errors.New("visitor message content is required")
	}
	if contentLength > visitorMessageContentLimit {
		return visitorMessage{}, errors.New("visitor message content is too long")
	}
	return visitorMessage{Nickname: nickname, Email: emailAddress, Content: content}, nil
}

func sanitizedVisitorMessageText(value string) string {
	decoded := html.UnescapeString(value)
	return strings.TrimSpace(visitorMessageTextPolicy.Sanitize(decoded))
}

func publicVisitorMessage(message visitorMessage) publicVisitorMessageResponse {
	return publicVisitorMessageResponse{
		ID:        message.ID,
		Nickname:  message.Nickname,
		Content:   message.Content,
		CreatedAt: message.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func adminVisitorMessage(message visitorMessage) adminVisitorMessageResponse {
	return adminVisitorMessageResponse{
		ID:        message.ID,
		Nickname:  message.Nickname,
		Email:     message.Email,
		Content:   message.Content,
		CreatedAt: message.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func visitorMessageRateLimitSource(c *fiber.Ctx) string {
	forwardedFor := strings.Split(c.Get(fiber.HeaderXForwardedFor), ",")
	sourceIP := strings.TrimSpace(forwardedFor[len(forwardedFor)-1])
	if net.ParseIP(sourceIP) == nil {
		sourceIP = c.IP()
	}
	return sourceIP + "|POST:/api/v1/visitor-messages"
}
