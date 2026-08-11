package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
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

type memoryGitHubCollector struct {
	data  githubActivityData
	err   error
	calls int
}

func (collector *memoryGitHubCollector) Collect(context.Context) (githubActivityData, error) {
	collector.calls++
	return collector.data, collector.err
}

type memoryGitHubCache struct {
	data     githubActivityData
	err      error
	last     githubActivityData
	lastErr  error
	gets     int
	lastGets int
	sets     int
	ttl      time.Duration
}

func (cache *memoryGitHubCache) Get(context.Context) (githubActivityData, error) {
	cache.gets++
	if cache.err != nil {
		return githubActivityData{}, cache.err
	}
	return cache.data, nil
}

func (cache *memoryGitHubCache) GetLastSuccess(context.Context) (githubActivityData, error) {
	cache.lastGets++
	if cache.lastErr != nil {
		return githubActivityData{}, cache.lastErr
	}
	return cache.last, nil
}

func (cache *memoryGitHubCache) Set(_ context.Context, data githubActivityData, ttl time.Duration) error {
	cache.sets++
	cache.ttl = ttl
	cache.data = data
	cache.err = nil
	cache.last = data
	cache.lastErr = nil
	return nil
}

func TestGitHubActivityPublicAPIUsesCacheAndDegradesSafely(t *testing.T) {
	fresh := githubActivityData{
		Availability: availabilityOperational,
		Contributions: []githubContributionDay{
			{Date: "2026-08-10", Level: 3},
		},
		Activities: []githubActivity{
			{Kind: "pushed", Repository: "key-Naka/kagari", OccurredAt: "2026-08-10T12:00:00Z"},
		},
		Repositories: []githubRepository{
			{Name: "kagari", URL: "https://github.com/key-Naka/kagari", Description: "Personal site", UpdatedAt: "2026-08-10T12:00:00Z"},
		},
		SampledAt: "2026-08-10T12:00:00Z",
	}

	t.Run("caches successful public data for subsequent requests", func(t *testing.T) {
		collector := &memoryGitHubCollector{data: fresh}
		cache := &memoryGitHubCache{err: errors.New("cache miss")}
		app := newApp(nil, appServices{
			github:     &githubActivityService{collector: collector, cache: cache, ttl: time.Hour},
			corsOrigin: "https://ykagari.top",
		})

		for requestNumber := 0; requestNumber < 2; requestNumber++ {
			response, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/github", nil))
			if err != nil {
				t.Fatalf("request GitHub activity: %v", err)
			}
			defer response.Body.Close()

			if response.StatusCode != fiber.StatusOK {
				t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusOK)
			}

			var payload githubActivityData
			if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
				t.Fatalf("decode GitHub activity: %v", err)
			}
			if !payload.valid() || payload.Availability != availabilityOperational || len(payload.Contributions) != 1 || len(payload.Activities) != 1 || len(payload.Repositories) != 1 {
				t.Errorf("payload = %#v, want cached public GitHub activity", payload)
			}
		}

		if collector.calls != 1 || cache.sets != 1 || cache.ttl != time.Hour {
			t.Errorf("collector calls = %d, cache sets = %d, cache ttl = %s, want 1, 1 and 1h", collector.calls, cache.sets, cache.ttl)
		}
	})

	t.Run("returns the latest cached snapshot when collection fails", func(t *testing.T) {
		collector := &memoryGitHubCollector{err: errors.New("upstream unavailable")}
		cache := &memoryGitHubCache{err: errors.New("cache miss"), last: fresh}
		app := newApp(nil, appServices{
			github:     &githubActivityService{collector: collector, cache: cache, ttl: time.Hour},
			corsOrigin: "https://ykagari.top",
		})

		response, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/github", nil))
		if err != nil {
			t.Fatalf("request GitHub activity: %v", err)
		}
		defer response.Body.Close()

		var payload githubActivityData
		if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
			t.Fatalf("decode GitHub activity: %v", err)
		}
		if payload.Availability != availabilityDegraded || len(payload.Contributions) != 1 || collector.calls != 1 || cache.lastGets != 1 {
			t.Errorf("payload = %#v, collector calls = %d, last cache gets = %d, want a degraded cached snapshot", payload, collector.calls, cache.lastGets)
		}
	})

	t.Run("returns a complete safe degraded response when no snapshot exists", func(t *testing.T) {
		collector := &memoryGitHubCollector{err: errors.New("upstream unavailable")}
		cache := &memoryGitHubCache{err: errors.New("cache miss"), lastErr: errors.New("no snapshot")}
		app := newApp(nil, appServices{
			github:     &githubActivityService{collector: collector, cache: cache, ttl: time.Hour},
			corsOrigin: "https://ykagari.top",
		})

		response, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/github", nil))
		if err != nil {
			t.Fatalf("request GitHub activity: %v", err)
		}
		defer response.Body.Close()

		var payload githubActivityData
		if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
			t.Fatalf("decode GitHub activity: %v", err)
		}
		if !payload.valid() || payload.Availability != availabilityDegraded || len(payload.Contributions) != 0 || len(payload.Activities) != 0 || len(payload.Repositories) != 0 {
			t.Errorf("payload = %#v, want complete degraded response", payload)
		}
	})
}

