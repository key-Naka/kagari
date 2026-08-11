package main

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
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

const serviceStatusCacheKey = "service-status:v2"

type availability string

const (
	availabilityOperational availability = "operational"
	availabilityDegraded    availability = "degraded"
	availabilityUnavailable availability = "unavailable"
)

type resourceStatus struct {
	CPU     string `json:"cpu"`
	Memory  string `json:"memory"`
	Disk    string `json:"disk"`
	Network string `json:"network"`
	Uptime  string `json:"uptime"`
}

type namedStatus struct {
	Name  string       `json:"name"`
	State availability `json:"state"`
}

type containerStatus struct {
	Name      string       `json:"name"`
	State     availability `json:"state"`
	Resources availability `json:"resources"`
}

// serviceStatus is the exact, sanitized public status contract.
type serviceStatus struct {
	Availability availability      `json:"availability"`
	Resources    resourceStatus    `json:"resources"`
	Containers   []containerStatus `json:"containers"`
	Applications []namedStatus     `json:"applications"`
	SampledAt    string            `json:"sampledAt"`
}

type statusCollector interface {
	Collect(context.Context) (serviceStatus, error)
}

type statusCache interface {
	Get(context.Context) (serviceStatus, error)
	Set(context.Context, serviceStatus, time.Duration) error
}

type redisStatusCache struct{ client *redis.Client }

func (cache redisStatusCache) Get(ctx context.Context) (serviceStatus, error) {
	encoded, err := cache.client.Get(ctx, serviceStatusCacheKey).Bytes()
	if err != nil {
		return serviceStatus{}, fmt.Errorf("get service status cache: %w", err)
	}
	var status serviceStatus
	if err := json.Unmarshal(encoded, &status); err != nil {
		return serviceStatus{}, fmt.Errorf("decode service status cache: %w", err)
	}
	if !status.valid() {
		return serviceStatus{}, errors.New("cached service status has an invalid public contract")
	}
	return status, nil
}

func (cache redisStatusCache) Set(ctx context.Context, status serviceStatus, ttl time.Duration) error {
	encoded, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("encode service status cache: %w", err)
	}
	if err := cache.client.Set(ctx, serviceStatusCacheKey, encoded, ttl).Err(); err != nil {
		return fmt.Errorf("set service status cache: %w", err)
	}
	return nil
}

func (status serviceStatus) valid() bool {
	if !validAvailability(status.Availability) || status.SampledAt == "" {
		return false
	}
	if _, err := time.Parse(time.RFC3339, status.SampledAt); err != nil {
		return false
	}
	for _, value := range []string{status.Resources.CPU, status.Resources.Memory, status.Resources.Disk, status.Resources.Network, status.Resources.Uptime} {
		if !validAvailability(availability(value)) {
			return false
		}
	}
	return validContainerStatuses(status.Containers, []string{"Web", "API", "Database", "Cache"}) && validNamedStatuses(status.Applications, []string{"API", "HTTP", "MySQL", "Redis"})
}

func validAvailability(value availability) bool {
	return value == availabilityOperational || value == availabilityDegraded || value == availabilityUnavailable
}

func validNamedStatuses(values []namedStatus, names []string) bool {
	if len(values) != len(names) {
		return false
	}
	for index, name := range names {
		if values[index].Name != name || !validAvailability(values[index].State) {
			return false
		}
	}
	return true
}

func validContainerStatuses(values []containerStatus, names []string) bool {
	if len(values) != len(names) {
		return false
	}
	for index, name := range names {
		if values[index].Name != name || !validAvailability(values[index].State) || !validAvailability(values[index].Resources) {
			return false
		}
	}
	return true
}

type serviceStatusService struct {
	collector statusCollector
	cache     statusCache
	ttl       time.Duration
	mu        sync.Mutex
}

func degradedServiceStatus() serviceStatus {
	return statusFromStates(availabilityDegraded, availabilityUnavailable, availabilityUnavailable, availabilityUnavailable, availabilityUnavailable, availabilityUnavailable, unavailableContainers(), nil)
}

