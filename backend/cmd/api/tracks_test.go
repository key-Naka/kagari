package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

type memoryTrackRepository struct {
	nextID uint
	tracks map[uint]track
}

type memoryMediaLookup struct {
	media map[uint]mediaRecord
	err   error
}

func (lookup memoryMediaLookup) FindByID(_ context.Context, id uint) (mediaRecord, error) {
	if lookup.err != nil {
		return mediaRecord{}, lookup.err
	}
	value, ok := lookup.media[id]
	if !ok {
		return mediaRecord{}, errMediaObjectNotFound
	}
	return value, nil
}

func TestCreateTrackReportsMediaLookupFailureAsServerError(t *testing.T) {
	app := newApp(nil, appServices{
		sessions:     &memorySessionRepository{sessions: map[string]uint{"test-session": 1}},
		tracks:       &memoryTrackRepository{tracks: make(map[uint]track)},
		mediaRecords: memoryMediaLookup{err: errors.New("database unavailable")},
		media:        &mediaService{publicBaseURL: "https://cdn.example.com"},
		corsOrigin:   "https://ykagari.top",
	})
	request := httptest.NewRequest(fiber.MethodPost, "/api/v1/admin/tracks", strings.NewReader(`{"title":"Night Archive","coverMediaId":1,"audioMediaId":2,"enabled":true,"sortOrder":0}`))
	request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	request.Header.Set(fiber.HeaderCookie, sessionCookieName+"=test-session")

	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("create track during media lookup failure: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusInternalServerError)
	}
}

func (repository *memoryTrackRepository) List(context.Context) ([]track, error) {
	tracks := make([]track, 0, len(repository.tracks))
	for _, value := range repository.tracks {
		tracks = append(tracks, value)
	}
	return tracks, nil
}

func TestAdministratorManagesTrackAvailabilityAndOrder(t *testing.T) {
	repository := &memoryTrackRepository{tracks: make(map[uint]track)}
	media := map[uint]mediaRecord{
		1: {ID: 1, ObjectKey: "media/image/2026/08/cover-a.webp", Kind: mediaKindImage, MimeType: "image/webp", Size: 1024, OriginalName: "cover-a.webp", Width: 1200, Height: 1200},
		2: {ID: 2, ObjectKey: "media/audio/2026/08/track-a.mp3", Kind: mediaKindAudio, MimeType: "audio/mpeg", Size: 4096, OriginalName: "track-a.mp3", DurationMs: 180000},
		3: {ID: 3, ObjectKey: "media/image/2026/08/cover-b.webp", Kind: mediaKindImage, MimeType: "image/webp", Size: 2048, OriginalName: "cover-b.webp", Width: 1200, Height: 1200},
		4: {ID: 4, ObjectKey: "media/audio/2026/08/track-b.ogg", Kind: mediaKindAudio, MimeType: "audio/ogg", Size: 8192, OriginalName: "track-b.ogg", DurationMs: 95000},
	}
	app := newApp(nil, appServices{
		sessions:     &memorySessionRepository{sessions: map[string]uint{"test-session": 1}},
		tracks:       repository,
		mediaRecords: memoryMediaLookup{media: media},
		media:        &mediaService{publicBaseURL: "https://cdn.example.com"},
		ttl:          60,
		corsOrigin:   "https://ykagari.top",
	})

	request := httptest.NewRequest(fiber.MethodPost, "/api/v1/admin/tracks", strings.NewReader(`{}`))
	request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	unauthenticated, err := app.Test(request)
	if err != nil {
		t.Fatalf("create track without a session: %v", err)
	}
	unauthenticated.Body.Close()
	if unauthenticated.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d, want %d", unauthenticated.StatusCode, fiber.StatusUnauthorized)
	}

	adminRequest := func(method, path, body string) *http.Request {
		request := httptest.NewRequest(method, path, strings.NewReader(body))
		request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
		request.Header.Set(fiber.HeaderCookie, sessionCookieName+"=test-session")
		return request
	}

	invalid := `{"title":"Wrong media","coverMediaId":2,"audioMediaId":1,"enabled":true,"sortOrder":0}`
	invalidResponse, err := app.Test(adminRequest(fiber.MethodPost, "/api/v1/admin/tracks", invalid))
	if err != nil {
		t.Fatalf("create track with swapped media: %v", err)
	}
	invalidResponse.Body.Close()
	if invalidResponse.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("swapped media status = %d, want %d", invalidResponse.StatusCode, fiber.StatusBadRequest)
	}

	disabled := `{"title":"Night Archive","coverMediaId":1,"audioMediaId":2,"enabled":false,"sortOrder":20}`
	created, err := app.Test(adminRequest(fiber.MethodPost, "/api/v1/admin/tracks", disabled))
	if err != nil {
		t.Fatalf("create disabled track: %v", err)
	}
	created.Body.Close()
	if created.StatusCode != fiber.StatusCreated {
		t.Fatalf("create status = %d, want %d", created.StatusCode, fiber.StatusCreated)
	}

	publicBeforeEnable, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/tracks", nil))
	if err != nil {
		t.Fatalf("list tracks before enable: %v", err)
	}
	var hidden []publicTrackResponse
	if err := json.NewDecoder(publicBeforeEnable.Body).Decode(&hidden); err != nil {
		t.Fatalf("decode hidden tracks: %v", err)
	}
	publicBeforeEnable.Body.Close()
	if len(hidden) != 0 {
		t.Fatalf("public tracks = %#v, want disabled track hidden", hidden)
	}

	enabled := `{"title":"Night Archive","coverMediaId":1,"audioMediaId":2,"enabled":true,"sortOrder":20}`
	updated, err := app.Test(adminRequest(fiber.MethodPut, "/api/v1/admin/tracks/1", enabled))
	if err != nil {
		t.Fatalf("enable track: %v", err)
	}
	updated.Body.Close()
	if updated.StatusCode != fiber.StatusOK {
		t.Fatalf("update status = %d, want %d", updated.StatusCode, fiber.StatusOK)
	}

	second := `{"title":"First Light","coverMediaId":3,"audioMediaId":4,"enabled":true,"sortOrder":5}`
	secondResponse, err := app.Test(adminRequest(fiber.MethodPost, "/api/v1/admin/tracks", second))
	if err != nil {
		t.Fatalf("create second track: %v", err)
	}
	secondResponse.Body.Close()
	if secondResponse.StatusCode != fiber.StatusCreated {
		t.Fatalf("second create status = %d, want %d", secondResponse.StatusCode, fiber.StatusCreated)
	}

	adminList, err := app.Test(adminRequest(fiber.MethodGet, "/api/v1/admin/tracks", ""))
	if err != nil {
		t.Fatalf("list administrator tracks: %v", err)
	}
	defer adminList.Body.Close()
	var tracks []adminTrackResponse
	if err := json.NewDecoder(adminList.Body).Decode(&tracks); err != nil {
		t.Fatalf("decode administrator tracks: %v", err)
	}
	if len(tracks) != 2 || tracks[0].Title != "First Light" || tracks[1].Audio.DurationMs != 180000 || !tracks[1].Enabled {
		t.Fatalf("administrator tracks = %#v, want sorted tracks with persisted duration and enabled state", tracks)
	}
}

