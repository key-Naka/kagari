package main

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestHealthReturnsServiceAndDependencyStatus(t *testing.T) {
	app := newApp([]dependency{
		dependencyStub{name: "mysql"},
		dependencyStub{name: "redis"},
	})

	response, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/health", nil))
	if err != nil {
		t.Fatalf("request health endpoint: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusOK)
	}

	var payload healthResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Status != "ok" {
		t.Errorf("service status = %q, want ok", payload.Status)
	}
	if payload.Dependencies["mysql"] != "ok" || payload.Dependencies["redis"] != "ok" {
		t.Errorf("dependency status = %#v, want all ok", payload.Dependencies)
	}
}

func TestHealthReturnsServiceUnavailableWhenDependencyFails(t *testing.T) {
	app := newApp([]dependency{
		dependencyStub{name: "mysql", err: errors.New("mysql unavailable")},
		dependencyStub{name: "redis"},
	})

	response, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/health", nil))
	if err != nil {
		t.Fatalf("request health endpoint: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != fiber.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusServiceUnavailable)
	}

	var payload healthResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Status != "degraded" {
		t.Errorf("service status = %q, want degraded", payload.Status)
	}
	if payload.Dependencies["mysql"] != "unavailable" || payload.Dependencies["redis"] != "ok" {
		t.Errorf("dependency status = %#v, want mysql unavailable and redis ok", payload.Dependencies)
	}
}
