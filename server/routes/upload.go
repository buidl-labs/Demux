package routes

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
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

	"github.com/buidl-labs/Demux/dataservice"
	"github.com/buidl-labs/Demux/model"
	"github.com/buidl-labs/Demux/util"
	"github.com/gorilla/mux"
	"github.com/ipfs/go-cid"
	powc "github.com/textileio/powergate/api/client"
)

type uploadFile struct {
	file       *os.File
	name       string
	tempPath   string
	status     string
	size       int64
	transfered int64
}

var files = make(map[string]uploadFile)

// FileUploadHandler handles the asset uploads
func FileUploadHandler(w http.ResponseWriter, r *http.Request) {
	// w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	// w.Header().Set("Access-Control-Allow-Headers", "Content-Range, Content-Length")
	w.Header().Set("Access-Control-Allow-Methods", "*")

	if r.Method == "GET" {
		vars := mux.Vars(r)
		assetID := vars["asset_id"]
		fmt.Println("get req", vars)
		fmt.Println(assetID)
		if _, err := os.Stat("./assets/" + assetID); os.IsNotExist(err) {
			// path/to/whatever does not exist
			fmt.Println("doesn't exist")
			return
		}
		// exec.Command("mkdir", "./assets/"+assetID+"/temp").Output()
		tempFolder := "./assets/" + assetID //+ "/temp/"

		resumableIdentifier, _ := r.URL.Query()["resumableIdentifier"]
		resumableChunkNumber, _ := r.URL.Query()["resumableChunkNumber"]
		path := fmt.Sprintf("%s/%s", tempFolder, resumableIdentifier[0])
		relativeChunk := fmt.Sprintf("%s%s%s%s", path, "/", "part", resumableChunkNumber[0])

		if _, err := os.Stat(path); os.IsNotExist(err) {
			os.Mkdir(path, os.ModePerm)
		}

		fmt.Println("GresumableChunkNumber", resumableChunkNumber)
		fmt.Println("GPath", path)
		fmt.Println("RCh", relativeChunk)
		if _, err := os.Stat(relativeChunk); os.IsNotExist(err) {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusMethodNotAllowed)
		} else {
			http.Error(w, "Chunk already exist", http.StatusCreated)
		}
	} else {
		fmt.Println("ELSE CASE", r.Method)
		// var responded = false
		uploaded := false

		var videoFileSize int64

		vars := mux.Vars(r)
		assetID := vars["asset_id"]
		if !dataservice.IfAssetExists(assetID) {
			// responded = true
			fmt.Println("not found")
			// http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		fmt.Println("post req", vars, r)
		fmt.Println(assetID)
		if _, err := os.Stat("./assets/" + assetID); os.IsNotExist(err) {
			// path/to/whatever does not exist
			dataservice.UpdateAssetStatus(assetID, 0, "asset created", true)
			// http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			fmt.Println("doesn't exist")
			return
		}

		tempFolder := "./assets/" + assetID

		r.ParseMultipartForm(15 << 21)
		file, _, err := r.FormFile("file")
		if err != nil {
			dataservice.UpdateAssetStatus(assetID, 0, "asset created", true)
			// http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			fmt.Println(err)
			return
		}
		defer file.Close()
		fmt.Println(tempFolder)
		resumableIdentifier, _ := r.URL.Query()["resumableIdentifier"]
		resumableChunkNumber, _ := r.URL.Query()["resumableChunkNumber"]
		path := fmt.Sprintf("%s/%s", tempFolder, resumableIdentifier[0])
		relativeChunk := fmt.Sprintf("%s%s%s%s", path, "/", "part", resumableChunkNumber[0])

		if _, err := os.Stat(path); os.IsNotExist(err) {
			os.Mkdir(path, os.ModePerm)
		}

		f, err := os.OpenFile(relativeChunk, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			dataservice.UpdateAssetStatus(assetID, 0, "asset created", true)
			// http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			fmt.Println(err)
		}
		defer f.Close()
		io.Copy(f, file)

		/*
			If it is the last chunk, trigger the recombination of chunks
		*/
		resumableTotalChunks, _ := r.URL.Query()["resumableTotalChunks"]

		current, err := strconv.Atoi(resumableChunkNumber[0])
		total, err := strconv.Atoi(resumableTotalChunks[0])
		fmt.Println("resumableTotalChunks", resumableTotalChunks)
		if current == total {
			fmt.Println("Combining chunks into one file")
			fmt.Println("resumableChunkSize", r.URL.Query()["resumableChunkSize"])
			fmt.Println("resumableTotalSize", r.URL.Query()["resumableTotalSize"])
			resumableTotalSize, _ := r.URL.Query()["resumableTotalSize"]
			fmt.Println("resumableTotalChunks", resumableTotalChunks)
			videoFileSizeInt, _ := strconv.Atoi(resumableTotalSize[0])
			if videoFileSizeInt > 30*1024*1024 {
				dataservice.UpdateAssetStatus(assetID, 0, "asset created", true)
				return
			}
			videoFileSize = int64(videoFileSizeInt)
			iterations, _ := strconv.Atoi(resumableTotalChunks[0])
			fmt.Println("iterations", iterations)
			// resumableChunkSizeStr := r.URL.Query()["resumableChunkSize"]
			// resumableTotalSizeStr := r.URL.Query()["resumableTotalSize"]

			// f, err := os.Create("./assets/" + assetID + "/testfile.mp4")
			// if err != nil {
			// 	fmt.Printf("Error: %s", err)
			// }
			// defer f.Close()
			// chunkSize, err := DirSize("./assets/" + assetID)
			// if err != nil {
			// 	log.Println("finding dirsize", err)
			// }
			chunkSizeInBytesStr, _ := r.URL.Query()["resumableChunkSize"]
			chunkSizeInBytes, _ := strconv.Atoi(chunkSizeInBytesStr[0])
			// chunkSizeInBytes := 1048576
			// chunkSizeInBytes := uint64(chunkSize * 1024 * 1024)
			// chunksDir := "./assets/" + assetID + "/temp"
			chunksDir := path

			/*
			   Generate an empty file
			*/
			f, err := os.Create("./assets/" + assetID + "/testfile.mp4")
			if err != nil {
				dataservice.UpdateAssetStatus(assetID, 0, "asset created", true)
				// http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				fmt.Printf("Error: %s", err)
			}
			defer f.Close()

			// resumableTotalSize := r.URL.Query()["resumableTotalSize"]
			// totalChunks := resumableTotalSize / chunkSizeInBytes
			// fmt.Println("totalChunks", totalChunks)

			//For every chunk, write it to the empty file. The number of iterations is determined from resumable.js

			for i := 1; i <= iterations; i++ {
				relativePath := fmt.Sprintf("%s%s%d", chunksDir, "/part", i)
				fmt.Println("Chnk path: " + relativePath)

				writeOffset := int64(chunkSizeInBytes * (i - 1))
				if i == 1 {
					writeOffset = 0
				}
				dat, err := ioutil.ReadFile(relativePath)
				size, err := f.WriteAt(dat, writeOffset)
				if err != nil {
					dataservice.UpdateAssetStatus(assetID, 0, "asset created", true)
					// http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					fmt.Printf("Error: %s", err)
					return
				}
				fmt.Printf("%d bytes written offset %d \n", size, writeOffset)
			}
			uploaded = true
			dataservice.UpdateUploadStatus(assetID, true)
			fmt.Println("rming", tempFolder+"/"+resumableIdentifier[0])
			exec.Command("rm", "-rf", tempFolder+"/"+resumableIdentifier[0]).Output()
		} else {
			dataservice.UpdateAssetStatus(assetID, 0, "asset created", true)
			// http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		// exec.Command("rm", "-rf", tempFolder+"/"+resumableIdentifier[0]).Output()

		// videoFileSize := int64(7)
		demuxFileName := "./assets/" + assetID + "/testfile.mp4"

		if uploaded {
			go func() {

				livepeerPullCompleted := false

				livepeerAPIKey, livepeerAPIKeyExists := os.LookupEnv("LIVEPEER_COM_API_KEY")
				if !livepeerAPIKeyExists {
					log.Println("please provide the environment variable `LIVEPEER_COM_API_KEY`")
					dataservice.UpdateAssetStatus(assetID, 1, "processing in livepeer", true)
					return
				}

				// Set AssetStatus to 1 (processing in livepeer)
				dataservice.UpdateAssetStatus(assetID, 1, "processing in livepeer", false)

				// Start transcoding

				log.Println("Starting livepeer transcoding")

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
				var limit int64 = 30 * 1024 * 1024

				fmt.Println("videoFileSize", videoFileSize)

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
					log.Println("Livepeer pull timed out")
					// Set AssetError to true
					dataservice.UpdateAssetStatus(assetID, 1, "processing in livepeer", true)
					return
				case err := <-done:
					if err != nil {
						// Set AssetError to true
						dataservice.UpdateAssetStatus(assetID, 1, "processing in livepeer", true)
						return
					}
					livepeerPullCompleted = true
					log.Println("Done running livepeer cmd")
				}

				// Transcode using ffmpeg if livepeer pull fails or times out
				if livepeerPullCompleted == false {
					// Set AssetError to true
					dataservice.UpdateAssetStatus(assetID, 1, "processing in livepeer", true)
					return
				}

				items, err := ioutil.ReadDir("./assets/" + assetID)
				if err != nil {
					log.Println(err)
					return
				}
				for _, f := range items {
					fmt.Println(f.Name())
					if f.IsDir() {
						fmt.Println("dirName", f.Name())
						resos := [4]string{"source", "1080p", "720p", "360p"}

						var pWg sync.WaitGroup
						pWg.Add(4)
						for _, res := range resos {
							go func(res string) {
								segments, err := ioutil.ReadDir("./assets/" + assetID + "/" + f.Name() + "/" + res)
								if err != nil {
									log.Println(err)
									return
								}
								fmt.Println("segmentsLength", len(segments))
								if len(segments) > 6 {
									durations := make([]string, len(segments))
									durSum := float64(0)

									for i, seg := range segments {
										segName := seg.Name()
										fmt.Println("segName", segName)

										stdout, err := exec.Command("ffprobe", "-i", "./assets/"+assetID+"/"+f.Name()+"/"+res+"/"+segName, "-show_entries", "format=duration", "-v", "quiet", "-of", "csv=p=0").Output()
										if err != nil {
											log.Println(err)
											return
										}
										duration, err := strconv.ParseFloat(string(stdout)[:len(string(stdout))-2], 64)
										if err != nil {
											log.Println(err)
											return
										}
										fmt.Println("duree", duration)
										durSum += duration
										durations[i] = fmt.Sprintf("%.3f", duration)
									}
									fmt.Println("durs", durations)
									fmt.Println("total", durSum, int(durSum))
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
									// txt := "#EXTM3U\n" +
									// 	"#EXT-X-VERSION:3\n" +
									// 	"#EXT-X-TARGETDURATION:" + strconv.Itoa(int(durSum)) + "\n" +
									// 	"#EXT-X-MEDIA-SEQUENCE:0\n" +
									// 	"#EXTINF:" + fmt.Sprintf("%.6f", durSum) + ",\n" +
									// 	"myvid0.ts\n" +
									// 	"#EXT-X-ENDLIST\n"
									// fmt.Println(txt)
									fmt.Println("m3u8str", m3u8str.String())

									m3u8strFile, err := os.Create("./assets/" + assetID + "/" + f.Name() + "/" + res + ".m3u8")
									bWriter := bufio.NewWriter(m3u8strFile)
									n, err := bWriter.WriteString(m3u8str.String())
									if err != nil {
										log.Println(err)
										return
									}
									log.Println("created the internal m3u8 file")
									log.Printf("Wrote %d bytes\n", n)
									bWriter.Flush()

								}
								pWg.Done()
							}(res)
						}
						pWg.Wait()
					}
				}

				// Calculate transcoding cost of the video.

				// var transcodingCostEstimated uint64
				transcodingCostEstimated := big.NewInt(0)
				transcodingCostEstimated, err = util.CalculateTranscodingCost(demuxFileName, float64(0))
				if err != nil {
					log.Println(err)
					// Couldn't calculate transcoding cost. Set it to 0
				}

				dataservice.CreateTranscodingDeal(model.TranscodingDeal{
					AssetID:                  assetID,
					TranscodingCost:          big.NewInt(0).String(),
					TranscodingCostEstimated: transcodingCostEstimated.String(),
				})

				// Set AssetStatus to 2 (attempting to pin to ipfs)
				dataservice.UpdateAssetStatus(assetID, 2, "attempting to pin to ipfs", false)

				if livepeerPullCompleted == false {
					log.Println("lpcfalse")
					/*
						util.RemoveContents("./assets/" + assetID)
					*/
				} else {
					// generate thumbnail
					exec.Command("ffmpeg", "-i", demuxFileName, "-ss", "00:00:01.000", "-vframes", "1", "./assets/"+assetID+"/thumbnail.png").Output()

					rmcmd := exec.Command("rm", "-rf", demuxFileName)
					_, err := rmcmd.Output()
					if err != nil {
						log.Println(err)
						// Set AssetError to true
						dataservice.UpdateAssetStatus(assetID, 2, "attempting to pin to ipfs", true)
						return
					}

					pattern := "./assets/" + assetID + "/*.mp4"
					matches, err := filepath.Glob(pattern)
					if err != nil {
						log.Println(err)
						// Set AssetError to true
						dataservice.UpdateAssetStatus(assetID, 2, "attempting to pin to ipfs", true)
						return
					}
					fmt.Println("mpfmatches", matches)
					for _, match := range matches {
						rmcmd = exec.Command("rm", "-rf", match)
						_, err := rmcmd.Output()
						if err != nil {
							log.Println(err)
							// Set AssetError to true
							dataservice.UpdateAssetStatus(assetID, 2, "attempting to pin to ipfs", true)
							return
						}
					}
				}

				if livepeerPullCompleted {
					pattern := "./assets/" + assetID + "/*.m3u8"
					matches, err := filepath.Glob(pattern)
					if err != nil {
						log.Println(err)
						// Set AssetError to true
						dataservice.UpdateAssetStatus(assetID, 2, "attempting to pin to ipfs", true)
						return
					}
					if len(matches) == 1 {
						fmt.Println("matchesm3u8", matches)
						renameCmd := exec.Command("cp", matches[0], "./assets/"+assetID+"/root.m3u8")
						stdout, err := renameCmd.Output()
						if err != nil {
							log.Println(err)
							// Set AssetError to true
							dataservice.UpdateAssetStatus(assetID, 2, "attempting to pin to ipfs", true)
							return
						}
						_ = stdout
					}
				}
				dirsize, err := DirSize("./assets/" + assetID)
				if err != nil {
					log.Println("finding dirsize", err)
				}
				dirsize = dirsize / (1024 * 1024)
				videoFileSize = videoFileSize / (1024 * 1024)
				fmt.Println("dirsize", dirsize)
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

				//************************* Compute estimated storage price
				// estimatedPriceStr := ""
				estimatedPrice := float64(0)
				storageDurationInt := 31536000          // deal duration currently set to 1 year. 15768000-> 6 months
				duration := float64(storageDurationInt) //duration of deal in seconds (provided by user)
				epochs := float64(duration / float64(30))
				folderSize := dirsize //size of folder in MiB
				fmt.Println("folderSize", folderSize, "videoFileSize", videoFileSize)
				fmt.Println("duration", duration, "epochs", epochs)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				pgClient, _ := powc.NewClient(util.InitialPowergateSetup.PowergateAddr)
				defer func() {
					if err := pgClient.Close(); err != nil {
						log.Printf("closing powergate client: %s\n", err)
					}
				}()

				index, err := pgClient.Asks.Get(ctx)
				if err != nil {
					log.Printf("getting asks: %s\n", err)
				}
				if len(index.Storage) > 0 {
					log.Printf("Storage median price: %v\n", index.StorageMedianPrice)
					log.Printf("Last updated: %v\n", index.LastUpdated.Format("01/02/06 15:04 MST"))
					// fmt.Println("index:\n", index)
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
					meanEpochPrice := float64(float64(pricesSum) / float64(len(index.Storage)))
					fmt.Println("pricesSum", pricesSum)
					fmt.Println("meanEpochPrice", meanEpochPrice)
					// estimatedPrice = meanEpochPrice * epochs * folderSize / 1024
					estimatedPrice = meanEpochPrice * float64(epochs) * float64(folderSize) / float64(1024)
					fmt.Println("estimatedPrice", estimatedPrice)
					// estimatedPriceStr = fmt.Sprintf("%f", estimatedPrice)
				}
				//*************************

				ctx = context.Background()

				var currCID cid.Cid
				var streamURL string
				// var pinataIpfsGateway = "https://gateway.pinata.cloud/ipfs/"
				// var ipfsioGateway = "https://ipfs.io/ipfs/"
				var ipfsGateway = os.Getenv("IPFS_GATEWAY")
				var jid string
				var currFolderName string
				currCID, currFolderName, minerName, tok, jid, storagePrice, expiry, staged, err := util.RunPow(ctx, util.InitialPowergateSetup, "./assets/"+assetID)
				fmt.Println("minerName", minerName)
				fmt.Println("storagePrice", storagePrice)
				fmt.Println("expiry", expiry)
				fmt.Println("streamURL", streamURL)
				if err != nil {
					log.Println(err)
					if staged {
						currCIDStr := fmt.Sprintf("%s", currCID)
						// pinataCID, pinataErr := util.PinFolder("assets/"+assetID, "POW"+currCIDStr)
						// if pinataErr != nil {
						// 	// dataservice.SetAssetError(assetID, fmt.Sprintf("pinning to pinata: %s", pinataErr), http.StatusFailedDependency)
						// 	// return
						// }
						// if pinataCID == currCIDStr {
						// 	fmt.Println("EQ")
						// 	streamURL = pinataIpfsGateway + pinataCID
						// } else {
						// 	fmt.Println("NOTEQ", pinataCID, currCIDStr)
						// 	streamURL = ipfsioGateway + currCIDStr
						// }
						streamURL = ipfsGateway + currCIDStr + "/root.m3u8"

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
					}
					return
				}

				log.Printf("CID: %s, currFolderName: %s\n", currCID, currFolderName)
				currCIDStr := fmt.Sprintf("%s", currCID)
				// pinataCID, pinataErr := util.PinFolder("assets/"+assetID, "POW"+currCIDStr)
				// if pinataErr != nil {
				// 	// dataservice.SetAssetError(assetID, fmt.Sprintf("pinning to pinata: %s", pinataErr), http.StatusFailedDependency)
				// 	// return
				// }
				// if pinataCID == currCIDStr {
				// 	fmt.Println("EQ")
				// 	streamURL = pinataIpfsGateway + pinataCID
				// } else {
				// 	fmt.Println("NOTEQ", pinataCID, currCIDStr)
				// 	streamURL = ipfsioGateway + currCIDStr
				// }
				streamURL = ipfsGateway + currCIDStr + "/root.m3u8"

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

			// if responded == false {
			w.WriteHeader(http.StatusOK)
			data := map[string]interface{}{
				"asset_id": assetID,
			}
			util.WriteResponse(data, w)
			// 	responded = true
			// }
		}
	}
}

func UploadHandler(w http.ResponseWriter, r *http.Request) {
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

func DirSize(path string) (uint64, error) {
	var size uint64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += uint64(info.Size())
		}
		return err
	})
	return size, err
}

func parseContentRange(contentRange string) (totalSize int64, partFrom int64, partTo int64) {
	contentRange = strings.Replace(contentRange, "bytes ", "", -1)
	fmt.Println("contentRange", contentRange)
	fromTo := strings.Split(contentRange, "/")[0]
	fmt.Println("fromTo", fromTo)
	totalSize, err := strconv.ParseInt(strings.Split(contentRange, "/")[1], 10, 64)
	if err != nil {
		fmt.Println(err)
		return
	}

	splitted := strings.Split(fromTo, "-")

	partFrom, err = strconv.ParseInt(splitted[0], 10, 64)
	if err != nil {
		fmt.Println(err)
		return
	}
	partTo, err = strconv.ParseInt(splitted[1], 10, 64)
	if err != nil {
		fmt.Println(err)
		return
	}
	return totalSize, partFrom, partTo
}
