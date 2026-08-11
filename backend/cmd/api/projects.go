package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

const (
	projectStatusDraft     = "draft"
	projectStatusPublished = "published"
)

var (
	errProjectNotFound  = errors.New("portfolio project not found")
	errProjectSlugTaken = errors.New("portfolio project slug already exists")
	projectSlugPattern  = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
)

type portfolioProject struct {
	ID               uint      `gorm:"primaryKey"`
	Title            string    `gorm:"size:160;not null"`
	Slug             string    `gorm:"size:160;uniqueIndex;not null"`
	CoverURL         string    `gorm:"type:text;not null"`
	Description      string    `gorm:"type:text;not null"`
	TechnologiesJSON string    `gorm:"column:technologies;type:json;not null"`
	TypesJSON        string    `gorm:"column:types;type:json;not null"`
	Featured         bool      `gorm:"not null"`
	SortOrder        int       `gorm:"not null"`
	Status           string    `gorm:"size:16;not null"`
	WebsiteURL       string    `gorm:"type:text;not null"`
	RepositoryURL    string    `gorm:"type:text;not null"`
	CreatedAt        time.Time `gorm:"not null"`
	UpdatedAt        time.Time `gorm:"not null"`
}

type projectRepository interface {
	List(context.Context) ([]portfolioProject, error)
	FindByID(context.Context, uint) (portfolioProject, error)
	FindBySlug(context.Context, string) (portfolioProject, error)
	Save(context.Context, portfolioProject) (portfolioProject, error)
	Delete(context.Context, uint) error
}

type gormProjectRepository struct{ db *gorm.DB }

func (repository gormProjectRepository) List(ctx context.Context) ([]portfolioProject, error) {
	var projects []portfolioProject
	if err := repository.db.WithContext(ctx).Find(&projects).Error; err != nil {
		return nil, fmt.Errorf("list portfolio projects: %w", err)
	}
	return projects, nil
}

func (repository gormProjectRepository) FindByID(ctx context.Context, id uint) (portfolioProject, error) {
	var project portfolioProject
	if err := repository.db.WithContext(ctx).First(&project, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return portfolioProject{}, errProjectNotFound
		}
		return portfolioProject{}, fmt.Errorf("find portfolio project: %w", err)
	}
	return project, nil
}

func (repository gormProjectRepository) FindBySlug(ctx context.Context, slug string) (portfolioProject, error) {
	var project portfolioProject
	if err := repository.db.WithContext(ctx).Where("slug = ?", slug).First(&project).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return portfolioProject{}, errProjectNotFound
		}
		return portfolioProject{}, fmt.Errorf("find portfolio project by slug: %w", err)
	}
	return project, nil
}

func (repository gormProjectRepository) Save(ctx context.Context, project portfolioProject) (portfolioProject, error) {
	existing, err := repository.FindBySlug(ctx, project.Slug)
	if err == nil && existing.ID != project.ID {
		return portfolioProject{}, errProjectSlugTaken
	}
	if err != nil && !errors.Is(err, errProjectNotFound) {
		return portfolioProject{}, err
	}
	if project.ID == 0 {
		if err := repository.db.WithContext(ctx).Create(&project).Error; err != nil {
			return portfolioProject{}, fmt.Errorf("create portfolio project: %w", err)
		}
		return project, nil
	}
	if err := repository.db.WithContext(ctx).Save(&project).Error; err != nil {
		return portfolioProject{}, fmt.Errorf("save portfolio project: %w", err)
	}
	return project, nil
}

