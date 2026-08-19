package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gofiber/fiber/v2"
	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
	"gorm.io/gorm"
)

const (
	postStatusDraft     = "draft"
	postStatusPublished = "published"
	postStatusArchived  = "archived"
)

var (
	errPostNotFound   = errors.New("blog post not found")
	errPostSlugTaken  = errors.New("blog post slug already exists")
	postSlugPattern   = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	archivePattern    = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)
	reservedPostSlugs = map[string]struct{}{
		"archives": {},
		"tags":     {},
	}
	blogMarkdown = goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithRendererOptions(renderer.WithNodeRenderers(util.Prioritized(blogImageRenderer{}, 500))),
	)
	blogContentPolicy = newBlogContentPolicy()
)

type blogPost struct {
	ID          uint       `gorm:"primaryKey"`
	Title       string     `gorm:"size:160;not null"`
	Slug        string     `gorm:"size:160;uniqueIndex;not null"`
	Summary     string     `gorm:"type:text;not null"`
	Content     string     `gorm:"type:longtext;not null"`
	TagsJSON    string     `gorm:"column:tags;type:json;not null"`
	Status      string     `gorm:"size:16;not null"`
	PublishedAt *time.Time `gorm:"index"`
	CreatedAt   time.Time  `gorm:"not null"`
	UpdatedAt   time.Time  `gorm:"not null"`
}

type postRepository interface {
	List(context.Context) ([]blogPost, error)
	FindByID(context.Context, uint) (blogPost, error)
	FindBySlug(context.Context, string) (blogPost, error)
	Save(context.Context, blogPost) (blogPost, error)
	Delete(context.Context, uint) error
}

type gormPostRepository struct{ db *gorm.DB }

func (repository gormPostRepository) List(ctx context.Context) ([]blogPost, error) {
	var posts []blogPost
	if err := repository.db.WithContext(ctx).Find(&posts).Error; err != nil {
		return nil, fmt.Errorf("list blog posts: %w", err)
	}
	return posts, nil
}

func (repository gormPostRepository) FindByID(ctx context.Context, id uint) (blogPost, error) {
	var post blogPost
	if err := repository.db.WithContext(ctx).First(&post, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return blogPost{}, errPostNotFound
		}
		return blogPost{}, fmt.Errorf("find blog post: %w", err)
	}
	return post, nil
}

func (repository gormPostRepository) FindBySlug(ctx context.Context, slug string) (blogPost, error) {
	var post blogPost
	if err := repository.db.WithContext(ctx).Where("slug = ?", slug).First(&post).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return blogPost{}, errPostNotFound
		}
		return blogPost{}, fmt.Errorf("find blog post by slug: %w", err)
	}
	return post, nil
}

func (repository gormPostRepository) Save(ctx context.Context, post blogPost) (blogPost, error) {
	existing, err := repository.FindBySlug(ctx, post.Slug)
	if err == nil && existing.ID != post.ID {
		return blogPost{}, errPostSlugTaken
	}
	if err != nil && !errors.Is(err, errPostNotFound) {
		return blogPost{}, err
	}
	if post.ID == 0 {
		if err := repository.db.WithContext(ctx).Create(&post).Error; err != nil {
			return blogPost{}, fmt.Errorf("create blog post: %w", err)
		}
		return post, nil
	}
	if err := repository.db.WithContext(ctx).Save(&post).Error; err != nil {
		return blogPost{}, fmt.Errorf("save blog post: %w", err)
	}
	return post, nil
}

