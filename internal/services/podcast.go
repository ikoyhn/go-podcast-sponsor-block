package services

import (
	"errors"
	"ikoyhn/podcast-sponsorblock/internal/database"
	"ikoyhn/podcast-sponsorblock/internal/models"
)

func GetAllPodcasts() ([]models.Podcast, error) {
	podcasts := database.GetAllPodcasts()
	if len(podcasts) == 0 {
		return nil, errors.New("no podcasts found")
	}
	return podcasts, nil
}
