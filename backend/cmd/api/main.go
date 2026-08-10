package main

import (
	"context"
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type dependency interface {
	Name() string
	Ping(context.Context) error
}

type databaseDependency struct {
	db *gorm.DB
}

func (dependency databaseDependency) Name() string { return "mysql" }
func (dependency databaseDependency) Ping(context.Context) error {
	sqlDB, err := dependency.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

type redisDependency struct {
	client *redis.Client
}

func (dependency redisDependency) Name() string { return "redis" }
func (dependency redisDependency) Ping(ctx context.Context) error {
	return dependency.client.Ping(ctx).Err()
}

type healthResponse struct {
	Status       string            `json:"status"`
	Dependencies map[string]string `json:"dependencies"`
}

func newApp(dependencies []dependency) *fiber.App {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/health", func(c *fiber.Ctx) error {
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
	})
	return app
}

func main() {
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}
	if _, err := strconv.Atoi(port); err != nil {
		panic("APP_PORT must be numeric")
	}

	var dependencies []dependency
	if dsn := os.Getenv("MYSQL_DSN"); dsn != "" {
		db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err != nil {
			panic(err)
		}
		dependencies = append(dependencies, databaseDependency{db: db})
	}
	if addr := os.Getenv("REDIS_ADDR"); addr != "" {
		client := redis.NewClient(&redis.Options{Addr: addr, Password: os.Getenv("REDIS_PASSWORD")})
		dependencies = append(dependencies, redisDependency{client: client})
	}
	if len(dependencies) == 0 {
		dependencies = append(dependencies, dependencyStub{name: "configuration", err: errors.New("dependencies are not configured")})
	}

	if err := newApp(dependencies).Listen(":" + port); err != nil {
		panic(err)
	}
}

type dependencyStub struct {
	name string
	err  error
}

func (stub dependencyStub) Name() string               { return stub.name }
func (stub dependencyStub) Ping(context.Context) error { return stub.err }
