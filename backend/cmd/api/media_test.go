package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

type memoryMediaRepository struct {
	nextID uint
	media  map[string]mediaRecord
}

func (repository *memoryMediaRepository) Save(_ context.Context, record mediaRecord) (mediaRecord, error) {
	if _, exists := repository.media[record.ObjectKey]; exists {
		return mediaRecord{}, errMediaObjectKeyTaken
	}
	repository.nextID++
	record.ID = repository.nextID
	repository.media[record.ObjectKey] = record
	return record, nil
}

type memoryMediaObjectInspector struct {
	objects map[string]mediaObjectDetails
	err     error
}

func (inspector memoryMediaObjectInspector) Stat(_ context.Context, objectKey string) (mediaObjectDetails, error) {
	if inspector.err != nil {
		return mediaObjectDetails{}, inspector.err
	}
	details, ok := inspector.objects[objectKey]
	if !ok {
		return mediaObjectDetails{}, errMediaObjectNotFound
	}
	return details, nil
}

type memoryMediaObjectReader struct {
	objects   map[string][]byte
	err       error
	lastRange string
}

type closeAwareMediaBody struct {
	reader *bytes.Reader
	closed bool
}

func (body *closeAwareMediaBody) Read(buffer []byte) (int, error) {
	if body.closed {
		return 0, errors.New("read closed media body")
	}
	return body.reader.Read(buffer)
}

func (body *closeAwareMediaBody) Close() error {
	body.closed = true
	return nil
}

func (reader *memoryMediaObjectReader) Open(_ context.Context, objectKey, byteRange string) (io.ReadCloser, error) {
	if reader.err != nil {
		return nil, reader.err
	}
	contents, ok := reader.objects[objectKey]
	if !ok {
		return nil, errMediaObjectNotFound
	}
	reader.lastRange = byteRange
	switch byteRange {
	case "":
	case "bytes=1-3":
		contents = contents[1:4]
	case "bytes=4-5":
		contents = contents[4:6]
	default:
		return nil, errors.New("unexpected byte range")
	}
	return &closeAwareMediaBody{reader: bytes.NewReader(contents)}, nil
}

func newMediaTestApp(t *testing.T) (*fiber.App, *memoryMediaRepository) {
	t.Helper()

	administrators := &memoryAdminRepository{administrators: make(map[string]admin)}
	if err := administrators.Initialize(context.Background(), "admin", "correct horse battery staple"); err != nil {
		t.Fatalf("initialize administrator: %v", err)
	}
	repository := &memoryMediaRepository{media: make(map[string]mediaRecord)}
	objectKey := "media/image/2026/08/018f4f4e-6e80-7de3-a769-6d62a0df4f31.webp"
	publicObjectKey := "media/image/2026/08/public-object.webp"
	media := &mediaService{
		repository: repository,
		issuer: qiniuUploadTokenIssuer{
			accessKey: "test-access-key",
			secretKey: "test-secret-key",
			bucket:    "test-bucket",
		},
		inspector: memoryMediaObjectInspector{objects: map[string]mediaObjectDetails{
			objectKey: {
				MimeType: "image/webp",
				Size:     1048576,
			},
			publicObjectKey: {
				MimeType: "image/webp",
				Size:     6,
			},
			"media/audio/2026/08/018f4f4e-6e80-7de3-a769-6d62a0df4f31.mp3": {
				MimeType: "audio/mpeg",
				Size:     1048576,
			},
		}},
		reader: &memoryMediaObjectReader{objects: map[string][]byte{
			publicObjectKey: []byte("abcdef"),
		}},
		publicBaseURL: "https://cdn.example.com",
		uploadURL:     "https://up-z2.qiniup.com",
		now: func() time.Time {
			return time.Date(2026, time.August, 12, 12, 0, 0, 0, time.UTC)
		},
		newID: func() string {
			return "018f4f4e-6e80-7de3-a769-6d62a0df4f31"
		},
		tokenTTL: 10 * time.Minute,
	}

	return newApp(nil, appServices{
		administrators: administrators,
		config:         &memoryConfigRepository{configuration: map[string]any{}},
		sessions:       &memorySessionRepository{sessions: map[string]uint{"test-session": 1}},
		media:          media,
		ttl:            time.Hour,
		cookieDomain:   ".ykagari.top",
		corsOrigin:     "https://ykagari.top",
	}), repository
}

