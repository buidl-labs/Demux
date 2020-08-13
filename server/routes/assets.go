package routes

import (
	"bufio"
	"context"
	"fmt"
	"github.com/buidl-labs/Demux/dataservice"
	"github.com/buidl-labs/Demux/model"
	"github.com/buidl-labs/Demux/util"
	guuid "github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/ipfs/go-cid"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
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
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			data := map[string]interface{}{
				"Error": fmt.Sprintf("please upload a file of size less than 30MB: %s", err),
			}
			util.WriteResponse(data, w)
			responded = true
			return
		}

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

			//orchWebhook, orchWebhookExists := os.LookupEnv("ORCH_WEBHOOK_URL")
			//if !orchWebhookExists {
			//	dataservice.SetAssetError(id.String(), "please provide the environment variable `ORCH_WEBHOOK_URL`", http.StatusFailedDependency)
			//	return
			//}
			//
			//log.Println("Starting livepeer transcoding")
			//
			//lpCmd := exec.Command("./livepeerPull/livepeer", "-pull", demuxFileName,
			//	"-recordingDir", "./assets/"+id.String(), "-transcodingOptions",
			//	"./livepeerPull/configs/profiles.json", "-orchWebhookUrl",
			//	orchWebhook, "-v", "99")
			//lpStdout, err := lpCmd.Output()
			//if err != nil {
			//	dataservice.SetAssetError(id.String(), fmt.Sprintf("livepeer transcoding: %s", err), http.StatusFailedDependency)
			//	return
			//}
			//_ = lpStdout
			//
			//log.Println("Done running livepeer cmd")

			// Transcode using ffmpeg if livepeer pull fails

			cmd1 := exec.Command("ffmpeg", "-i", demuxFileName, "-vf", "scale=-1:1080", "-c:v", "libx264", "-crf", "18", "-preset", "ultrafast", "-c:a", "copy", "./assets/"+id.String()+"/random1080p.mp4")
			stdout1, err := cmd1.Output()
			cmd2 := exec.Command("ffmpeg", "-i", demuxFileName, "-vf", "scale=-1:720", "-c:v", "libx264", "-crf", "18", "-preset", "ultrafast", "-c:a", "copy", "./assets/"+id.String()+"/random720p.mp4")
			stdout2, err := cmd2.Output()
			cmd3 := exec.Command("ffmpeg", "-i", demuxFileName, "-vf", "scale=-1:360", "-c:v", "libx264", "-crf", "18", "-preset", "ultrafast", "-c:a", "copy", "./assets/"+id.String()+"/random360p.mp4")
			stdout3, err := cmd3.Output()

			if err != nil {
				dataservice.SetAssetError(id.String(), fmt.Sprintf("ffmpeg transcoding: %s", err), http.StatusFailedDependency)
				return
			}
			_ = stdout1
			_ = stdout2
			_ = stdout3

			var transcodingCostWEI string
			transcodingCostWEI, err = util.CalculateTranscodingCost(demuxFileName)
			if err != nil {
				dataservice.SetAssetError(id.String(), fmt.Sprintf("calculating transcoding cost: %s", err), http.StatusFailedDependency)
				return
			}

			transcodingID := guuid.New()

			dataservice.CreateTranscodingDeal(model.TranscodingDeal{
				TranscodingID:   transcodingID.String(),
				TranscodingCost: transcodingCostWEI,
				Directory:       id.String(),
				StorageStatus:   false,
			})

			allFiles, err := ioutil.ReadDir("./assets/" + id.String())
			if err != nil {
				dataservice.SetAssetError(id.String(), fmt.Sprintf("reading asset directory: %s", err), http.StatusFailedDependency)
				return
			}

			exec.Command("mkdir", "./assets/"+id.String()+"/1080p").Output()
			exec.Command("mkdir", "./assets/"+id.String()+"/720p").Output()
			exec.Command("mkdir", "./assets/"+id.String()+"/360p").Output()

			log.Println("segmenting the transcoded videos...")

			for _, f := range allFiles {
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
			}

			log.Println("completed segmentation")

			// Delete original mp4 file
			exec.Command("rm", "-rf", demuxFileName).Output()

			// Set AssetStatus to 2 (storing in ipfs+filecoin network)
			dataservice.UpdateAssetStatus(id.String(), 2)

			// Store video in ipfs/filecoin network

			ctx := context.Background()

			var currCID cid.Cid
			var currFolderName string
			currCID, currFolderName, minerName, storagePrice, expiry, err := util.RunPow(ctx, util.InitialPowergateSetup, "./assets/"+id.String())
			if err != nil {
				dataservice.SetAssetError(id.String(), fmt.Sprintf("creating storage deal: %s", err), http.StatusFailedDependency)
				return
			}

			log.Printf("CID: %s, currFolderName: %s\n", currCID, currFolderName)
			currCIDStr := fmt.Sprintf("%s", currCID)

			dataservice.CreateStorageDeal(model.StorageDeal{
				CID:           currCIDStr,
				Name:          currFolderName,
				AssetID:       id.String(),
				Miner:         minerName,
				StorageCost:   float64(storagePrice),
				Expiry:        uint32(expiry),
				TranscodingID: transcodingID.String(),
			})

			var IpfsHostWithCID string
			IpfsHost, IpfsHostExists := os.LookupEnv("IPFS_HOST")
			if !IpfsHostExists {
				dataservice.SetAssetError(id.String(), "please provide the environment variable `IPFS_HOST`", http.StatusFailedDependency)
				return
			}

			if IpfsHost[len(IpfsHost)-1:] == "/" {
				IpfsHostWithCID = IpfsHost + currCIDStr
			} else {
				IpfsHostWithCID = IpfsHost + "/" + currCIDStr
			}

			abrStreamFile, err := os.Create("./assets/" + id.String() + "/root.m3u8")

			bWriter := bufio.NewWriter(abrStreamFile)

			n, err := bWriter.WriteString("#EXTM3U\n" +
				"#EXT-X-STREAM-INF:BANDWIDTH=6000000,RESOLUTION=1920x1080\n" +
				IpfsHostWithCID + "/1080p/myvid.m3u8\n" +
				"#EXT-X-STREAM-INF:BANDWIDTH=2000000,RESOLUTION=1280x720\n" +
				IpfsHostWithCID + "/720p/myvid.m3u8\n" +
				"#EXT-X-STREAM-INF:BANDWIDTH=500000,RESOLUTION=640x360\n" +
				IpfsHostWithCID + "/360p/myvid.m3u8\n")

			if err != nil {
				dataservice.SetAssetError(id.String(), fmt.Sprintf("creating the root m3u8 file: %s", err), http.StatusFailedDependency)
				return
			}

			log.Println("created the root m3u8 file")
			log.Printf("Wrote %d bytes\n", n)
			bWriter.Flush()

			abrStreamCID, abrStreamFileName, minerName, abrStreamStoragePrice, abrStreamExpiry, err := util.RunPow(ctx, util.InitialPowergateSetup, "./assets/"+id.String()+"/root.m3u8")
			if err != nil {
				dataservice.SetAssetError(id.String(), fmt.Sprintf("creating storage deal: %s", err), http.StatusFailedDependency)
				return
			}
			log.Printf("abrStreamCID: %s, abrStreamFileName: %s\n", abrStreamCID, abrStreamFileName)
			totalStoragePrice := abrStreamStoragePrice + storagePrice

			finalExpiry := expiry
			if abrStreamExpiry < expiry {
				finalExpiry = abrStreamExpiry
			}

			abrStreamCIDStr := fmt.Sprintf("%s", abrStreamCID)

			dataservice.UpdateStorageDeal(currCIDStr, abrStreamCIDStr, float64(totalStoragePrice), uint32(finalExpiry))
			dataservice.UpdateAsset(id.String(), transcodingCostWEI, minerName, float64(totalStoragePrice), uint32(expiry))

			// Set AssetStatus to 3 (completed storage process)
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
			return
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
					data["CID"], data["RootCID"] = dataservice.GetCIDsForAsset(vars["asset_id"])
					asset := dataservice.GetAsset(vars["asset_id"])
					data["TranscodingCost"] = asset.TranscodingCost
					data["Miner"] = asset.Miner
					data["StorageCost"] = asset.StorageCost
					data["Expiry"] = asset.Expiry
				}
				util.WriteResponse(data, w)
				return
			} else {
				w.WriteHeader(http.StatusOK)
				data := map[string]interface{}{
					"AssetID": vars["asset_id"],
					"Error":   dataservice.GetAssetError(vars["asset_id"]),
				}
				util.WriteResponse(data, w)
				return
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
			data := map[string]interface{}{
				"AssetID": nil,
				"Error":   "no such asset",
			}
			util.WriteResponse(data, w)
			return
		}
	}
}