func (repository *memoryTrackRepository) FindByID(_ context.Context, id uint) (track, error) {
	value, ok := repository.tracks[id]
	if !ok {
		return track{}, errTrackNotFound
	}
	return value, nil
}

func (repository *memoryTrackRepository) Save(_ context.Context, value track) (track, error) {
	if value.ID == 0 {
		repository.nextID++
		value.ID = repository.nextID
	}
	repository.tracks[value.ID] = value
	return value, nil
}

func TestPublicTracksReturnsOnlyEnabledTracksInSortOrder(t *testing.T) {
	repository := &memoryTrackRepository{tracks: map[uint]track{
		1: {
			ID: 1, Title: "Second Signal", Enabled: true, SortOrder: 20,
			Cover: mediaRecord{ID: 11, ObjectKey: "media/image/2026/08/cover-b.webp", Kind: mediaKindImage, MimeType: "image/webp", Size: 2048, OriginalName: "cover-b.webp", Width: 1200, Height: 1200},
			Audio: mediaRecord{ID: 12, ObjectKey: "media/audio/2026/08/track-b.mp3", Kind: mediaKindAudio, MimeType: "audio/mpeg", Size: 4096, OriginalName: "track-b.mp3", DurationMs: 181000},
		},
		2: {
			ID: 2, Title: "Hidden Signal", Enabled: false, SortOrder: 0,
			Cover: mediaRecord{ID: 21, ObjectKey: "media/image/2026/08/cover-hidden.webp", Kind: mediaKindImage},
			Audio: mediaRecord{ID: 22, ObjectKey: "media/audio/2026/08/track-hidden.mp3", Kind: mediaKindAudio},
		},
		3: {
			ID: 3, Title: "First Signal", Enabled: true, SortOrder: 10,
			Cover: mediaRecord{ID: 31, ObjectKey: "media/image/2026/08/cover-a.webp", Kind: mediaKindImage, MimeType: "image/webp", Size: 1024, OriginalName: "cover-a.webp", Width: 1200, Height: 1200},
			Audio: mediaRecord{ID: 32, ObjectKey: "media/audio/2026/08/track-a.ogg", Kind: mediaKindAudio, MimeType: "audio/ogg", Size: 8192, OriginalName: "track-a.ogg", DurationMs: 95000},
		},
	}}
	app := newApp(nil, appServices{
		tracks:     repository,
		media:      &mediaService{publicBaseURL: "https://cdn.example.com"},
		corsOrigin: "https://ykagari.top",
	})

	response, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/tracks", nil))
	if err != nil {
		t.Fatalf("list public tracks: %v", err)
	}
	defer response.Body.Close()

	var tracks []publicTrackResponse
	if err := json.NewDecoder(response.Body).Decode(&tracks); err != nil {
		t.Fatalf("decode public tracks: %v", err)
	}
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusOK)
	}
	if len(tracks) != 2 || tracks[0].Title != "First Signal" || tracks[1].Title != "Second Signal" {
		t.Fatalf("tracks = %#v, want enabled tracks ordered by sortOrder", tracks)
	}
	if tracks[0].Audio.DurationMs != 95000 || tracks[0].Audio.PublicURL != "https://cdn.example.com/media/audio/2026/08/track-a.ogg" {
		t.Errorf("audio = %#v, want persisted duration and public URL", tracks[0].Audio)
	}
}