func (repository gormPostRepository) Delete(ctx context.Context, id uint) error {
	result := repository.db.WithContext(ctx).Delete(&blogPost{}, id)
	if result.Error != nil {
		return fmt.Errorf("delete blog post: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errPostNotFound
	}
	return nil
}

type postRequest struct {
	Title   string   `json:"title"`
	Slug    string   `json:"slug"`
	Summary string   `json:"summary"`
	Content string   `json:"content"`
	Tags    []string `json:"tags"`
	Status  string   `json:"status"`
}

type publicPostResponse struct {
	Title       string   `json:"title"`
	Slug        string   `json:"slug"`
	Summary     string   `json:"summary"`
	Tags        []string `json:"tags"`
	PublishedAt string   `json:"publishedAt"`
}

type publicPostDetailResponse struct {
	publicPostResponse
	Content string `json:"content"`
}

type adminPostResponse struct {
	ID uint `json:"id"`
	postRequest
	PublishedAt string `json:"publishedAt"`
}

type postTagResponse struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type archiveResponse struct {
	Key   string `json:"key"`
	Year  int    `json:"year"`
	Month int    `json:"month"`
	Count int    `json:"count"`
}

func postFromRequest(request postRequest) (blogPost, error) {
	request.Title = strings.TrimSpace(request.Title)
	request.Slug = strings.TrimSpace(request.Slug)
	request.Summary = strings.TrimSpace(request.Summary)
	request.Content = strings.TrimSpace(request.Content)
	request.Status = strings.TrimSpace(request.Status)

	if utf8.RuneCountInString(request.Title) == 0 || utf8.RuneCountInString(request.Title) > 160 {
		return blogPost{}, errors.New("title must contain between 1 and 160 characters")
	}
	if !postSlugPattern.MatchString(request.Slug) || len(request.Slug) > 160 {
		return blogPost{}, errors.New("slug must use lowercase letters, numbers, and single hyphens")
	}
	if _, reserved := reservedPostSlugs[request.Slug]; reserved {
		return blogPost{}, errors.New("slug is reserved")
	}
	if utf8.RuneCountInString(request.Summary) == 0 || utf8.RuneCountInString(request.Summary) > 600 {
		return blogPost{}, errors.New("summary must contain between 1 and 600 characters")
	}
	if utf8.RuneCountInString(request.Content) == 0 || utf8.RuneCountInString(request.Content) > 100000 {
		return blogPost{}, errors.New("content must contain between 1 and 100000 characters")
	}
	tags, err := normalizedPostTags(request.Tags)
	if err != nil {
		return blogPost{}, fmt.Errorf("tags: %w", err)
	}
	if request.Status != postStatusDraft && request.Status != postStatusPublished && request.Status != postStatusArchived {
		return blogPost{}, errors.New("status must be draft, published, or archived")
	}
	encodedTags, err := json.Marshal(tags)
	if err != nil {
		return blogPost{}, fmt.Errorf("encode tags: %w", err)
	}
	return blogPost{
		Title:    request.Title,
		Slug:     request.Slug,
		Summary:  request.Summary,
		Content:  request.Content,
		TagsJSON: string(encodedTags),
		Status:   request.Status,
	}, nil
}

func normalizedPostTags(tags []string) ([]string, error) {
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

func postTags(encoded string) ([]string, error) {
	var tags []string
	if err := json.Unmarshal([]byte(encoded), &tags); err != nil {
		return nil, errors.New("invalid stored post tags")
	}
	return tags, nil
}

func publicPost(post blogPost) (publicPostResponse, error) {
	if post.PublishedAt == nil {
		return publicPostResponse{}, errors.New("published post is missing publishedAt")
	}
	tags, err := postTags(post.TagsJSON)
	if err != nil {
		return publicPostResponse{}, err
	}
	return publicPostResponse{
		Title:       post.Title,
		Slug:        post.Slug,
		Summary:     post.Summary,
		Tags:        tags,
		PublishedAt: post.PublishedAt.UTC().Format(time.RFC3339),
	}, nil
}

func publicPostDetail(post blogPost) (publicPostDetailResponse, error) {
	summary, err := publicPost(post)
	if err != nil {
		return publicPostDetailResponse{}, err
	}
	content, err := renderPostMarkdown(post.Content)
	if err != nil {
		return publicPostDetailResponse{}, err
	}
	return publicPostDetailResponse{publicPostResponse: summary, Content: content}, nil
}

func adminPost(post blogPost) (adminPostResponse, error) {
	tags, err := postTags(post.TagsJSON)
	if err != nil {
		return adminPostResponse{}, err
	}
	publishedAt := ""
	if post.PublishedAt != nil {
		publishedAt = post.PublishedAt.UTC().Format(time.RFC3339)
	}
	return adminPostResponse{
		ID: post.ID,
		postRequest: postRequest{
			Title:   post.Title,
			Slug:    post.Slug,
			Summary: post.Summary,
			Content: post.Content,
			Tags:    tags,
			Status:  post.Status,
		},
		PublishedAt: publishedAt,
	}, nil
}

func sortedPosts(posts []blogPost) {
	sort.Slice(posts, func(left, right int) bool {
		if posts[left].PublishedAt != nil && posts[right].PublishedAt != nil && !posts[left].PublishedAt.Equal(*posts[right].PublishedAt) {
			return posts[left].PublishedAt.After(*posts[right].PublishedAt)
		}
		if posts[left].PublishedAt != nil {
			return true
		}
		if posts[right].PublishedAt != nil {
			return false
		}
		return posts[left].ID > posts[right].ID
	})
}

func postMatchesFilter(post blogPost, tag, archive string) (bool, error) {
	if tag != "" {
		tags, err := postTags(post.TagsJSON)
		if err != nil {
			return false, err
		}
		found := false
		for _, value := range tags {
			if strings.EqualFold(value, tag) {
				found = true
				break
			}
		}
		if !found {
			return false, nil
		}
	}
	if archive != "" {
		if post.PublishedAt == nil || post.PublishedAt.UTC().Format("2006-01") != archive {
			return false, nil
		}
	}
	return true, nil
}

func publishedPosts(posts []blogPost, tag, archive string) ([]blogPost, error) {
	published := make([]blogPost, 0, len(posts))
	for _, post := range posts {
		if post.Status != postStatusPublished {
			continue
		}
		matches, err := postMatchesFilter(post, tag, archive)
		if err != nil {
			return nil, err
		}
		if matches {
			published = append(published, post)
		}
	}
	sortedPosts(published)
	return published, nil
}

func (service appServices) publicPosts(c *fiber.Ctx) error {
	tag := strings.TrimSpace(c.Query("tag"))
	archive := strings.TrimSpace(c.Query("archive"))
	if archive != "" && !archivePattern.MatchString(archive) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "archive must use YYYY-MM"})
	}
	posts, err := service.posts.List(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "list blog posts"})
	}
	published, err := publishedPosts(posts, tag, archive)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "decode blog post tags"})
	}
	response := make([]publicPostResponse, 0, len(published))
	for _, post := range published {
		value, err := publicPost(post)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "decode blog posts"})
		}
		response = append(response, value)
	}
	return c.JSON(response)
}

