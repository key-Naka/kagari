package main

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/argon2"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const sessionCookieName = "kagari_admin_session"

type dependency interface {
	Name() string
	Ping(context.Context) error
}

type databaseDependency struct{ db *gorm.DB }

func (dependency databaseDependency) Name() string { return "mysql" }
func (dependency databaseDependency) Ping(context.Context) error {
	sqlDB, err := dependency.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

type redisDependency struct{ client *redis.Client }

func (dependency redisDependency) Name() string { return "redis" }
func (dependency redisDependency) Ping(ctx context.Context) error {
	return dependency.client.Ping(ctx).Err()
}

type healthResponse struct {
	Status       string            `json:"status"`
	Dependencies map[string]string `json:"dependencies"`
}

type admin struct {
	ID           uint   `gorm:"primaryKey"`
	Username     string `gorm:"size:255;uniqueIndex;not null"`
	PasswordHash string `gorm:"type:text;not null"`
}

type siteConfig struct {
	ID       uint   `gorm:"primaryKey"`
	Contents string `gorm:"type:json;not null"`
}

type adminRepository interface {
	Initialize(context.Context, string, string) error
	FindByUsername(context.Context, string) (admin, error)
}

type configRepository interface {
	Get(context.Context) (map[string]any, error)
	Save(context.Context, map[string]any) error
}

type sessionRepository interface {
	Create(context.Context, uint, time.Duration) (string, error)
	Get(context.Context, string) (uint, error)
	Touch(context.Context, string, time.Duration) error
	Delete(context.Context, string) error
}

type gormAdminRepository struct{ db *gorm.DB }

func (repository gormAdminRepository) Initialize(ctx context.Context, username, password string) error {
	var count int64
	if err := repository.db.WithContext(ctx).Model(&admin{}).Count(&count).Error; err != nil {
		return fmt.Errorf("count administrators: %w", err)
	}
	if count > 0 {
		return nil
	}
	if username == "" || password == "" {
		return errors.New("ADMIN_USERNAME and ADMIN_PASSWORD are required to initialize the administrator")
	}
	hash, err := hashPassword(password)
	if err != nil {
		return fmt.Errorf("hash administrator password: %w", err)
	}
	if err := repository.db.WithContext(ctx).Create(&admin{Username: username, PasswordHash: hash}).Error; err != nil {
		return fmt.Errorf("create administrator: %w", err)
	}
	return nil
}

func (repository gormAdminRepository) FindByUsername(ctx context.Context, username string) (admin, error) {
	var administrator admin
	err := repository.db.WithContext(ctx).Where("username = ?", username).First(&administrator).Error
	if err != nil {
		return admin{}, fmt.Errorf("find administrator: %w", err)
	}
	return administrator, nil
}

type gormConfigRepository struct{ db *gorm.DB }

func (repository gormConfigRepository) Get(ctx context.Context) (map[string]any, error) {
	configuration := siteConfig{ID: 1, Contents: "{}"}
	if err := repository.db.WithContext(ctx).FirstOrCreate(&configuration, siteConfig{ID: 1}).Error; err != nil {
		return nil, fmt.Errorf("get site configuration: %w", err)
	}
	var contents map[string]any
	if err := json.Unmarshal([]byte(configuration.Contents), &contents); err != nil {
		return nil, fmt.Errorf("decode site configuration: %w", err)
	}
	return contents, nil
}

func (repository gormConfigRepository) Save(ctx context.Context, contents map[string]any) error {
	encoded, err := json.Marshal(contents)
	if err != nil {
		return fmt.Errorf("encode site configuration: %w", err)
	}
	configuration := siteConfig{ID: 1, Contents: string(encoded)}
	if err := repository.db.WithContext(ctx).Save(&configuration).Error; err != nil {
		return fmt.Errorf("save site configuration: %w", err)
	}
	return nil
}

type redisSessionRepository struct{ client *redis.Client }

func (repository redisSessionRepository) Create(ctx context.Context, administratorID uint, ttl time.Duration) (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(bytes)
	if err := repository.client.Set(ctx, "admin-session:"+token, strconv.FormatUint(uint64(administratorID), 10), ttl).Err(); err != nil {
		return "", fmt.Errorf("store session: %w", err)
	}
	return token, nil
}

func (repository redisSessionRepository) Get(ctx context.Context, token string) (uint, error) {
	value, err := repository.client.Get(ctx, "admin-session:"+token).Result()
	if err != nil {
		return 0, fmt.Errorf("get session: %w", err)
	}
	id, err := strconv.ParseUint(value, 10, 0)
	if err != nil {
		return 0, fmt.Errorf("parse session administrator: %w", err)
	}
	return uint(id), nil
}

func (repository redisSessionRepository) Touch(ctx context.Context, token string, ttl time.Duration) error {
	return repository.client.Expire(ctx, "admin-session:"+token, ttl).Err()
}

func (repository redisSessionRepository) Delete(ctx context.Context, token string) error {
	return repository.client.Del(ctx, "admin-session:"+token).Err()
}

type appServices struct {
	administrators adminRepository
	config         configRepository
	sessions       sessionRepository
	ttl            time.Duration
	cookieDomain   string
	corsOrigin     string
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func newApp(dependencies []dependency, services ...appServices) *fiber.App {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/health", healthHandler(dependencies))
	if len(services) == 0 {
		return app
	}

	service := services[0]
	app.Use(cors.New(cors.Config{AllowOrigins: service.corsOrigin, AllowMethods: "GET,POST,PUT,DELETE,OPTIONS", AllowHeaders: "Content-Type", AllowCredentials: true}))
	app.Post("/api/v1/admin/session", service.login)
	app.Delete("/api/v1/admin/session", service.logout)
	app.Get("/api/v1/admin/session", service.requireSession(service.sessionStatus))
	app.Get("/api/v1/admin/site-config", service.requireSession(service.getSiteConfig))
	app.Put("/api/v1/admin/site-config", service.requireSession(service.putSiteConfig))
	return app
}

func healthHandler(dependencies []dependency) fiber.Handler {
	return func(c *fiber.Ctx) error {
		status := healthResponse{Status: "ok", Dependencies: make(map[string]string, len(dependencies))}
		ctx, cancel := context.WithTimeout(c.Context(), 2*time.Second)
		defer cancel()
		for _, dependency := range dependencies {
			status.Dependencies[dependency.Name()] = "ok"
			if err := dependency.Ping(ctx); err != nil {
				status.Status = "degraded"
				status.Dependencies[dependency.Name()] = "unavailable"
			}
		}
		if status.Status == "degraded" {
			return c.Status(fiber.StatusServiceUnavailable).JSON(status)
		}
		return c.Status(fiber.StatusOK).JSON(status)
	}
}

func (service appServices) login(c *fiber.Ctx) error {
	var request loginRequest
	if err := c.BodyParser(&request); err != nil || request.Username == "" || request.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "username and password are required"})
	}
	administrator, err := service.administrators.FindByUsername(c.Context(), request.Username)
	if err != nil || !verifyPassword(administrator.PasswordHash, request.Password) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid credentials"})
	}
	token, err := service.sessions.Create(c.Context(), administrator.ID, service.ttl)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "create session"})
	}
	service.setSessionCookie(c, token, int(service.ttl.Seconds()))
	return c.SendStatus(fiber.StatusNoContent)
}