func TestPublicMediaObjectSupportsGetHeadAndRanges(t *testing.T) {
	app, _ := newMediaTestApp(t)
	objectPath := "/api/v1/media/media/image/2026/08/public-object.webp"

	tests := []struct {
		name             string
		method           string
		rangeHeader      string
		wantStatus       int
		wantBody         string
		wantContentRange string
		wantLength       string
	}{
		{
			name:       "full object",
			method:     fiber.MethodGet,
			wantStatus: fiber.StatusOK,
			wantBody:   "abcdef",
			wantLength: "6",
		},
		{
			name:       "head",
			method:     fiber.MethodHead,
			wantStatus: fiber.StatusOK,
			wantLength: "6",
		},
		{
			name:             "bounded range",
			method:           fiber.MethodGet,
			rangeHeader:      "bytes=1-3",
			wantStatus:       fiber.StatusPartialContent,
			wantBody:         "bcd",
			wantContentRange: "bytes 1-3/6",
			wantLength:       "3",
		},
		{
			name:             "suffix range",
			method:           fiber.MethodGet,
			rangeHeader:      "bytes=-2",
			wantStatus:       fiber.StatusPartialContent,
			wantBody:         "ef",
			wantContentRange: "bytes 4-5/6",
			wantLength:       "2",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, objectPath, nil)
			if test.rangeHeader != "" {
				request.Header.Set(fiber.HeaderRange, test.rangeHeader)
			}
			response, err := app.Test(request)
			if err != nil {
				t.Fatalf("read public media: %v", err)
			}
			defer response.Body.Close()
			body, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatalf("read response body: %v", err)
			}
			if response.StatusCode != test.wantStatus {
				t.Errorf("status = %d, want %d", response.StatusCode, test.wantStatus)
			}
			if string(body) != test.wantBody {
				t.Errorf("body = %q, want %q", body, test.wantBody)
			}
			if response.Header.Get(fiber.HeaderContentType) != "image/webp" {
				t.Errorf("content type = %q, want image/webp", response.Header.Get(fiber.HeaderContentType))
			}
			if response.Header.Get(fiber.HeaderAcceptRanges) != "bytes" {
				t.Errorf("accept ranges = %q, want bytes", response.Header.Get(fiber.HeaderAcceptRanges))
			}
			if response.Header.Get(fiber.HeaderContentRange) != test.wantContentRange {
				t.Errorf("content range = %q, want %q", response.Header.Get(fiber.HeaderContentRange), test.wantContentRange)
			}
			if response.Header.Get(fiber.HeaderContentLength) != test.wantLength {
				t.Errorf("content length = %q, want %q", response.Header.Get(fiber.HeaderContentLength), test.wantLength)
			}
			if response.Header.Get(fiber.HeaderCacheControl) != "public, max-age=31536000, immutable" {
				t.Errorf("cache control = %q, want immutable public caching", response.Header.Get(fiber.HeaderCacheControl))
			}
		})
	}
}

