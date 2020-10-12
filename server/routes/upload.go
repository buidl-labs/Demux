package routes

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
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
	"github.com/buidl-labs/Demux/model"
	"github.com/buidl-labs/Demux/util"
	"github.com/gorilla/mux"
	"github.com/ipfs/go-cid"
	powc "github.com/textileio/powergate/api/client"
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
	} else {
		uploaded := false

		var videoFileSize int64

		vars := mux.Vars(r)
		assetID := vars["asset_id"]
		if !dataservice.IfAssetExists(assetID) {
			// Set AssetError to true
			dataservice.UpdateAssetStatus(assetID, 0, "asset created", true)
			log.Error("asset not found")
			return
		}

		if _, err := os.Stat("./assets/" + assetID); os.IsNotExist(err) {
			// Set AssetError to true
			dataservice.UpdateAssetStatus(assetID, 0, "asset created", true)
			log.Error("asset doesn't exist:", err)
			return
		}

		tempFolder := "./assets/" + assetID

		r.ParseMultipartForm(15 << 21)
		file, _, err := r.FormFile("file")
		if err != nil {
			dataservice.UpdateAssetStatus(assetID, 0, "asset created", true)
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
			dataservice.UpdateAssetStatus(assetID, 0, "asset created", true)
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
				dataservice.UpdateAssetStatus(assetID, 0, "asset created", true)
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
				dataservice.UpdateAssetStatus(assetID, 0, "asset created", true)
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
					dataservice.UpdateAssetStatus(assetID, 0, "asset created", true)
					log.Error("couldn't create file from chunks", err)
					return
				}
				log.Infof("%d bytes written offset %d\n", size, writeOffset)
			}

			uploaded = true
			dataservice.UpdateUploadStatus(assetID, true)

			// Delete the temporary chunks
			exec.Command("rm", "-rf", tempFolder+"/"+resumableIdentifier[0]).Output()
		} else {
			log.Infof("currentChunk: %d, totalChunks: %d\n", currentChunk, totalChunks)
		}

		demuxFileName := "./assets/" + assetID + "/testfile.mp4"

		if uploaded {
			go func() {

				livepeerPullCompleted := false

				livepeerAPIKey, livepeerAPIKeyExists := os.LookupEnv("LIVEPEER_COM_API_KEY")
				if !livepeerAPIKeyExists {
					dataservice.UpdateAssetStatus(assetID, 1, "processing in livepeer", true)
					log.Error("please provide the environment variable `LIVEPEER_COM_API_KEY`")
					return
				}

				// Set AssetStatus to 1 (processing in livepeer)
				dataservice.UpdateAssetStatus(assetID, 1, "processing in livepeer", false)

				// Start transcoding

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
					// Set AssetError to true
					dataservice.UpdateAssetStatus(assetID, 1, "processing in livepeer", true)
					log.Error("livepeer pull timed out")
					return
				case err := <-done:
					if err != nil {
						// Set AssetError to true
						dataservice.UpdateAssetStatus(assetID, 1, "processing in livepeer", true)
						log.Error("livepeer transcoding unsuccessful")
						return
					}
					livepeerPullCompleted = true
					log.Info("Completed livepeer transcoding")
				}

				// End process if livepeer pull fails or times out
				if livepeerPullCompleted == false {
					// Set AssetError to true
					dataservice.UpdateAssetStatus(assetID, 1, "processing in livepeer", true)
					log.Error("livepeer transcoding unsuccessful")
					return
				}

				items, err := ioutil.ReadDir("./assets/" + assetID)
				if err != nil {
					dataservice.UpdateAssetStatus(assetID, 1, "processing in livepeer", true)
					log.Error(err)
					return
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
									dataservice.UpdateAssetStatus(assetID, 1, "processing in livepeer", true)
									log.Error(err)
									return
								}

								durations := make([]string, len(segments))
								durSum := float64(0)

								for i, seg := range segments {
									segName := seg.Name()

									stdout, err := exec.Command("ffprobe", "-i", "./assets/"+assetID+"/"+f.Name()+"/"+res+"/"+segName, "-show_entries", "format=duration", "-v", "quiet", "-of", "csv=p=0").Output()
									if err != nil {
										dataservice.UpdateAssetStatus(assetID, 1, "processing in livepeer", true)
										log.Error(err)
										return
									}
									duration, err := strconv.ParseFloat(string(stdout)[:len(string(stdout))-2], 64)
									if err != nil {
										dataservice.UpdateAssetStatus(assetID, 1, "processing in livepeer", true)
										log.Error(err)
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
									dataservice.UpdateAssetStatus(assetID, 1, "processing in livepeer", true)
									log.Error(err)
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

				// Calculate transcoding cost of the video.

				transcodingCostEstimated := big.NewInt(0)
				transcodingCostEstimated, err = util.CalculateTranscodingCost(demuxFileName, float64(0))
				if err != nil {
					log.Warn("Couldn't calculate transcoding cost:", err)
					// Couldn't calculate transcoding cost. Set it to 0
				}

				dataservice.CreateTranscodingDeal(model.TranscodingDeal{
					AssetID:                  assetID,
					TranscodingCost:          big.NewInt(0).String(),
					TranscodingCostEstimated: transcodingCostEstimated.String(),
				})

				// Set AssetStatus to 2 (attempting to pin to ipfs)
				dataservice.UpdateAssetStatus(assetID, 2, "attempting to pin to ipfs", false)

				if livepeerPullCompleted {
					// generate thumbnail
					exec.Command("ffmpeg", "-i", demuxFileName, "-ss", "00:00:01.000", "-vframes", "1", "./assets/"+assetID+"/thumbnail.png").Output()

					rmcmd := exec.Command("rm", "-rf", demuxFileName)
					_, err := rmcmd.Output()
					if err != nil {
						// Set AssetError to true
						dataservice.UpdateAssetStatus(assetID, 2, "attempting to pin to ipfs", true)
						log.Error(err)
						return
					}

					pattern := "./assets/" + assetID + "/*.mp4"
					matches, err := filepath.Glob(pattern)
					if err != nil {
						// Set AssetError to true
						dataservice.UpdateAssetStatus(assetID, 2, "attempting to pin to ipfs", true)
						log.Error(err)
						return
					}

					for _, match := range matches {
						rmcmd = exec.Command("rm", "-rf", match)
						_, err := rmcmd.Output()
						if err != nil {
							// Set AssetError to true
							dataservice.UpdateAssetStatus(assetID, 2, "attempting to pin to ipfs", true)
							log.Error(err)
							return
						}
					}

					pattern = "./assets/" + assetID + "/*.m3u8"
					matches, err = filepath.Glob(pattern)
					if err != nil {
						// Set AssetError to true
						dataservice.UpdateAssetStatus(assetID, 2, "attempting to pin to ipfs", true)
						log.Error(err)
						return
					}
					if len(matches) == 1 {
						renameCmd := exec.Command("cp", matches[0], "./assets/"+assetID+"/root.m3u8")
						stdout, err := renameCmd.Output()
						if err != nil {
							// Set AssetError to true
							dataservice.UpdateAssetStatus(assetID, 2, "attempting to pin to ipfs", true)
							log.Error(err)
							return
						}
						_ = stdout
					}
				}
				dirsize, err := util.DirSize("./assets/" + assetID)
				if err != nil {
					// Set AssetError to true
					dataservice.UpdateAssetStatus(assetID, 2, "attempting to pin to ipfs", true)
					log.Error("finding dirsize:", err)
					return
				}
				dirsize = dirsize / (1024 * 1024)
				videoFileSize = videoFileSize / (1024 * 1024)
				ratio := float64(dirsize) / float64(videoFileSize)
				dataservice.AddSizeRatio(model.SizeRatio{
					AssetID:          assetID,
					SizeRatio:        ratio,
					VideoFileSize:    uint64(videoFileSize),
					StreamFolderSize: dirsize,
				})
				msr := dataservice.GetMeanSizeRatio()
				currRatioSum := msr.RatioSum
				currCount := msr.Count
				dataservice.UpdateMeanSizeRatio((ratio+currRatioSum)/float64(currCount+1), ratio+currRatioSum, currCount+1)

				// Compute estimated storage price

				estimatedPrice := float64(0)
				storageDurationInt := 31536000          // deal duration currently set to 1 year. 15768000-> 6 months
				duration := float64(storageDurationInt) // duration of deal in seconds (provided by user)
				epochs := float64(duration / float64(30))
				folderSize := dirsize //size of folder in MiB
				log.Info("folderSize", folderSize, "videoFileSize", videoFileSize)
				log.Info("duration", duration, "epochs", epochs)

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				pgClient, _ := powc.NewClient(util.InitialPowergateSetup.PowergateAddr)
				defer func() {
					if err := pgClient.Close(); err != nil {
						log.Warn("closing powergate client:", err)
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
					estimatedPrice = meanEpochPrice * float64(epochs) * float64(folderSize) / float64(1024)
					log.Info("estimatedPrice", estimatedPrice, ", meanEpochPrice", meanEpochPrice, ", pricesSum", pricesSum)
				}

				ctx = context.Background()

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

						dataservice.CreateStorageDeal(model.StorageDeal{
							AssetID:              assetID,
							StorageStatusCode:    0,
							StorageStatus:        "pinned to ipfs, attempting to store in filecoin",
							CID:                  currCIDStr,
							Miner:                "",
							StorageCost:          big.NewInt(0).String(),
							StorageCostEstimated: big.NewInt(int64(estimatedPrice)).String(),
							FilecoinDealExpiry:   int64(0),
							FFSToken:             tok,
							JobID:                jid,
						})

						// Update streamURL of the asset
						dataservice.UpdateStreamURL(assetID, streamURL)

						// Update thumbnail of the asset
						dataservice.UpdateThumbnail(assetID, ipfsGateway+currCIDStr+"/thumbnail.png")

						// Set AssetStatus to 3 (pinned to ipfs, attempting to store in filecoin)
						dataservice.UpdateAssetStatus(assetID, 3, "pinned to ipfs, attempting to store in filecoin", false)
					} else {
						// Set AssetError to true
						dataservice.UpdateAssetStatus(assetID, 2, "attempting to pin to ipfs", true)
						log.Error(err)
					}
					return
				}

				currCIDStr := fmt.Sprintf("%s", currCID)
				streamURL = ipfsGateway + currCIDStr + "/root.m3u8"

				log.Infof("CID: %s, currFolderName: %s\n", currCIDStr, currFolderName)

				dataservice.CreateStorageDeal(model.StorageDeal{
					AssetID:              assetID,
					StorageStatusCode:    0,
					StorageStatus:        "pinned to ipfs, attempting to store in filecoin",
					CID:                  currCIDStr,
					Miner:                "",
					StorageCost:          big.NewInt(0).String(),
					StorageCostEstimated: big.NewInt(int64(estimatedPrice)).String(),
					FilecoinDealExpiry:   int64(0),
					FFSToken:             tok,
					JobID:                jid,
				})

				// Update streamURL of the asset
				dataservice.UpdateStreamURL(assetID, streamURL)

				// Update thumbnail of the asset
				dataservice.UpdateThumbnail(assetID, ipfsGateway+currCIDStr+"/thumbnail.png")

				// Set AssetStatus to 3 (pinned to ipfs, attempting to store in filecoin)
				dataservice.UpdateAssetStatus(assetID, 3, "pinned to ipfs, attempting to store in filecoin", false)

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
		if dataservice.IfUploadExists(vars["asset_id"]) {
			upload := dataservice.GetUpload(vars["asset_id"])
			w.WriteHeader(http.StatusOK)
			data := map[string]interface{}{
				"asset_id": upload.AssetID,
				"status":   upload.Status,
				"url":      upload.URL,
			}
			util.WriteResponse(data, w)
		} else {
			w.WriteHeader(http.StatusNotFound)
			data := map[string]interface{}{
				"asset_id": vars["asset_id"],
				"status":   false,
				"url":      "",
			}
			util.WriteResponse(data, w)
		}
	}
}
