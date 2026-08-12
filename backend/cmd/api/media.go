package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	driverMysql "github.com/go-sql-driver/mysql"
	"github.com/gofiber/fiber/v2"
	qiniuClient "github.com/qiniu/go-sdk/v7/client"
	"github.com/qiniu/go-sdk/v7/storagev2/credentials"
	httpclient "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	"github.com/qiniu/go-sdk/v7/storagev2/objects"
	"github.com/qiniu/go-sdk/v7/storagev2/uptoken"
	"gorm.io/gorm"
)

const (
	mediaKindImage = "image"
	mediaKindAudio = "audio"

	maxImageSize = 20 * 1024 * 1024
	maxAudioSize = 50 * 1024 * 1024
)

var (
	errMediaObjectKeyTaken = errors.New("media object key already exists")
	errMediaObjectNotFound = errors.New("media object not found")
	mediaObjectKeyPattern  = regexp.MustCompile(`^media/(image|audio)/[0-9]{4}/(0[1-9]|1[0-2])/[A-Za-z0-9_-]+\.[a-z0-9]+$`)
)

type mediaRecord struct {
	ID           uint      `gorm:"primaryKey"`
	ObjectKey    string    `gorm:"size:512;uniqueIndex;not null"`
	Kind         string    `gorm:"size:16;index;not null"`
	MimeType     string    `gorm:"size:128;not null"`
	Size         int64     `gorm:"not null"`
	OriginalName string    `gorm:"size:255;not null"`
	Width        int       `gorm:"not null"`
	Height       int       `gorm:"not null"`
	DurationMs   int64     `gorm:"not null"`
	CreatedAt    time.Time `gorm:"not null"`
}

type mediaRepository interface {
	Save(context.Context, mediaRecord) (mediaRecord, error)
}

type gormMediaRepository struct{ db *gorm.DB }

func (repository gormMediaRepository) Save(ctx context.Context, record mediaRecord) (mediaRecord, error) {
	if err := repository.db.WithContext(ctx).Create(&record).Error; err != nil {
		var mysqlError *driverMysql.MySQLError
		if errors.As(err, &mysqlError) && mysqlError.Number == 1062 {
			return mediaRecord{}, errMediaObjectKeyTaken
		}
		return mediaRecord{}, fmt.Errorf("create media record: %w", err)
	}
	return record, nil
}

type mediaUploadRequest struct {
	Kind     string `json:"kind"`
	MimeType string `json:"mimeType"`
	Size     int64  `json:"size"`
	Filename string `json:"filename"`
}

