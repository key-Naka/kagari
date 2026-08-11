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

	"github.com/gofiber/fiber/v2"
)

type memoryPostRepository struct {
	nextID uint
	posts  map[uint]blogPost
}

func (repository *memoryPostRepository) List(context.Context) ([]blogPost, error) {
	posts := make([]blogPost, 0, len(repository.posts))
	for _, post := range repository.posts {
		posts = append(posts, post)
	}
	return posts, nil
}

func (repository *memoryPostRepository) FindByID(_ context.Context, id uint) (blogPost, error) {
	post, ok := repository.posts[id]
	if !ok {
		return blogPost{}, errPostNotFound
	}
	return post, nil
}

func (repository *memoryPostRepository) FindBySlug(_ context.Context, slug string) (blogPost, error) {
	for _, post := range repository.posts {
		if post.Slug == slug {
			return post, nil
		}
	}
	return blogPost{}, errPostNotFound
}

func (repository *memoryPostRepository) Save(_ context.Context, post blogPost) (blogPost, error) {
	for id, existing := range repository.posts {
		if id != post.ID && existing.Slug == post.Slug {
			return blogPost{}, errPostSlugTaken
		}
	}
	if post.ID == 0 {
		repository.nextID++
		post.ID = repository.nextID
	}
	repository.posts[post.ID] = post
	return post, nil
}

func (repository *memoryPostRepository) Delete(_ context.Context, id uint) error {
	if _, ok := repository.posts[id]; !ok {
		return errPostNotFound
	}
	delete(repository.posts, id)
	return nil
}

