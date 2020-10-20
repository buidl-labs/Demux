package routes

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"math/big"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/buidl-labs/Demux/dataservice"
	"github.com/buidl-labs/Demux/internal"
	"github.com/buidl-labs/Demux/model"
	"github.com/buidl-labs/Demux/util"
	"github.com/gorilla/mux"
	"github.com/ipfs/go-cid"
)

var maxFileSize = 30 * 1024 * 1024

// FileUploadHandler handles the asset uploads using resumable.js
func FileUploadHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.Header().Set("Access-Control-Allow-Methods", "*")

	if r.Method == "GET" {
		vars := mux.Vars(r)
		assetID := vars["asset_id"]

		if _, err := os.Stat("./assets/" + assetID); os.IsNotExist(err) {
			return
		}

		tempFolder := "./assets/" + assetID

		resumableIdentifier, _ := r.URL.Query()["resumableIdentifier"]
		resumableChunkNumber, _ := r.URL.Query()["resumableChunkNumber"]
		path := fmt.Sprintf("%s/%s", tempFolder, resumableIdentifier[0])
		relativeChunk := fmt.Sprintf("%s%s%s%s", path, "/", "part", resumableChunkNumber[0])

		if _, err := os.Stat(path); os.IsNotExist(err) {
			os.Mkdir(path, os.ModePerm)
		}

		if _, err := os.Stat(relativeChunk); os.IsNotExist(err) {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusMethodNotAllowed)
		} else {
			http.Error(w, "Chunk already exist", http.StatusCreated)
		}
	} else if r.Method == "POST" {
		uploaded := false

		var videoFileSize int64

		vars := mux.Vars(r)
		assetID := vars["asset_id"]
		if !dataservice.IfAssetExists(assetID) {
			// Set AssetError to true
			dataservice.UpdateAssetStatus(assetID, -1, internal.AssetStatusMap[-1], true)
			log.Error("asset not found")
			return
		}

		if _, err := os.Stat("./assets/" + assetID); os.IsNotExist(err) {
			// Set AssetError to true
			dataservice.UpdateAssetStatus(assetID, -1, internal.AssetStatusMap[-1], true)
			log.Error("asset doesn't exist:", err)
			return
		}

		tempFolder := "./assets/" + assetID

		// r.ParseMultipartForm(15 << 21)
		file, _, err := r.FormFile("file")
		if err != nil {
			dataservice.UpdateAssetStatus(assetID, -1, internal.AssetStatusMap[-1], true)
			log.Error(err)
			return
		}
		defer file.Close()

		resumableIdentifier, _ := r.URL.Query()["resumableIdentifier"]
		resumableChunkNumber, _ := r.URL.Query()["resumableChunkNumber"]
		resumableTotalChunks, _ := r.URL.Query()["resumableTotalChunks"]

		path := fmt.Sprintf("%s/%s", tempFolder, resumableIdentifier[0])

		relativeChunk := fmt.Sprintf("%s%s%s%s", path, "/", "part", resumableChunkNumber[0])

		if _, err := os.Stat(path); os.IsNotExist(err) {
			os.Mkdir(path, os.ModePerm)
		}

		f, err := os.OpenFile(relativeChunk, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			dataservice.UpdateAssetStatus(assetID, -1, internal.AssetStatusMap[-1], true)
			log.Error(err)
			return
		}
		defer f.Close()
		io.Copy(f, file)

		currentChunk, err := strconv.Atoi(resumableChunkNumber[0])
		totalChunks, err := strconv.Atoi(resumableTotalChunks[0])

		// If it is the last chunk, trigger the recombination of chunks
		if currentChunk == totalChunks {
			log.Info("Combining chunks into one file")
			resumableTotalSize, _ := r.URL.Query()["resumableTotalSize"]
			videoFileSizeInt, _ := strconv.Atoi(resumableTotalSize[0])

			if videoFileSizeInt > maxFileSize {
				dataservice.UpdateAssetStatus(assetID, -1, internal.AssetStatusMap[-1], true)
				log.Warn("Maximum file size is 30 MiB")
				return
			}
			videoFileSize = int64(videoFileSizeInt)

			chunkSizeInBytesStr, _ := r.URL.Query()["resumableChunkSize"]
			chunkSizeInBytes, _ := strconv.Atoi(chunkSizeInBytesStr[0])

			chunksDir := path

			// Generate an empty file
			f, err := os.Create("./assets/" + assetID + "/testfile.mp4")
			if err != nil {
				dataservice.UpdateAssetStatus(assetID, -1, internal.AssetStatusMap[-1], true)
				log.Error("couldn't create file from chunks", err)
				return
			}
			defer f.Close()

			// For every chunk, write it to the empty file.

			for i := 1; i <= totalChunks; i++ {
				relativePath := fmt.Sprintf("%s%s%d", chunksDir, "/part", i)

				writeOffset := int64(chunkSizeInBytes * (i - 1))
				if i == 1 {
					writeOffset = 0
				}
				dat, err := ioutil.ReadFile(relativePath)
				size, err := f.WriteAt(dat, writeOffset)
				if err != nil {
					// Set AssetError true
					dataservice.UpdateAssetStatus(assetID, -1, internal.AssetStatusMap[-1], true)
					log.Error("couldn't create file from chunks", err)
					return
				}
				log.Infof("%d bytes written offset %d\n", size, writeOffset)
			}

			uploaded = true
			dataservice.UpdateUploadStatus(assetID, true)
			dataservice.UpdateAssetStatus(assetID, 0, internal.AssetStatusMap[0], true)

			// Delete the temporary chunks
			exec.Command("rm", "-rf", tempFolder+"/"+resumableIdentifier[0]).Output()
		} else {
			log.Infof("currentChunk: %d, totalChunks: %d\n", currentChunk, totalChunks)
		}

		demuxFileName := "./assets/" + assetID + "/testfile.mp4"

		if uploaded {
			go func() {

				// Set AssetStatus to 1 (processing in livepeer)
				dataservice.UpdateAssetStatus(assetID, 1, internal.AssetStatusMap[1], false)

				if err := Transcode(assetID, demuxFileName, videoFileSize); err != nil {
					log.Error(err)
					// Set AssetError to true
					dataservice.UpdateAssetStatus(assetID, 1, internal.AssetStatusMap[1], true)
					return
				}

				// Calculate transcoding cost of the video.
				transcodingCostEstimated := big.NewInt(0)
				transcodingCostEstimated, err = util.CalculateTranscodingCost(demuxFileName, float64(0))
				if err != nil {
					log.Warn("Couldn't calculate transcoding cost: ", err)
					// Couldn't calculate transcoding cost. Let it be 0.
				}

				if err := CreateStream(assetID, demuxFileName); err != nil {
					log.Error(err)
					// Set AssetError to true
					dataservice.UpdateAssetStatus(assetID, 1, internal.AssetStatusMap[1], true)
					return
				}

				dataservice.InsertTranscodingDeal(model.TranscodingDeal{
					AssetID:                  assetID,
					TranscodingCost:          big.NewInt(0).String(),
					TranscodingCostEstimated: transcodingCostEstimated.String(),
				})

				// Set AssetStatus to 2 (attempting to pin to ipfs)
				dataservice.UpdateAssetStatus(assetID, 2, internal.AssetStatusMap[2], false)

				// Set storageCostEstimated to 0.
				storageCostEstimated := big.NewInt(0)

				dirsize, err := util.DirSize("./assets/" + assetID)
				if err != nil {
					// dataservice.UpdateAssetStatus(assetID, 2, internal.AssetStatusMap[2], true)
					log.Warn("finding dirsize: ", err)
				} else {
					// Update sizeRatio
					_, _, _, _, err = UpdateSizeRatio(assetID, videoFileSize, dirsize)
					if err != nil {
						log.Warn("Couldn't process folder")
					}
					// Calculate storage cost of the video
					storageCostEstimated, err = util.CalculateStorageCost(dirsize, 31536000)
					if err != nil {
						log.Warn("Couldn't calculate storage cost: ", err)
						// Couldn't calculate storage cost. Let it be 0.
					}
				}

				ctx := context.Background()

				var currCID cid.Cid
				var streamURL string
				var ipfsGateway = os.Getenv("IPFS_GATEWAY")
				var jid string
				var currFolderName string
				currCID, currFolderName, minerName, tok, jid, storagePrice, expiry, staged, err := util.RunPow(ctx, util.InitialPowergateSetup, "./assets/"+assetID)
				_ = minerName
				_ = storagePrice
				_ = expiry
				if err != nil {
					if staged {
						log.Warn(err)
						currCIDStr := fmt.Sprintf("%s", currCID)
						streamURL = ipfsGateway + currCIDStr + "/root.m3u8"

						log.Infof("CID: %s, currFolderName: %s\n", currCIDStr, currFolderName)

						dataservice.InsertStorageDeal(model.StorageDeal{
							AssetID:              assetID,
							StorageStatusCode:    0,
							StorageStatus:        internal.AssetStatusMap[3],
							CID:                  currCIDStr,
							Miner:                "",
							StorageCost:          big.NewInt(0).String(),
							StorageCostEstimated: storageCostEstimated.String(),
							FilecoinDealExpiry:   int64(0),
							FFSToken:             tok,
							JobID:                jid,
						})

						// Update streamURL of the asset
						dataservice.UpdateStreamURL(assetID, streamURL)

						// Update thumbnail of the asset
						dataservice.UpdateThumbnail(assetID, ipfsGateway+currCIDStr+"/thumbnail.png")

						// Set AssetStatus to 3 (pinned to ipfs, attempting to store in filecoin)
						dataservice.UpdateAssetStatus(assetID, 3, internal.AssetStatusMap[3], false)
					} else {
						// Set AssetError to true
						dataservice.UpdateAssetStatus(assetID, 2, internal.AssetStatusMap[2], true)
						log.Error(err)
					}
					return
				}

				currCIDStr := fmt.Sprintf("%s", currCID)
				streamURL = ipfsGateway + currCIDStr + "/root.m3u8"

				log.Infof("CID: %s, currFolderName: %s\n", currCIDStr, currFolderName)

				dataservice.InsertStorageDeal(model.StorageDeal{
					AssetID:              assetID,
					StorageStatusCode:    0,
					StorageStatus:        internal.AssetStatusMap[3],
					CID:                  currCIDStr,
					Miner:                "",
					StorageCost:          big.NewInt(0).String(),
					StorageCostEstimated: storageCostEstimated.String(),
					FilecoinDealExpiry:   int64(0),
					FFSToken:             tok,
					JobID:                jid,
				})

				// Update streamURL of the asset
				dataservice.UpdateStreamURL(assetID, streamURL)

				// Update thumbnail of the asset
				dataservice.UpdateThumbnail(assetID, ipfsGateway+currCIDStr+"/thumbnail.png")

				// Set AssetStatus to 3 (pinned to ipfs, attempting to store in filecoin)
				dataservice.UpdateAssetStatus(assetID, 3, internal.AssetStatusMap[3], false)

				// Set AssetReady to true
				dataservice.UpdateAssetReady(assetID, true)
			}()

			w.WriteHeader(http.StatusOK)
			data := map[string]interface{}{
				"asset_id": assetID,
			}
			util.WriteResponse(data, w)
		}
	}
}

