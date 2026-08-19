package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

type memoryAlbumItemRepository struct {
	nextID uint
	items  map[uint]albumItem
}

func albumMediaID(id uint) *uint { return &id }

func (repository *memoryAlbumItemRepository) List(context.Context) ([]albumItem, error) {
	items := make([]albumItem, 0, len(repository.items))
	for _, item := range repository.items {
		items = append(items, item)
	}
	return items, nil
}

func (repository *memoryAlbumItemRepository) FindByID(_ context.Context, id uint) (albumItem, error) {
	item, ok := repository.items[id]
	if !ok {
		return albumItem{}, errAlbumItemNotFound
	}
	return item, nil
}

func (repository *memoryAlbumItemRepository) Save(_ context.Context, item albumItem) (albumItem, error) {
	if item.ID == 0 {
		repository.nextID++
		item.ID = repository.nextID
	}
	repository.items[item.ID] = item
	return item, nil
}

func (repository *memoryAlbumItemRepository) Delete(_ context.Context, id uint) error {
	if _, ok := repository.items[id]; !ok {
		return errAlbumItemNotFound
	}
	delete(repository.items, id)
	return nil
}

func albumAdminRequest(method, path, body string) *http.Request {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	request.Header.Set(fiber.HeaderCookie, sessionCookieName+"=album-session")
	return request
}

func TestAdministratorCreatesDraftAlbumItemWithoutPublishingIt(t *testing.T) {
	repository := &memoryAlbumItemRepository{items: make(map[uint]albumItem)}
	image := mediaRecord{
		ID: 1, ObjectKey: "media/image/2026/08/album.jpg", Kind: mediaKindImage,
		MimeType: "image/jpeg", Size: 2048, OriginalName: "album.jpg", Width: 1200, Height: 800,
	}
	app := newApp(nil, appServices{
		sessions:     &memorySessionRepository{sessions: map[string]uint{"album-session": 1}},
		albumItems:   repository,
		mediaRecords: memoryMediaLookup{media: map[uint]mediaRecord{1: image}},
		media:        &mediaService{publicBaseURL: "https://cdn.example.com"},
		corsOrigin:   "https://ykagari.top",
	})

	createResponse, err := app.Test(albumAdminRequest(fiber.MethodPost, "/api/v1/admin/gallery-items", `{
		"title":"Violet Wake",
		"note":"afterimage / 00:14",
		"alt":"紫色光晕穿过深色轨道",
		"year":"2026",
		"imageMediaId":1,
		"anchorX":0.25,
		"anchorY":0.4,
		"width":"12vw",
		"aspectRatio":"3 / 2",
		"colors":["#100f18","#352157","#9f7aea"],
		"published":false,
		"sortOrder":2
	}`))
	if err != nil {
		t.Fatalf("create Album Item: %v", err)
	}
	defer createResponse.Body.Close()
	if createResponse.StatusCode != fiber.StatusCreated {
		t.Fatalf("create Album Item status = %d, want %d", createResponse.StatusCode, fiber.StatusCreated)
	}
	var created adminAlbumItemResponse
	if err := json.NewDecoder(createResponse.Body).Decode(&created); err != nil {
		t.Fatalf("decode created Album Item: %v", err)
	}
	if created.ID != 1 || created.Image.ID != image.ID || created.Published {
		t.Errorf("created Album Item = %#v, want draft backed by registered image", created)
	}

	publicResponse, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/gallery-items", nil))
	if err != nil {
		t.Fatalf("list public Album Items: %v", err)
	}
	defer publicResponse.Body.Close()
	var publicItems []publicAlbumItemResponse
	if err := json.NewDecoder(publicResponse.Body).Decode(&publicItems); err != nil {
		t.Fatalf("decode public Album Items: %v", err)
	}
	if len(publicItems) != 0 {
		t.Errorf("public draft Album Items = %#v, want none", publicItems)
	}

	adminResponse, err := app.Test(albumAdminRequest(fiber.MethodGet, "/api/v1/admin/gallery-items", ""))
	if err != nil {
		t.Fatalf("list administrator Album Items: %v", err)
	}
	defer adminResponse.Body.Close()
	var adminItems []adminAlbumItemResponse
	if err := json.NewDecoder(adminResponse.Body).Decode(&adminItems); err != nil {
		t.Fatalf("decode administrator Album Items: %v", err)
	}
	if len(adminItems) != 1 || adminItems[0].Title != "Violet Wake" {
		t.Errorf("administrator Album Items = %#v, want created draft", adminItems)
	}
}