func TestPublicMediaObjectRejectsInvalidRequests(t *testing.T) {
	objectKey := "media/image/2026/08/018f4f4e-6e80-7de3-a769-6d62a0df4f31.webp"
	tests := []struct {
		name        string
		path        string
		rangeHeader string
		inspector   mediaObjectInspector
		reader      mediaObjectReader
		wantStatus  int
	}{
		{
			name:       "invalid managed key",
			path:       "/api/v1/media/other/image.webp",
			wantStatus: fiber.StatusNotFound,
		},
		{
			name:        "multiple ranges",
			path:        "/api/v1/media/" + objectKey,
			rangeHeader: "bytes=0-1,3-4",
			inspector: memoryMediaObjectInspector{objects: map[string]mediaObjectDetails{
				objectKey: {MimeType: "image/webp", Size: 6},
			}},
			wantStatus: fiber.StatusRequestedRangeNotSatisfiable,
		},
		{
			name:        "range starts past object",
			path:        "/api/v1/media/" + objectKey,
			rangeHeader: "bytes=6-",
			inspector: memoryMediaObjectInspector{objects: map[string]mediaObjectDetails{
				objectKey: {MimeType: "image/webp", Size: 6},
			}},
			wantStatus: fiber.StatusRequestedRangeNotSatisfiable,
		},
		{
			name:       "object does not exist",
			path:       "/api/v1/media/" + objectKey,
			inspector:  memoryMediaObjectInspector{objects: map[string]mediaObjectDetails{}},
			wantStatus: fiber.StatusNotFound,
		},
		{
			name:       "object metadata lookup fails",
			path:       "/api/v1/media/" + objectKey,
			inspector:  memoryMediaObjectInspector{err: errors.New("Qiniu unavailable")},
			wantStatus: fiber.StatusBadGateway,
		},
		{
			name: "object read fails",
			path: "/api/v1/media/" + objectKey,
			inspector: memoryMediaObjectInspector{objects: map[string]mediaObjectDetails{
				objectKey: {MimeType: "image/webp", Size: 6},
			}},
			reader:     &memoryMediaObjectReader{err: errors.New("Qiniu unavailable")},
			wantStatus: fiber.StatusBadGateway,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			app := newApp(nil, appServices{
				media: &mediaService{
					inspector: test.inspector,
					reader:    test.reader,
				},
				corsOrigin: "https://ykagari.top",
			})
			request := httptest.NewRequest(fiber.MethodGet, test.path, nil)
			if test.rangeHeader != "" {
				request.Header.Set(fiber.HeaderRange, test.rangeHeader)
			}
			response, err := app.Test(request)
			if err != nil {
				t.Fatalf("read public media: %v", err)
			}
			defer response.Body.Close()
			if response.StatusCode != test.wantStatus {
				t.Errorf("status = %d, want %d", response.StatusCode, test.wantStatus)
			}
			if test.wantStatus == fiber.StatusRequestedRangeNotSatisfiable &&
				response.Header.Get(fiber.HeaderContentRange) != "bytes */6" {
				t.Errorf("content range = %q, want bytes */6", response.Header.Get(fiber.HeaderContentRange))
			}
		})
	}
}

func adminMediaRequest(method, path, body string) *http.Request {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	request.Header.Set(fiber.HeaderCookie, sessionCookieName+"=test-session")
	return request
}

func TestAdminCanRequestRestrictedQiniuUploadCredentials(t *testing.T) {
	app, _ := newMediaTestApp(t)

	t.Run("authentication is required", func(t *testing.T) {
		request := httptest.NewRequest(
			fiber.MethodPost,
			"/api/v1/admin/media/upload-credentials",
			strings.NewReader(`{"kind":"image","mimeType":"image/webp","size":1048576,"filename":"cover.webp"}`),
		)
		request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
		response, err := app.Test(request)
		if err != nil {
			t.Fatalf("request upload credentials: %v", err)
		}
		defer response.Body.Close()
		if response.StatusCode != fiber.StatusUnauthorized {
			t.Errorf("status = %d, want %d", response.StatusCode, fiber.StatusUnauthorized)
		}
	})

	request := adminMediaRequest(
		fiber.MethodPost,
		"/api/v1/admin/media/upload-credentials",
		`{"kind":"image","mimeType":"image/webp","size":1048576,"filename":"cover.webp"}`,
	)
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("request upload credentials: %v", err)
	}
	defer response.Body.Close()

	var credentials mediaUploadCredentialsResponse
	if err := json.NewDecoder(response.Body).Decode(&credentials); err != nil {
		t.Fatalf("decode upload credentials: %v", err)
	}
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusOK)
	}
	if credentials.ObjectKey != "media/image/2026/08/018f4f4e-6e80-7de3-a769-6d62a0df4f31.webp" {
		t.Errorf("object key = %q, want a server-generated image key", credentials.ObjectKey)
	}
	if credentials.PublicURL != "https://cdn.example.com/"+credentials.ObjectKey {
		t.Errorf("public URL = %q, want configured domain and object key", credentials.PublicURL)
	}
	if credentials.UploadToken == "" {
		t.Fatal("upload token is empty")
	}
	if credentials.UploadURL != "https://up-z2.qiniup.com" {
		t.Errorf("upload URL = %q, want configured Qiniu region endpoint", credentials.UploadURL)
	}
	if credentials.ExpiresAt != "2026-08-12T12:10:00Z" {
		t.Errorf("expiresAt = %q, want 2026-08-12T12:10:00Z", credentials.ExpiresAt)
	}

	encodedPolicy := strings.Split(credentials.UploadToken, ":")
	if len(encodedPolicy) != 3 {
		t.Fatalf("upload token has %d parts, want 3", len(encodedPolicy))
	}
	policyBytes, err := base64.RawURLEncoding.DecodeString(encodedPolicy[2])
	if err != nil {
		t.Fatalf("decode upload policy: %v", err)
	}
	var policy struct {
		Scope      string `json:"scope"`
		Deadline   int64  `json:"deadline"`
		InsertOnly int    `json:"insertOnly"`
		DetectMime int    `json:"detectMime"`
		MimeLimit  string `json:"mimeLimit"`
		FsizeMin   int64  `json:"fsizeMin"`
		FsizeLimit int64  `json:"fsizeLimit"`
	}
	if err := json.Unmarshal(policyBytes, &policy); err != nil {
		t.Fatalf("decode upload policy JSON: %v", err)
	}
	if policy.Scope != "test-bucket:"+credentials.ObjectKey {
		t.Errorf("scope = %q, want exact bucket and object key", policy.Scope)
	}
	if policy.Deadline != time.Date(2026, time.August, 12, 12, 10, 0, 0, time.UTC).Unix() {
		t.Errorf("deadline = %d, want fixed ten-minute expiry", policy.Deadline)
	}
	if policy.InsertOnly != 1 || policy.DetectMime != 1 {
		t.Errorf("upload policy = %#v, want insert-only MIME detection", policy)
	}
	if policy.MimeLimit != "image/webp" || policy.FsizeMin != 1048576 || policy.FsizeLimit != 1048576 {
		t.Errorf("upload policy = %#v, want exact MIME and file size", policy)
	}

	encoded, err := json.Marshal(credentials)
	if err != nil {
		t.Fatalf("encode credentials response: %v", err)
	}
	for _, secret := range []string{"test-secret-key", `"accessKey"`, `"secretKey"`} {
		if strings.Contains(string(encoded), secret) {
			t.Errorf("response leaked %q: %s", secret, encoded)
		}
	}
}