func statusFromStates(overall, cpu, memory, disk, network, uptime availability, containers map[string]containerStatus, applications map[string]availability) serviceStatus {
	status := serviceStatus{
		Availability: overall,
		Resources:    resourceStatus{CPU: string(cpu), Memory: string(memory), Disk: string(disk), Network: string(network), Uptime: string(uptime)},
		Containers:   []containerStatus{{Name: "Web", State: availabilityUnavailable, Resources: availabilityUnavailable}, {Name: "API", State: availabilityUnavailable, Resources: availabilityUnavailable}, {Name: "Database", State: availabilityUnavailable, Resources: availabilityUnavailable}, {Name: "Cache", State: availabilityUnavailable, Resources: availabilityUnavailable}},
		Applications: []namedStatus{{Name: "API", State: availabilityUnavailable}, {Name: "HTTP", State: availabilityUnavailable}, {Name: "MySQL", State: availabilityUnavailable}, {Name: "Redis", State: availabilityUnavailable}},
		SampledAt:    time.Now().UTC().Format(time.RFC3339),
	}
	for index := range status.Containers {
		if container, ok := containers[status.Containers[index].Name]; ok {
			status.Containers[index] = container
		}
	}
	for index := range status.Applications {
		if state, ok := applications[status.Applications[index].Name]; ok {
			status.Applications[index].State = state
		}
	}
	return status
}

func (service *serviceStatusService) current(ctx context.Context) serviceStatus {
	if service == nil || service.collector == nil {
		return degradedServiceStatus()
	}
	if service.cache != nil {
		if status, err := service.cache.Get(ctx); err == nil {
			return status
		}
	}
	service.mu.Lock()
	defer service.mu.Unlock()
	if service.cache != nil {
		if status, err := service.cache.Get(ctx); err == nil {
			return status
		}
	}
	return service.collectAndCache(ctx)
}

func (service *serviceStatusService) collectAndCache(ctx context.Context) serviceStatus {
	status, err := service.collector.Collect(ctx)
	if err != nil || !status.valid() {
		return degradedServiceStatus()
	}
	if service.cache != nil {
		_ = service.cache.Set(ctx, status, service.ttl)
	}
	return status
}

func (service *serviceStatusService) refresh(ctx context.Context) {
	if service == nil || service.collector == nil {
		return
	}
	service.mu.Lock()
	defer service.mu.Unlock()
	service.collectAndCache(ctx)
}

func (service *serviceStatusService) start(ctx context.Context) {
	if service == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				refreshCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
				service.refresh(refreshCtx)
				cancel()
			}
		}
	}()
}

type productionStatusCollector struct {
	mysql          dependency
	redis          dependency
	dockerURL      string
	hostMetricsURL string
	httpURL        string
	httpClient     *http.Client
	requestTimeout time.Duration
}

func (collector productionStatusCollector) Collect(ctx context.Context) (serviceStatus, error) {
	host := collector.hostMetrics(ctx)
	containers := collector.docker(ctx)
	applications := map[string]availability{
		"API":   availabilityOperational,
		"HTTP":  collector.http(ctx),
		"MySQL": dependencyAvailability(ctx, collector.mysql),
		"Redis": dependencyAvailability(ctx, collector.redis),
	}
	overall := availabilityOperational
	for _, state := range []availability{host.cpu, host.memory, host.disk, host.network, host.uptime, applications["HTTP"], applications["MySQL"], applications["Redis"]} {
		if state != availabilityOperational {
			overall = availabilityDegraded
			break
		}
	}
	for _, container := range containers {
		if container.State != availabilityOperational || container.Resources != availabilityOperational {
			overall = availabilityDegraded
			break
		}
	}
	return statusFromStates(overall, host.cpu, host.memory, host.disk, host.network, host.uptime, containers, applications), nil
}

func dependencyAvailability(ctx context.Context, service dependency) availability {
	if service == nil {
		return availabilityUnavailable
	}
	checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := service.Ping(checkCtx); err != nil {
		return availabilityUnavailable
	}
	return availabilityOperational
}

type hostMetricsStatus struct {
	cpu     availability
	memory  availability
	disk    availability
	network availability
	uptime  availability
}

type hostMetricsDTO struct {
	CPU     availability `json:"cpu"`
	Memory  availability `json:"memory"`
	Disk    availability `json:"disk"`
	Network availability `json:"network"`
	Uptime  availability `json:"uptime"`
}

func (collector productionStatusCollector) hostMetrics(ctx context.Context) hostMetricsStatus {
	unavailable := hostMetricsStatus{cpu: availabilityUnavailable, memory: availabilityUnavailable, disk: availabilityUnavailable, network: availabilityUnavailable, uptime: availabilityUnavailable}
	endpoint, err := url.Parse(collector.hostMetricsURL)
	if err != nil {
		return unavailable
	}
	endpoint.Path = "/status"
	endpoint.RawQuery = ""
	requestCtx, cancel := context.WithTimeout(ctx, collector.requestTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return unavailable
	}
	response, err := collector.httpClient.Do(request)
	if err != nil || response.StatusCode != http.StatusOK {
		if response != nil {
			response.Body.Close()
		}
		return unavailable
	}
	defer response.Body.Close()
	var metrics hostMetricsDTO
	decoder := json.NewDecoder(io.LimitReader(response.Body, 4096))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&metrics) != nil || decoder.Decode(&struct{}{}) != io.EOF || !validAvailability(metrics.CPU) || !validAvailability(metrics.Memory) || !validAvailability(metrics.Disk) || !validAvailability(metrics.Network) || !validAvailability(metrics.Uptime) {
		return unavailable
	}
	return hostMetricsStatus{cpu: metrics.CPU, memory: metrics.Memory, disk: metrics.Disk, network: metrics.Network, uptime: metrics.Uptime}
}

