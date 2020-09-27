package routes

import (
	"context"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/buidl-labs/Demux/dataservice"
	"github.com/buidl-labs/Demux/model"
	"github.com/buidl-labs/Demux/util"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	powc "github.com/textileio/powergate/api/client"
	// "github.com/textileio/powergate/ffs"
	// "github.com/textileio/powergate/health"
)

// type Data struct {
// 	StorageDuration int64
// 	VideoFileSize   int64
// }

// PriceEstimateHandler handles the /pricing endpoint
func PriceEstimateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {

		var responded = false

		// decoder := json.NewDecoder(r.Body)
		// var d Data
		// err := decoder.Decode(&d)
		// if err != nil {
		// 	// panic(err)
		// 	fmt.Println(err)
		// 	responded = true
		// 	return
		// }
		// log.Println(d.StorageDuration)
		// log.Println(d.VideoFileSize)

		storageDuration := r.FormValue("storage_duration")
		storageDurationInt, _ := strconv.ParseInt(storageDuration, 10, 64)
		fmt.Println("storageDurationInt", storageDurationInt)
		// 1 month <= storageDurationInt <= 10 years
		if storageDurationInt > 315360000 || storageDurationInt < 2628003 {
			w.WriteHeader(http.StatusExpectationFailed)
			data := map[string]interface{}{
				"error": "please specify a value of `storage_duration` between 2628003 and 315360000",
			}
			util.WriteResponse(data, w)
			responded = true
			return
		}

		// TODO: handle the case when a remote file is sent
		// example: https://file-examples-com.github.io/uploads/2017/04/file_example_MP4_1280_10MG.mp4
		r.Body = http.MaxBytesReader(w, r.Body, 30*1024*1024)
		clientFile, handler, err := r.FormFile("input_file")
		if err != nil {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			data := map[string]interface{}{
				"error": "please upload a file of size less than 30MB",
			}
			util.WriteResponse(data, w)
			responded = true
			return
		}

		videoFileSize := uint64(handler.Size)
		log.Println("videoFileSize:", videoFileSize)

		defer clientFile.Close()

		ss := strings.Split(handler.Filename, ".")

		if ss[len(ss)-1] != "mp4" {
			log.Println("not mp4")
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			data := map[string]interface{}{
				"error": "please upload an mp4 file",
			}
			util.WriteResponse(data, w)
			responded = true
			return
		}

		// Generate a new assetID.
		id := uuid.New()

		cmd := exec.Command("mkdir", "./assets/"+id.String())
		stdout, err := cmd.Output()
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusFailedDependency)
			data := map[string]interface{}{
				"error": "could not create asset",
			}
			util.WriteResponse(data, w)
			responded = true
			return
		}
		_ = stdout

		f, err := os.OpenFile("./assets/"+id.String()+"/"+handler.Filename, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusFailedDependency)
			data := map[string]interface{}{
				"error": "could not create asset",
			}
			util.WriteResponse(data, w)
			responded = true
			return
		}
		demuxFileName := f.Name()
		defer f.Close()
		_, err = io.Copy(f, clientFile) // copy file to demux server
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusFailedDependency)
			data := map[string]interface{}{
				"error": "could not create asset",
			}
			util.WriteResponse(data, w)
			responded = true
			return
		}
		// Create a new asset.
		dataservice.CreateAsset(model.Asset{
			AssetID:         id.String(),
			AssetStatusCode: 0,
			AssetStatus:     "video uploaded successfully",
			AssetError:      false,
		})

		transcodingCostWEI, err := util.CalculateTranscodingCost(demuxFileName)
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
