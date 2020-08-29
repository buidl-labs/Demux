package util

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"os"
	"os/exec"
	"strconv"
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
		return nil, err
	}
	return *s, nil
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

// CalculateTranscodingCost computes the transcoding cost
// of a video in wei and returns it.
func CalculateTranscodingCost(fileName string) (string, error) {
	stdout, err := exec.Command("ffprobe", "-i", fileName, "-show_entries", "format=duration", "-v", "quiet", "-of", "csv=p=0").Output()
	if err != nil {
		return "", fmt.Errorf("finding video duration: %s", err)
	}
	duration, err := strconv.ParseFloat(string(stdout)[:len(string(stdout))-2], 64)
	if err != nil {
		return "", fmt.Errorf("finding video duration: %s", err)
	}

	// Fetch orchestrator stats from livepeer pricing tool:
	// GET https://livepeer-pricing-tool.com/orchestratorStats

	livepeerPricingToolURL, livepeerPricingToolURLExists := os.LookupEnv("LIVEPEER_PRICING_TOOL")
	if !livepeerPricingToolURLExists {
		return "", fmt.Errorf("`LIVEPEER_PRICING_TOOL` env variable not provided")
	}

	var orchestratorStats string
	if livepeerPricingToolURL[len(livepeerPricingToolURL)-1:] == "/" {
		orchestratorStats = livepeerPricingToolURL + "orchestratorStats"
	} else {
		orchestratorStats = livepeerPricingToolURL + "/orchestratorStats"
	}

	resp, err := http.Get(orchestratorStats)
	if err != nil {
		return "", fmt.Errorf("couldn't fetch orchestrator stats: %s", err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading the orchestrator stats: %s", err)
	}

	orchStats, err := GetOrchestratorStats([]byte(body))
	if err != nil {
		return "", fmt.Errorf("getting the orchestrator stats: %s", err)
	}

	weightSum := big.NewInt(0)
	productSum := big.NewFloat(0)

	for _, i := range orchStats {
		stake := new(big.Float).SetInt(i.DelegatedStake)
		ppp := new(big.Float).SetFloat64(i.PricePerPixel)
		product := stake.Mul(stake, ppp)

		weightSum.Add(weightSum, i.DelegatedStake)
		productSum.Add(productSum, product)
	}

	// Calculate weighted average price per pixel (weighted by Delegated Stake)

	// Weighted pricePerPixel
	pricePerPixel := new(big.Float).Quo(productSum, new(big.Float).SetInt(weightSum))

	pixels := GetTotalPixels(int(duration))

	// Calculate livepeer price for uploaded video

	livepeerPrice := new(big.Float).SetInt(big.NewInt(int64(1)))
	livepeerPrice = livepeerPrice.Mul(new(big.Float).SetInt(big.NewInt(int64(pixels))), pricePerPixel)

	// Transcoding cost of the video in wei
	transcodingCostWEI := livepeerPrice.String()

	return transcodingCostWEI, nil
}
