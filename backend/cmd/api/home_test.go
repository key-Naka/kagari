package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

func TestHomeArchiveSummarizesPublishedPublicModules(t *testing.T) {
	publishedAt := time.Date(2026, time.August, 19, 8, 0, 0, 0, time.UTC)
	services := appServices{
		projects: &memoryProjectRepository{projects: map[uint]portfolioProject{
			1: {
				ID: 1, Title: "Draft", Status: projectStatusDraft,
			},
			2: {
				ID: 2, Title: "Kagari Core", CoverURL: "https://cdn.example.com/kagari.webp",
				Description: "A public archive system.", TechnologiesJSON: "[]", TypesJSON: "[]",
				Featured: true, Status: projectStatusPublished,
			},
		}},
		posts: &memoryPostRepository{posts: map[uint]blogPost{
			1: {
				ID: 1, Title: "Night Index", Summary: "A boundary note.", Status: postStatusPublished,
				TagsJSON: "[]", PublishedAt: &publishedAt,
			},
		}},
		tracks: &memoryTrackRepository{tracks: map[uint]track{
			1: {
				ID: 1, Title: "Hidden Track", Enabled: false,
			},
			2: {
				ID: 2, Title: "Ash Choir", Enabled: true,
				Cover: mediaRecord{ObjectKey: "media/image/2026/08/ash.webp"},
			},
		}},
		visitorMessages: &memoryVisitorMessageRepository{messages: map[uint]visitorMessage{
			1: {
				ID: 1, Nickname: "Aya", Email: "private@example.com", Content: "Hello from the edge",
				CreatedAt: publishedAt,
			},
		}},
		media:      &mediaService{publicBaseURL: "https://cdn.example.com"},
		corsOrigin: "https://ykagari.top",
	}

	response, err := newApp(nil, services).Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/home", nil))
	if err != nil {
		t.Fatalf("get home archive: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusOK)
	}

	encoded, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read home archive: %v", err)
	}
	if strings.Contains(string(encoded), "private@example.com") {
		t.Fatal("home archive leaked a visitor email")
	}
	var archive homeArchiveResponse
	if err := json.Unmarshal(encoded, &archive); err != nil {
		t.Fatalf("decode home archive: %v", err)
	}
	if archive.Works.Count != 1 || archive.Works.Item == nil || archive.Works.Item.Title != "Kagari Core" {
		t.Fatalf("works = %#v, want one published project", archive.Works)
	}
	if archive.Blog.Count != 1 || archive.Blog.Item == nil || archive.Blog.Item.Title != "Night Index" {
		t.Fatalf("blog = %#v, want latest published post", archive.Blog)
	}
	if archive.Music.Count != 1 || archive.Music.Item == nil || archive.Music.Item.Title != "Ash Choir" {
		t.Fatalf("music = %#v, want one enabled track", archive.Music)
	}
	if archive.Music.Item.CoverURL != "https://cdn.example.com/media/image/2026/08/ash.webp" {
		t.Fatalf("music cover URL = %q", archive.Music.Item.CoverURL)
	}
	if archive.Gallery.Availability != availabilityOperational || archive.Gallery.Count != len(seededGalleryItems) {
		t.Fatalf("gallery = %#v, want seeded public Album Item count", archive.Gallery)
	}
	if archive.VisitorMessages.Count != 1 || archive.VisitorMessages.Item == nil {
		t.Fatalf("visitor messages = %#v, want latest public message", archive.VisitorMessages)
	}
}

func TestHomeArchiveDegradesOnlyTheFailingModule(t *testing.T) {
	services := appServices{
		projects: &memoryProjectRepository{projects: map[uint]portfolioProject{
			1: {
				ID: 1, Title: "Still Available", TechnologiesJSON: "[]", TypesJSON: "[]",
				Status: projectStatusPublished,
			},
		}},
		posts:      failingPostRepository{err: errors.New("database unavailable")},
		corsOrigin: "https://ykagari.top",
	}

	response, err := newApp(nil, services).Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/home", nil))
	if err != nil {
		t.Fatalf("get partially available home archive: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusOK)
	}

	var archive homeArchiveResponse
	if err := json.NewDecoder(response.Body).Decode(&archive); err != nil {
		t.Fatalf("decode home archive: %v", err)
	}
	if archive.Works.Availability != availabilityOperational || archive.Works.Count != 1 {
		t.Fatalf("works = %#v, want operational summary", archive.Works)
	}
	if archive.Blog.Availability != availabilityUnavailable || archive.Blog.Item != nil {
		t.Fatalf("blog = %#v, want isolated unavailable summary", archive.Blog)
	}
	if archive.Status.Availability == availabilityUnavailable {
		t.Fatalf("status = %#v, want independent status summary", archive.Status)
	}
}

func TestGalleryItemsExposeTheSeededPublicArchive(t *testing.T) {
	response, err := newApp(nil, appServices{corsOrigin: "https://ykagari.top"}).Test(
		httptest.NewRequest(fiber.MethodGet, "/api/v1/gallery-items", nil),
	)
	if err != nil {
		t.Fatalf("get gallery items: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusOK)
	}
	var items []map[string]any
	if err := json.NewDecoder(response.Body).Decode(&items); err != nil {
		t.Fatalf("decode gallery items: %v", err)
	}
	if len(items) != 12 {
		t.Fatalf("gallery item count = %d, want 12", len(items))
	}
	if items[0]["id"] != "A-01" || items[0]["title"] != "Violet Wake" {
		t.Fatalf("first gallery item = %#v", items[0])
	}
}
