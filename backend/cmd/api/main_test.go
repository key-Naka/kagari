package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

type memoryAdminRepository struct{ administrators map[string]admin }

func (repository *memoryAdminRepository) Initialize(_ context.Context, username, password string) error {
	if len(repository.administrators) > 0 {
		return nil
	}
	if username == "" || password == "" {
		return errors.New("ADMIN_USERNAME and ADMIN_PASSWORD are required to initialize the administrator")
	}
	hash, err := hashPassword(password)
	if err != nil {
		return err
	}
	repository.administrators[username] = admin{ID: 1, Username: username, PasswordHash: hash}
	return nil
}

func (repository *memoryAdminRepository) FindByUsername(_ context.Context, username string) (admin, error) {
	administrator, ok := repository.administrators[username]
	if !ok {
		return admin{}, errors.New("administrator not found")
	}
	return administrator, nil
}

type memoryConfigRepository struct{ configuration map[string]any }

func (repository *memoryConfigRepository) Get(context.Context) (map[string]any, error) {
	return repository.configuration, nil
}

func (repository *memoryConfigRepository) Save(_ context.Context, configuration map[string]any) error {
	repository.configuration = configuration
	return nil
}

type memorySessionRepository struct{ sessions map[string]uint }

func (repository *memorySessionRepository) Create(_ context.Context, administratorID uint, _ time.Duration) (string, error) {
	token := "test-session"
	repository.sessions[token] = administratorID
	return token, nil
}

func (repository *memorySessionRepository) Get(_ context.Context, token string) (uint, error) {
	administratorID, ok := repository.sessions[token]
	if !ok {
		return 0, errors.New("session not found")
	}
	return administratorID, nil
}

func (repository *memorySessionRepository) Touch(_ context.Context, token string, _ time.Duration) error {
	if _, ok := repository.sessions[token]; !ok {
		return errors.New("session not found")
	}
	return nil
}

func (repository *memorySessionRepository) Delete(_ context.Context, token string) error {
	delete(repository.sessions, token)
	return nil
}

func newTestApp(t *testing.T) (*fiber.App, *memoryAdminRepository, *memoryConfigRepository) {
	t.Helper()
	administrators := &memoryAdminRepository{administrators: make(map[string]admin)}
	if err := administrators.Initialize(context.Background(), "admin", "correct horse battery staple"); err != nil {
		t.Fatalf("initialize administrator: %v", err)
	}
	configuration := &memoryConfigRepository{configuration: map[string]any{"title": "Kagari"}}
	return newApp(nil, appServices{
		administrators: administrators,
		config:         configuration,
		sessions:       &memorySessionRepository{sessions: make(map[string]uint)},
		ttl:            time.Hour,
		cookieDomain:   ".ykagari.top",
		corsOrigin:     "https://ykagari.top",
	}), administrators, configuration
}

func TestHealthReturnsServiceAndDependencyStatus(t *testing.T) {
	app := newApp([]dependency{dependencyStub{name: "mysql"}, dependencyStub{name: "redis"}})

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
	app := newApp([]dependency{dependencyStub{name: "mysql", err: errors.New("mysql unavailable")}, dependencyStub{name: "redis"}})

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

func TestAdministratorInitialization(t *testing.T) {
	repository := &memoryAdminRepository{administrators: make(map[string]admin)}
	if err := repository.Initialize(context.Background(), "admin", "password"); err != nil {
		t.Fatalf("initialize administrator: %v", err)
	}
	administrator, err := repository.FindByUsername(context.Background(), "admin")
	if err != nil {
		t.Fatalf("find initialized administrator: %v", err)
	}
	if administrator.PasswordHash == "password" || !verifyPassword(administrator.PasswordHash, "password") {
		t.Error("initialized administrator password was not stored as an Argon2id hash")
	}
	if err := repository.Initialize(context.Background(), "other", "other-password"); err != nil {
		t.Fatalf("repeat initialization: %v", err)
	}
	if len(repository.administrators) != 1 {
		t.Errorf("administrator count = %d, want 1", len(repository.administrators))
	}
}

func TestAdminSessionAuthenticationAndSiteConfiguration(t *testing.T) {
	app, _, configuration := newTestApp(t)

	t.Run("unauthenticated request is rejected", func(t *testing.T) {
		response, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/admin/site-config", nil))
		if err != nil {
			t.Fatalf("request protected configuration: %v", err)
		}
		defer response.Body.Close()
		if response.StatusCode != fiber.StatusUnauthorized {
			t.Errorf("status = %d, want %d", response.StatusCode, fiber.StatusUnauthorized)
		}
	})

	t.Run("invalid credentials are rejected", func(t *testing.T) {
		request := httptest.NewRequest(fiber.MethodPost, "/api/v1/admin/session", strings.NewReader(`{"username":"admin","password":"incorrect"}`))
		request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
		response, err := app.Test(request)
		if err != nil {
			t.Fatalf("login: %v", err)
		}
		defer response.Body.Close()
		if response.StatusCode != fiber.StatusUnauthorized {
			t.Errorf("status = %d, want %d", response.StatusCode, fiber.StatusUnauthorized)
		}
	})

	request := httptest.NewRequest(fiber.MethodPost, "/api/v1/admin/session", strings.NewReader(`{"username":"admin","password":"correct horse battery staple"}`))
	request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	response.Body.Close()
	if response.StatusCode != fiber.StatusNoContent {
		t.Fatalf("login status = %d, want %d", response.StatusCode, fiber.StatusNoContent)
	}
	cookie := strings.ToLower(response.Header.Get(fiber.HeaderSetCookie))
	for _, attribute := range []string{"httponly", "secure", "samesite=lax", "domain=.ykagari.top"} {
		if !strings.Contains(cookie, attribute) {
			t.Errorf("session cookie %q does not include %q", cookie, attribute)
		}
	}

	t.Run("authenticated session status and configuration are available", func(t *testing.T) {
		statusRequest := httptest.NewRequest(fiber.MethodGet, "/api/v1/admin/session", nil)
		statusRequest.Header.Set(fiber.HeaderCookie, sessionCookieName+"=test-session")
		statusResponse, err := app.Test(statusRequest)
		if err != nil {
			t.Fatalf("session status: %v", err)
		}
		defer statusResponse.Body.Close()
		if statusResponse.StatusCode != fiber.StatusOK {
			t.Fatalf("session status = %d, want %d", statusResponse.StatusCode, fiber.StatusOK)
		}

		putRequest := httptest.NewRequest(fiber.MethodPut, "/api/v1/admin/site-config", strings.NewReader(`{"title":"Updated"}`))
		putRequest.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
		putRequest.Header.Set(fiber.HeaderCookie, sessionCookieName+"=test-session")
		putResponse, err := app.Test(putRequest)
		if err != nil {
			t.Fatalf("save configuration: %v", err)
		}
		putResponse.Body.Close()
		if putResponse.StatusCode != fiber.StatusOK {
			t.Fatalf("save status = %d, want %d", putResponse.StatusCode, fiber.StatusOK)
		}
		if configuration.configuration["title"] != "Updated" {
			t.Errorf("saved title = %v, want Updated", configuration.configuration["title"])
		}
	})
}