func TestProductionGitHubActivityCollectorUsesOnlyPublicDTOFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/users/key-Naka/contributions":
			_, _ = writer.Write([]byte(`<td data-date="2026-08-09" data-level="2"></td><td data-date="2026-08-10" data-level="4"></td>`))
		case "/users/key-Naka/events/public":
			_, _ = writer.Write([]byte(`[{"type":"PushEvent","created_at":"2026-08-10T12:00:00Z","repo":{"name":"key-Naka/kagari"},"payload":{"private":"ignored"}},{"type":"IssueCommentEvent","created_at":"2026-08-10T13:00:00Z","repo":{"name":"key-Naka/kagari"}}]`))
		case "/users/key-Naka/repos":
			_, _ = writer.Write([]byte(`[{"name":"kagari","html_url":"https://github.com/key-Naka/kagari","description":"Personal site","language":"Go","stargazers_count":3,"updated_at":"2026-08-10T12:00:00Z","owner":{"login":"key-Naka"}}]`))
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	collector := productionGitHubActivityCollector{
		username:       "key-Naka",
		apiBase:        server.URL,
		contributions:  server.URL,
		httpClient:     server.Client(),
		requestTimeout: time.Second,
	}
	data, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatalf("collect GitHub activity: %v", err)
	}
	if !data.valid() || data.Availability != availabilityOperational {
		t.Fatalf("data = %#v, want a valid operational public DTO", data)
	}
	if len(data.Contributions) != 2 || data.Contributions[0].Date != "2026-08-09" || data.Contributions[1].Level != 4 {
		t.Errorf("contributions = %#v, want sorted public contribution days", data.Contributions)
	}
	if len(data.Activities) != 2 || data.Activities[0].Kind != "pushed" || data.Activities[1].Kind != "commented on issue" || data.Activities[0].Repository != "key-Naka/kagari" {
		t.Errorf("activities = %#v, want normalized public activity", data.Activities)
	}
	if len(data.Repositories) != 1 || data.Repositories[0].Name != "kagari" || data.Repositories[0].URL != "https://github.com/key-Naka/kagari" {
		t.Errorf("repositories = %#v, want sanitized public repository", data.Repositories)
	}
	encoded, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal GitHub activity: %v", err)
	}
	if strings.Contains(string(encoded), "payload") || strings.Contains(string(encoded), "private") || strings.Contains(string(encoded), "owner") {
		t.Errorf("public GitHub DTO leaked source fields: %s", encoded)
	}
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

type memoryStatusCache struct {
	status serviceStatus
	err    error
	gets   int
	sets   int
}

func (cache *memoryStatusCache) Get(context.Context) (serviceStatus, error) {
	cache.gets++
	if cache.err != nil {
		return serviceStatus{}, cache.err
	}
	return cache.status, nil
}

func (cache *memoryStatusCache) Set(_ context.Context, status serviceStatus, _ time.Duration) error {
	cache.sets++
	cache.status = status
	return nil
}

type memoryStatusCollector struct {
	status serviceStatus
	err    error
	calls  int
}

