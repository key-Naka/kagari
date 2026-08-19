package main

import (
	"context"

	"github.com/gofiber/fiber/v2"
)

type homeCollectionSummary[T any] struct {
	Availability availability `json:"availability"`
	Count        int          `json:"count"`
	Item         *T           `json:"item,omitempty"`
}

type homeProjectSummary struct {
	Title       string `json:"title"`
	CoverURL    string `json:"coverUrl"`
	Description string `json:"description"`
	Featured    bool   `json:"featured"`
}

type homePostSummary struct {
	Title       string `json:"title"`
	Summary     string `json:"summary"`
	PublishedAt string `json:"publishedAt"`
}

type homeTrackSummary struct {
	Title    string `json:"title"`
	CoverURL string `json:"coverUrl"`
}

type homeGitHubSummary struct {
	Repository  string `json:"repository"`
	Description string `json:"description"`
}

type homeVisitorMessageSummary struct {
	Nickname string `json:"nickname"`
	Content  string `json:"content"`
}

type homeGallerySummary struct {
	Availability availability `json:"availability"`
	Count        int          `json:"count"`
}

type homeStatusSummary struct {
	Availability availability `json:"availability"`
	Operational  int          `json:"operational"`
	Total        int          `json:"total"`
}

type homeArchiveResponse struct {
	Works           homeCollectionSummary[homeProjectSummary]        `json:"works"`
	Blog            homeCollectionSummary[homePostSummary]           `json:"blog"`
	Music           homeCollectionSummary[homeTrackSummary]          `json:"music"`
	GitHub          homeCollectionSummary[homeGitHubSummary]         `json:"github"`
	Gallery         homeGallerySummary                               `json:"gallery"`
	Status          homeStatusSummary                                `json:"status"`
	VisitorMessages homeCollectionSummary[homeVisitorMessageSummary] `json:"visitorMessages"`
}

func unavailableHomeCollection[T any]() homeCollectionSummary[T] {
	return homeCollectionSummary[T]{Availability: availabilityUnavailable}
}

func (service appServices) homeArchive(c *fiber.Ctx) error {
	ctx := c.Context()
	return c.JSON(homeArchiveResponse{
		Works:           service.homeProjects(ctx),
		Blog:            service.homePosts(ctx),
		Music:           service.homeTracks(ctx),
		GitHub:          service.homeGitHub(ctx),
		Gallery:         service.homeGallery(ctx),
		Status:          service.homeStatus(ctx),
		VisitorMessages: service.homeVisitorMessages(ctx),
	})
}

func (service appServices) homeGallery(ctx context.Context) homeGallerySummary {
	if service.albumItems == nil {
		return homeGallerySummary{Availability: availabilityOperational, Count: len(seededGalleryItems)}
	}
	items, err := service.albumItems.List(ctx)
	if err != nil {
		return homeGallerySummary{Availability: availabilityUnavailable}
	}
	return homeGallerySummary{
		Availability: availabilityOperational,
		Count:        len(publishedAlbumItems(items)),
	}
}

func (service appServices) homeProjects(ctx context.Context) homeCollectionSummary[homeProjectSummary] {
	if service.projects == nil {
		return unavailableHomeCollection[homeProjectSummary]()
	}
	projects, err := service.projects.List(ctx)
	if err != nil {
		return unavailableHomeCollection[homeProjectSummary]()
	}
	published := publishedProjects(projects)
	summary := homeCollectionSummary[homeProjectSummary]{
		Availability: availabilityOperational,
		Count:        len(published),
	}
	if len(published) == 0 {
		return summary
	}
	project, err := publicProject(published[0])
	if err != nil {
		return unavailableHomeCollection[homeProjectSummary]()
	}
	summary.Item = &homeProjectSummary{
		Title:       project.Title,
		CoverURL:    project.CoverURL,
		Description: project.Description,
		Featured:    project.Featured,
	}
	return summary
}

func (service appServices) homePosts(ctx context.Context) homeCollectionSummary[homePostSummary] {
	if service.posts == nil {
		return unavailableHomeCollection[homePostSummary]()
	}
	posts, err := service.posts.List(ctx)
	if err != nil {
		return unavailableHomeCollection[homePostSummary]()
	}
	published, err := publishedPosts(posts, "", "")
	if err != nil {
		return unavailableHomeCollection[homePostSummary]()
	}
	summary := homeCollectionSummary[homePostSummary]{
		Availability: availabilityOperational,
		Count:        len(published),
	}
	if len(published) == 0 {
		return summary
	}
	post, err := publicPost(published[0])
	if err != nil {
		return unavailableHomeCollection[homePostSummary]()
	}
	summary.Item = &homePostSummary{
		Title:       post.Title,
		Summary:     post.Summary,
		PublishedAt: post.PublishedAt,
	}
	return summary
}

func (service appServices) homeTracks(ctx context.Context) homeCollectionSummary[homeTrackSummary] {
	if service.tracks == nil || service.media == nil {
		return unavailableHomeCollection[homeTrackSummary]()
	}
	tracks, err := service.tracks.List(ctx)
	if err != nil {
		return unavailableHomeCollection[homeTrackSummary]()
	}
	enabled := enabledTracks(tracks)
	summary := homeCollectionSummary[homeTrackSummary]{
		Availability: availabilityOperational,
		Count:        len(enabled),
	}
	if len(enabled) == 0 {
		return summary
	}
	track := service.publicTrack(enabled[0])
	summary.Item = &homeTrackSummary{Title: track.Title, CoverURL: track.Cover.PublicURL}
	return summary
}

func (service appServices) homeGitHub(ctx context.Context) homeCollectionSummary[homeGitHubSummary] {
	data := service.github.current(ctx)
	summary := homeCollectionSummary[homeGitHubSummary]{
		Availability: data.Availability,
		Count:        len(data.Repositories),
	}
	var repository string
	if len(data.Activities) > 0 {
		repository = data.Activities[0].Repository
	}
	var description string
	if len(data.Repositories) > 0 {
		if repository == "" {
			repository = data.Repositories[0].Name
		}
		description = data.Repositories[0].Description
	}
	if repository != "" {
		summary.Item = &homeGitHubSummary{Repository: repository, Description: description}
	}
	return summary
}

func (service appServices) homeStatus(ctx context.Context) homeStatusSummary {
	status := service.status.current(ctx)
	operational := 0
	for _, application := range status.Applications {
		if application.State == availabilityOperational {
			operational++
		}
	}
	return homeStatusSummary{
		Availability: status.Availability,
		Operational:  operational,
		Total:        len(status.Applications),
	}
}

func (service appServices) homeVisitorMessages(ctx context.Context) homeCollectionSummary[homeVisitorMessageSummary] {
	if service.visitorMessages == nil {
		return unavailableHomeCollection[homeVisitorMessageSummary]()
	}
	messages, err := service.visitorMessages.ListNewestFirst(ctx)
	if err != nil {
		return unavailableHomeCollection[homeVisitorMessageSummary]()
	}
	summary := homeCollectionSummary[homeVisitorMessageSummary]{
		Availability: availabilityOperational,
		Count:        len(messages),
	}
	if len(messages) == 0 {
		return summary
	}
	message := publicVisitorMessage(messages[0])
	summary.Item = &homeVisitorMessageSummary{Nickname: message.Nickname, Content: message.Content}
	return summary
}
