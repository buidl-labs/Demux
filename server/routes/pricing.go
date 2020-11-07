package routes

import (
	"encoding/json"
	"math/big"
	"net/http"

	"github.com/buidl-labs/Demux/dataservice"
	"github.com/buidl-labs/Demux/util"

	log "github.com/sirupsen/logrus"
)

// VideoData contains some data which is used to predict the streaming cost.
type VideoData struct {
	StorageDuration int64 `json:"storage_duration"`
	VideoFileSize   int64 `json:"video_file_size"`
	VideoDuration   int64 `json:"video_duration"`
}

// PriceEstimateHandler handles the /pricing endpoint
func PriceEstimateHandler(w http.ResponseWriter, r *http.Request, msr dataservice.MeanSizeRatioDatabase) {
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

		transcodingCostWEI := big.NewInt(0)
		transcodingCostWEI, err = util.CalculateTranscodingCost("", float64(videoDurationInt))
		if err != nil {
			log.Warn("Couldn't calculate transcoding cost: ", err)
			// Couldn't calculate transcoding cost. Let it be 0.
		}

		// Calculate powergate (filecoin) storage price

		storageCostEstimated := big.NewInt(0)

		duration := float64(storageDurationInt) // duration of deal in seconds (provided by user)
		epochs := float64(duration / float64(30))
		folderSize, err := GetFolderSizeEstimate(float64(videoFileSize), msr) // size of folder in MiB (to be predicted by estimation algorithm)
		if err != nil {
			w.WriteHeader(http.StatusExpectationFailed)
			data := map[string]interface{}{
				"error": "estimating folder size",
			}
			util.WriteResponse(data, w)
			responded = true
			return
		}
		log.Info("folderSize", folderSize, "videoFileSize", videoFileSize)
		log.Info("duration", duration, "epochs", epochs)

		// Calculate storage cost of the video
		storageCostEstimated, err = util.CalculateStorageCost(uint64(folderSize), int64(duration))
		if err != nil {
			log.Warn("Couldn't calculate storage cost: ", err)
			// Couldn't calculate storage cost. Let it be 0.
		}

		// TODO: Convert total price to USD and return

		if responded == false {
			w.WriteHeader(http.StatusOK)
			data := map[string]interface{}{
				"transcoding_cost_estimated": transcodingCostWEI,
				"storage_cost_estimated":     storageCostEstimated,
			}
			util.WriteResponse(data, w)
		}
	}
}

// GetFolderSizeEstimate estimates the folderSize
// (after transcoding) of an mp4 video using the meanSizeRatio.
func GetFolderSizeEstimate(fileSize float64, msr dataservice.MeanSizeRatioDatabase) (float64, error) {
	msrObj, err := msr.GetMeanSizeRatio()

	return fileSize * float64(msrObj.MeanSizeRatio), err
}
