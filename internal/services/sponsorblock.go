package services

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"

	log "github.com/labstack/gommon/log"
)

const SPONSORBLOCK_API_URL = "https://sponsor.ajay.app/api/skipSegments?videoID="

func TotalSponsorTimeSkipped(youtubeVideoId string) float64 {
	log.Debug("[SponsorBlock] Looking up podcast in SponsorBlock API...")
	endURL := SPONSORBLOCK_API_URL + youtubeVideoId
	
	if categories := getCategories(); categories != nil {
		for _, category := range categories {
			endURL += "&category=" + strings.TrimSpace(category)
		}
	}

	resp, err := http.Get(endURL)
	if err != nil {
		log.Error(err)
		return 0
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		log.Errorf("Video not found on SponsorBlock API: %s", youtubeVideoId)
		return 0
	}

	body, bodyErr := io.ReadAll(resp.Body)
	if bodyErr != nil {
		log.Error(bodyErr)
		return 0
	}
	sponsorBlockResponse, marshErr := unmarshalSponsorBlockResponse(body)
	if marshErr != nil {
		log.Error(marshErr)
		return 0
	}

	totalTimeSkipped := calculateSkippedTime(sponsorBlockResponse)

	return totalTimeSkipped
}

func unmarshalSponsorBlockResponse(data []byte) ([]SponsorBlockResponse, error) {
	var res []SponsorBlockResponse

	if err := json.Unmarshal(data, &res); err != nil {
		return []SponsorBlockResponse{}, err
	}

	return res, nil
}

func calculateSkippedTime(segments []SponsorBlockResponse) float64 {
	skippedTime := float64(0)
	prevStopTime := float64(0)

	for _, segment := range segments {
		startTime := segment.Segment[0]
		stopTime := segment.Segment[1]

		if startTime > prevStopTime {
			skippedTime += stopTime - startTime
		} else {
			skippedTime += stopTime - prevStopTime
		}

		prevStopTime = stopTime
	}

	return skippedTime
}

func getCategories() []string {
	categories := os.Getenv("SPONSORBLOCK_CATEGORIES")
	if categories == "" {
		return nil
	}
	return strings.Split(categories, ",")
}

type SponsorBlockResponse struct {
	Segment       []float64 `json:"segment"`
	UUID          string    `json:"UUID"`
	Category      string    `json:"category"`
	VideoDuration float64   `json:"videoDuration"`
	ActionType    string    `json:"actionType"`
	Locked        int16     `json:"locked"`
	Votes         int16     `json:"votes"`
	Description   string    `json:"description"`
}
