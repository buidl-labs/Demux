package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strconv"

	"github.com/buidl-labs/Demux/dataservice"
	"github.com/buidl-labs/Demux/util"

	log "github.com/sirupsen/logrus"
	powc "github.com/textileio/powergate/api/client"
	// "github.com/textileio/powergate/ffs"
	// "github.com/textileio/powergate/health"
)

type Data struct {
	StorageDuration int64 `json:"storage_duration"`
	VideoFileSize   int64 `json:"video_file_size"`
	VideoDuration   int64 `json:"video_duration"`
}

// PriceEstimateHandler handles the /pricing endpoint
func PriceEstimateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {

		var responded = false

		decoder := json.NewDecoder(r.Body)
		var d Data
		err := decoder.Decode(&d)
		if err != nil {
			fmt.Println(err)
			responded = true
			return
		}
		log.Println(d.StorageDuration)
		log.Println(d.VideoFileSize)
		log.Println(d.VideoDuration)
		videoDurationInt := d.VideoDuration
		storageDurationInt := d.StorageDuration
		videoFileSize := uint64(d.VideoFileSize)

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

		// storageDuration := r.FormValue("storage_duration")
		// storageDurationInt, _ := strconv.ParseInt(storageDuration, 10, 64)
		// fmt.Println("storageDurationInt", storageDurationInt)
		// // 1 month <= storageDurationInt <= 10 years
		// if storageDurationInt > 315360000 || storageDurationInt < 2628003 {
		// 	w.WriteHeader(http.StatusExpectationFailed)
		// 	data := map[string]interface{}{
		// 		"error": "please specify a value of `storage_duration` between 2628003 and 315360000",
		// 	}
		// 	util.WriteResponse(data, w)
		// 	responded = true
		// 	return
		// }

		// videoDuration := r.FormValue("video_duration")
		// videoDurationInt, _ := strconv.ParseInt(videoDuration, 10, 64)
		// fmt.Println("videoDurationInt", videoDurationInt)
		// if videoDurationInt < 1 {
		// 	w.WriteHeader(http.StatusExpectationFailed)
		// 	data := map[string]interface{}{
		// 		"error": "please specify a value of `video_duration` >= 1",
		// 	}
		// 	util.WriteResponse(data, w)
		// 	responded = true
		// 	return
		// }

		// videoFileSizeStr := r.FormValue("video_file_size")
		// videoFileSizeInt, _ := strconv.ParseInt(videoFileSizeStr, 10, 64)
		// fmt.Println("videoFileSizeInt", videoFileSizeInt)
		// if videoFileSizeInt <= 0 {
		// 	w.WriteHeader(http.StatusExpectationFailed)
		// 	data := map[string]interface{}{
		// 		"error": "please specify a value of `video_file_size` > 0",
		// 	}
		// 	util.WriteResponse(data, w)
		// 	responded = true
		// 	return
		// }
		// videoFileSize := uint64(videoFileSizeInt)
		// log.Println("videoFileSize:", videoFileSize)

		transcodingCostWEI, err := util.CalculateTranscodingCost("", float64(videoDurationInt))
		if err != nil {
			// TODO: handle this case
			log.Println(err)
			transcodingCostWEI = big.NewInt(0)
		}

		// TODO: Calculate powergate (filecoin) storage price

		estimatedPrice := uint64(0)

		duration := uint64(storageDurationInt) //duration of deal in seconds (provided by user)
		epochs := uint64(duration / 30)
		folderSize := getFolderSizeEstimate(videoFileSize) //size of folder in MiB (to be predicted by estimation algorithm)
		fmt.Println("folderSize", folderSize, "videoFileSize", videoFileSize)

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
			log.Errorf("getting asks: %s", err)
		}
		if len(index.Storage) > 0 {
			log.Printf("Storage median price: %v\n", index.StorageMedianPrice)
			log.Printf("Last updated: %v\n", index.LastUpdated.Format("01/02/06 15:04 MST"))
			fmt.Println("index:\n", index)
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
				// fmt.Printf("ask %d: %v\n", i, data[i])
				i++
			}
			meanEpochPrice := uint64(pricesSum / len(index.Storage))
			fmt.Println("pricesSum", pricesSum)
			fmt.Println("meanEpochPrice", meanEpochPrice)
			estimatedPrice = meanEpochPrice * epochs * folderSize / 1024
			fmt.Println("estimatedPrice", estimatedPrice)
		}

		// TODO: Convert total price to USD and return

		if responded == false {
			w.WriteHeader(http.StatusOK)
			data := map[string]interface{}{
				"transcoding_cost_estimated": transcodingCostWEI,
				"storage_cost_estimated":     estimatedPrice,
			}
			util.WriteResponse(data, w)
		}
	}
}
func getFolderSizeEstimate(fileSize uint64) uint64 {
	msr := dataservice.GetMeanSizeRatio()
	fmt.Println("MeanSizeRatio", msr.MeanSizeRatio)
	fmt.Println("fileSize", fileSize)

	return fileSize * uint64(msr.MeanSizeRatio)
}