func (service appServices) publicPost(c *fiber.Ctx) error {
	post, err := service.posts.FindBySlug(c.Context(), c.Params("slug"))
	if errors.Is(err, errPostNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "blog post not found"})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "find blog post"})
	}
	if post.Status != postStatusPublished {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "blog post not found"})
	}
	response, err := publicPostDetail(post)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "render blog post"})
	}
	return c.JSON(response)
}

func (service appServices) publicPostTags(c *fiber.Ctx) error {
	posts, err := service.posts.List(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "list blog posts"})
	}
	counts := make(map[string]int)
	names := make(map[string]string)
	for _, post := range posts {
		if post.Status != postStatusPublished {
			continue
		}
		tags, err := postTags(post.TagsJSON)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "decode blog post tags"})
		}
		for _, tag := range tags {
			key := strings.ToLower(tag)
			counts[key]++
			names[key] = tag
		}
	}
	response := make([]postTagResponse, 0, len(counts))
	for key, count := range counts {
		response = append(response, postTagResponse{Name: names[key], Count: count})
	}
	sort.Slice(response, func(left, right int) bool {
		return strings.ToLower(response[left].Name) < strings.ToLower(response[right].Name)
	})
	return c.JSON(response)
}

func (service appServices) publicPostArchives(c *fiber.Ctx) error {
	posts, err := service.posts.List(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "list blog posts"})
	}
	counts := make(map[string]int)
	for _, post := range posts {
		if post.Status == postStatusPublished && post.PublishedAt != nil {
			counts[post.PublishedAt.UTC().Format("2006-01")]++
		}
	}
	response := make([]archiveResponse, 0, len(counts))
	for key, count := range counts {
		year, _ := strconv.Atoi(key[:4])
		month, _ := strconv.Atoi(key[5:])
		response = append(response, archiveResponse{Key: key, Year: year, Month: month, Count: count})
	}
	sort.Slice(response, func(left, right int) bool {
		return response[left].Key > response[right].Key
	})
	return c.JSON(response)
}

func (service appServices) adminPosts(c *fiber.Ctx) error {
	posts, err := service.posts.List(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "list blog posts"})
	}
	sortedPosts(posts)
	response := make([]adminPostResponse, 0, len(posts))
	for _, post := range posts {
		value, err := adminPost(post)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "decode blog posts"})
		}
		response = append(response, value)
	}
	return c.JSON(response)
}