func TestUploadCredentialsRejectUnsupportedOrOversizedMedia(t *testing.T) {
	app, _ := newMediaTestApp(t)

	tests := []struct {
		name string
		body string
	}{
		{
			name: "unsupported image MIME",
			body: `{"kind":"image","mimeType":"image/gif","size":1024,"filename":"animation.gif"}`,
		},
		{
			name: "oversized image",
			body: `{"kind":"image","mimeType":"image/jpeg","size":20971521,"filename":"large.jpg"}`,
		},
		{
			name: "unsupported audio MIME",
			body: `{"kind":"audio","mimeType":"audio/aac","size":1024,"filename":"track.aac"}`,
		},
		{
			name: "oversized audio",
			body: `{"kind":"audio","mimeType":"audio/mpeg","size":52428801,"filename":"large.mp3"}`,
		},
		{
			name: "extension does not match MIME",
			body: `{"kind":"image","mimeType":"image/png","size":1024,"filename":"cover.webp"}`,
		},
		{
			name: "empty file",
			body: `{"kind":"image","mimeType":"image/png","size":0,"filename":"cover.png"}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response, err := app.Test(adminMediaRequest(
				fiber.MethodPost,
				"/api/v1/admin/media/upload-credentials",
				test.body,
			))
			if err != nil {
				t.Fatalf("request upload credentials: %v", err)
			}
			defer response.Body.Close()
			if response.StatusCode != fiber.StatusBadRequest {
				t.Errorf("status = %d, want %d", response.StatusCode, fiber.StatusBadRequest)
			}
		})
	}
}

func TestUploadCredentialsAcceptEveryAllowedMediaTypeAtItsLimit(t *testing.T) {
	app, _ := newMediaTestApp(t)

	tests := []struct {
		name     string
		kind     string
		mimeType string
		size     int64
		filename string
		suffix   string
	}{
		{name: "JPEG", kind: "image", mimeType: "image/jpeg", size: maxImageSize, filename: "photo.jpeg", suffix: ".jpg"},
		{name: "PNG", kind: "image", mimeType: "image/png", size: maxImageSize, filename: "photo.png", suffix: ".png"},
		{name: "WebP", kind: "image", mimeType: "image/webp", size: maxImageSize, filename: "photo.webp", suffix: ".webp"},
		{name: "AVIF", kind: "image", mimeType: "image/avif", size: maxImageSize, filename: "photo.avif", suffix: ".avif"},
		{name: "MP3", kind: "audio", mimeType: "audio/mpeg", size: maxAudioSize, filename: "track.mp3", suffix: ".mp3"},
		{name: "OGG", kind: "audio", mimeType: "audio/ogg", size: maxAudioSize, filename: "track.ogg", suffix: ".ogg"},
		{name: "WAV", kind: "audio", mimeType: "audio/wav", size: maxAudioSize, filename: "track.wav", suffix: ".wav"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body, err := json.Marshal(mediaUploadRequest{
				Kind:     test.kind,
				MimeType: test.mimeType,
				Size:     test.size,
				Filename: test.filename,
			})
			if err != nil {
				t.Fatalf("encode request: %v", err)
			}
			response, err := app.Test(adminMediaRequest(
				fiber.MethodPost,
				"/api/v1/admin/media/upload-credentials",
				string(body),
			))
			if err != nil {
				t.Fatalf("request upload credentials: %v", err)
			}
			defer response.Body.Close()
			if response.StatusCode != fiber.StatusOK {
				t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusOK)
			}
			var credentials mediaUploadCredentialsResponse
			if err := json.NewDecoder(response.Body).Decode(&credentials); err != nil {
				t.Fatalf("decode upload credentials: %v", err)
			}
			if !strings.HasSuffix(credentials.ObjectKey, test.suffix) {
				t.Errorf("object key = %q, want suffix %q", credentials.ObjectKey, test.suffix)
			}
		})
	}
}

func TestAdminCanRegisterUploadedMediaMetadata(t *testing.T) {
	app, repository := newMediaTestApp(t)

	t.Run("authentication is required", func(t *testing.T) {
		request := httptest.NewRequest(fiber.MethodPost, "/api/v1/admin/media", strings.NewReader(`{}`))
		request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
		response, err := app.Test(request)
		if err != nil {
			t.Fatalf("register media without a session: %v", err)
		}
		defer response.Body.Close()
		if response.StatusCode != fiber.StatusUnauthorized {
			t.Errorf("status = %d, want %d", response.StatusCode, fiber.StatusUnauthorized)
		}
	})

	imageBody := `{
		"objectKey":"media/image/2026/08/018f4f4e-6e80-7de3-a769-6d62a0df4f31.webp",
		"kind":"image",
		"mimeType":"image/webp",
		"size":1048576,
		"originalName":"cover.webp",
		"width":1920,
		"height":1080
	}`
	response, err := app.Test(adminMediaRequest(fiber.MethodPost, "/api/v1/admin/media", imageBody))
	if err != nil {
		t.Fatalf("register image metadata: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusCreated {
		t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusCreated)
	}

	var registered mediaResponse
	if err := json.NewDecoder(response.Body).Decode(&registered); err != nil {
		t.Fatalf("decode registered media: %v", err)
	}
	if registered.ID != 1 || registered.PublicURL != "https://cdn.example.com/"+registered.ObjectKey {
		t.Errorf("registered media = %#v, want stored object with computed public URL", registered)
	}
	stored, ok := repository.media[registered.ObjectKey]
	if !ok {
		t.Fatalf("object key %q was not persisted", registered.ObjectKey)
	}
	if stored.MimeType != "image/webp" || stored.Size != 1048576 || stored.Width != 1920 || stored.Height != 1080 {
		t.Errorf("stored media = %#v, want validated image metadata", stored)
	}
}

func TestMediaRegistrationVerifiesTheUploadedQiniuObject(t *testing.T) {
	tests := []struct {
		name       string
		inspector  mediaObjectInspector
		wantStatus int
	}{
		{
			name:       "object does not exist",
			inspector:  memoryMediaObjectInspector{objects: map[string]mediaObjectDetails{}},
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name: "object MIME does not match",
			inspector: memoryMediaObjectInspector{objects: map[string]mediaObjectDetails{
				"media/image/2026/08/018f4f4e-6e80-7de3-a769-6d62a0df4f31.webp": {
					MimeType: "application/octet-stream",
					Size:     1048576,
				},
			}},
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name: "object size does not match",
			inspector: memoryMediaObjectInspector{objects: map[string]mediaObjectDetails{
				"media/image/2026/08/018f4f4e-6e80-7de3-a769-6d62a0df4f31.webp": {
					MimeType: "image/webp",
					Size:     1048577,
				},
			}},
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "Qiniu lookup fails",
			inspector:  memoryMediaObjectInspector{err: errors.New("Qiniu unavailable")},
			wantStatus: fiber.StatusBadGateway,
		},
	}

	body := `{
		"objectKey":"media/image/2026/08/018f4f4e-6e80-7de3-a769-6d62a0df4f31.webp",
		"kind":"image",
		"mimeType":"image/webp",
		"size":1048576,
		"originalName":"cover.webp",
		"width":1920,
		"height":1080
	}`
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			app, _ := newMediaTestApp(t)
			service := appServices{
				sessions: &memorySessionRepository{sessions: map[string]uint{"test-session": 1}},
				media: &mediaService{
					repository:    &memoryMediaRepository{media: make(map[string]mediaRecord)},
					inspector:     test.inspector,
					publicBaseURL: "https://cdn.example.com",
				},
				ttl:        time.Hour,
				corsOrigin: "https://ykagari.top",
			}
			app = newApp(nil, service)

			response, err := app.Test(adminMediaRequest(fiber.MethodPost, "/api/v1/admin/media", body))
			if err != nil {
				t.Fatalf("register media: %v", err)
			}
			defer response.Body.Close()
			if response.StatusCode != test.wantStatus {
				t.Errorf("status = %d, want %d", response.StatusCode, test.wantStatus)
			}
		})
	}
}

func TestMediaRegistrationRejectsInvalidMetadata(t *testing.T) {
	app, _ := newMediaTestApp(t)

	tests := []struct {
		name string
		body string
	}{
		{
			name: "unsupported MIME",
			body: `{"objectKey":"media/image/2026/08/id.gif","kind":"image","mimeType":"image/gif","size":1024,"originalName":"image.gif","width":100,"height":100}`,
		},
		{
			name: "oversized image",
			body: `{"objectKey":"media/image/2026/08/id.jpg","kind":"image","mimeType":"image/jpeg","size":20971521,"originalName":"image.jpg","width":100,"height":100}`,
		},
		{
			name: "invalid object key",
			body: `{"objectKey":"other/image.webp","kind":"image","mimeType":"image/webp","size":1024,"originalName":"image.webp","width":100,"height":100}`,
		},
		{
			name: "image dimensions are required",
			body: `{"objectKey":"media/image/2026/08/id.webp","kind":"image","mimeType":"image/webp","size":1024,"originalName":"image.webp"}`,
		},
		{
			name: "audio duration is required",
			body: `{"objectKey":"media/audio/2026/08/id.mp3","kind":"audio","mimeType":"audio/mpeg","size":1024,"originalName":"track.mp3"}`,
		},
		{
			name: "image cannot contain audio duration",
			body: `{"objectKey":"media/image/2026/08/id.png","kind":"image","mimeType":"image/png","size":1024,"originalName":"image.png","width":100,"height":100,"durationMs":1000}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response, err := app.Test(adminMediaRequest(fiber.MethodPost, "/api/v1/admin/media", test.body))
			if err != nil {
				t.Fatalf("register media: %v", err)
			}
			defer response.Body.Close()
			if response.StatusCode != fiber.StatusBadRequest {
				t.Errorf("status = %d, want %d", response.StatusCode, fiber.StatusBadRequest)
			}
		})
	}
}

