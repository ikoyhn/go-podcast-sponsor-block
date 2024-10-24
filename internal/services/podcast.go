package services

import (
	"flag"
	"github.com/labstack/echo/v4"
	log "github.com/labstack/gommon/log"
	"google.golang.org/api/youtube/v3"
	"gorm.io/gorm"
	"ikoyhn/podcast-sponsorblock/internal/models"
)

var (
	maxResults = flag.Int64("max-results", 50, "Max YouTube results")
)

var UNAVAILABLE_STATUSES = []string{"private", "privacyStatusUnspecified"}

func BuildRssFeed(db *gorm.DB, c echo.Context, youtubePlaylistId string) error {
	log.Info("[RSS FEED] Building rss feed...")
	ytData := getYoutubeData(youtubePlaylistId)
	podcastRss := buildMainPodcast(ytData)
	return GenerateRssFeed(podcastRss, c)
}

func buildMainPodcast(allItems []*youtube.PlaylistItem) models.Podcast {
	allItems = cleanPlaylistItems(allItems)
	item := allItems[0]
	itunesResponse := GetApplePodcastData(item.Snippet.ChannelTitle)
	closestApplePodcastData := findClosestResult(itunesResponse.Results, len(allItems))

	return models.Podcast{
		AppleId:          closestApplePodcastData.CollectionId,
		YoutubePodcastId: item.Snippet.PlaylistId,
		PodcastName:      item.Snippet.ChannelTitle,
		Description:      item.Snippet.Description,
		Category:         closestApplePodcastData.PrimaryGenreName,
		Language:         "en",
		PostedDate:       closestApplePodcastData.ReleaseDate,
		ImageUrl:         closestApplePodcastData.ArtworkUrl100,
		PodcastEpisodes:  buildPodcastEpisodes(allItems),
	}
}

func buildPodcastEpisodes(allItems []*youtube.PlaylistItem) []models.PodcastEpisode {
	podcastEpisodes := []models.PodcastEpisode{}
	for _, item := range allItems {
		tempPodcast := models.PodcastEpisode{
			YoutubeVideoId:     item.Snippet.ResourceId.VideoId,
			EpisodeName:        item.Snippet.Title,
			EpisodeDescription: item.Snippet.Description,
			Position:           item.Snippet.Position,
		}
		podcastEpisodes = append(podcastEpisodes, tempPodcast)

	}
	return podcastEpisodes
}