func (repository gormProjectRepository) Delete(ctx context.Context, id uint) error {
	result := repository.db.WithContext(ctx).Delete(&portfolioProject{}, id)
	if result.Error != nil {
		return fmt.Errorf("delete portfolio project: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errProjectNotFound
	}
	return nil
}

type projectRequest struct {
	Title         string   `json:"title"`
	Slug          string   `json:"slug"`
	CoverURL      string   `json:"coverUrl"`
	Description   string   `json:"description"`
	Technologies  []string `json:"technologies"`
	Types         []string `json:"types"`
	Featured      bool     `json:"featured"`
	SortOrder     int      `json:"sortOrder"`
	Status        string   `json:"status"`
	WebsiteURL    string   `json:"websiteUrl"`
	RepositoryURL string   `json:"repositoryUrl"`
}

type publicProjectResponse struct {
	Title         string   `json:"title"`
	Slug          string   `json:"slug"`
	CoverURL      string   `json:"coverUrl"`
	Description   string   `json:"description"`
	Technologies  []string `json:"technologies"`
	Types         []string `json:"types"`
	Featured      bool     `json:"featured"`
	SortOrder     int      `json:"sortOrder"`
	WebsiteURL    string   `json:"websiteUrl"`
	RepositoryURL string   `json:"repositoryUrl"`
}

type adminProjectResponse struct {
	ID uint `json:"id"`
	publicProjectResponse
	Status string `json:"status"`
}

func projectFromRequest(request projectRequest) (portfolioProject, error) {
	request.Title = strings.TrimSpace(request.Title)
	request.Slug = strings.TrimSpace(request.Slug)
	request.CoverURL = strings.TrimSpace(request.CoverURL)
	request.Description = strings.TrimSpace(request.Description)
	request.Status = strings.TrimSpace(request.Status)
	request.WebsiteURL = strings.TrimSpace(request.WebsiteURL)
	request.RepositoryURL = strings.TrimSpace(request.RepositoryURL)

	if utf8.RuneCountInString(request.Title) == 0 || utf8.RuneCountInString(request.Title) > 160 {
		return portfolioProject{}, errors.New("title must contain between 1 and 160 characters")
	}
	if !projectSlugPattern.MatchString(request.Slug) || len(request.Slug) > 160 {
		return portfolioProject{}, errors.New("slug must use lowercase letters, numbers, and single hyphens")
	}
	if !validHTTPSURL(request.CoverURL) {
		return portfolioProject{}, errors.New("coverUrl must be an HTTPS URL")
	}
	if utf8.RuneCountInString(request.Description) == 0 || utf8.RuneCountInString(request.Description) > 6000 {
		return portfolioProject{}, errors.New("description must contain between 1 and 6000 characters")
	}
	technologies, err := normalizedProjectTags(request.Technologies)
	if err != nil {
		return portfolioProject{}, fmt.Errorf("technologies: %w", err)
	}
	types, err := normalizedProjectTags(request.Types)
	if err != nil {
		return portfolioProject{}, fmt.Errorf("types: %w", err)
	}
	if request.SortOrder < 0 {
		return portfolioProject{}, errors.New("sortOrder must not be negative")
	}
	if request.Status != projectStatusDraft && request.Status != projectStatusPublished {
		return portfolioProject{}, errors.New("status must be draft or published")
	}
	if request.WebsiteURL != "" && !validHTTPSURL(request.WebsiteURL) {
		return portfolioProject{}, errors.New("websiteUrl must be an HTTPS URL")
	}
	if request.RepositoryURL != "" && !validHTTPSURL(request.RepositoryURL) {
		return portfolioProject{}, errors.New("repositoryUrl must be an HTTPS URL")
	}

	encodedTechnologies, err := json.Marshal(technologies)
	if err != nil {
		return portfolioProject{}, fmt.Errorf("encode technologies: %w", err)
	}
	encodedTypes, err := json.Marshal(types)
	if err != nil {
		return portfolioProject{}, fmt.Errorf("encode types: %w", err)
	}
	return portfolioProject{
		Title:            request.Title,
		Slug:             request.Slug,
		CoverURL:         request.CoverURL,
		Description:      request.Description,
		TechnologiesJSON: string(encodedTechnologies),
		TypesJSON:        string(encodedTypes),
		Featured:         request.Featured,
		SortOrder:        request.SortOrder,
		Status:           request.Status,
		WebsiteURL:       request.WebsiteURL,
		RepositoryURL:    request.RepositoryURL,
	}, nil
}

func normalizedProjectTags(tags []string) ([]string, error) {
	if len(tags) > 24 {
		return nil, errors.New("must contain at most 24 values")
	}
	normalized := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if utf8.RuneCountInString(tag) == 0 || utf8.RuneCountInString(tag) > 64 {
			return nil, errors.New("each value must contain between 1 and 64 characters")
		}
		key := strings.ToLower(tag)
		if _, ok := seen[key]; ok {
			return nil, errors.New("values must be unique")
		}
		seen[key] = struct{}{}
		normalized = append(normalized, tag)
	}
	return normalized, nil
}

func validHTTPSURL(value string) bool {
	parsed, err := url.ParseRequestURI(value)
	return err == nil && parsed.Scheme == "https" && parsed.Host != ""
}

func projectTags(encoded string) ([]string, error) {
	var tags []string
	if err := json.Unmarshal([]byte(encoded), &tags); err != nil {
		return nil, errors.New("invalid stored project tags")
	}
	return tags, nil
}

