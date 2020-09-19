package routes

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
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
		// r.Body = http.MaxBytesReader(w, r.Body, 30*1024*1024)
		clientFile, handler, err := r.FormFile("inputfile")
		if err != nil {
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
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			data := map[string]interface{}{
				"Error": "please upload an mp4 file",
			}
			util.WriteResponse(data, w)
			responded = true
			return
		}

		// Generate a new assetID and create asset.

		id := guuid.New()
		dataservice.CreateAsset(model.Asset{
			AssetID:     id.String(),
			AssetName:   handler.Filename,
			AssetStatus: 0,
			Error:       "",
		})

		cmd := exec.Command("mkdir", "./assets/"+id.String())
		stdout, err := cmd.Output()
		if err != nil {
			log.Println(err)
			return
		}
		_ = stdout

		f, err := os.OpenFile("./assets/"+id.String()+"/"+handler.Filename, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			log.Println(err)
			return
		}
		demuxFileName := f.Name()
		defer f.Close()
		io.Copy(f, clientFile) // copy file to demux server

		go func() {
			// Set AssetStatus to 1 (transcoding)
			dataservice.UpdateAssetStatus(id.String(), 1)

			// Start transcoding

			livepeerPullCompleted := false

			livepeerAPIKey, livepeerAPIKeyExists := os.LookupEnv("LIVEPEER_COM_API_KEY")
			if !livepeerAPIKeyExists {
				dataservice.SetAssetError(id.String(), "please provide the environment variable `LIVEPEER_COM_API_KEY`", http.StatusFailedDependency)
				return
			}

			log.Println("Starting livepeer transcoding")

			goos := runtime.GOOS
			lpCmd := exec.Command("./livepeerPull/"+goos+"/livepeer", "-pull", demuxFileName,
				"-recordingDir", "./assets/"+id.String(), "-transcodingOptions",
				"./livepeerPull/configs/profiles.json", "-apiKey",
				livepeerAPIKey, "-v", "99")

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
			case err := <-done:
				if err != nil {
					dataservice.SetAssetError(id.String(), fmt.Sprintf("livepeer transcoding: %s", err), http.StatusFailedDependency)
					return
				}
				livepeerPullCompleted = true
				log.Println("Done running livepeer cmd")
			}

			// Transcode using ffmpeg if livepeer pull fails or times out
			if livepeerPullCompleted == false {
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
			}

			// Calculate transcoding cost of the video.

			var transcodingCostWEI string
			transcodingCostWEI, err = util.CalculateTranscodingCost(demuxFileName)
			if err != nil {
				dataservice.SetAssetError(id.String(), fmt.Sprintf("calculating transcoding cost: %s", err), http.StatusFailedDependency)
				transcodingCostWEI = "0"
			}

			transcodingID := guuid.New()

			dataservice.CreateTranscodingDeal(model.TranscodingDeal{
				TranscodingID:   transcodingID.String(),
				TranscodingCost: transcodingCostWEI,
				Directory:       id.String(),
				StorageStatus:   false,
			})

			if livepeerPullCompleted == false {
				log.Println("lpcfalse")
				util.RemoveContents("./assets/" + id.String())
			} else {
				rmcmd := exec.Command("rm", "-rf", demuxFileName)
				_, err := rmcmd.Output()
				if err != nil {
					log.Println(err)
					return
				}

				pattern := "./assets/" + id.String() + "/*_source.mp4"
				matches, err := filepath.Glob(pattern)
				if err != nil {
					log.Println(err)
					return
				}
				if len(matches) == 1 {
					rmcmd = exec.Command("rm", "-rf", matches[0])
					_, err := rmcmd.Output()
					if err != nil {
						log.Println(err)
						return
					}
				}
			}

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

			// Set AssetStatus to 2 (storing in ipfs+filecoin network)
			dataservice.UpdateAssetStatus(id.String(), 2)

			ctx := context.Background()

			var currCID cid.Cid
			var streamURL string
			var pinataIpfsGateway = "https://gateway.pinata.cloud/ipfs/"
			var ipfsioGateway = "https://ipfs.io/ipfs/"
			var jid string
			var currFolderName string
			currCID, currFolderName, minerName, tok, jid, storagePrice, expiry, staged, err := util.RunPow(ctx, util.InitialPowergateSetup, "./assets/"+id.String())
			if err != nil {
				log.Println(err)
				if staged {
					currCIDStr := fmt.Sprintf("%s", currCID)
					pinataCID, pinataErr := util.PinFolder("assets/"+id.String(), "POW"+currCIDStr)
					if pinataErr != nil {
						dataservice.SetAssetError(id.String(), fmt.Sprintf("pinning to pinata: %s", pinataErr), http.StatusFailedDependency)
						return
					}
					if pinataCID == currCIDStr {
						fmt.Println("EQ")
						streamURL = pinataIpfsGateway + pinataCID
					} else {
						fmt.Println("NOTEQ", pinataCID, currCIDStr)
						streamURL = ipfsioGateway + currCIDStr
					}

					dataservice.CreateStorageDeal(model.StorageDeal{
						CID:           currCIDStr,
						Name:          currFolderName,
						AssetID:       id.String(),
						Miner:         "",
						StorageCost:   float64(0),
						Expiry:        uint32(0),
						TranscodingID: transcodingID.String(),
						Token:         tok,
						Status:        0, //pinned to ipfs
						JID:           jid,
					})
					dataservice.UpdateAsset(id.String(), transcodingCostWEI, "", float64(0), uint32(0), streamURL+"/root.m3u8")
					// Set AssetStatus to 3 (pushed CID for storage deal job)
					dataservice.UpdateAssetStatus(id.String(), 3)
				} else {
					dataservice.SetAssetError(id.String(), fmt.Sprintf("creating storage deal: %s", err), http.StatusFailedDependency)
				}
				return
			}

			log.Printf("CID: %s, currFolderName: %s\n", currCID, currFolderName)
			currCIDStr := fmt.Sprintf("%s", currCID)
			pinataCID, pinataErr := util.PinFolder("assets/"+id.String(), "POW"+currCIDStr)
			if pinataErr != nil {
				dataservice.SetAssetError(id.String(), fmt.Sprintf("pinning to pinata: %s", pinataErr), http.StatusFailedDependency)
				return
			}
			if pinataCID == currCIDStr {
				fmt.Println("EQ")
				streamURL = pinataIpfsGateway + pinataCID
			} else {
				fmt.Println("NOTEQ", pinataCID, currCIDStr)
				streamURL = ipfsioGateway + currCIDStr
			}

			dataservice.CreateStorageDeal(model.StorageDeal{
				CID:           currCIDStr,
				Name:          currFolderName,
				AssetID:       id.String(),
				Miner:         minerName,
				StorageCost:   float64(0),
				Expiry:        uint32(0),
				TranscodingID: transcodingID.String(),
				Token:         tok,
				Status:        0, //pinned to ipfs
				JID:           jid,
			})

			dataservice.UpdateAsset(id.String(), transcodingCostWEI, minerName, float64(storagePrice), uint32(expiry), streamURL+"/root.m3u8")

			// Set AssetStatus to 3 (pushed CID for storage deal job)
			dataservice.UpdateAssetStatus(id.String(), 3)
			dataservice.SetAssetError(id.String(), "", http.StatusOK)
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
			assetStatus := dataservice.GetAssetStatusIfExists(vars["asset_id"])
			assetError := dataservice.GetAssetError(vars["asset_id"])

			if assetError == "" {
				w.WriteHeader(http.StatusOK)
				var data = make(map[string]interface{})
				data["AssetID"] = vars["asset_id"]
				data["AssetStatus"] = assetStatus

				if assetStatus == 3 {
					data["CID"] = dataservice.GetCIDForAsset(vars["asset_id"])
					asset := dataservice.GetAsset(vars["asset_id"])
					data["TranscodingCost"] = asset.TranscodingCost
					data["StreamURL"] = asset.StreamURL

					if dataservice.GetStorageDealStatus(vars["asset_id"]) == 0 {
						data["Status"] = "Filecoin deal pending"
						data["Miner"] = asset.Miner
						data["StorageCost"] = asset.StorageCost
						data["Expiry"] = asset.Expiry
					} else if dataservice.GetStorageDealStatus(vars["asset_id"]) == 1 {
						data["Status"] = "Completed Filecoin storage deal"
						storageDeal := dataservice.GetStorageDeal(vars["asset_id"])
						data["Miner"] = storageDeal.Miner
						data["StorageCost"] = storageDeal.StorageCost
						data["Expiry"] = storageDeal.Expiry
					} else {
						data["Status"] = "Failed to store in Filecoin"
						data["Miner"] = asset.Miner
						data["StorageCost"] = asset.StorageCost
						data["Expiry"] = asset.Expiry
					}
				}
				util.WriteResponse(data, w)
			} else {
				w.WriteHeader(http.StatusOK)
				data := map[string]interface{}{
					"AssetID": vars["asset_id"],
					"Error":   dataservice.GetAssetError(vars["asset_id"]),
				}
				util.WriteResponse(data, w)
			}
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
