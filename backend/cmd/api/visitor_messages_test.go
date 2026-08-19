package main

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

type memoryVisitorMessageRepository struct {
	nextID   uint
	messages map[uint]visitorMessage
}

func TestRedisRateLimitIsScopedByVisitorIPAndRoute(t *testing.T) {
	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() {
		if err := redisClient.Close(); err != nil {
			t.Errorf("close Redis client: %v", err)
		}
	})
	repository := &memoryVisitorMessageRepository{messages: make(map[uint]visitorMessage)}
	app := newApp(nil, appServices{
		visitorMessages: repository,
		messageLimiter: redisVisitorMessageRateLimiter{
			client: redisClient,
			limit:  3,
			window: 10 * time.Minute,
		},
		corsOrigin: "https://ykagari.top",
	})

	submit := func(t *testing.T, sourceIP string) int {
		t.Helper()
		request := httptest.NewRequest(fiber.MethodPost, "/api/v1/visitor-messages", strings.NewReader(`{"content":"A finite public signal"}`))
		request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
		request.Header.Set(fiber.HeaderXForwardedFor, sourceIP)
		response, err := app.Test(request)
		if err != nil {
			t.Fatalf("submit visitor message: %v", err)
		}
		response.Body.Close()
		return response.StatusCode
	}

	for submission := 1; submission <= 3; submission++ {
		if status := submit(t, "203.0.113.10"); status != fiber.StatusCreated {
			t.Fatalf("submission %d status = %d, want %d", submission, status, fiber.StatusCreated)
		}
	}
	if status := submit(t, "198.51.100.99, 203.0.113.10"); status != fiber.StatusTooManyRequests {
		t.Errorf("fourth submission status = %d, want %d", status, fiber.StatusTooManyRequests)
	}
	if status := submit(t, "203.0.113.11"); status != fiber.StatusCreated {
		t.Errorf("different IP submission status = %d, want %d", status, fiber.StatusCreated)
	}
}

func TestAdministratorCanReviewPrivateEmailAndPermanentlyDeleteVisitorMessage(t *testing.T) {
	repository := &memoryVisitorMessageRepository{
		nextID: 1,
		messages: map[uint]visitorMessage{
			1: {
				ID:        1,
				Nickname:  "Aya",
				Email:     "aya@example.com",
				Content:   "A message for the archive",
				CreatedAt: time.Date(2026, time.August, 19, 9, 0, 0, 0, time.UTC),
			},
		},
	}
	app := newApp(nil, appServices{
		visitorMessages: repository,
		sessions:        &memorySessionRepository{sessions: map[string]uint{"admin-token": 1}},
		ttl:             time.Hour,
		corsOrigin:      "https://ykagari.top",
	})

	unauthenticated, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/admin/visitor-messages", nil))
	if err != nil {
		t.Fatalf("list visitor messages without session: %v", err)
	}
	unauthenticated.Body.Close()
	if unauthenticated.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d, want %d", unauthenticated.StatusCode, fiber.StatusUnauthorized)
	}

	listRequest := httptest.NewRequest(fiber.MethodGet, "/api/v1/admin/visitor-messages", nil)
	listRequest.Header.Set(fiber.HeaderCookie, sessionCookieName+"=admin-token")
	listResponse, err := app.Test(listRequest)
	if err != nil {
		t.Fatalf("list visitor messages as administrator: %v", err)
	}
	defer listResponse.Body.Close()
	var messages []adminVisitorMessageResponse
	if err := json.NewDecoder(listResponse.Body).Decode(&messages); err != nil {
		t.Fatalf("decode administrator visitor messages: %v", err)
	}
	if len(messages) != 1 || messages[0].Email != "aya@example.com" {
		t.Fatalf("administrator visitor messages = %#v, want private email", messages)
	}

	deleteRequest := httptest.NewRequest(fiber.MethodDelete, "/api/v1/admin/visitor-messages/1", nil)
	deleteRequest.Header.Set(fiber.HeaderCookie, sessionCookieName+"=admin-token")
	deleteResponse, err := app.Test(deleteRequest)
	if err != nil {
		t.Fatalf("delete visitor message: %v", err)
	}
	deleteResponse.Body.Close()
	if deleteResponse.StatusCode != fiber.StatusNoContent {
		t.Fatalf("delete status = %d, want %d", deleteResponse.StatusCode, fiber.StatusNoContent)
	}
	if len(repository.messages) != 0 {
		t.Errorf("visitor message count after deletion = %d, want 0", len(repository.messages))
	}
}

func (repository *memoryVisitorMessageRepository) ListNewestFirst(context.Context) ([]visitorMessage, error) {
	messages := make([]visitorMessage, 0, len(repository.messages))
	for _, message := range repository.messages {
		messages = append(messages, message)
	}
	sort.Slice(messages, func(left, right int) bool {
		return messages[left].CreatedAt.After(messages[right].CreatedAt)
	})
	return messages, nil
}

func (repository *memoryVisitorMessageRepository) Save(_ context.Context, message visitorMessage) (visitorMessage, error) {
	repository.nextID++
	message.ID = repository.nextID
	message.CreatedAt = time.Date(2026, time.August, 19, 8, 30, 0, 0, time.UTC)
	repository.messages[message.ID] = message
	return message, nil
}

