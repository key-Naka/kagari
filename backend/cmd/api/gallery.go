package main

import (
	"context"
	_ "embed"
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
	"gorm.io/gorm"
)

var (
	errAlbumItemNotFound = errors.New("Album Item not found")
	yearPattern          = regexp.MustCompile(`^\d{4}$`)
	galleryWidthPattern  = regexp.MustCompile(`^(\d+(?:\.\d+)?)vw$`)
	aspectRatioPattern   = regexp.MustCompile(`^(\d+(?:\.\d+)?) / (\d+(?:\.\d+)?)$`)
	galleryColorPattern  = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
)

type albumItemValidationError struct{ message string }

func (err albumItemValidationError) Error() string { return err.message }

type albumItem struct {
	ID           uint        `gorm:"primaryKey"`
	Title        string      `gorm:"size:160;not null"`
	Note         string      `gorm:"size:255;not null"`
	Alt          string      `gorm:"size:500;not null"`
	Year         string      `gorm:"size:4;not null;index"`
	ImageMediaID *uint       `gorm:"index"`
	Image        mediaRecord `gorm:"foreignKey:ImageMediaID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	AnchorX      float64     `gorm:"not null"`
	AnchorY      float64     `gorm:"not null"`
	Width        string      `gorm:"size:32;not null"`
	AspectRatio  string      `gorm:"size:32;not null"`
	ColorA       string      `gorm:"size:7;not null"`
	ColorB       string      `gorm:"size:7;not null"`
	ColorC       string      `gorm:"size:7;not null"`
	Published    bool        `gorm:"not null;index"`
	SortOrder    int         `gorm:"not null;index"`
	CreatedAt    time.Time   `gorm:"not null"`
	UpdatedAt    time.Time   `gorm:"not null"`
}

type albumItemRepository interface {
	List(context.Context) ([]albumItem, error)
	FindByID(context.Context, uint) (albumItem, error)
	Save(context.Context, albumItem) (albumItem, error)
	Delete(context.Context, uint) error
}

type gormAlbumItemRepository struct{ db *gorm.DB }

func (repository gormAlbumItemRepository) List(ctx context.Context) ([]albumItem, error) {
	var items []albumItem
	if err := repository.db.WithContext(ctx).Preload("Image").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("list Album Items: %w", err)
	}
	return items, nil
}

func (repository gormAlbumItemRepository) FindByID(ctx context.Context, id uint) (albumItem, error) {
	var item albumItem
	if err := repository.db.WithContext(ctx).Preload("Image").First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return albumItem{}, errAlbumItemNotFound
		}
		return albumItem{}, fmt.Errorf("find Album Item: %w", err)
	}
	return item, nil
}

func (repository gormAlbumItemRepository) Save(ctx context.Context, item albumItem) (albumItem, error) {
	query := repository.db.WithContext(ctx).Omit("Image")
	if item.ID == 0 {
		if err := query.Create(&item).Error; err != nil {
			return albumItem{}, fmt.Errorf("create Album Item: %w", err)
		}
	} else if err := query.Save(&item).Error; err != nil {
		return albumItem{}, fmt.Errorf("save Album Item: %w", err)
	}
	return repository.FindByID(ctx, item.ID)
}

func (repository gormAlbumItemRepository) Delete(ctx context.Context, id uint) error {
	result := repository.db.WithContext(ctx).Delete(&albumItem{}, id)
	if result.Error != nil {
		return fmt.Errorf("delete Album Item: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errAlbumItemNotFound
	}
	return nil
}

type albumItemRequest struct {
	Title        string    `json:"title"`
	Note         string    `json:"note"`
	Alt          string    `json:"alt"`
	Year         string    `json:"year"`
	ImageMediaID uint      `json:"imageMediaId"`
	AnchorX      float64   `json:"anchorX"`
	AnchorY      float64   `json:"anchorY"`
	Width        string    `json:"width"`
	AspectRatio  string    `json:"aspectRatio"`
	Colors       [3]string `json:"colors"`
	Published    bool      `json:"published"`
	SortOrder    int       `json:"sortOrder"`
}

type publicAlbumItemResponse struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Note        string    `json:"note"`
	Alt         string    `json:"alt"`
	Year        string    `json:"year"`
	ImageURL    string    `json:"imageUrl,omitempty"`
	AnchorX     float64   `json:"anchorX"`
	AnchorY     float64   `json:"anchorY"`
	Width       string    `json:"width"`
	AspectRatio string    `json:"aspectRatio"`
	Colors      [3]string `json:"colors"`
}

type adminAlbumItemResponse struct {
	ID          uint           `json:"id"`
	Title       string         `json:"title"`
	Note        string         `json:"note"`
	Alt         string         `json:"alt"`
	Year        string         `json:"year"`
	Image       *mediaResponse `json:"image"`
	AnchorX     float64        `json:"anchorX"`
	AnchorY     float64        `json:"anchorY"`
	Width       string         `json:"width"`
	AspectRatio string         `json:"aspectRatio"`
	Colors      [3]string      `json:"colors"`
	Published   bool           `json:"published"`
	SortOrder   int            `json:"sortOrder"`
}

//go:embed gallery_items.json
var galleryItemsJSON []byte

var seededGalleryItems = decodeSeededGalleryItems(galleryItemsJSON)

const albumSeedMigrationName = "2026-08-19-seed-album-items"

type contentMigration struct {
	Name      string    `gorm:"primaryKey;size:160"`
	CreatedAt time.Time `gorm:"not null"`
}

func decodeSeededGalleryItems(encoded []byte) []publicAlbumItemResponse {
	items := []publicAlbumItemResponse{}
	if err := json.Unmarshal(encoded, &items); err != nil {
		panic("decode seeded gallery items: " + err.Error())
	}
	return items
}

func seededAlbumItemRecords() []albumItem {
	records := make([]albumItem, 0, len(seededGalleryItems))
	for index, item := range seededGalleryItems {
		records = append(records, albumItem{
			ID: uint(index + 1), Title: item.Title, Note: item.Note, Alt: item.Alt, Year: item.Year,
			AnchorX: item.AnchorX, AnchorY: item.AnchorY, Width: item.Width, AspectRatio: item.AspectRatio,
			ColorA: item.Colors[0], ColorB: item.Colors[1], ColorC: item.Colors[2],
			Published: true, SortOrder: index,
		})
	}
	return records
}

func migrateSeededAlbumItems(ctx context.Context, db *gorm.DB) error {
	return db.WithContext(ctx).Transaction(func(transaction *gorm.DB) error {
		var completed int64
		if err := transaction.Model(&contentMigration{}).Where("name = ?", albumSeedMigrationName).Count(&completed).Error; err != nil {
			return fmt.Errorf("check Album Item seed migration: %w", err)
		}
		var existingItems int64
		if completed == 0 {
			if err := transaction.Model(&albumItem{}).Count(&existingItems).Error; err != nil {
				return fmt.Errorf("check existing Album Items: %w", err)
			}
		}
		return applyAlbumSeedMigration(completed > 0, existingItems,
			func(records []albumItem) error { return transaction.Omit("Image").Create(&records).Error },
			func() error { return transaction.Create(&contentMigration{Name: albumSeedMigrationName}).Error },
		)
	})
}

func applyAlbumSeedMigration(completed bool, existingItems int64, seed func([]albumItem) error, markComplete func() error) error {
	if completed {
		return nil
	}
	records := seededAlbumItemRecords()
	if existingItems == 0 && len(records) > 0 {
		if err := seed(records); err != nil {
			return fmt.Errorf("seed initial Album Items: %w", err)
		}
	}
	if err := markComplete(); err != nil {
		return fmt.Errorf("complete Album Item seed migration: %w", err)
	}
	return nil
}

func sortedAlbumItems(items []albumItem) {
	sort.SliceStable(items, func(left, right int) bool {
		if items[left].SortOrder != items[right].SortOrder {
			return items[left].SortOrder < items[right].SortOrder
		}
		return items[left].ID < items[right].ID
	})
}

func publishedAlbumItems(items []albumItem) []albumItem {
	published := make([]albumItem, 0, len(items))
	for _, item := range items {
		if item.Published {
			published = append(published, item)
		}
	}
	sortedAlbumItems(published)
	return published
}

func (service appServices) publicGalleryItems(c *fiber.Ctx) error {
	if service.albumItems == nil {
		return c.JSON(seededGalleryItems)
	}
	items, err := service.albumItems.List(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "list Album Items"})
	}
	published := publishedAlbumItems(items)
	response := make([]publicAlbumItemResponse, 0, len(published))
	for _, item := range published {
		response = append(response, service.publicAlbumItem(item))
	}
	return c.JSON(response)
}

func (service appServices) adminGalleryItems(c *fiber.Ctx) error {
	items, err := service.albumItems.List(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "list Album Items"})
	}
	sortedAlbumItems(items)
	response := make([]adminAlbumItemResponse, 0, len(items))
	for _, item := range items {
		response = append(response, service.adminAlbumItem(item))
	}
	return c.JSON(response)
}

func (service appServices) createAlbumItem(c *fiber.Ctx) error {
	item, err := service.parseAlbumItemRequest(c)
	if err != nil {
		return albumItemRequestFailure(c, err)
	}
	item, err = service.albumItems.Save(c.Context(), item)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "create Album Item"})
	}
	return c.Status(fiber.StatusCreated).JSON(service.adminAlbumItem(item))
}

func (service appServices) updateAlbumItem(c *fiber.Ctx) error {
	id, err := parseAlbumItemID(c)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Album Item not found"})
	}
	existing, err := service.albumItems.FindByID(c.Context(), id)
	if errors.Is(err, errAlbumItemNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Album Item not found"})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "find Album Item"})
	}
	item, err := service.parseAlbumItemRequest(c)
	if err != nil {
		return albumItemRequestFailure(c, err)
	}
	item.ID = id
	item.CreatedAt = existing.CreatedAt
	item, err = service.albumItems.Save(c.Context(), item)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "save Album Item"})
	}
	return c.JSON(service.adminAlbumItem(item))
}

func (service appServices) deleteAlbumItem(c *fiber.Ctx) error {
	id, err := parseAlbumItemID(c)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Album Item not found"})
	}
	err = service.albumItems.Delete(c.Context(), id)
	if errors.Is(err, errAlbumItemNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Album Item not found"})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "delete Album Item"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (service appServices) parseAlbumItemRequest(c *fiber.Ctx) (albumItem, error) {
	var request albumItemRequest
	if err := c.BodyParser(&request); err != nil {
		return albumItem{}, albumItemValidationError{"Album Item must be a JSON object"}
	}
	request.Title = strings.TrimSpace(request.Title)
	request.Note = strings.TrimSpace(request.Note)
	request.Alt = strings.TrimSpace(request.Alt)
	request.Year = strings.TrimSpace(request.Year)
	request.Width = strings.TrimSpace(request.Width)
	request.AspectRatio = strings.TrimSpace(request.AspectRatio)
	if utf8.RuneCountInString(request.Title) == 0 || utf8.RuneCountInString(request.Title) > 160 {
		return albumItem{}, albumItemValidationError{"title must contain between 1 and 160 characters"}
	}
	if utf8.RuneCountInString(request.Note) == 0 || utf8.RuneCountInString(request.Note) > 255 {
		return albumItem{}, albumItemValidationError{"note must contain between 1 and 255 characters"}
	}
	if utf8.RuneCountInString(request.Alt) == 0 || utf8.RuneCountInString(request.Alt) > 500 {
		return albumItem{}, albumItemValidationError{"alt must contain between 1 and 500 characters"}
	}
	if !yearPattern.MatchString(request.Year) {
		return albumItem{}, albumItemValidationError{"year must contain four digits"}
	}
	if request.AnchorX < 0 || request.AnchorX > 1 || request.AnchorY < 0 || request.AnchorY > 1 {
		return albumItem{}, albumItemValidationError{"anchor coordinates must be between 0 and 1"}
	}
	if !positiveGalleryWidth(request.Width) {
		return albumItem{}, albumItemValidationError{"width must be expressed in vw"}
	}
	if !positiveAspectRatio(request.AspectRatio) {
		return albumItem{}, albumItemValidationError{"aspectRatio must contain two positive numbers"}
	}
	for _, color := range request.Colors {
		if !galleryColorPattern.MatchString(color) {
			return albumItem{}, albumItemValidationError{"colors must contain three hexadecimal colors"}
		}
	}
	if request.SortOrder < 0 {
		return albumItem{}, albumItemValidationError{"sortOrder must not be negative"}
	}
	image, err := service.mediaRecords.FindByID(c.Context(), request.ImageMediaID)
	if err != nil {
		if errors.Is(err, errMediaObjectNotFound) {
			return albumItem{}, albumItemValidationError{"imageMediaId must reference registered image media"}
		}
		return albumItem{}, fmt.Errorf("find Album Item image: %w", err)
	}
	if image.Kind != mediaKindImage || image.Width <= 0 || image.Height <= 0 {
		return albumItem{}, albumItemValidationError{"imageMediaId must reference registered image media"}
	}
	imageID := image.ID
	return albumItem{
		Title: request.Title, Note: request.Note, Alt: request.Alt, Year: request.Year,
		ImageMediaID: &imageID, Image: image, AnchorX: request.AnchorX, AnchorY: request.AnchorY,
		Width: request.Width, AspectRatio: request.AspectRatio,
		ColorA: request.Colors[0], ColorB: request.Colors[1], ColorC: request.Colors[2],
		Published: request.Published, SortOrder: request.SortOrder,
	}, nil
}

func albumItemRequestFailure(c *fiber.Ctx, err error) error {
	var validation albumItemValidationError
	if errors.As(err, &validation) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": validation.Error()})
	}
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "resolve Album Item media"})
}

func positiveGalleryWidth(value string) bool {
	matches := galleryWidthPattern.FindStringSubmatch(value)
	if len(matches) != 2 {
		return false
	}
	width, err := strconv.ParseFloat(matches[1], 64)
	return err == nil && width > 0
}

func positiveAspectRatio(value string) bool {
	matches := aspectRatioPattern.FindStringSubmatch(value)
	if len(matches) != 3 {
		return false
	}
	width, widthError := strconv.ParseFloat(matches[1], 64)
	height, heightError := strconv.ParseFloat(matches[2], 64)
	return widthError == nil && heightError == nil && width > 0 && height > 0
}

func (service appServices) publicAlbumItem(item albumItem) publicAlbumItemResponse {
	imageURL := ""
	if item.ImageMediaID != nil && service.media != nil {
		imageURL = service.media.publicURL(item.Image.ObjectKey)
	}
	return publicAlbumItemResponse{
		ID: fmt.Sprintf("A-%02d", item.ID), Title: item.Title, Note: item.Note, Alt: item.Alt,
		Year: item.Year, ImageURL: imageURL,
		AnchorX: item.AnchorX, AnchorY: item.AnchorY, Width: item.Width, AspectRatio: item.AspectRatio,
		Colors: [3]string{item.ColorA, item.ColorB, item.ColorC},
	}
}

func (service appServices) adminAlbumItem(item albumItem) adminAlbumItemResponse {
	var image *mediaResponse
	if item.ImageMediaID != nil && service.media != nil {
		response := service.media.response(item.Image)
		image = &response
	}
	return adminAlbumItemResponse{
		ID: item.ID, Title: item.Title, Note: item.Note, Alt: item.Alt, Year: item.Year,
		Image: image, AnchorX: item.AnchorX, AnchorY: item.AnchorY,
		Width: item.Width, AspectRatio: item.AspectRatio,
		Colors:    [3]string{item.ColorA, item.ColorB, item.ColorC},
		Published: item.Published, SortOrder: item.SortOrder,
	}
}

func parseAlbumItemID(c *fiber.Ctx) (uint, error) {
	id, err := strconv.ParseUint(c.Params("id"), 10, 0)
	if err != nil || id == 0 {
		return 0, errors.New("invalid Album Item ID")
	}
	return uint(id), nil
}