func publicProject(project portfolioProject) (publicProjectResponse, error) {
	technologies, err := projectTags(project.TechnologiesJSON)
	if err != nil {
		return publicProjectResponse{}, err
	}
	types, err := projectTags(project.TypesJSON)
	if err != nil {
		return publicProjectResponse{}, err
	}
	return publicProjectResponse{
		Title:         project.Title,
		Slug:          project.Slug,
		CoverURL:      project.CoverURL,
		Description:   project.Description,
		Technologies:  technologies,
		Types:         types,
		Featured:      project.Featured,
		SortOrder:     project.SortOrder,
		WebsiteURL:    project.WebsiteURL,
		RepositoryURL: project.RepositoryURL,
	}, nil
}

func adminProject(project portfolioProject) (adminProjectResponse, error) {
	public, err := publicProject(project)
	if err != nil {
		return adminProjectResponse{}, err
	}
	return adminProjectResponse{ID: project.ID, publicProjectResponse: public, Status: project.Status}, nil
}

func sortedProjects(projects []portfolioProject) {
	sort.Slice(projects, func(left, right int) bool {
		if projects[left].Featured != projects[right].Featured {
			return projects[left].Featured
		}
		if projects[left].SortOrder != projects[right].SortOrder {
			return projects[left].SortOrder < projects[right].SortOrder
		}
		return projects[left].ID < projects[right].ID
	})
}

func (service appServices) publicProjects(c *fiber.Ctx) error {
	projects, err := service.projects.List(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "list portfolio projects"})
	}
	published := make([]portfolioProject, 0, len(projects))
	for _, project := range projects {
		if project.Status == projectStatusPublished {
			published = append(published, project)
		}
	}
	sortedProjects(published)
	response := make([]publicProjectResponse, 0, len(published))
	for _, project := range published {
		value, err := publicProject(project)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "decode portfolio projects"})
		}
		response = append(response, value)
	}
	return c.JSON(response)
}

func (service appServices) publicProject(c *fiber.Ctx) error {
	project, err := service.projects.FindBySlug(c.Context(), c.Params("slug"))
	if errors.Is(err, errProjectNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "portfolio project not found"})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "find portfolio project"})
	}
	if project.Status != projectStatusPublished {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "portfolio project not found"})
	}
	response, err := publicProject(project)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "decode portfolio project"})
	}
	return c.JSON(response)
}

func (service appServices) adminProjects(c *fiber.Ctx) error {
	projects, err := service.projects.List(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "list portfolio projects"})
	}
	sortedProjects(projects)
	response := make([]adminProjectResponse, 0, len(projects))
	for _, project := range projects {
		value, err := adminProject(project)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "decode portfolio projects"})
		}
		response = append(response, value)
	}
	return c.JSON(response)
}

func (service appServices) createProject(c *fiber.Ctx) error {
	project, err := parseProjectRequest(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	project, err = service.projects.Save(c.Context(), project)
	if errors.Is(err, errProjectSlugTaken) {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "portfolio project slug already exists"})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "create portfolio project"})
	}
	response, err := adminProject(project)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "decode portfolio project"})
	}
	return c.Status(fiber.StatusCreated).JSON(response)
}

func (service appServices) updateProject(c *fiber.Ctx) error {
	id, err := parseProjectID(c)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "portfolio project not found"})
	}
	project, err := parseProjectRequest(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	existing, err := service.projects.FindByID(c.Context(), id)
	if errors.Is(err, errProjectNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "portfolio project not found"})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "find portfolio project"})
	}
	project.ID = id
	project.CreatedAt = existing.CreatedAt
	project, err = service.projects.Save(c.Context(), project)
	if errors.Is(err, errProjectSlugTaken) {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "portfolio project slug already exists"})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "save portfolio project"})
	}
	response, err := adminProject(project)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "decode portfolio project"})
	}
	return c.JSON(response)
}

func (service appServices) deleteProject(c *fiber.Ctx) error {
	id, err := parseProjectID(c)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "portfolio project not found"})
	}
	if err := service.projects.Delete(c.Context(), id); errors.Is(err, errProjectNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "portfolio project not found"})
	} else if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "delete portfolio project"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func parseProjectRequest(c *fiber.Ctx) (portfolioProject, error) {
	var request projectRequest
	if err := c.BodyParser(&request); err != nil {
		return portfolioProject{}, errors.New("portfolio project must be a JSON object")
	}
	return projectFromRequest(request)
}

func parseProjectID(c *fiber.Ctx) (uint, error) {
	id, err := strconv.ParseUint(c.Params("id"), 10, 0)
	if err != nil || id == 0 {
		return 0, errors.New("invalid portfolio project ID")
	}
	return uint(id), nil
}