type failingMediaRepository struct{ err error }

func (repository failingMediaRepository) Save(context.Context, mediaRecord) (mediaRecord, error) {
	return mediaRecord{}, repository.err
}

func TestMediaRegistrationReturnsConflictForDuplicateObjectKey(t *testing.T) {
	app, _ := newMediaTestApp(t)
	body := `{
		"objectKey":"media/audio/2026/08/018f4f4e-6e80-7de3-a769-6d62a0df4f31.mp3",
		"kind":"audio",
		"mimeType":"audio/mpeg",
		"size":1048576,
		"originalName":"track.mp3",
		"durationMs":180000
	}`

	first, err := app.Test(adminMediaRequest(fiber.MethodPost, "/api/v1/admin/media", body))
	if err != nil {
		t.Fatalf("register media: %v", err)
	}
	first.Body.Close()
	if first.StatusCode != fiber.StatusCreated {
		t.Fatalf("first status = %d, want %d", first.StatusCode, fiber.StatusCreated)
	}

	second, err := app.Test(adminMediaRequest(fiber.MethodPost, "/api/v1/admin/media", body))
	if err != nil {
		t.Fatalf("register duplicate media: %v", err)
	}
	defer second.Body.Close()
	if second.StatusCode != fiber.StatusConflict {
		t.Errorf("duplicate status = %d, want %d", second.StatusCode, fiber.StatusConflict)
	}
}