func TestBlogPostsAreManagedPrivatelyAndPublishedPublicly(t *testing.T) {
	app, _, _ := newTestApp(t)

	adminRequest := func(method, path, body string) *http.Request {
		request := httptest.NewRequest(method, path, strings.NewReader(body))
		request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
		request.Header.Set(fiber.HeaderCookie, sessionCookieName+"=test-session")
		return request
	}

	t.Run("unauthenticated changes are rejected", func(t *testing.T) {
		response, err := app.Test(httptest.NewRequest(fiber.MethodPost, "/api/v1/admin/posts", strings.NewReader(`{}`)))
		if err != nil {
			t.Fatalf("create post without a session: %v", err)
		}
		defer response.Body.Close()
		if response.StatusCode != fiber.StatusUnauthorized {
			t.Errorf("status = %d, want %d", response.StatusCode, fiber.StatusUnauthorized)
		}
	})

	loginRequest := httptest.NewRequest(fiber.MethodPost, "/api/v1/admin/session", strings.NewReader(`{"username":"admin","password":"correct horse battery staple"}`))
	loginRequest.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	loginResponse, err := app.Test(loginRequest)
	if err != nil {
		t.Fatalf("login administrator: %v", err)
	}
	loginResponse.Body.Close()
	if loginResponse.StatusCode != fiber.StatusNoContent {
		t.Fatalf("login status = %d, want %d", loginResponse.StatusCode, fiber.StatusNoContent)
	}

	draft := `{"title":"Archive Signals","slug":"archive-signals","summary":"A short public summary.","content":"# Archive Signals\n\n**Published** writing.\n\n![Diagram](https://cdn.example.com/diagram.webp \"System diagram\")\n\n[Reference](https://example.com)\n\n[Unsafe](javascript:alert('xss'))\n\n<script>alert('xss')</script>","tags":["Go","Vue"],"status":"draft"}`
	reservedSlugResponse, err := app.Test(adminRequest(fiber.MethodPost, "/api/v1/admin/posts", strings.Replace(draft, `"archive-signals"`, `"tags"`, 1)))
	if err != nil {
		t.Fatalf("create post with a reserved slug: %v", err)
	}
	reservedSlugResponse.Body.Close()
	if reservedSlugResponse.StatusCode != fiber.StatusBadRequest {
		t.Errorf("reserved slug status = %d, want %d", reservedSlugResponse.StatusCode, fiber.StatusBadRequest)
	}

	createResponse, err := app.Test(adminRequest(fiber.MethodPost, "/api/v1/admin/posts", draft))
	if err != nil {
		t.Fatalf("create draft post: %v", err)
	}
	defer createResponse.Body.Close()
	if createResponse.StatusCode != fiber.StatusCreated {
		t.Fatalf("create status = %d, want %d", createResponse.StatusCode, fiber.StatusCreated)
	}

	listResponse, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/posts", nil))
	if err != nil {
		t.Fatalf("list drafts publicly: %v", err)
	}
	defer listResponse.Body.Close()
	var drafts []publicPostResponse
	if err := json.NewDecoder(listResponse.Body).Decode(&drafts); err != nil {
		t.Fatalf("decode public draft list: %v", err)
	}
	if listResponse.StatusCode != fiber.StatusOK || len(drafts) != 0 {
		t.Errorf("public draft list = %#v with status %d, want no posts", drafts, listResponse.StatusCode)
	}

	published := strings.Replace(draft, `"status":"draft"`, `"status":"published"`, 1)
	updateResponse, err := app.Test(adminRequest(fiber.MethodPut, "/api/v1/admin/posts/1", published))
	if err != nil {
		t.Fatalf("publish post: %v", err)
	}
	updateResponse.Body.Close()
	if updateResponse.StatusCode != fiber.StatusOK {
		t.Fatalf("publish status = %d, want %d", updateResponse.StatusCode, fiber.StatusOK)
	}

	detailResponse, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/posts/archive-signals", nil))
	if err != nil {
		t.Fatalf("get public post: %v", err)
	}
	defer detailResponse.Body.Close()
	body, err := io.ReadAll(detailResponse.Body)
	if err != nil {
		t.Fatalf("read public post: %v", err)
	}
	var detail publicPostDetailResponse
	if err := json.Unmarshal(body, &detail); err != nil {
		t.Fatalf("decode public post: %v", err)
	}
	if detailResponse.StatusCode != fiber.StatusOK || detail.Title != "Archive Signals" || detail.PublishedAt == "" || !strings.Contains(detail.Content, "<figure>") || !strings.Contains(detail.Content, "<figcaption>System diagram</figcaption>") || !strings.Contains(detail.Content, "<strong>Published</strong>") {
		t.Errorf("public detail = %#v with status %d, want rendered published post", detail, detailResponse.StatusCode)
	}
	for _, unsafe := range []string{"<script", "alert('xss')", "javascript:"} {
		if strings.Contains(strings.ToLower(string(body)), unsafe) {
			t.Errorf("public content contains unsafe value %q: %s", unsafe, body)
		}
	}

	archive := detail.PublishedAt[:7]
	filteredResponse, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/posts?tag=vue&archive="+archive, nil))
	if err != nil {
		t.Fatalf("filter public posts: %v", err)
	}
	defer filteredResponse.Body.Close()
	var filtered []publicPostResponse
	if err := json.NewDecoder(filteredResponse.Body).Decode(&filtered); err != nil {
		t.Fatalf("decode filtered posts: %v", err)
	}
	if filteredResponse.StatusCode != fiber.StatusOK || len(filtered) != 1 || filtered[0].Slug != "archive-signals" || strings.Contains(string(body), `"status"`) {
		t.Errorf("filtered posts = %#v with status %d, want one sanitized published post", filtered, filteredResponse.StatusCode)
	}

	tagsResponse, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/posts/tags", nil))
	if err != nil {
		t.Fatalf("list public tags: %v", err)
	}
	defer tagsResponse.Body.Close()
	var tags []postTagResponse
	if err := json.NewDecoder(tagsResponse.Body).Decode(&tags); err != nil {
		t.Fatalf("decode public tags: %v", err)
	}
	if len(tags) != 2 || tags[0].Name != "Go" || tags[1].Name != "Vue" || tags[0].Count != 1 {
		t.Errorf("tags = %#v, want published tag counts", tags)
	}

	archivesResponse, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/posts/archives", nil))
	if err != nil {
		t.Fatalf("list public archives: %v", err)
	}
	defer archivesResponse.Body.Close()
	var archives []archiveResponse
	if err := json.NewDecoder(archivesResponse.Body).Decode(&archives); err != nil {
		t.Fatalf("decode public archives: %v", err)
	}
	if len(archives) != 1 || archives[0].Key != archive || archives[0].Count != 1 {
		t.Errorf("archives = %#v, want one published archive", archives)
	}

	archived := strings.Replace(published, `"status":"published"`, `"status":"archived"`, 1)
	archiveResponse, err := app.Test(adminRequest(fiber.MethodPut, "/api/v1/admin/posts/1", archived))
	if err != nil {
		t.Fatalf("archive post: %v", err)
	}
	archiveResponse.Body.Close()
	if archiveResponse.StatusCode != fiber.StatusOK {
		t.Fatalf("archive status = %d, want %d", archiveResponse.StatusCode, fiber.StatusOK)
	}

	hiddenResponse, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/posts/archive-signals", nil))
	if err != nil {
		t.Fatalf("get archived post: %v", err)
	}
	hiddenResponse.Body.Close()
	if hiddenResponse.StatusCode != fiber.StatusNotFound {
		t.Errorf("archived post status = %d, want %d", hiddenResponse.StatusCode, fiber.StatusNotFound)
	}

	deleteResponse, err := app.Test(adminRequest(fiber.MethodDelete, "/api/v1/admin/posts/1", ""))
	if err != nil {
		t.Fatalf("delete post: %v", err)
	}
	deleteResponse.Body.Close()
	if deleteResponse.StatusCode != fiber.StatusNoContent {
		t.Fatalf("delete status = %d, want %d", deleteResponse.StatusCode, fiber.StatusNoContent)
	}

	adminListResponse, err := app.Test(adminRequest(fiber.MethodGet, "/api/v1/admin/posts", ""))
	if err != nil {
		t.Fatalf("list posts after delete: %v", err)
	}
	defer adminListResponse.Body.Close()
	var remaining []adminPostResponse
	if err := json.NewDecoder(adminListResponse.Body).Decode(&remaining); err != nil {
		t.Fatalf("decode administrator posts: %v", err)
	}
	if adminListResponse.StatusCode != fiber.StatusOK || len(remaining) != 0 {
		t.Errorf("administrator posts = %#v with status %d, want no remaining posts", remaining, adminListResponse.StatusCode)
	}
}

func TestPublicPostReturnsServerErrorOnRepositoryFailure(t *testing.T) {
	app := newApp(nil, appServices{
		posts:      failingPostRepository{err: errors.New("database unavailable")},
		corsOrigin: "https://ykagari.top",
	})

	response, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/posts/archive-signals", nil))
	if err != nil {
		t.Fatalf("get public post when storage fails: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusInternalServerError {
		t.Errorf("status = %d, want %d", response.StatusCode, fiber.StatusInternalServerError)
	}
}

type failingPostRepository struct{ err error }

func (repository failingPostRepository) List(context.Context) ([]blogPost, error) {
	return nil, repository.err
}

func (repository failingPostRepository) FindByID(context.Context, uint) (blogPost, error) {
	return blogPost{}, repository.err
}

func (repository failingPostRepository) FindBySlug(context.Context, string) (blogPost, error) {
	return blogPost{}, repository.err
}

func (repository failingPostRepository) Save(context.Context, blogPost) (blogPost, error) {
	return blogPost{}, repository.err
}

func (repository failingPostRepository) Delete(context.Context, uint) error {
	return repository.err
}
