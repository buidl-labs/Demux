package routes

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/buidl-labs/Demux/dataservice"
	"github.com/buidl-labs/Demux/model"
	"github.com/buidl-labs/Demux/util"
	guuid "github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/ipfs/go-cid"
)

// AssetsHandler handles the asset uploads
func AssetsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {

		var responded = false

		// TODO: handle the case when a remote file is sent
		// example: https://file-examples-com.github.io/uploads/2017/04/file_example_MP4_1280_10MG.mp4
		r.Body = http.MaxBytesReader(w, r.Body, 30*1024*1024)
		clientFile, handler, err := r.FormFile("inputfile")
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			data := map[string]interface{}{
				"Error": fmt.Sprintf("please upload a file of size less than 30MB: %s", err),
			}
			util.WriteResponse(data, w)
			responded = true
			return
		}

		videoFileSize := handler.Size
		log.Println("videoFileSize:", videoFileSize)

		defer clientFile.Close()

		ss := strings.Split(handler.Filename, ".")

		if ss[len(ss)-1] != "mp4" {
			log.Println("not mp4")
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			data := map[string]interface{}{
				"Error": "please upload an mp4 file",
			}
			util.WriteResponse(data, w)
			responded = true
			return
		}

		// Generate a new assetID.
		id := guuid.New()

		cmd := exec.Command("mkdir", "./assets/"+id.String())
		stdout, err := cmd.Output()
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusFailedDependency)
			data := map[string]interface{}{
				"Error": "could not create asset",
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
				"Error": "could not create asset",
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
				"Error": "could not create asset",
			}
			util.WriteResponse(data, w)
			responded = true
			return
		}

		// Create a new asset.
		dataservice.CreateAsset(model.Asset{
			AssetID:         id.String(),
			AssetStatusCode: 0,
			AssetStatus:     "Video uploaded to API successfully",
			AssetError:      false,
			CreatedAt:       time.Now().Unix(),
		})

		go func() {

			livepeerPullCompleted := false

			livepeerAPIKey, livepeerAPIKeyExists := os.LookupEnv("LIVEPEER_COM_API_KEY")
			if !livepeerAPIKeyExists {
				log.Println("please provide the environment variable `LIVEPEER_COM_API_KEY`")
				dataservice.UpdateAssetStatus(id.String(), 1, "Processing in Livepeer", true)
				return
			}

			// Set AssetStatus to 1 (Processing in Livepeer)
			dataservice.UpdateAssetStatus(id.String(), 1, "Processing in Livepeer", false)

			// Start transcoding

			log.Println("Starting livepeer transcoding")

			goos := runtime.GOOS
			lpCmd := exec.Command("./livepeerPull/"+goos+"/livepeer", "-pull", demuxFileName,
				"-recordingDir", "./assets/"+id.String(), "-transcodingOptions",
				"./livepeerPull/configs/profiles.json", "-apiKey",
				livepeerAPIKey, "-v", "99", "-mediaDir", "./assets/"+id.String())

			var buf bytes.Buffer
			lpCmd.Stdout = &buf

			lpCmd.Start()

			done := make(chan error)
			go func() { done <- lpCmd.Wait() }()

			var timeout <-chan time.Time
			var limit int64 = 30 * 1024 * 1024

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
				dataservice.UpdateAssetStatus(id.String(), 1, "Processing in Livepeer", true)
				// TODO: this is temporary:
				livepeerPullCompleted = true
				// return
			case err := <-done:
				if err != nil {
					// Set AssetError to true
					dataservice.UpdateAssetStatus(id.String(), 1, "Processing in Livepeer", true)
					return
				}
				livepeerPullCompleted = true
				log.Println("Done running livepeer cmd")
			}

			// Transcode using ffmpeg if livepeer pull fails or times out
			if livepeerPullCompleted == false {
				// Set AssetError to true
				dataservice.UpdateAssetStatus(id.String(), 1, "Processing in Livepeer", true)
				return
				/*
					resos := [3]string{"1080", "720", "360"}

					var transcodeWg sync.WaitGroup
					transcodeWg.Add(3)
					for _, res := range resos {
						go func(res string) {
							cmd1 := exec.Command("ffmpeg", "-i", demuxFileName, "-vf", "scale=-1:"+res, "-c:v", "libx264", "-crf", "18", "-preset", "ultrafast", "-c:a", "copy", "./assets/"+id.String()+"/random"+res+"p.mp4")
							stdout1, err := cmd1.Output()
							if err != nil {
								dataservice.SetAssetError(id.String(), fmt.Sprintf("ffmpeg transcoding: %s", err), http.StatusFailedDependency)
								return
							}
							_ = stdout1

							transcodeWg.Done()
						}(res)
					}
					transcodeWg.Wait()
				*/
			} else {
				// dataservice.UpdateAssetStatus(id.String(), 2, "Calculating transcoding cost", false)
			}

			// Calculate transcoding cost of the video.

			// var transcodingCostEstimated uint64
			transcodingCostEstimated := big.NewInt(0)
			transcodingCostEstimated, err = util.CalculateTranscodingCost(demuxFileName)
			if err != nil {
				log.Println(err)
				// Couldn't calculate transcoding cost. Set it to 0
			}

			dataservice.CreateTranscodingDeal(model.TranscodingDeal{
				AssetID:                  id.String(),
				TranscodingCost:          big.NewInt(0).String(),
				TranscodingCostEstimated: transcodingCostEstimated.String(),
			})

			// Set AssetStatus to 2 (Attempting to pin to IPFS)
			dataservice.UpdateAssetStatus(id.String(), 2, "Attempting to pin to IPFS", false)

			if livepeerPullCompleted == false {
				log.Println("lpcfalse")
				/*
					util.RemoveContents("./assets/" + id.String())
				*/
			} else {
				rmcmd := exec.Command("rm", "-rf", demuxFileName)
				_, err := rmcmd.Output()
				if err != nil {
					log.Println(err)
					// Set AssetError to true
					dataservice.UpdateAssetStatus(id.String(), 2, "Attempting to pin to IPFS", true)
					return
				}

				pattern := "./assets/" + id.String() + "/*.mp4"
				matches, err := filepath.Glob(pattern)
				if err != nil {
					log.Println(err)
					// Set AssetError to true
					dataservice.UpdateAssetStatus(id.String(), 2, "Attempting to pin to IPFS", true)
					return
				}
				fmt.Println("mpfmatches", matches)
				for _, match := range matches {
					rmcmd = exec.Command("rm", "-rf", match)
					_, err := rmcmd.Output()
					if err != nil {
						log.Println(err)
						// Set AssetError to true
						dataservice.UpdateAssetStatus(id.String(), 2, "Attempting to pin to IPFS", true)
						return
					}
				}
			}

			if livepeerPullCompleted == false {
				/*
					allFiles, err := ioutil.ReadDir("./assets/" + id.String())
					if err != nil {
						dataservice.SetAssetError(id.String(), fmt.Sprintf("reading asset directory: %s", err), http.StatusFailedDependency)
						return
					}

					exec.Command("mkdir", "./assets/"+id.String()+"/1080p").Output()
					exec.Command("mkdir", "./assets/"+id.String()+"/720p").Output()
					exec.Command("mkdir", "./assets/"+id.String()+"/360p").Output()

					log.Println("segmenting the transcoded videos...")

					var wg sync.WaitGroup
					wg.Add(3)

					for _, f := range allFiles {
						go func(f os.FileInfo) {
							fname := f.Name()
							nm := strings.Split(fname, "/")[len(strings.Split(fname, "/"))-1]
							name := strings.Split(nm, ".")[0]
							if len(name) > 5 {
								if name[len(name)-5:] == "1080p" {
									// 1080p
									_, err := util.CreateSegments(fname, "1080p", id)
									if err != nil {
										dataservice.SetAssetError(id.String(), fmt.Sprintf("creating segments: %s", err), http.StatusFailedDependency)
										return
									}
								} else if name[len(name)-4:] == "720p" {
									// 720p
									_, err := util.CreateSegments(fname, "720p", id)
									if err != nil {
										dataservice.SetAssetError(id.String(), fmt.Sprintf("creating segments: %s", err), http.StatusFailedDependency)
										return
									}
								} else if name[len(name)-4:] == "360p" {
									// 360p
									_, err := util.CreateSegments(fname, "360p", id)
									if err != nil {
										dataservice.SetAssetError(id.String(), fmt.Sprintf("creating segments: %s", err), http.StatusFailedDependency)
										return
									}
								}
							} else if len(name) > 4 {
								if name[len(name)-4:] == "720p" {
									// 720p
									_, err := util.CreateSegments(fname, "720p", id)
									if err != nil {
										dataservice.SetAssetError(id.String(), fmt.Sprintf("creating segments: %s", err), http.StatusFailedDependency)
										return
									}
								} else if name[len(name)-4:] == "360p" {
									// 360p
									_, err := util.CreateSegments(fname, "360p", id)
									if err != nil {
										dataservice.SetAssetError(id.String(), fmt.Sprintf("creating segments: %s", err), http.StatusFailedDependency)
										return
									}
								}
							}
							wg.Done()
						}(f)
					}

					wg.Wait()

					log.Println("completed segmentation")

					// Create root abrStreamFile
					abrStreamFile, err := os.Create("./assets/" + id.String() + "/root.m3u8")

					bWriter := bufio.NewWriter(abrStreamFile)

					n, err := bWriter.WriteString("#EXTM3U\n" +
						"#EXT-X-STREAM-INF:BANDWIDTH=6000000,RESOLUTION=1920x1080\n" +
						"1080p/myvid.m3u8\n" +
						"#EXT-X-STREAM-INF:BANDWIDTH=2000000,RESOLUTION=1280x720\n" +
						"720p/myvid.m3u8\n" +
						"#EXT-X-STREAM-INF:BANDWIDTH=500000,RESOLUTION=640x360\n" +
						"360p/myvid.m3u8\n")

					if err != nil {
						dataservice.SetAssetError(id.String(), fmt.Sprintf("creating the root m3u8 file: %s", err), http.StatusFailedDependency)
						return
					}

					log.Println("created the root m3u8 file")
					log.Printf("Wrote %d bytes\n", n)
					bWriter.Flush()
				*/
			} else {
				pattern := "./assets/" + id.String() + "/*.m3u8"
				matches, err := filepath.Glob(pattern)
				if err != nil {
					log.Println(err)
					// Set AssetError to true
					dataservice.UpdateAssetStatus(id.String(), 2, "Attempting to pin to IPFS", true)
					return
				}
				if len(matches) == 1 {
					fmt.Println("matchesm3u8", matches)
					renameCmd := exec.Command("cp", matches[0], "./assets/"+id.String()+"/root.m3u8")
					stdout, err := renameCmd.Output()
					if err != nil {
						log.Println(err)
						// Set AssetError to true
						dataservice.UpdateAssetStatus(id.String(), 2, "Attempting to pin to IPFS", true)
						return
					}
					_ = stdout
				}
			}

			ctx := context.Background()

			var currCID cid.Cid
			var streamURL string
			var pinataIpfsGateway = "https://gateway.pinata.cloud/ipfs/"
			var ipfsioGateway = "https://ipfs.io/ipfs/"
			var ipfsGateway = os.Getenv("IPFS_GATEWAY")
			var jid string
			var currFolderName string
			currCID, currFolderName, minerName, tok, jid, storagePrice, expiry, staged, err := util.RunPow(ctx, util.InitialPowergateSetup, "./assets/"+id.String())
			fmt.Println("minerName", minerName)
			fmt.Println("storagePrice", storagePrice)
			fmt.Println("expiry", expiry)
			fmt.Println("streamURL", streamURL)
			if err != nil {
				log.Println(err)
				if staged {
					currCIDStr := fmt.Sprintf("%s", currCID)
					pinataCID, pinataErr := util.PinFolder("assets/"+id.String(), "POW"+currCIDStr)
					if pinataErr != nil {
						// dataservice.SetAssetError(id.String(), fmt.Sprintf("pinning to pinata: %s", pinataErr), http.StatusFailedDependency)
						// return
					}
					if pinataCID == currCIDStr {
						fmt.Println("EQ")
						streamURL = pinataIpfsGateway + pinataCID
					} else {
						fmt.Println("NOTEQ", pinataCID, currCIDStr)
						streamURL = ipfsioGateway + currCIDStr
					}
					streamURL = ipfsGateway + currCIDStr + "/root.m3u8"

					dataservice.CreateStorageDeal(model.StorageDeal{
						AssetID:              id.String(),
						StorageStatusCode:    0,
						StorageStatus:        "Pinned to IPFS. Attempting to store in Filecoin",
						CID:                  currCIDStr,
						Miner:                "",
						StorageCost:          big.NewInt(0).String(),
						StorageCostEstimated: big.NewInt(0).String(),
						FilecoinDealExpiry:   int64(0),
						FFSToken:             tok,
						JobID:                jid,
					})

					// Update streamURL of the asset
					dataservice.UpdateStreamURL(id.String(), streamURL)

					// Set AssetStatus to 3 (Pinned to IPFS. Attempting to store in Filecoin)
					dataservice.UpdateAssetStatus(id.String(), 3, "Pinned to IPFS. Attempting to store in Filecoin", false)
				} else {
					// Set AssetError to true
					dataservice.UpdateAssetStatus(id.String(), 2, "Attempting to pin to IPFS", true)
				}
				return
			}

			log.Printf("CID: %s, currFolderName: %s\n", currCID, currFolderName)
			currCIDStr := fmt.Sprintf("%s", currCID)
			pinataCID, pinataErr := util.PinFolder("assets/"+id.String(), "POW"+currCIDStr)
			if pinataErr != nil {
				// dataservice.SetAssetError(id.String(), fmt.Sprintf("pinning to pinata: %s", pinataErr), http.StatusFailedDependency)
				// return
			}
			if pinataCID == currCIDStr {
				fmt.Println("EQ")
				streamURL = pinataIpfsGateway + pinataCID
			} else {
				fmt.Println("NOTEQ", pinataCID, currCIDStr)
				streamURL = ipfsioGateway + currCIDStr
			}
			streamURL = ipfsGateway + currCIDStr + "/root.m3u8"

			dataservice.CreateStorageDeal(model.StorageDeal{
				AssetID:              id.String(),
				StorageStatusCode:    0,
				StorageStatus:        "Pinned to IPFS. Attempting to store in Filecoin",
				CID:                  currCIDStr,
				Miner:                "",
				StorageCost:          big.NewInt(0).String(),
				StorageCostEstimated: big.NewInt(0).String(),
				FilecoinDealExpiry:   int64(0),
				FFSToken:             tok,
				JobID:                jid,
			})

			// Update streamURL of the asset
			dataservice.UpdateStreamURL(id.String(), streamURL)

			// Set AssetStatus to 3 (Pinned to IPFS. Attempting to store in Filecoin)
			dataservice.UpdateAssetStatus(id.String(), 3, "Pinned to IPFS. Attempting to store in Filecoin", false)
		}()

		if responded == false {
			w.WriteHeader(http.StatusOK)
			data := map[string]interface{}{
				"AssetID": id.String(),
			}
			util.WriteResponse(data, w)
			responded = true
		}
	}
}