// UploadStatusHandler returns the upload details and status.
func UploadStatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == "GET" {
		vars := mux.Vars(r)

		unknownStatusData := map[string]interface{}{
			"asset_id": vars["asset_id"],
			"status":   false,
			"error":    false,
			"url":      "",
		}

		if dataservice.IfUploadExists(vars["asset_id"]) {
			upload, err := dataservice.GetUpload(vars["asset_id"])
			if err != nil {
				w.WriteHeader(http.StatusNotFound)
				util.WriteResponse(unknownStatusData, w)
			}
			w.WriteHeader(http.StatusOK)
			data := map[string]interface{}{
				"asset_id": upload.AssetID,
				"status":   upload.Status,
				"error":    upload.Error,
				"url":      upload.URL,
			}
			util.WriteResponse(data, w)
		} else {
			w.WriteHeader(http.StatusNotFound)
			util.WriteResponse(unknownStatusData, w)
		}
	}
}

// Transcode transcodes the video in the livepeer network.
func Transcode(assetID string, demuxFileName string, videoFileSize int64) error {
	livepeerPullCompleted := false

	livepeerAPIKey, livepeerAPIKeyExists := os.LookupEnv("LIVEPEER_COM_API_KEY")
	if !livepeerAPIKeyExists {
		return fmt.Errorf("please provide the environment variable `LIVEPEER_COM_API_KEY`")
	}

	log.Info("Starting livepeer transcoding")

	goos := runtime.GOOS
	lpCmd := exec.Command("./livepeerPull/"+goos+"/livepeer", "-pull", demuxFileName,
		"-recordingDir", "./assets/"+assetID, "-transcodingOptions",
		"./livepeerPull/configs/profiles.json", "-apiKey",
		livepeerAPIKey, "-v", "99", "-mediaDir", "./assets/"+assetID)

	var buf bytes.Buffer
	lpCmd.Stdout = &buf

	lpCmd.Start()

	done := make(chan error)
	go func() { done <- lpCmd.Wait() }()

	var timeout <-chan time.Time
	var limit int64 = int64(maxFileSize)

	if videoFileSize <= limit/4 {
		timeout = time.After(2 * time.Minute)
	} else if videoFileSize <= limit/2 {
		timeout = time.After(3 * time.Minute)
	} else if videoFileSize <= limit*3/4 {
		timeout = time.After(4 * time.Minute)
	} else {
		timeout = time.After(5 * time.Minute)
	}

	select {
	case <-timeout:
		lpCmd.Process.Kill()
		return fmt.Errorf("livepeer pull timed out")
	case err := <-done:
		if err != nil {
			return fmt.Errorf("livepeer transcoding unsuccessful")
		}
		livepeerPullCompleted = true
		log.Info("Completed livepeer transcoding")
	}

	// Return error if livepeer pull fails or times out
	if livepeerPullCompleted == false {
		return fmt.Errorf("livepeer transcoding unsuccessful")
	}
	return nil
}