type mediaRegistrationRequest struct {
	ObjectKey    string `json:"objectKey"`
	Kind         string `json:"kind"`
	MimeType     string `json:"mimeType"`
	Size         int64  `json:"size"`
	OriginalName string `json:"originalName"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	DurationMs   int64  `json:"durationMs"`
}

type mediaUploadCredentialsResponse struct {
	UploadToken string `json:"uploadToken"`
	UploadURL   string `json:"uploadUrl"`
	ObjectKey   string `json:"objectKey"`
	PublicURL   string `json:"publicUrl"`
	ExpiresAt   string `json:"expiresAt"`
}

type mediaResponse struct {
	ID           uint   `json:"id"`
	ObjectKey    string `json:"objectKey"`
	PublicURL    string `json:"publicUrl"`
	Kind         string `json:"kind"`
	MimeType     string `json:"mimeType"`
	Size         int64  `json:"size"`
	OriginalName string `json:"originalName"`
	Width        int    `json:"width,omitempty"`
	Height       int    `json:"height,omitempty"`
	DurationMs   int64  `json:"durationMs,omitempty"`
	CreatedAt    string `json:"createdAt"`
}

type uploadTokenPolicy struct {
	ObjectKey string
	MimeType  string
	Size      int64
	Expires   time.Time
}

type uploadTokenIssuer interface {
	Issue(uploadTokenPolicy) (string, error)
}

type mediaObjectDetails struct {
	MimeType string
	Size     int64
}

type mediaObjectInspector interface {
	Stat(context.Context, string) (mediaObjectDetails, error)
}

type qiniuUploadTokenIssuer struct {
	accessKey string
	secretKey string
	bucket    string
}

func (issuer qiniuUploadTokenIssuer) Issue(policy uploadTokenPolicy) (string, error) {
	if issuer.accessKey == "" || issuer.secretKey == "" || issuer.bucket == "" {
		return "", errors.New("qiniu upload credentials are not configured")
	}
	putPolicy, err := uptoken.NewPutPolicyWithKey(issuer.bucket, policy.ObjectKey, policy.Expires)
	if err != nil {
		return "", fmt.Errorf("create qiniu upload policy: %w", err)
	}
	putPolicy = putPolicy.
		SetInsertOnly(1).
		SetReturnBody(`{"key":"$(key)","hash":"$(etag)","size":$(fsize),"mimeType":"$(mimeType)"}`).
		SetFsizeMin(policy.Size).
		SetFsizeLimit(policy.Size).
		SetDetectMime(1).
		SetMimeLimit(policy.MimeType)
	token, err := uptoken.NewSigner(
		putPolicy,
		credentials.NewCredentials(issuer.accessKey, issuer.secretKey),
	).GetUpToken(context.Background())
	if err != nil {
		return "", fmt.Errorf("sign qiniu upload policy: %w", err)
	}
	return token, nil
}

type qiniuObjectInspector struct {
	manager *objects.ObjectsManager
	bucket  string
}

func (inspector qiniuObjectInspector) Stat(ctx context.Context, objectKey string) (mediaObjectDetails, error) {
	details, err := inspector.manager.Bucket(inspector.bucket).Object(objectKey).Stat().Call(ctx)
	if err != nil {
		var qiniuError *qiniuClient.ErrorInfo
		if errors.As(err, &qiniuError) && qiniuError.Code == 612 {
			return mediaObjectDetails{}, errMediaObjectNotFound
		}
		return mediaObjectDetails{}, fmt.Errorf("stat qiniu object: %w", err)
	}
	return mediaObjectDetails{MimeType: details.MimeType, Size: details.Size}, nil
}

type mediaService struct {
	repository    mediaRepository
	issuer        uploadTokenIssuer
	inspector     mediaObjectInspector
	publicBaseURL string
	uploadURL     string
	now           func() time.Time
	newID         func() string
	tokenTTL      time.Duration
}

func (service *mediaService) uploadCredentials(c *fiber.Ctx) error {
	var request mediaUploadRequest
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "media upload request must be a JSON object"})
	}
	specification, err := validateMediaFile(request.Kind, request.MimeType, request.Size, request.Filename)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	now := service.currentTime()
	objectID, err := service.objectID()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "create media object key"})
	}
	objectKey := fmt.Sprintf(
		"media/%s/%s/%s.%s",
		specification.kind,
		now.Format("2006/01"),
		objectID,
		specification.extension,
	)
	expires := now.Add(service.uploadTokenTTL())
	token, err := service.issuer.Issue(uploadTokenPolicy{
		ObjectKey: objectKey,
		MimeType:  specification.mimeType,
		Size:      request.Size,
		Expires:   expires,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "create media upload credentials"})
	}

	return c.JSON(mediaUploadCredentialsResponse{
		UploadToken: token,
		UploadURL:   service.uploadURL,
		ObjectKey:   objectKey,
		PublicURL:   service.publicURL(objectKey),
		ExpiresAt:   expires.UTC().Format(time.RFC3339),
	})
}

func (service *mediaService) register(c *fiber.Ctx) error {
	var request mediaRegistrationRequest
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "media metadata must be a JSON object"})
	}
	record, err := mediaFromRequest(request)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	actual, err := service.inspector.Stat(c.Context(), record.ObjectKey)
	if errors.Is(err, errMediaObjectNotFound) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "uploaded media object does not exist"})
	}
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "verify uploaded media object"})
	}
	if actual.MimeType != record.MimeType || actual.Size != record.Size {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "uploaded media object does not match its metadata"})
	}
	record.CreatedAt = service.currentTime()
	record, err = service.repository.Save(c.Context(), record)
	if errors.Is(err, errMediaObjectKeyTaken) {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "media object key already exists"})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "register media metadata"})
	}
	return c.Status(fiber.StatusCreated).JSON(service.response(record))
}

func mediaFromRequest(request mediaRegistrationRequest) (mediaRecord, error) {
	specification, err := validateMediaFile(request.Kind, request.MimeType, request.Size, request.OriginalName)
	if err != nil {
		return mediaRecord{}, err
	}
	request.ObjectKey = strings.TrimSpace(request.ObjectKey)
	if !mediaObjectKeyPattern.MatchString(request.ObjectKey) {
		return mediaRecord{}, errors.New("objectKey must use the managed media namespace")
	}
	parts := strings.Split(request.ObjectKey, "/")
	if len(parts) != 5 || parts[1] != specification.kind || path.Ext(request.ObjectKey) != "."+specification.extension {
		return mediaRecord{}, errors.New("objectKey does not match the media metadata")
	}

	switch specification.kind {
	case mediaKindImage:
		if request.Width <= 0 || request.Height <= 0 {
			return mediaRecord{}, errors.New("image width and height must be positive")
		}
		if request.DurationMs != 0 {
			return mediaRecord{}, errors.New("image metadata must not include durationMs")
		}
	case mediaKindAudio:
		if request.DurationMs <= 0 {
			return mediaRecord{}, errors.New("audio durationMs must be positive")
		}
		if request.Width != 0 || request.Height != 0 {
			return mediaRecord{}, errors.New("audio metadata must not include image dimensions")
		}
	}

	return mediaRecord{
		ObjectKey:    request.ObjectKey,
		Kind:         specification.kind,
		MimeType:     specification.mimeType,
		Size:         request.Size,
		OriginalName: strings.TrimSpace(request.OriginalName),
		Width:        request.Width,
		Height:       request.Height,
		DurationMs:   request.DurationMs,
	}, nil
}

type mediaFileSpecification struct {
	kind      string
	mimeType  string
	extension string
}

var allowedMediaFiles = map[string]mediaFileSpecification{
	"image/jpeg": {kind: mediaKindImage, mimeType: "image/jpeg", extension: "jpg"},
	"image/png":  {kind: mediaKindImage, mimeType: "image/png", extension: "png"},
	"image/webp": {kind: mediaKindImage, mimeType: "image/webp", extension: "webp"},
	"image/avif": {kind: mediaKindImage, mimeType: "image/avif", extension: "avif"},
	"audio/mpeg": {kind: mediaKindAudio, mimeType: "audio/mpeg", extension: "mp3"},
	"audio/ogg":  {kind: mediaKindAudio, mimeType: "audio/ogg", extension: "ogg"},
	"audio/wav":  {kind: mediaKindAudio, mimeType: "audio/wav", extension: "wav"},
}

func validateMediaFile(kind, mimeType string, size int64, filename string) (mediaFileSpecification, error) {
	kind = strings.TrimSpace(kind)
	mimeType = strings.ToLower(strings.TrimSpace(mimeType))
	filename = strings.TrimSpace(filename)
	specification, ok := allowedMediaFiles[mimeType]
	if !ok || specification.kind != kind {
		return mediaFileSpecification{}, errors.New("media type is not allowed")
	}
	if size <= 0 {
		return mediaFileSpecification{}, errors.New("media size must be positive")
	}
	maximum := int64(maxImageSize)
	if kind == mediaKindAudio {
		maximum = maxAudioSize
	}
	if size > maximum {
		return mediaFileSpecification{}, fmt.Errorf("%s files must not exceed %d bytes", kind, maximum)
	}
	if utf8.RuneCountInString(filename) == 0 || utf8.RuneCountInString(filename) > 255 ||
		strings.ContainsAny(filename, `/\`) {
		return mediaFileSpecification{}, errors.New("filename must contain between 1 and 255 characters without a path")
	}
	extension := strings.ToLower(strings.TrimPrefix(path.Ext(filename), "."))
	if mimeType == "image/jpeg" && extension == "jpeg" {
		extension = "jpg"
	}
	if extension != specification.extension {
		return mediaFileSpecification{}, errors.New("filename extension does not match mimeType")
	}
	return specification, nil
}

func (service *mediaService) response(record mediaRecord) mediaResponse {
	return mediaResponse{
		ID:           record.ID,
		ObjectKey:    record.ObjectKey,
		PublicURL:    service.publicURL(record.ObjectKey),
		Kind:         record.Kind,
		MimeType:     record.MimeType,
		Size:         record.Size,
		OriginalName: record.OriginalName,
		Width:        record.Width,
		Height:       record.Height,
		DurationMs:   record.DurationMs,
		CreatedAt:    record.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func (service *mediaService) publicURL(objectKey string) string {
	return strings.TrimRight(service.publicBaseURL, "/") + "/" + objectKey
}

func (service *mediaService) currentTime() time.Time {
	if service.now == nil {
		return time.Now().UTC()
	}
	return service.now().UTC()
}

func (service *mediaService) objectID() (string, error) {
	if service.newID != nil {
		id := service.newID()
		if id == "" {
			return "", errors.New("generated media object ID is empty")
		}
		return id, nil
	}
	bytes := make([]byte, 18)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate media object ID: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func (service *mediaService) uploadTokenTTL() time.Duration {
	if service.tokenTTL <= 0 {
		return 10 * time.Minute
	}
	return service.tokenTTL
}

func validateQiniuPublicBaseURL(value string) error {
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.RawQuery != "" ||
		parsed.Fragment != "" || parsed.User != nil || (parsed.Path != "" && parsed.Path != "/") {
		return errors.New("QINIU_PUBLIC_BASE_URL must be an HTTPS origin")
	}
	return nil
}

func validateQiniuUploadURL(value string) error {
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.RawQuery != "" ||
		parsed.Fragment != "" || parsed.User != nil || (parsed.Path != "" && parsed.Path != "/") {
		return errors.New("QINIU_UPLOAD_URL must be an official Qiniu HTTPS upload origin")
	}
	hostname := strings.ToLower(parsed.Hostname())
	if hostname != "upload.qiniup.com" && !strings.HasSuffix(hostname, ".qiniup.com") {
		return errors.New("QINIU_UPLOAD_URL must be an official Qiniu HTTPS upload origin")
	}
	return nil
}

func mediaServiceFromEnvironment(db *gorm.DB) (*mediaService, error) {
	accessKey := os.Getenv("QINIU_ACCESS_KEY")
	secretKey := os.Getenv("QINIU_SECRET_KEY")
	bucket := os.Getenv("QINIU_BUCKET")
	publicBaseURL := strings.TrimRight(os.Getenv("QINIU_PUBLIC_BASE_URL"), "/")
	uploadURL := strings.TrimRight(os.Getenv("QINIU_UPLOAD_URL"), "/")
	if accessKey == "" || secretKey == "" || bucket == "" || publicBaseURL == "" || uploadURL == "" {
		return nil, errors.New("QINIU_ACCESS_KEY, QINIU_SECRET_KEY, QINIU_BUCKET, QINIU_PUBLIC_BASE_URL and QINIU_UPLOAD_URL are required")
	}
	if err := validateQiniuPublicBaseURL(publicBaseURL); err != nil {
		return nil, err
	}
	if err := validateQiniuUploadURL(uploadURL); err != nil {
		return nil, err
	}
	qiniuCredentials := credentials.NewCredentials(accessKey, secretKey)
	return &mediaService{
		repository: gormMediaRepository{db: db},
		issuer: qiniuUploadTokenIssuer{
			accessKey: accessKey,
			secretKey: secretKey,
			bucket:    bucket,
		},
		inspector: qiniuObjectInspector{
			manager: objects.NewObjectsManager(&objects.ObjectsManagerOptions{
				Options: httpclient.Options{Credentials: qiniuCredentials},
			}),
			bucket: bucket,
		},
		publicBaseURL: publicBaseURL,
		uploadURL:     uploadURL,
		tokenTTL:      10 * time.Minute,
	}, nil
}