func (collector *memoryStatusCollector) Collect(context.Context) (serviceStatus, error) {
	collector.calls++
	return collector.status, collector.err
}

func successfulServiceStatus() serviceStatus {
	return statusFromStates(
		availabilityOperational,
		availabilityOperational,
		availabilityOperational,
		availabilityOperational,
		availabilityOperational,
		availabilityOperational,
		map[string]containerStatus{
			"Web":      {Name: "Web", State: availabilityOperational, Resources: availabilityOperational},
			"API":      {Name: "API", State: availabilityOperational, Resources: availabilityOperational},
			"Database": {Name: "Database", State: availabilityOperational, Resources: availabilityOperational},
			"Cache":    {Name: "Cache", State: availabilityOperational, Resources: availabilityOperational},
		},
		map[string]availability{"API": availabilityOperational, "HTTP": availabilityOperational, "MySQL": availabilityOperational, "Redis": availabilityOperational},
	)
}

func TestProductionStatusCollectorSanitizesProxyFailuresAndDockerStats(t *testing.T) {
	t.Run("invalid host metrics are unavailable without leaking the proxy response", func(t *testing.T) {
		proxy := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"cpu":"operational","memory":"secret-hostname","disk":"operational","network":"operational","uptime":"operational"}`))
		}))
		defer proxy.Close()
		collector := productionStatusCollector{hostMetricsURL: proxy.URL, httpClient: proxy.Client(), requestTimeout: time.Second}
		metrics := collector.hostMetrics(context.Background())
		if metrics.cpu != availabilityUnavailable || metrics.memory != availabilityUnavailable {
			t.Errorf("metrics = %#v, want all unavailable", metrics)
		}
	})

	t.Run("HTTP non-2xx and resource thresholds degrade public status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			switch request.URL.Path {
			case "/status":
				_, _ = writer.Write([]byte(`{"cpu":"operational","memory":"operational","disk":"operational","network":"operational","uptime":"operational"}`))
			case "/containers/json":
				_, _ = writer.Write([]byte(`[{"Id":"web-id","State":"running","Labels":{"com.docker.compose.service":"frontend"}}]`))
			case "/containers/web-id/stats":
				_, _ = writer.Write([]byte(`{"cpu_stats":{"cpu_usage":{"total_usage":850},"system_cpu_usage":1000,"online_cpus":1},"precpu_stats":{"cpu_usage":{"total_usage":0},"system_cpu_usage":0},"memory_stats":{"usage":10,"limit":100},"networks":{"eth0":{}}}`))
			default:
				writer.WriteHeader(http.StatusInternalServerError)
			}
		}))
		defer server.Close()
		collector := productionStatusCollector{hostMetricsURL: server.URL, dockerURL: server.URL, httpURL: server.URL + "/http-failure", httpClient: server.Client(), requestTimeout: time.Second}
		status, err := collector.Collect(context.Background())
		if err != nil {
			t.Fatalf("collect status: %v", err)
		}
		if status.Availability != availabilityDegraded || status.Applications[1].State != availabilityUnavailable || status.Containers[0].Resources != availabilityDegraded {
			t.Errorf("status = %#v, want degraded HTTP and container resources", status)
		}
		encoded, err := json.Marshal(status)
		if err != nil {
			t.Fatalf("marshal public status: %v", err)
		}
		for _, leaked := range []string{"web-id", "850", "1000", "http-failure"} {
			if strings.Contains(string(encoded), leaked) {
				t.Errorf("public status leaked %q: %s", leaked, encoded)
			}
		}
	})

	t.Run("container statistics receive independent timeouts", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			switch request.URL.Path {
			case "/containers/json":
				_, _ = writer.Write([]byte(`[
					{"Id":"web","State":"running","Labels":{"com.docker.compose.service":"frontend"}},
					{"Id":"api","State":"running","Labels":{"com.docker.compose.service":"backend"}},
					{"Id":"database","State":"running","Labels":{"com.docker.compose.service":"mysql"}},
					{"Id":"cache","State":"running","Labels":{"com.docker.compose.service":"redis"}}
				]`))
			default:
				time.Sleep(35 * time.Millisecond)
				_, _ = writer.Write([]byte(`{"cpu_stats":{"cpu_usage":{"total_usage":850},"system_cpu_usage":1000,"online_cpus":1},"precpu_stats":{"cpu_usage":{"total_usage":0},"system_cpu_usage":0},"memory_stats":{"usage":10,"limit":100},"networks":{"eth0":{}}}`))
			}
		}))
		defer server.Close()

		collector := productionStatusCollector{dockerURL: server.URL, httpClient: server.Client(), requestTimeout: 90 * time.Millisecond}
		for _, container := range collector.docker(context.Background()) {
			if container.Resources != availabilityDegraded {
				t.Errorf("container %q resources = %q, want %q", container.Name, container.Resources, availabilityDegraded)
			}
		}
	})
}

func TestServiceStatus(t *testing.T) {
	newStatusApp := func(service *serviceStatusService) *fiber.App {
		return newApp(nil, appServices{status: service, corsOrigin: "https://ykagari.top"})
	}

	t.Run("successful collection returns the exact public DTO", func(t *testing.T) {
		collector := &memoryStatusCollector{status: successfulServiceStatus()}
		cache := &memoryStatusCache{err: errors.New("cache miss")}
		response, err := newStatusApp(&serviceStatusService{collector: collector, cache: cache, ttl: time.Minute}).Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/service-status", nil))
		if err != nil {
			t.Fatalf("request service status: %v", err)
		}
		defer response.Body.Close()
		if response.StatusCode != fiber.StatusOK {
			t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusOK)
		}
		var payload serviceStatus
		if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
			t.Fatalf("decode service status: %v", err)
		}
		if !payload.valid() || payload.Availability != availabilityOperational {
			t.Errorf("payload = %#v, want valid operational DTO", payload)
		}
		if collector.calls != 1 || cache.sets != 1 {
			t.Errorf("collector calls = %d, cache sets = %d, want 1 and 1", collector.calls, cache.sets)
		}
	})

	t.Run("collection failure returns complete sanitized degraded DTO", func(t *testing.T) {
		collector := &memoryStatusCollector{err: errors.New("mysql://db.internal:3306 password=secret")}
		response, err := newStatusApp(&serviceStatusService{collector: collector}).Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/service-status", nil))
		if err != nil {
			t.Fatalf("request service status: %v", err)
		}
		defer response.Body.Close()
		body, err := io.ReadAll(response.Body)
		if err != nil {
			t.Fatalf("read service status: %v", err)
		}
		var payload serviceStatus
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode degraded status: %v", err)
		}
		if !payload.valid() || payload.Availability != availabilityDegraded {
			t.Errorf("payload = %#v, want valid degraded DTO", payload)
		}
		for _, token := range []string{"password", "3306", "db.internal", "port", "hostname", "container id", "command", "env", "192.168", "12345"} {
			if strings.Contains(strings.ToLower(string(body)), token) {
				t.Errorf("response leaked %q: %s", token, body)
			}
		}
	})

	t.Run("cache hit avoids collection", func(t *testing.T) {
		cached := successfulServiceStatus()
		collector := &memoryStatusCollector{err: errors.New("collector must not run")}
		cache := &memoryStatusCache{status: cached}
		response, err := newStatusApp(&serviceStatusService{collector: collector, cache: cache, ttl: time.Minute}).Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/service-status", nil))
		if err != nil {
			t.Fatalf("request service status: %v", err)
		}
		response.Body.Close()
		if collector.calls != 0 || cache.gets != 1 {
			t.Errorf("collector calls = %d, cache gets = %d, want 0 and 1", collector.calls, cache.gets)
		}
	})

	t.Run("refresh shares the collection lock with current", func(t *testing.T) {
		collector := &memoryStatusCollector{status: successfulServiceStatus()}
		service := &serviceStatusService{collector: collector}
		service.mu.Lock()
		done := make(chan struct{})
		go func() {
			service.refresh(context.Background())
			close(done)
		}()
		select {
		case <-done:
			t.Fatal("refresh did not wait for current collection lock")
		case <-time.After(20 * time.Millisecond):
		}
		service.mu.Unlock()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("refresh did not complete after collection lock was released")
		}
	})
}