func (repository *memoryVisitorMessageRepository) Delete(_ context.Context, id uint) error {
	if _, exists := repository.messages[id]; !exists {
		return errVisitorMessageNotFound
	}
	delete(repository.messages, id)
	return nil
}

type allowVisitorMessageRateLimiter struct{}

func (allowVisitorMessageRateLimiter) Allow(context.Context, string) (time.Duration, bool, error) {
	return 0, true, nil
}

type denyVisitorMessageRateLimiter struct{ retryAfter time.Duration }

func (limiter denyVisitorMessageRateLimiter) Allow(context.Context, string) (time.Duration, bool, error) {
	return limiter.retryAfter, false, nil
}

func TestVisitorCanSubmitSanitizedMessageWithoutPublishingEmail(t *testing.T) {
	repository := &memoryVisitorMessageRepository{messages: make(map[uint]visitorMessage)}
	app := newApp(nil, appServices{
		visitorMessages: repository,
		messageLimiter:  allowVisitorMessageRateLimiter{},
		corsOrigin:      "https://ykagari.top",
	})

	request := httptest.NewRequest(fiber.MethodPost, "/api/v1/visitor-messages", strings.NewReader(`{
		"nickname":" <b>Aya</b> ",
		"email":"Aya@Example.COM",
		"content":"Hello &lt;img src=x onerror=private()&gt;<script>private()</script><strong>world</strong>"
	}`))
	request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	request.Header.Set(fiber.HeaderXForwardedFor, "203.0.113.8")
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("submit visitor message: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusCreated {
		t.Fatalf("submit status = %d, want %d", response.StatusCode, fiber.StatusCreated)
	}
	var created publicVisitorMessageResponse
	if err := json.NewDecoder(response.Body).Decode(&created); err != nil {
		t.Fatalf("decode created visitor message: %v", err)
	}
	if created.Nickname != "Aya" || created.Content != "Hello world" {
		t.Errorf("created visitor message = %#v, want sanitized nickname and content", created)
	}

	stored := repository.messages[created.ID]
	if stored.Email != "aya@example.com" {
		t.Errorf("stored private email = %q, want normalized address", stored.Email)
	}

	listResponse, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/visitor-messages", nil))
	if err != nil {
		t.Fatalf("list visitor messages: %v", err)
	}
	defer listResponse.Body.Close()
	var listed []map[string]any
	if err := json.NewDecoder(listResponse.Body).Decode(&listed); err != nil {
		t.Fatalf("decode visitor messages: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("visitor message count = %d, want 1", len(listed))
	}
	if _, exposed := listed[0]["email"]; exposed {
		t.Error("public visitor message exposed the private email")
	}
}

func TestVisitorMessageRateLimitRejectsSubmissionWithoutPersistingIt(t *testing.T) {
	repository := &memoryVisitorMessageRepository{messages: make(map[uint]visitorMessage)}
	app := newApp(nil, appServices{
		visitorMessages: repository,
		messageLimiter:  denyVisitorMessageRateLimiter{retryAfter: 90 * time.Second},
		corsOrigin:      "https://ykagari.top",
	})
	request := httptest.NewRequest(fiber.MethodPost, "/api/v1/visitor-messages", strings.NewReader(`{"content":"Please let this through"}`))
	request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	request.Header.Set(fiber.HeaderXForwardedFor, "203.0.113.9")

	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("submit rate-limited visitor message: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusTooManyRequests)
	}
	if response.Header.Get(fiber.HeaderRetryAfter) != "90" {
		t.Errorf("Retry-After = %q, want 90", response.Header.Get(fiber.HeaderRetryAfter))
	}
	if len(repository.messages) != 0 {
		t.Errorf("persisted visitor message count = %d, want 0", len(repository.messages))
	}
}

func TestVisitorMessageSubmissionValidatesEveryPublicInput(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "content is required", body: `{}`},
		{name: "HTML-only content is empty after cleaning", body: `{"content":"<script>private()</script>"}`},
		{name: "content length is bounded", body: `{"content":"` + strings.Repeat("讯", visitorMessageContentLimit+1) + `"}`},
		{name: "nickname length is bounded", body: `{"nickname":"` + strings.Repeat("访", visitorMessageNicknameLimit+1) + `","content":"hello"}`},
		{name: "email must be an address", body: `{"email":"not-an-address","content":"hello"}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &memoryVisitorMessageRepository{messages: make(map[uint]visitorMessage)}
			app := newApp(nil, appServices{
				visitorMessages: repository,
				messageLimiter:  allowVisitorMessageRateLimiter{},
				corsOrigin:      "https://ykagari.top",
			})
			request := httptest.NewRequest(fiber.MethodPost, "/api/v1/visitor-messages", strings.NewReader(test.body))
			request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
			response, err := app.Test(request)
			if err != nil {
				t.Fatalf("submit invalid visitor message: %v", err)
			}
			response.Body.Close()
			if response.StatusCode != fiber.StatusBadRequest {
				t.Errorf("status = %d, want %d", response.StatusCode, fiber.StatusBadRequest)
			}
			if len(repository.messages) != 0 {
				t.Errorf("persisted invalid visitor message count = %d, want 0", len(repository.messages))
			}
		})
	}
}