func (service appServices) logout(c *fiber.Ctx) error {
	token := c.Cookies(sessionCookieName)
	if token != "" {
		if err := service.sessions.Delete(c.Context(), token); err != nil && !errors.Is(err, redis.Nil) {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "delete session"})
		}
	}
	service.setSessionCookie(c, "", -1)
	return c.SendStatus(fiber.StatusNoContent)
}

func (service appServices) requireSession(next fiber.Handler) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := c.Cookies(sessionCookieName)
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "authentication required"})
		}
		administratorID, err := service.sessions.Get(c.Context(), token)
		if err != nil || administratorID == 0 {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "authentication required"})
		}
		if err := service.sessions.Touch(c.Context(), token, service.ttl); err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "authentication required"})
		}
		return next(c)
	}
}

func (service appServices) sessionStatus(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"authenticated": true})
}

func (service appServices) getSiteConfig(c *fiber.Ctx) error {
	configuration, err := service.config.Get(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "get site configuration"})
	}
	return c.JSON(configuration)
}

func (service appServices) putSiteConfig(c *fiber.Ctx) error {
	var configuration map[string]any
	if err := c.BodyParser(&configuration); err != nil || configuration == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "site configuration must be a JSON object"})
	}
	if err := service.config.Save(c.Context(), configuration); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "save site configuration"})
	}
	return c.JSON(configuration)
}

