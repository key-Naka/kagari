package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	githubActivityCacheKey     = "github-activity:v1"
	githubLastSuccessCacheKey  = "github-activity:last-success:v1"
	githubActivityCacheTTL     = time.Hour
	githubLastSuccessCacheTTL  = 7 * 24 * time.Hour
	githubPublicAPIBase        = "https://api.github.com"
	githubContributionsBaseURL = "https://github.com"
)

type githubContributionDay struct {
	Date  string `json:"date"`
	Level int    `json:"level"`
}

type githubActivity struct {
	Kind       string `json:"kind"`
	Repository string `json:"repository"`
	OccurredAt string `json:"occurredAt"`
}

type githubRepository struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Language    string `json:"language"`
	Stars       int    `json:"stars"`
	UpdatedAt   string `json:"updatedAt"`
}

type githubActivityData struct {
	Availability  availability            `json:"availability"`
	Contributions []githubContributionDay `json:"contributions"`
	Activities    []githubActivity        `json:"activities"`
	Repositories  []githubRepository      `json:"repositories"`
	SampledAt     string                  `json:"sampledAt"`
}

func (data githubActivityData) valid() bool {
	if !validAvailability(data.Availability) {
		return false
	}
	if _, err := time.Parse(time.RFC3339, data.SampledAt); err != nil {
		return false
	}
	for _, contribution := range data.Contributions {
		if _, err := time.Parse("2006-01-02", contribution.Date); err != nil || contribution.Level < 0 || contribution.Level > 4 {
			return false
		}
	}
	for _, activity := range data.Activities {
		if activity.Kind == "" || activity.Repository == "" {
			return false
		}
		if _, err := time.Parse(time.RFC3339, activity.OccurredAt); err != nil {
			return false
		}
	}
	for _, repository := range data.Repositories {
		if repository.Name == "" || repository.URL == "" {
			return false
		}
		parsedURL, err := url.Parse(repository.URL)
		if err != nil || parsedURL.Scheme != "https" || parsedURL.Host != "github.com" {
			return false
		}
		if _, err := time.Parse(time.RFC3339, repository.UpdatedAt); err != nil {
			return false
		}
	}
	return true
}

func degradedGitHubActivityData() githubActivityData {
	return githubActivityData{
		Availability:  availabilityDegraded,
		Contributions: []githubContributionDay{},
		Activities:    []githubActivity{},
		Repositories:  []githubRepository{},
		SampledAt:     time.Now().UTC().Format(time.RFC3339),
	}
}

type githubActivityCollector interface {
	Collect(context.Context) (githubActivityData, error)
}

type githubActivityCache interface {
	Get(context.Context) (githubActivityData, error)
	GetLastSuccess(context.Context) (githubActivityData, error)
	Set(context.Context, githubActivityData, time.Duration) error
}

type redisGitHubActivityCache struct{ client *redis.Client }

func (cache redisGitHubActivityCache) Get(ctx context.Context) (githubActivityData, error) {
	return cache.get(ctx, githubActivityCacheKey)
}

func (cache redisGitHubActivityCache) GetLastSuccess(ctx context.Context) (githubActivityData, error) {
	return cache.get(ctx, githubLastSuccessCacheKey)
}

func (cache redisGitHubActivityCache) get(ctx context.Context, key string) (githubActivityData, error) {
	encoded, err := cache.client.Get(ctx, key).Bytes()
	if err != nil {
		return githubActivityData{}, fmt.Errorf("get GitHub activity cache: %w", err)
	}
	var data githubActivityData
	if err := json.Unmarshal(encoded, &data); err != nil || !data.valid() {
		return githubActivityData{}, errors.New("decode GitHub activity cache")
	}
	return data, nil
}

func (cache redisGitHubActivityCache) Set(ctx context.Context, data githubActivityData, ttl time.Duration) error {
	encoded, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("encode GitHub activity cache: %w", err)
	}
	if err := cache.client.Set(ctx, githubActivityCacheKey, encoded, ttl).Err(); err != nil {
		return fmt.Errorf("set GitHub activity cache: %w", err)
	}
	if err := cache.client.Set(ctx, githubLastSuccessCacheKey, encoded, githubLastSuccessCacheTTL).Err(); err != nil {
		return fmt.Errorf("set GitHub activity fallback: %w", err)
	}
	return nil
}

type githubActivityService struct {
	collector githubActivityCollector
	cache     githubActivityCache
	ttl       time.Duration
	mu        sync.Mutex
}

func (service *githubActivityService) current(ctx context.Context) githubActivityData {
	if service == nil || service.collector == nil {
		return degradedGitHubActivityData()
	}
	if data, ok := service.cached(ctx); ok {
		return data
	}

	service.mu.Lock()
	defer service.mu.Unlock()
	if data, ok := service.cached(ctx); ok {
		return data
	}

	data, err := service.collector.Collect(ctx)
	if err == nil && data.valid() && data.Availability == availabilityOperational {
		if service.cache != nil {
			_ = service.cache.Set(ctx, data, service.ttl)
		}
		return data
	}
	if service.cache != nil {
		if lastSuccess, cacheErr := service.cache.GetLastSuccess(ctx); cacheErr == nil && lastSuccess.valid() {
			lastSuccess.Availability = availabilityDegraded
			return lastSuccess
		}
	}
	return degradedGitHubActivityData()
}

func (service *githubActivityService) cached(ctx context.Context) (githubActivityData, bool) {
	if service.cache == nil {
		return githubActivityData{}, false
	}
	data, err := service.cache.Get(ctx)
	return data, err == nil && data.valid()
}

