package main

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

var errTrackNotFound = errors.New("track not found")

type trackValidationError struct{ message string }

func (err trackValidationError) Error() string { return err.message }

type track struct {
	ID           uint        `gorm:"primaryKey"`
	Title        string      `gorm:"size:160;not null"`
	CoverMediaID uint        `gorm:"not null;index"`
	Cover        mediaRecord `gorm:"foreignKey:CoverMediaID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	AudioMediaID uint        `gorm:"not null;index"`
	Audio        mediaRecord `gorm:"foreignKey:AudioMediaID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	Enabled      bool        `gorm:"not null;index"`
	SortOrder    int         `gorm:"not null;index"`
	CreatedAt    time.Time   `gorm:"not null"`
	UpdatedAt    time.Time   `gorm:"not null"`
}

type trackRepository interface {
	List(context.Context) ([]track, error)
	FindByID(context.Context, uint) (track, error)
	Save(context.Context, track) (track, error)
}

type gormTrackRepository struct{ db *gorm.DB }

func (repository gormTrackRepository) List(ctx context.Context) ([]track, error) {
	var tracks []track
	if err := repository.db.WithContext(ctx).Preload("Cover").Preload("Audio").Find(&tracks).Error; err != nil {
		return nil, fmt.Errorf("list tracks: %w", err)
	}
	return tracks, nil
}

func (repository gormTrackRepository) FindByID(ctx context.Context, id uint) (track, error) {
	var value track
	if err := repository.db.WithContext(ctx).Preload("Cover").Preload("Audio").First(&value, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return track{}, errTrackNotFound
		}
		return track{}, fmt.Errorf("find track: %w", err)
	}
	return value, nil
}

func (repository gormTrackRepository) Save(ctx context.Context, value track) (track, error) {
	if value.ID == 0 {
		if err := repository.db.WithContext(ctx).Omit("Cover", "Audio").Create(&value).Error; err != nil {
			return track{}, fmt.Errorf("create track: %w", err)
		}
	} else if err := repository.db.WithContext(ctx).Omit("Cover", "Audio").Save(&value).Error; err != nil {
		return track{}, fmt.Errorf("save track: %w", err)
	}
	return repository.FindByID(ctx, value.ID)
}

type publicTrackResponse struct {
	ID        uint          `json:"id"`
	Title     string        `json:"title"`
	Cover     mediaResponse `json:"cover"`
	Audio     mediaResponse `json:"audio"`
	SortOrder int           `json:"sortOrder"`
}

type trackRequest struct {
	Title        string `json:"title"`
	CoverMediaID uint   `json:"coverMediaId"`
	AudioMediaID uint   `json:"audioMediaId"`
	Enabled      bool   `json:"enabled"`
	SortOrder    int    `json:"sortOrder"`
}

type adminTrackResponse struct {
	ID        uint          `json:"id"`
	Title     string        `json:"title"`
	Cover     mediaResponse `json:"cover"`
	Audio     mediaResponse `json:"audio"`
	Enabled   bool          `json:"enabled"`
	SortOrder int           `json:"sortOrder"`
}

func sortedTracks(tracks []track) {
	sort.SliceStable(tracks, func(left, right int) bool {
		if tracks[left].SortOrder != tracks[right].SortOrder {
			return tracks[left].SortOrder < tracks[right].SortOrder
		}
		return tracks[left].ID < tracks[right].ID
	})
}

func enabledTracks(tracks []track) []track {
	enabled := make([]track, 0, len(tracks))
	for _, value := range tracks {
		if value.Enabled {
			enabled = append(enabled, value)
		}
	}
	sortedTracks(enabled)
	return enabled
}

func (service appServices) publicTracks(c *fiber.Ctx) error {
	tracks, err := service.tracks.List(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "list tracks"})
	}
	enabled := enabledTracks(tracks)
	response := make([]publicTrackResponse, 0, len(enabled))
	for _, value := range enabled {
		response = append(response, service.publicTrack(value))
	}
	return c.JSON(response)
}

