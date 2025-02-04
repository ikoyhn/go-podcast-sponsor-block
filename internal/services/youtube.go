package services

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	log "github.com/labstack/gommon/log"
	"github.com/lrstanley/go-ytdlp"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

const youtubeVideoUrl = "https://www.youtube.com/watch?v="

var youtubeVideoMutexes = &sync.Map{}

// Get all youtube playlist items and meta data for the RSS feed
func getYoutubeData(youtubePlaylistId string) []*youtube.PlaylistItem {

	log.Info("[RSS FEED] Getting youtube data...")
	ctx := context.Background()

	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		log.Fatalf("GOOGLE_API_KEY is not set")
	}

	service, err := youtube.NewService(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		log.Errorf("Error creating new YouTube client: %v", err)
		return nil
	}
	if service == nil {
		log.Errorf("Failed to create YouTube service: %v", err)
		return nil
	}

	continue_requesting_playlist_items := true
	allItems := []*youtube.PlaylistItem{}
	pageToken := "first_call"
	for continue_requesting_playlist_items {
		call := service.PlaylistItems.List([]string{"snippet", "status"}).
			PlaylistId(youtubePlaylistId).
			MaxResults(50)
		if pageToken != "first_call" {
			call.PageToken(pageToken)
		}

		response, ytAgainErr := call.Do()
		if ytAgainErr != nil {
			log.Fatalf("Error calling YouTube API: %v. Ensure your API key is valid", response)
		}
		if response.HTTPStatusCode != http.StatusOK {
			log.Errorf("YouTube API returned status code %v", response.HTTPStatusCode)
			return nil
		}

		pageToken = response.NextPageToken
		allItems = append(allItems, response.Items...)
		if response.NextPageToken == "" {
			continue_requesting_playlist_items = false
		}
	}
	return allItems
}

// Remove unavailable youtube videos used during the RSS feed generation
func cleanPlaylistItems(items []*youtube.PlaylistItem) []*youtube.PlaylistItem {
	unavailableStatuses := map[string]bool{
		"private":                  true,
		"unlisted":                 true,
		"privacyStatusUnspecified": true,
	}
	cleanItems := make([]*youtube.PlaylistItem, 0)
	for _, item := range items {
		if item.Status != nil {
			if !unavailableStatuses[item.Status.PrivacyStatus] {
				cleanItems = append(cleanItems, item)
			}
		}
	}
	return cleanItems
}

func GetYoutubeVideo(youtubeVideoId string) (string, <-chan struct{}) {
	mutex, ok := youtubeVideoMutexes.Load(youtubeVideoId)
	if !ok {
		mutex = &sync.Mutex{}
		youtubeVideoMutexes.Store(youtubeVideoId, mutex)
	}

	mutex.(*sync.Mutex).Lock()

	// Check if the file is already being processed
	filePath := "/config/audio/" + youtubeVideoId + ".m4a"
	if _, err := os.Stat(filePath); err == nil {
		mutex.(*sync.Mutex).Unlock()
		return youtubeVideoId, make(chan struct{})
	}

	// If not, proceed with the download
	youtubeVideoId = strings.TrimSuffix(youtubeVideoId, ".m4a")
	ytdlp.Install(context.TODO(), nil)

	categories := os.Getenv("SPONSORBLOCK_CATEGORIES")
	if categories == "" {
		categories = "sponsor"
	}
	strings.trimSpace(categories)

	dl := ytdlp.New().
		NoProgress().
		FormatSort("ext::m4a").
		SponsorblockRemove(categories).
		ExtractAudio().
		NoPlaylist().
		FFmpegLocation("/usr/bin/ffmpeg").
		Continue().
		Paths("/config/audio").
		ProgressFunc(500*time.Millisecond, func(prog ytdlp.ProgressUpdate) {
			fmt.Printf(
				"%s @ %s [eta: %s] :: %s\n",
				prog.Status,
				prog.PercentString(),
				prog.ETA(),
				prog.Filename,
			)
		}).
		Output(youtubeVideoId + ".%(ext)s")

	done := make(chan struct{})
	go func() {
		r, err := dl.Run(context.TODO(), youtubeVideoUrl+youtubeVideoId)
		if err != nil {
			log.Errorf("Error downloading YouTube video: %v", err)
		}
		if r.ExitCode != 0 {
			log.Errorf("YouTube video download failed with exit code %d", r.ExitCode)
		}
		mutex.(*sync.Mutex).Unlock()

		close(done)
	}()

	return youtubeVideoId, done
}