func TestAdministratorPublishesSortsAndPermanentlyDeletesAlbumItems(t *testing.T) {
	image := mediaRecord{
		ID: 1, ObjectKey: "media/image/2026/08/album.jpg", Kind: mediaKindImage,
		MimeType: "image/jpeg", Size: 2048, OriginalName: "album.jpg", Width: 1200, Height: 800,
	}
	repository := &memoryAlbumItemRepository{
		nextID: 2,
		items: map[uint]albumItem{
			1: {ID: 1, Title: "Later", Note: "later", Alt: "later image", Year: "2025", ImageMediaID: albumMediaID(1), Image: image, AnchorX: 0.2, AnchorY: 0.2, Width: "10vw", AspectRatio: "1 / 1", ColorA: "#111111", ColorB: "#222222", ColorC: "#333333", Published: true, SortOrder: 5},
			2: {ID: 2, Title: "Draft", Note: "draft", Alt: "draft image", Year: "2026", ImageMediaID: albumMediaID(1), Image: image, AnchorX: 0.4, AnchorY: 0.4, Width: "12vw", AspectRatio: "3 / 2", ColorA: "#444444", ColorB: "#555555", ColorC: "#666666", Published: false, SortOrder: 9},
		},
	}
	app := newApp(nil, appServices{
		sessions:     &memorySessionRepository{sessions: map[string]uint{"album-session": 1}},
		albumItems:   repository,
		mediaRecords: memoryMediaLookup{media: map[uint]mediaRecord{1: image}},
		media:        &mediaService{publicBaseURL: "https://cdn.example.com"},
		corsOrigin:   "https://ykagari.top",
	})

	updateResponse, err := app.Test(albumAdminRequest(fiber.MethodPut, "/api/v1/admin/gallery-items/2", `{
		"title":"First Signal","note":"published","alt":"first image","year":"2026",
		"imageMediaId":1,"anchorX":0.4,"anchorY":0.4,"width":"12vw","aspectRatio":"3 / 2",
		"colors":["#444444","#555555","#666666"],"published":true,"sortOrder":0
	}`))
	if err != nil {
		t.Fatalf("publish Album Item: %v", err)
	}
	updateResponse.Body.Close()
	if updateResponse.StatusCode != fiber.StatusOK {
		t.Fatalf("publish Album Item status = %d, want %d", updateResponse.StatusCode, fiber.StatusOK)
	}

	publicResponse, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/gallery-items", nil))
	if err != nil {
		t.Fatalf("list published Album Items: %v", err)
	}
	defer publicResponse.Body.Close()
	var publicItems []publicAlbumItemResponse
	if err := json.NewDecoder(publicResponse.Body).Decode(&publicItems); err != nil {
		t.Fatalf("decode published Album Items: %v", err)
	}
	if len(publicItems) != 2 || publicItems[0].ID != "A-02" || publicItems[1].ID != "A-01" {
		t.Errorf("published Album Items = %#v, want configured sort order", publicItems)
	}
	if publicItems[0].ImageURL != "https://cdn.example.com/media/image/2026/08/album.jpg" {
		t.Errorf("published image URL = %q, want managed media URL", publicItems[0].ImageURL)
	}

	deleteResponse, err := app.Test(albumAdminRequest(fiber.MethodDelete, "/api/v1/admin/gallery-items/2", ""))
	if err != nil {
		t.Fatalf("delete Album Item: %v", err)
	}
	deleteResponse.Body.Close()
	if deleteResponse.StatusCode != fiber.StatusNoContent {
		t.Fatalf("delete Album Item status = %d, want %d", deleteResponse.StatusCode, fiber.StatusNoContent)
	}
	if _, ok := repository.items[2]; ok {
		t.Error("deleted Album Item remains in repository")
	}
}