func (collector productionStatusCollector) http(ctx context.Context) availability {
	requestCtx, cancel := context.WithTimeout(ctx, collector.requestTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, collector.httpURL, nil)
	if err != nil {
		return availabilityUnavailable
	}
	response, err := collector.httpClient.Do(request)
	if err != nil {
		return availabilityUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return availabilityUnavailable
	}
	return availabilityOperational
}

func (collector productionStatusCollector) docker(ctx context.Context) map[string]containerStatus {
	states := unavailableContainers()
	endpoint, err := url.Parse(collector.dockerURL)
	if err != nil {
		return states
	}
	endpoint.Path = "/containers/json"
	query := endpoint.Query()
	query.Set("filters", `{"label":["com.docker.compose.project=kagari"]}`)
	endpoint.RawQuery = query.Encode()
	requestCtx, cancel := context.WithTimeout(ctx, collector.requestTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return states
	}
	response, err := collector.httpClient.Do(request)
	if err != nil || response.StatusCode != http.StatusOK {
		if response != nil {
			response.Body.Close()
		}
		return states
	}
	defer response.Body.Close()
	var containers []dockerContainer
	if json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&containers) != nil {
		return states
	}
	for _, container := range containers {
		component := dockerComponent(container.Labels, container.Names)
		if component == "" {
			continue
		}
		if container.State != "running" || container.ID == "" {
			states[component] = containerStatus{Name: component, State: availabilityDegraded, Resources: availabilityUnavailable}
			continue
		}
		states[component] = containerStatus{Name: component, State: availabilityOperational, Resources: collector.dockerResources(ctx, *endpoint, container.ID)}
	}
	return states
}

type dockerContainer struct {
	ID     string            `json:"Id"`
	Names  []string          `json:"Names"`
	State  string            `json:"State"`
	Labels map[string]string `json:"Labels"`
}

type dockerStats struct {
	CPUStats struct {
		CPUUsage struct {
			TotalUsage float64 `json:"total_usage"`
		} `json:"cpu_usage"`
		SystemCPUUsage float64 `json:"system_cpu_usage"`
		OnlineCPUs     float64 `json:"online_cpus"`
	} `json:"cpu_stats"`
	PreCPUStats struct {
		CPUUsage struct {
			TotalUsage float64 `json:"total_usage"`
		} `json:"cpu_usage"`
		SystemCPUUsage float64 `json:"system_cpu_usage"`
	} `json:"precpu_stats"`
	MemoryStats struct {
		Usage float64            `json:"usage"`
		Limit float64            `json:"limit"`
		Stats map[string]float64 `json:"stats"`
	} `json:"memory_stats"`
	Networks map[string]json.RawMessage `json:"networks"`
}

func unavailableContainers() map[string]containerStatus {
	return map[string]containerStatus{
		"Web":      {Name: "Web", State: availabilityUnavailable, Resources: availabilityUnavailable},
		"API":      {Name: "API", State: availabilityUnavailable, Resources: availabilityUnavailable},
		"Database": {Name: "Database", State: availabilityUnavailable, Resources: availabilityUnavailable},
		"Cache":    {Name: "Cache", State: availabilityUnavailable, Resources: availabilityUnavailable},
	}
}

func (collector productionStatusCollector) dockerResources(ctx context.Context, endpoint url.URL, id string) availability {
	endpoint.Path = "/containers/" + url.PathEscape(id) + "/stats"
	endpoint.RawQuery = "stream=false"
	requestCtx, cancel := context.WithTimeout(ctx, collector.requestTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return availabilityUnavailable
	}
	response, err := collector.httpClient.Do(request)
	if err != nil || response.StatusCode != http.StatusOK {
		if response != nil {
			response.Body.Close()
		}
		return availabilityUnavailable
	}
	defer response.Body.Close()
	var stats dockerStats
	if json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&stats) != nil {
		return availabilityUnavailable
	}
	return dockerResourceAvailability(stats)
}