type productionGitHubActivityCollector struct {
	username       string
	apiBase        string
	contributions  string
	httpClient     *http.Client
	requestTimeout time.Duration
}

type githubEventDTO struct {
	Type      string `json:"type"`
	CreatedAt string `json:"created_at"`
	Repo      struct {
		Name string `json:"name"`
	} `json:"repo"`
}

type githubRepositoryDTO struct {
	Name            string  `json:"name"`
	HTMLURL         string  `json:"html_url"`
	Description     *string `json:"description"`
	Language        *string `json:"language"`
	StargazersCount int     `json:"stargazers_count"`
	UpdatedAt       string  `json:"updated_at"`
}

var contributionDayPattern = regexp.MustCompile(`data-date="(\d{4}-\d{2}-\d{2})"[^>]*data-level="([0-4])"`)

func (collector productionGitHubActivityCollector) Collect(ctx context.Context) (githubActivityData, error) {
	contributions, err := collector.contributionDays(ctx)
	if err != nil {
		return githubActivityData{}, err
	}
	activities, err := collector.recentActivities(ctx)
	if err != nil {
		return githubActivityData{}, err
	}
	repositories, err := collector.repositories(ctx)
	if err != nil {
		return githubActivityData{}, err
	}
	return githubActivityData{
		Availability:  availabilityOperational,
		Contributions: contributions,
		Activities:    activities,
		Repositories:  repositories,
		SampledAt:     time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (collector productionGitHubActivityCollector) contributionDays(ctx context.Context) ([]githubContributionDay, error) {
	now := time.Now().UTC()
	from := now.AddDate(-1, 0, 1).Format("2006-01-02")
	to := now.Format("2006-01-02")
	endpoint := strings.TrimRight(collector.contributions, "/") + "/users/" + url.PathEscape(collector.username) + "/contributions?from=" + from + "&to=" + to
	body, err := collector.get(ctx, endpoint, "text/html")
	if err != nil {
		return nil, err
	}
	matches := contributionDayPattern.FindAllSubmatch(body, -1)
	if len(matches) == 0 {
		return nil, errors.New("GitHub contribution calendar has no daily values")
	}
	days := make([]githubContributionDay, 0, len(matches))
	for _, match := range matches {
		level, err := strconv.Atoi(string(match[2]))
		if err != nil {
			return nil, err
		}
		days = append(days, githubContributionDay{Date: string(match[1]), Level: level})
	}
	sort.Slice(days, func(left, right int) bool {
		return days[left].Date < days[right].Date
	})
	return days, nil
}

func (collector productionGitHubActivityCollector) recentActivities(ctx context.Context) ([]githubActivity, error) {
	endpoint := strings.TrimRight(collector.apiBase, "/") + "/users/" + url.PathEscape(collector.username) + "/events/public?per_page=12"
	body, err := collector.get(ctx, endpoint, "application/vnd.github+json")
	if err != nil {
		return nil, err
	}
	var events []githubEventDTO
	if err := json.Unmarshal(body, &events); err != nil {
		return nil, errors.New("decode GitHub public events")
	}
	activities := make([]githubActivity, 0, len(events))
	for _, event := range events {
		if event.Repo.Name == "" || event.CreatedAt == "" {
			continue
		}
		activities = append(activities, githubActivity{
			Kind:       githubActivityKind(event.Type),
			Repository: event.Repo.Name,
			OccurredAt: event.CreatedAt,
		})
	}
	return activities, nil
}

func (collector productionGitHubActivityCollector) repositories(ctx context.Context) ([]githubRepository, error) {
	endpoint := strings.TrimRight(collector.apiBase, "/") + "/users/" + url.PathEscape(collector.username) + "/repos?per_page=12&sort=updated"
	body, err := collector.get(ctx, endpoint, "application/vnd.github+json")
	if err != nil {
		return nil, err
	}
	var source []githubRepositoryDTO
	if err := json.Unmarshal(body, &source); err != nil {
		return nil, errors.New("decode GitHub public repositories")
	}
	repositories := make([]githubRepository, 0, len(source))
	for _, repository := range source {
		if repository.Name == "" || repository.HTMLURL == "" || repository.UpdatedAt == "" {
			continue
		}
		description := ""
		if repository.Description != nil {
			description = *repository.Description
		}
		language := ""
		if repository.Language != nil {
			language = *repository.Language
		}
		repositories = append(repositories, githubRepository{
			Name:        repository.Name,
			URL:         repository.HTMLURL,
			Description: description,
			Language:    language,
			Stars:       repository.StargazersCount,
			UpdatedAt:   repository.UpdatedAt,
		})
	}
	return repositories, nil
}

func (collector productionGitHubActivityCollector) get(ctx context.Context, endpoint, accept string) ([]byte, error) {
	requestCtx, cancel := context.WithTimeout(ctx, collector.requestTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", accept)
	request.Header.Set("User-Agent", "kagari-public-github-page")
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	response, err := collector.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, errors.New("GitHub public request failed")
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	return body, nil
}

func githubActivityKind(eventType string) string {
	switch eventType {
	case "PushEvent":
		return "pushed"
	case "CreateEvent":
		return "created"
	case "IssuesEvent":
		return "updated issue"
	case "IssueCommentEvent":
		return "commented on issue"
	case "PullRequestEvent":
		return "updated pull request"
	case "WatchEvent":
		return "starred"
	default:
		return strings.TrimSuffix(eventType, "Event")
	}
}