func TestMediaRegistrationReturnsServerErrorOnRepositoryFailure(t *testing.T) {
	app, _ := newMediaTestApp(t)
	app = newApp(nil, appServices{
		sessions: &memorySessionRepository{sessions: map[string]uint{"test-session": 1}},
		media: &mediaService{
			repository: failingMediaRepository{err: errors.New("database unavailable")},
			inspector: memoryMediaObjectInspector{objects: map[string]mediaObjectDetails{
				"media/audio/2026/08/018f4f4e-6e80-7de3-a769-6d62a0df4f31.mp3": {
					MimeType: "audio/mpeg",
					Size:     1048576,
				},
			}},
			publicBaseURL: "https://cdn.example.com",
		},
		ttl:        time.Hour,
		corsOrigin: "https://ykagari.top",
	})
	body := `{
		"objectKey":"media/audio/2026/08/018f4f4e-6e80-7de3-a769-6d62a0df4f31.mp3",
		"kind":"audio",
		"mimeType":"audio/mpeg",
		"size":1048576,
		"originalName":"track.mp3",
		"durationMs":180000
	}`

	response, err := app.Test(adminMediaRequest(fiber.MethodPost, "/api/v1/admin/media", body))
	if err != nil {
		t.Fatalf("register media when storage fails: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusInternalServerError {
		t.Errorf("status = %d, want %d", response.StatusCode, fiber.StatusInternalServerError)
	}
}

func TestMediaServiceEnvironmentConfiguration(t *testing.T) {
	names := []string{
		"QINIU_ACCESS_KEY",
		"QINIU_SECRET_KEY",
		"QINIU_BUCKET",
		"QINIU_PUBLIC_BASE_URL",
		"QINIU_UPLOAD_URL",
	}
	original := make(map[string]string, len(names))
	for _, name := range names {
		original[name] = os.Getenv(name)
		t.Cleanup(func() {
			if original[name] == "" {
				os.Unsetenv(name)
				return
			}
			os.Setenv(name, original[name])
		})
		os.Unsetenv(name)
	}

	if _, err := mediaServiceFromEnvironment(nil); err == nil {
		t.Fatal("missing Qiniu configuration did not return an error")
	}

	os.Setenv("QINIU_ACCESS_KEY", "test-access-key")
	os.Setenv("QINIU_SECRET_KEY", "test-secret-key")
	os.Setenv("QINIU_BUCKET", "test-bucket")
	os.Setenv("QINIU_UPLOAD_URL", "https://evil.example.com")
	os.Setenv("QINIU_PUBLIC_BASE_URL", "http://cdn.example.com")
	if _, err := mediaServiceFromEnvironment(nil); err == nil {
		t.Fatal("non-HTTPS public base URL did not return an error")
	}

	os.Setenv("QINIU_PUBLIC_BASE_URL", "https://user:password@cdn.example.com")
	if _, err := mediaServiceFromEnvironment(nil); err == nil {
		t.Fatal("public base URL with user information did not return an error")
	}

	os.Setenv("QINIU_PUBLIC_BASE_URL", "https://cdn.example.com")
	if _, err := mediaServiceFromEnvironment(nil); err == nil {
		t.Fatal("non-Qiniu upload URL did not return an error")
	}

	os.Setenv("QINIU_UPLOAD_URL", "http://up-z2.qiniup.com")
	if _, err := mediaServiceFromEnvironment(nil); err == nil {
		t.Fatal("non-HTTPS upload URL did not return an error")
	}

	os.Setenv("QINIU_UPLOAD_URL", "https://up-z2.qiniup.com")
	os.Setenv("QINIU_PUBLIC_BASE_URL", "https://cdn.example.com/")
	service, err := mediaServiceFromEnvironment(nil)
	if err != nil {
		t.Fatalf("build media service from environment: %v", err)
	}
	if service.publicBaseURL != "https://cdn.example.com" {
		t.Errorf("public base URL = %q, want normalized HTTPS origin", service.publicBaseURL)
	}
	if service.uploadURL != "https://up-z2.qiniup.com" {
		t.Errorf("upload URL = %q, want normalized Qiniu upload origin", service.uploadURL)
	}
}
