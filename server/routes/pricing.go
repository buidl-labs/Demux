package routes

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"strconv"

	"github.com/buidl-labs/Demux/dataservice"
	"github.com/buidl-labs/Demux/util"

	log "github.com/sirupsen/logrus"
	powc "github.com/textileio/powergate/api/client"
)

// VideoData contains some data which is used to predict the streaming cost.
type VideoData struct {
	StorageDuration int64 `json:"storage_duration"`
	VideoFileSize   int64 `json:"video_file_size"`
	VideoDuration   int64 `json:"video_duration"`
}

// PriceEstimateHandler handles the /pricing endpoint
func PriceEstimateHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "POST" {
		var responded = false

		decoder := json.NewDecoder(r.Body)
		var d VideoData
		err := decoder.Decode(&d)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			data := map[string]interface{}{
				"error": "please specify a value of `storage_duration` between 2628003 and 315360000",
			}
			util.WriteResponse(data, w)
			log.Errorln(err)
			responded = true
			return
		}

		if d.StorageDuration > 315360000 || d.StorageDuration < 2628003 {
			w.WriteHeader(http.StatusExpectationFailed)
			data := map[string]interface{}{
				"error": "please specify a value of `storage_duration` between 2628003 and 315360000",
			}
			util.WriteResponse(data, w)
			responded = true
			return
		}
		if d.VideoDuration < 1 {
			w.WriteHeader(http.StatusExpectationFailed)
			data := map[string]interface{}{
				"error": "please specify a value of `video_duration` >= 1",
			}
			util.WriteResponse(data, w)
			responded = true
			return
		}
		if d.VideoFileSize <= 0 {
			w.WriteHeader(http.StatusExpectationFailed)
			data := map[string]interface{}{
				"error": "please specify a value of `video_file_size` > 0",
			}
			util.WriteResponse(data, w)
			responded = true
			return
		}

		videoDurationInt := d.VideoDuration
		storageDurationInt := d.StorageDuration
		videoFileSize := uint64(d.VideoFileSize)

		transcodingCostWEI, err := util.CalculateTranscodingCost("", float64(videoDurationInt))
		if err != nil {
			log.Warn("Couldn't calculate transcoding cost:", err)
			transcodingCostWEI = big.NewInt(0)
		}

		// Calculate powergate (filecoin) storage price

		estimatedPrice := float64(0)

		duration := float64(storageDurationInt) // duration of deal in seconds (provided by user)
		epochs := float64(duration / float64(30))
		folderSize := getFolderSizeEstimate(float64(videoFileSize)) // size of folder in MiB (to be predicted by estimation algorithm)
		log.Info("folderSize", folderSize, "videoFileSize", videoFileSize)
		log.Info("duration", duration, "epochs", epochs)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		pgClient, _ := powc.NewClient(util.InitialPowergateSetup.PowergateAddr)
		defer func() {
			if err := pgClient.Close(); err != nil {
				log.Errorf("closing powergate client: %s", err)
			}
		}()

		index, err := pgClient.Asks.Get(ctx)
		if err != nil {
			log.Warn("getting asks:", err)
		}
		if len(index.Storage) > 0 {
			data := make([][]string, len(index.Storage))
			i := 0
			pricesSum := 0
			for _, ask := range index.Storage {
				pricesSum += int(ask.Price)
				data[i] = []string{
					ask.Miner,
					strconv.Itoa(int(ask.Price)),
					strconv.Itoa(int(ask.MinPieceSize)),
					strconv.FormatInt(ask.Timestamp, 10),
					strconv.FormatInt(ask.Expiry, 10),
				}
				i++
			}
			meanEpochPrice := float64(float64(pricesSum) / float64(len(index.Storage)))
			estimatedPrice = meanEpochPrice * float64(epochs) * folderSize / float64(1024)
			log.Info("estimatedPrice", estimatedPrice, ", meanEpochPrice", meanEpochPrice, ", pricesSum", pricesSum)
		}

		// TODO: Convert total price to USD and return

		if responded == false {
			w.WriteHeader(http.StatusOK)
			data := map[string]interface{}{
				"transcoding_cost_estimated": transcodingCostWEI,
				"storage_cost_estimated":     int64(estimatedPrice),
			}
			util.WriteResponse(data, w)
		}
	}
}
func getFolderSizeEstimate(fileSize float64) float64 {
	msr := dataservice.GetMeanSizeRatio()

	return fileSize * float64(msr.MeanSizeRatio)
}
