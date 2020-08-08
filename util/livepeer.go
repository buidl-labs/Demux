package util

import (
	"encoding/json"
	golog "log"
	"math/big"
)

// OrchestratorStat is an object that is received from the livepeer pricing tool.
type OrchestratorStat struct {
	Address           string   `json:"Address"`
	ServiceURI        string   `json:"ServiceURI"`
	LastRewardRound   int      `json:"LastRewardRound"`
	RewardCut         int      `json:"RewardCut"`
	FeeShare          int      `json:"FeeShare"`
	DelegatedStake    *big.Int `json:"DelegatedStake"`
	ActivationRound   int      `json:"ActivationRound"`
	DeactivationRound *big.Int `json:"DeactivationRound"`
	Active            bool     `json:"Active"`
	Status            string   `json:"Status"`
	PricePerPixel     float64  `json:"PricePerPixel"`
	UpdatedAt         int      `json:"UpdatedAt"`
}

// GetOrchestratorStats returns an array of OrchestratorStats
func GetOrchestratorStats(body []byte) ([]OrchestratorStat, error) {
	var s = new([]OrchestratorStat)
	err := json.Unmarshal(body, &s)
	if err != nil {
		golog.Println(err)
	}
	return *s, err
}

// GetPixelsInRendition returns the number of pixels in a particular rendition of a video.
func GetPixelsInRendition(width int, height int, fps int, numStreams int, duration int) int {
	return width * height * fps * numStreams * duration
}

// GetTotalPixels returns the total pixels for the 3 renditions combined.
// :param duration: duration of video in seconds
func GetTotalPixels(duration int) int {
	pixels1080p := GetPixelsInRendition(1920, 1080, 30, 1, duration)
	pixels720p := GetPixelsInRendition(1280, 720, 30, 1, duration)
	pixels360p := GetPixelsInRendition(640, 360, 30, 1, duration)

	return pixels1080p + pixels720p + pixels360p
}