func (service appServices) createPost(c *fiber.Ctx) error {
	post, err := parsePostRequest(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if post.Status == postStatusPublished {
		now := time.Now().UTC()
		post.PublishedAt = &now
	}
	post, err = service.posts.Save(c.Context(), post)
	if errors.Is(err, errPostSlugTaken) {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "blog post slug already exists"})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "create blog post"})
	}
	response, err := adminPost(post)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "decode blog post"})
	}
	return c.Status(fiber.StatusCreated).JSON(response)
}

func (service appServices) updatePost(c *fiber.Ctx) error {
	id, err := parsePostID(c)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "blog post not found"})
	}
	post, err := parsePostRequest(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	existing, err := service.posts.FindByID(c.Context(), id)
	if errors.Is(err, errPostNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "blog post not found"})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "find blog post"})
	}
	post.ID = id
	post.CreatedAt = existing.CreatedAt
	post.PublishedAt = existing.PublishedAt
	if post.Status == postStatusPublished && post.PublishedAt == nil {
		now := time.Now().UTC()
		post.PublishedAt = &now
	}
	post, err = service.posts.Save(c.Context(), post)
	if errors.Is(err, errPostSlugTaken) {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "blog post slug already exists"})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "save blog post"})
	}
	response, err := adminPost(post)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "decode blog post"})
	}
	return c.JSON(response)
}

func (service appServices) deletePost(c *fiber.Ctx) error {
	id, err := parsePostID(c)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "blog post not found"})
	}
	if err := service.posts.Delete(c.Context(), id); errors.Is(err, errPostNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "blog post not found"})
	} else if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "delete blog post"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func parsePostRequest(c *fiber.Ctx) (blogPost, error) {
	var request postRequest
	if err := c.BodyParser(&request); err != nil {
		return blogPost{}, errors.New("blog post must be a JSON object")
	}
	return postFromRequest(request)
}

func parsePostID(c *fiber.Ctx) (uint, error) {
	id, err := strconv.ParseUint(c.Params("id"), 10, 0)
	if err != nil || id == 0 {
		return 0, errors.New("invalid blog post ID")
	}
	return uint(id), nil
}

func renderPostMarkdown(markdown string) (string, error) {
	var rendered bytes.Buffer
	if err := blogMarkdown.Convert([]byte(markdown), &rendered); err != nil {
		return "", fmt.Errorf("render markdown: %w", err)
	}
	return blogContentPolicy.Sanitize(rendered.String()), nil
}

func newBlogContentPolicy() *bluemonday.Policy {
	policy := bluemonday.NewPolicy()
	policy.AllowElements(
		"p", "br", "hr", "blockquote", "pre", "code",
		"h1", "h2", "h3", "h4", "h5", "h6",
		"strong", "em", "del",
		"ul", "ol", "li",
		"table", "thead", "tbody", "tr", "th", "td",
		"a", "img", "figure", "figcaption",
	)
	policy.AllowAttrs("href").OnElements("a")
	policy.AllowAttrs("src", "alt").OnElements("img")
	policy.AllowURLSchemes("https")
	policy.RequireNoFollowOnLinks(true)
	policy.RequireNoReferrerOnLinks(true)
	return policy
}

type blogImageRenderer struct{}

func (blogImageRenderer) RegisterFuncs(registerer renderer.NodeRendererFuncRegisterer) {
	registerer.Register(ast.KindImage, renderBlogImage)
}

func renderBlogImage(writer util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	image := node.(*ast.Image)
	_, _ = writer.WriteString(`<figure><img src="`)
	_, _ = writer.Write(util.EscapeHTML(util.URLEscape(image.Destination, true)))
	_, _ = writer.WriteString(`" alt="`)
	_, _ = writer.Write(util.EscapeHTML(image.Text(source)))
	_, _ = writer.WriteString(`">`)
	if len(image.Title) > 0 {
		_, _ = writer.WriteString("<figcaption>")
		_, _ = writer.Write(util.EscapeHTML(image.Title))
		_, _ = writer.WriteString("</figcaption>")
	}
	_, _ = writer.WriteString("</figure>")
	return ast.WalkSkipChildren, nil
}