func (service appServices) setSessionCookie(c *fiber.Ctx, value string, maxAge int) {
	c.Cookie(&fiber.Cookie{Name: sessionCookieName, Value: value, Path: "/", Domain: service.cookieDomain, MaxAge: maxAge, HTTPOnly: true, Secure: true, SameSite: fiber.CookieSameSiteLaxMode})
}

func hashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	const memory uint32 = 64 * 1024
	const iterations uint32 = 3
	const parallelism uint8 = 2
	hash := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, 32)
	return fmt.Sprintf("argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", memory, iterations, parallelism, base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(hash)), nil
}

func verifyPassword(encoded, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 5 || parts[0] != "argon2id" || parts[1] != "v=19" {
		return false
	}
	var memory, iterations uint32
	var parallelism uint8
	if _, err := fmt.Sscanf(parts[2], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}
	actual := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, uint32(len(expected)))
	return subtle.ConstantTimeCompare(expected, actual) == 1
}

func main() {
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}
	if _, err := strconv.Atoi(port); err != nil {
		panic("APP_PORT must be numeric")
	}

	dsn := os.Getenv("MYSQL_DSN")
	redisAddress := os.Getenv("REDIS_ADDR")
	if dsn == "" || redisAddress == "" {
		panic("MYSQL_DSN and REDIS_ADDR are required")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	if err := db.AutoMigrate(&admin{}, &siteConfig{}); err != nil {
		panic(err)
	}
	redisDB, err := redisDatabaseFromEnvironment()
	if err != nil {
		panic(err)
	}
	client := redis.NewClient(&redis.Options{Addr: redisAddress, Password: os.Getenv("REDIS_PASSWORD"), DB: redisDB})
	administrators := gormAdminRepository{db: db}
	if err := administrators.Initialize(context.Background(), os.Getenv("ADMIN_USERNAME"), os.Getenv("ADMIN_PASSWORD")); err != nil {
		panic(err)
	}

	services := appServices{administrators: administrators, config: gormConfigRepository{db: db}, sessions: redisSessionRepository{client: client}, ttl: sessionTTL(), cookieDomain: cookieDomain(), corsOrigin: corsOrigin()}
	dependencies := []dependency{databaseDependency{db: db}, redisDependency{client: client}}
	if err := newApp(dependencies, services).Listen(":" + port); err != nil {
		panic(err)
	}
}

func redisDatabaseFromEnvironment() (int, error) {
	value := os.Getenv("REDIS_DB")
	if value == "" {
		return 0, nil
	}
	database, err := strconv.Atoi(value)
	if err != nil || database < 0 {
		return 0, errors.New("REDIS_DB must be a non-negative integer")
	}
	return database, nil
}

func sessionTTL() time.Duration {
	value := os.Getenv("SESSION_TTL")
	if value == "" {
		return 24 * time.Hour
	}
	ttl, err := time.ParseDuration(value)
	if err != nil || ttl <= 0 {
		panic("SESSION_TTL must be a positive Go duration")
	}
	return ttl
}

func corsOrigin() string {
	if origin := os.Getenv("CORS_ORIGIN"); origin != "" {
		return origin
	}
	return "https://ykagari.top"
}

func cookieDomain() string {
	if domain := os.Getenv("COOKIE_DOMAIN"); domain != "" {
		return domain
	}
	origin, err := url.Parse(corsOrigin())
	if err == nil && origin.Hostname() != "" {
		return "." + origin.Hostname()
	}
	return ".ykagari.top"
}

type dependencyStub struct {
	name string
	err  error
}

func (stub dependencyStub) Name() string               { return stub.name }
func (stub dependencyStub) Ping(context.Context) error { return stub.err }