// CreateStream generates the m3u8 files.
func CreateStream(assetID string, demuxFileName string) error {
	items, err := ioutil.ReadDir("./assets/" + assetID)
	if err != nil {
		return err
	}

	for _, f := range items {
		if f.IsDir() {
			resos := [4]string{"source", "1080p", "720p", "360p"}

			var pWg sync.WaitGroup
			pWg.Add(4)

			for _, res := range resos {
				go func(res string) {
					segments, err := ioutil.ReadDir("./assets/" + assetID + "/" + f.Name() + "/" + res)
					if err != nil {
						return
					}

					durations := make([]string, len(segments))
					durSum := float64(0)

					for i, seg := range segments {
						segName := seg.Name()

						stdout, err := exec.Command("ffprobe", "-i", "./assets/"+assetID+"/"+f.Name()+"/"+res+"/"+segName, "-show_entries", "format=duration", "-v", "quiet", "-of", "csv=p=0").Output()
						if err != nil {
							return
						}
						duration, err := strconv.ParseFloat(string(stdout)[:len(string(stdout))-2], 64)
						if err != nil {
							return
						}

						durSum += duration
						durations[i] = fmt.Sprintf("%.3f", duration)
					}

					var m3u8str strings.Builder
					m3u8str.WriteString("#EXTM3U\n" +
						"#EXT-X-VERSION:3\n" +
						"#EXT-X-TARGETDURATION:" + strconv.Itoa(int(durSum)) + "\n" +
						"#EXT-X-MEDIA-SEQUENCE:0\n")
					for i, dur := range durations {
						m3u8str.WriteString("#EXTINF:" + dur + ",\n" +
							res + "/" + strconv.Itoa(i) + ".ts\n")
					}
					m3u8str.WriteString("#EXT-X-ENDLIST\n")

					m3u8strFile, err := os.Create("./assets/" + assetID + "/" + f.Name() + "/" + res + ".m3u8")
					bWriter := bufio.NewWriter(m3u8strFile)
					n, err := bWriter.WriteString(m3u8str.String())
					if err != nil {
						return
					}
					_ = n
					bWriter.Flush()

					pWg.Done()
				}(res)
			}
			pWg.Wait()
		}
	}

	// generate thumbnail
	exec.Command("ffmpeg", "-i", demuxFileName, "-ss", "00:00:01.000", "-vframes", "1", "./assets/"+assetID+"/thumbnail.png").Output()

	rmcmd := exec.Command("rm", "-rf", demuxFileName)
	_, err = rmcmd.Output()
	if err != nil {
		return err
	}

	pattern := "./assets/" + assetID + "/*.mp4"
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}

	for _, match := range matches {
		rmcmd = exec.Command("rm", "-rf", match)
		_, err = rmcmd.Output()
		if err != nil {
			return err
		}
	}

	pattern = "./assets/" + assetID + "/*.m3u8"
	matches, err = filepath.Glob(pattern)
	if err != nil {
		return err
	}
	if len(matches) == 1 {
		renameCmd := exec.Command("cp", matches[0], "./assets/"+assetID+"/root.m3u8")
		stdout, err := renameCmd.Output()
		if err != nil {
			return err
		}
		_ = stdout
	}

	return nil
}