func dockerResourceAvailability(stats dockerStats) availability {
	if len(stats.Networks) == 0 || stats.MemoryStats.Limit <= 0 {
		return availabilityUnavailable
	}
	memoryUsage := stats.MemoryStats.Usage - stats.MemoryStats.Stats["cache"]
	if memoryUsage < 0 {
		memoryUsage = stats.MemoryStats.Usage
	}
	if memoryUsage/stats.MemoryStats.Limit >= .85 {
		return availabilityDegraded
	}
	cpuDelta := stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage
	systemDelta := stats.CPUStats.SystemCPUUsage - stats.PreCPUStats.SystemCPUUsage
	if cpuDelta < 0 || systemDelta <= 0 {
		return availabilityUnavailable
	}
	cpus := stats.CPUStats.OnlineCPUs
	if cpus <= 0 {
		cpus = 1
	}
	if (cpuDelta/systemDelta)*cpus*100 >= 85 {
		return availabilityDegraded
	}
	return availabilityOperational
}

func dockerComponent(labels map[string]string, names []string) string {
	service := labels["com.docker.compose.service"]
	if service == "" && len(names) > 0 {
		service = strings.TrimPrefix(names[0], "/kagari-")
		service = strings.TrimSuffix(service, "-1")
	}
	switch service {
	case "frontend":
		return "Web"
	case "backend":
		return "API"
	case "mysql":
		return "Database"
	case "redis":
		return "Cache"
	default:
		return ""
	}
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
	projects       projectRepository
	status         *serviceStatusService
	github         *githubActivityService
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
	app.Get("/api/v1/service-status", service.serviceStatus)
	app.Get("/api/v1/github", service.githubActivity)
	app.Get("/api/v1/projects", service.publicProjects)
	app.Get("/api/v1/projects/:slug", service.publicProject)
	app.Post("/api/v1/admin/session", service.login)
	app.Delete("/api/v1/admin/session", service.logout)
	app.Get("/api/v1/admin/session", service.requireSession(service.sessionStatus))
	app.Get("/api/v1/admin/site-config", service.requireSession(service.getSiteConfig))
	app.Put("/api/v1/admin/site-config", service.requireSession(service.putSiteConfig))
	app.Get("/api/v1/admin/projects", service.requireSession(service.adminProjects))
	app.Post("/api/v1/admin/projects", service.requireSession(service.createProject))
	app.Put("/api/v1/admin/projects/:id", service.requireSession(service.updateProject))
	app.Delete("/api/v1/admin/projects/:id", service.requireSession(service.deleteProject))
	return app
}

func (service appServices) serviceStatus(c *fiber.Ctx) error {
	return c.JSON(service.status.current(c.Context()))
}

func (service appServices) githubActivity(c *fiber.Ctx) error {
	return c.JSON(service.github.current(c.Context()))
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
	if err := db.AutoMigrate(&admin{}, &siteConfig{}, &portfolioProject{}); err != nil {
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

	services := appServices{administrators: administrators, config: gormConfigRepository{db: db}, sessions: redisSessionRepository{client: client}, projects: gormProjectRepository{db: db}, ttl: sessionTTL(), cookieDomain: cookieDomain(), corsOrigin: corsOrigin()}
	dependencies := []dependency{databaseDependency{db: db}, redisDependency{client: client}}
	statusContext, cancelStatus := context.WithCancel(context.Background())
	defer cancelStatus()
	services.status = &serviceStatusService{
		collector: productionStatusCollector{
			mysql:          dependencies[0],
			redis:          dependencies[1],
			dockerURL:      environmentOrDefault("DOCKER_PROXY_URL", "http://docker-proxy:2375"),
			hostMetricsURL: environmentOrDefault("HOST_METRICS_URL", "http://host-metrics:8090"),
			httpURL:        environmentOrDefault("STATUS_HTTP_URL", "http://frontend:3000/"),
			httpClient:     &http.Client{Timeout: 5 * time.Second},
			requestTimeout: 3 * time.Second,
		},
		cache: redisStatusCache{client: client},
		ttl:   time.Minute,
	}
	services.github = &githubActivityService{
		collector: productionGitHubActivityCollector{
			username:       environmentOrDefault("GITHUB_USERNAME", "key-Naka"),
			apiBase:        githubPublicAPIBase,
			contributions:  githubContributionsBaseURL,
			httpClient:     &http.Client{Timeout: 10 * time.Second},
			requestTimeout: 5 * time.Second,
		},
		cache: redisGitHubActivityCache{client: client},
		ttl:   githubActivityCacheTTL,
	}
	services.status.start(statusContext)
	if err := newApp(dependencies, services).Listen(":" + port); err != nil {
		panic(err)
	}
}

func environmentOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
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