func (service appServices) publicTrack(value track) publicTrackResponse {
	return publicTrackResponse{
		ID:        value.ID,
		Title:     value.Title,
		Cover:     service.media.response(value.Cover),
		Audio:     service.media.response(value.Audio),
		SortOrder: value.SortOrder,
	}
}

func (service appServices) adminTracks(c *fiber.Ctx) error {
	tracks, err := service.tracks.List(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "list tracks"})
	}
	sortedTracks(tracks)
	response := make([]adminTrackResponse, 0, len(tracks))
	for _, value := range tracks {
		response = append(response, service.adminTrack(value))
	}
	return c.JSON(response)
}

func (service appServices) createTrack(c *fiber.Ctx) error {
	value, err := service.parseTrackRequest(c)
	if err != nil {
		return trackRequestFailure(c, err)
	}
	value, err = service.tracks.Save(c.Context(), value)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "create track"})
	}
	return c.Status(fiber.StatusCreated).JSON(service.adminTrack(value))
}

func (service appServices) updateTrack(c *fiber.Ctx) error {
	id, err := parseTrackID(c)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "track not found"})
	}
	existing, err := service.tracks.FindByID(c.Context(), id)
	if errors.Is(err, errTrackNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "track not found"})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "find track"})
	}
	value, err := service.parseTrackRequest(c)
	if err != nil {
		return trackRequestFailure(c, err)
	}
	value.ID = id
	value.CreatedAt = existing.CreatedAt
	value, err = service.tracks.Save(c.Context(), value)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "save track"})
	}
	return c.JSON(service.adminTrack(value))
}

func (service appServices) parseTrackRequest(c *fiber.Ctx) (track, error) {
	var request trackRequest
	if err := c.BodyParser(&request); err != nil {
		return track{}, trackValidationError{"track must be a JSON object"}
	}
	request.Title = strings.TrimSpace(request.Title)
	if utf8.RuneCountInString(request.Title) == 0 || utf8.RuneCountInString(request.Title) > 160 {
		return track{}, trackValidationError{"title must contain between 1 and 160 characters"}
	}
	if request.SortOrder < 0 {
		return track{}, trackValidationError{"sortOrder must not be negative"}
	}
	cover, err := service.mediaRecords.FindByID(c.Context(), request.CoverMediaID)
	if err != nil {
		if errors.Is(err, errMediaObjectNotFound) {
			return track{}, trackValidationError{"coverMediaId must reference registered image media"}
		}
		return track{}, fmt.Errorf("find cover media: %w", err)
	}
	if cover.Kind != mediaKindImage {
		return track{}, trackValidationError{"coverMediaId must reference registered image media"}
	}
	audio, err := service.mediaRecords.FindByID(c.Context(), request.AudioMediaID)
	if err != nil {
		if errors.Is(err, errMediaObjectNotFound) {
			return track{}, trackValidationError{"audioMediaId must reference registered audio media with duration"}
		}
		return track{}, fmt.Errorf("find audio media: %w", err)
	}
	if audio.Kind != mediaKindAudio || audio.DurationMs <= 0 {
		return track{}, trackValidationError{"audioMediaId must reference registered audio media with duration"}
	}
	return track{
		Title: request.Title, CoverMediaID: cover.ID, Cover: cover, AudioMediaID: audio.ID, Audio: audio,
		Enabled: request.Enabled, SortOrder: request.SortOrder,
	}, nil
}

func trackRequestFailure(c *fiber.Ctx, err error) error {
	var validation trackValidationError
	if errors.As(err, &validation) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": validation.Error()})
	}
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "resolve track media"})
}

func (service appServices) adminTrack(value track) adminTrackResponse {
	return adminTrackResponse{
		ID: value.ID, Title: value.Title, Cover: service.media.response(value.Cover), Audio: service.media.response(value.Audio),
		Enabled: value.Enabled, SortOrder: value.SortOrder,
	}
}

func parseTrackID(c *fiber.Ctx) (uint, error) {
	id, err := strconv.ParseUint(c.Params("id"), 10, 0)
	if err != nil || id == 0 {
		return 0, errors.New("invalid track ID")
	}
	return uint(id), nil
}