// UpdateSizeRatio updates the sizeRatioCollection with the
// current asset file and folder size details.
func UpdateSizeRatio(assetID string, videoFileSize int64, dirsize uint64) (uint64, float64, float64, float64, error) {
	dirsize = dirsize / (1024 * 1024)
	videoFileSize = videoFileSize / (1024 * 1024)
	ratio := float64(dirsize) / float64(videoFileSize)
	ratio = math.Round(ratio*100) / 100
	dataservice.InsertSizeRatio(model.SizeRatio{
		AssetID:          assetID,
		SizeRatio:        ratio,
		VideoFileSize:    uint64(videoFileSize),
		StreamFolderSize: dirsize,
	})
	msr, err := dataservice.GetMeanSizeRatio()
	if err != nil {
		log.Error("getting msr:", err)
		return dirsize, ratio, msr.MeanSizeRatio, msr.MeanSizeRatio, err
	}
	currRatioSum := math.Round(msr.RatioSum*100) / 100
	currCount := msr.Count
	updatedMsr := math.Round(((ratio+currRatioSum)/float64(currCount+1))*100) / 100
	dataservice.UpdateMeanSizeRatio(updatedMsr, ratio+currRatioSum, currCount+1)

	return dirsize, ratio, msr.MeanSizeRatio, updatedMsr, nil
}