// AssetsStatusHandler enables checking the status of an asset in its demux lifecycle.
func AssetsStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		vars := mux.Vars(r)

		if dataservice.IfAssetExists(vars["asset_id"]) {
			asset := dataservice.GetAsset(vars["asset_id"])
			transcodingDeal := dataservice.GetTranscodingDeal(vars["asset_id"])
			storageDeal := dataservice.GetStorageDeal(vars["asset_id"])

			w.WriteHeader(http.StatusOK)
			data := map[string]interface{}{
				"AssetID":         asset.AssetID,
				"AssetStatusCode": asset.AssetStatusCode,
				"AssetStatus":     asset.AssetStatus,
				"AssetError":      asset.AssetError,
				"StreamURL":       asset.StreamURL,
				// "StorageCost":              storageDeal.StorageCost,
				// "StorageCostEstimated":     storageDeal.StorageCostEstimated,
				// "TranscodingCost":          transcodingDeal.TranscodingCost,
				// "TranscodingCostEstimated": transcodingDeal.TranscodingCostEstimated,
				"CreatedAt": asset.CreatedAt,
			}
			storageCostBigInt := new(big.Int)
			storageCostBigInt, ok := storageCostBigInt.SetString(storageDeal.StorageCost, 10)
			if !ok {
				fmt.Println("SetString: error", ok)
				data["StorageCost"] = storageDeal.StorageCost
			} else {
				fmt.Println(storageCostBigInt)
				data["StorageCost"] = storageCostBigInt
			}

			storageCostEstimatedBigInt := new(big.Int)
			storageCostEstimatedBigInt, ok = storageCostEstimatedBigInt.SetString(storageDeal.StorageCostEstimated, 10)
			if !ok {
				fmt.Println("SetString: error", ok)
				data["StorageCostEstimated"] = storageDeal.StorageCostEstimated
			} else {
				fmt.Println(storageCostBigInt)
				data["StorageCostEstimated"] = storageCostEstimatedBigInt
			}

			transcodingCostBigInt := new(big.Int)
			transcodingCostBigInt, ok = transcodingCostBigInt.SetString(transcodingDeal.TranscodingCost, 10)
			if !ok {
				fmt.Println("SetString: error", ok)
				data["TranscodingCost"] = transcodingDeal.TranscodingCost
			} else {
				fmt.Println(transcodingCostBigInt)
				data["TranscodingCost"] = transcodingCostBigInt
			}

			transcodingCostEstimatedBigInt := new(big.Int)
			transcodingCostEstimatedBigInt, ok = transcodingCostEstimatedBigInt.SetString(transcodingDeal.TranscodingCostEstimated, 10)
			if !ok {
				fmt.Println("SetString: error", ok)
				data["TranscodingCostEstimated"] = transcodingDeal.TranscodingCostEstimated
			} else {
				fmt.Println(storageCostBigInt)
				data["TranscodingCostEstimated"] = transcodingCostEstimatedBigInt
			}

			util.WriteResponse(data, w)

			// assetStatus := dataservice.GetAssetStatusIfExists(vars["asset_id"])
			// assetError := dataservice.GetAssetError(vars["asset_id"])

			// if assetError == "" {
			// 	w.WriteHeader(http.StatusOK)
			// 	var data = make(map[string]interface{})
			// 	data["AssetID"] = vars["asset_id"]
			// 	data["AssetStatus"] = assetStatus

			// 	if assetStatus == 3 {
			// 		data["CID"] = dataservice.GetCIDForAsset(vars["asset_id"])
			// 		asset := dataservice.GetAsset(vars["asset_id"])
			// 		data["TranscodingCost"] = asset.TranscodingCost
			// 		data["StreamURL"] = asset.StreamURL

			// 		if dataservice.GetStorageDealStatus(vars["asset_id"]) == 0 {
			// 			data["Status"] = "Filecoin deal pending"
			// 			data["Miner"] = asset.Miner
			// 			data["StorageCost"] = asset.StorageCost
			// 			data["Expiry"] = asset.Expiry
			// 		} else if dataservice.GetStorageDealStatus(vars["asset_id"]) == 1 {
			// 			data["Status"] = "Completed Filecoin storage deal"
			// 			storageDeal := dataservice.GetStorageDeal(vars["asset_id"])
			// 			data["Miner"] = storageDeal.Miner
			// 			data["StorageCost"] = storageDeal.StorageCost
			// 			data["Expiry"] = storageDeal.Expiry
			// 		} else {
			// 			data["Status"] = "Failed to store in Filecoin"
			// 			data["Miner"] = asset.Miner
			// 			data["StorageCost"] = asset.StorageCost
			// 			data["Expiry"] = asset.Expiry
			// 		}
			// 	}
			// 	util.WriteResponse(data, w)
			// } else {
			// 	w.WriteHeader(http.StatusOK)
			// 	data := map[string]interface{}{
			// 		"AssetID": vars["asset_id"],
			// 		"Error":   dataservice.GetAssetError(vars["asset_id"]),
			// 	}
			// 	util.WriteResponse(data, w)
			// }
		} else {
			w.WriteHeader(http.StatusNotFound)
			data := map[string]interface{}{
				"AssetID": nil,
				"Error":   "no such asset",
			}
			util.WriteResponse(data, w)
		}
	}
}
