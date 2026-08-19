package main

import (
	_ "embed"
	"encoding/json"

	"github.com/gofiber/fiber/v2"
)

type galleryItem struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Note        string    `json:"note"`
	Alt         string    `json:"alt"`
	Year        string    `json:"year"`
	AnchorX     float64   `json:"anchorX"`
	AnchorY     float64   `json:"anchorY"`
	Width       string    `json:"width"`
	AspectRatio string    `json:"aspectRatio"`
	Colors      [3]string `json:"colors"`
}

//go:embed gallery_items.json
var galleryItemsJSON []byte

var seededGalleryItems = decodeSeededGalleryItems(galleryItemsJSON)

func decodeSeededGalleryItems(encoded []byte) []galleryItem {
	items := []galleryItem{}
	if err := json.Unmarshal(encoded, &items); err != nil {
		panic("decode seeded gallery items: " + err.Error())
	}
	return items
}

func (appServices) publicGalleryItems(c *fiber.Ctx) error {
	return c.JSON(seededGalleryItems)
}