func TestAlbumItemRejectsNonImageMedia(t *testing.T) {
	audio := mediaRecord{ID: 2, Kind: mediaKindAudio, DurationMs: 42000}
	app := newApp(nil, appServices{
		sessions:     &memorySessionRepository{sessions: map[string]uint{"album-session": 1}},
		albumItems:   &memoryAlbumItemRepository{items: make(map[uint]albumItem)},
		mediaRecords: memoryMediaLookup{media: map[uint]mediaRecord{2: audio}},
		media:        &mediaService{publicBaseURL: "https://cdn.example.com"},
		corsOrigin:   "https://ykagari.top",
	})
	response, err := app.Test(albumAdminRequest(fiber.MethodPost, "/api/v1/admin/gallery-items", `{
		"title":"Audio","note":"not an image","alt":"invalid","year":"2026","imageMediaId":2,
		"anchorX":0.5,"anchorY":0.5,"width":"12vw","aspectRatio":"1 / 1",
		"colors":["#111111","#222222","#333333"],"published":false,"sortOrder":0
	}`))
	if err != nil {
		t.Fatalf("create Album Item with audio: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("create Album Item with audio status = %d, want %d", response.StatusCode, fiber.StatusBadRequest)
	}
}

func TestAlbumItemRejectsNonPositiveCanvasDimensions(t *testing.T) {
	image := mediaRecord{ID: 1, Kind: mediaKindImage, Width: 100, Height: 100}
	app := newApp(nil, appServices{
		sessions:     &memorySessionRepository{sessions: map[string]uint{"album-session": 1}},
		albumItems:   &memoryAlbumItemRepository{items: make(map[uint]albumItem)},
		mediaRecords: memoryMediaLookup{media: map[uint]mediaRecord{1: image}},
		media:        &mediaService{publicBaseURL: "https://cdn.example.com"},
		corsOrigin:   "https://ykagari.top",
	})

	for _, test := range []struct {
		name        string
		width       string
		aspectRatio string
	}{
		{name: "zero width", width: "0vw", aspectRatio: "4 / 5"},
		{name: "zero numerator", width: "12vw", aspectRatio: "0 / 5"},
		{name: "zero denominator", width: "12vw", aspectRatio: "4 / 0"},
	} {
		t.Run(test.name, func(t *testing.T) {
			body := fmt.Sprintf(`{
				"title":"Invalid","note":"invalid dimensions","alt":"invalid","year":"2026","imageMediaId":1,
				"anchorX":0.5,"anchorY":0.5,"width":%q,"aspectRatio":%q,
				"colors":["#111111","#222222","#333333"],"published":false,"sortOrder":0
			}`, test.width, test.aspectRatio)
			response, err := app.Test(albumAdminRequest(fiber.MethodPost, "/api/v1/admin/gallery-items", body))
			if err != nil {
				t.Fatalf("create Album Item: %v", err)
			}
			response.Body.Close()
			if response.StatusCode != fiber.StatusBadRequest {
				t.Errorf("status = %d, want %d", response.StatusCode, fiber.StatusBadRequest)
			}
		})
	}
}

func TestAlbumItemInitialMigrationPreservesSeededGallery(t *testing.T) {
	records := seededAlbumItemRecords()
	if len(records) != len(seededGalleryItems) || len(records) == 0 {
		t.Fatalf("seeded records = %d, want %d", len(records), len(seededGalleryItems))
	}
	for index, record := range records {
		if !record.Published || record.SortOrder != index || record.ImageMediaID != nil {
			t.Errorf("seeded record %d = %#v, want published media-free migration record", index, record)
		}
	}
	if albumSeedMigrationName == "" {
		t.Fatal("Album Item seed migration must have a persistent version name")
	}
}

func TestAlbumItemSeedMigrationRetriesFailureAndRunsOnce(t *testing.T) {
	completed := false
	seedAttempts := 0
	markAttempts := 0
	run := func() error {
		return applyAlbumSeedMigration(completed, 0, func([]albumItem) error {
			seedAttempts++
			if seedAttempts == 1 {
				return errors.New("temporary database failure")
			}
			return nil
		}, func() error {
			markAttempts++
			completed = true
			return nil
		})
	}

	if err := run(); err == nil {
		t.Fatal("first migration attempt succeeded, want temporary failure")
	}
	if err := run(); err != nil {
		t.Fatalf("retry migration: %v", err)
	}
	if err := run(); err != nil {
		t.Fatalf("repeat completed migration: %v", err)
	}
	if seedAttempts != 2 || markAttempts != 1 {
		t.Fatalf("seed attempts = %d, marks = %d; want 2 retries and 1 completion", seedAttempts, markAttempts)
	}
}

var _ albumItemRepository = (*memoryAlbumItemRepository)(nil)
