package routes

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/buidl-labs/Demux/dataservice"
	"github.com/buidl-labs/Demux/model"
	"github.com/buidl-labs/Demux/util"

	guuid "github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/ipfs/go-cid"
)

var (
	powergateAddr = "127.0.0.1:5002"
)

// AssetsHandler handles the asset uploads
func AssetsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {

		// TODO: handle the case when a remote file is sent
		// example: https://file-examples-com.github.io/uploads/2017/04/file_example_MP4_1280_10MG.mp4
		r.Body = http.MaxBytesReader(w, r.Body, 30*1024*1024)
		clientFile, handler, err := r.FormFile("inputfile")
		if err != nil {
			log.Println(err)
			if handler.Size > 30*1024*1024 {
				log.Println("Please upload file of size <= 30MB")
			}
			return
		}

		defer clientFile.Close()

		fmt.Printf("Uploaded File: %+v\n", handler.Filename)
		fmt.Printf("File Size: %+v\n", handler.Size)
		fmt.Printf("MIME Header: %+v\n", handler.Header)

		id := guuid.New()
		dataservice.CreateAsset(model.Asset{
			AssetID:     id.String(),
			AssetName:   handler.Filename,
			AssetStatus: 0,
		})

		ss := strings.Split(handler.Filename, ".")

		if ss[len(ss)-1] != "mp4" {
			log.Println("Please upload an mp4 file")
			return
		}

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
		io.Copy(f, clientFile)

		setup := util.PowergateSetup{
			PowergateAddr: powergateAddr,
			MinerAddr:     "t01000", // TODO: select miner by looking at Asks
			SampleSize:    700,
			MaxParallel:   1,
			TotalSamples:  1,
			RandSeed:      22,
		}

		go func() {
			// set AssetStatus to 1 (transcoding)
			dataservice.UpdateAssetStatus(id.String(), 1)

			// Start transcoding
			fmt.Println("start transcoding")

			// var lpWg sync.WaitGroup
			// lpWg.Add(1)

			// go func() {
			// 	lpCmd := exec.Command("./livepeerPull/livepeer", "-pull", demuxFileName,
			// 		"-recordingDir", "./assets/"+id.String(), "-transcodingOptions",
			// 		"./livepeerPull/configs/profiles.json", "-orchWebhookUrl",
			// 		os.Getenv("ORCH_WEBHOOK_URL"), "-v", "99")
			// 	lpStdout, err := lpCmd.Output()
			// 	if err != nil {
			// 		fmt.Println("Some issue with livepeer transcoding", err)
			// 	}
			// 	_ = lpStdout
			// 	lpWg.Done()
			// }()

			// lpWg.Wait()
			// fmt.Println("Done running livepeer cmd")

			// Transcode using ffmpeg if livepeer pull fails

			cmd1 := exec.Command("ffmpeg", "-i", demuxFileName, "-vf", "scale=-1:1080", "-c:v", "libx264", "-crf", "18", "-preset", "ultrafast", "-c:a", "copy", "./assets/"+id.String()+"/random1080p.mp4")
			stdout1, err := cmd1.Output()
			cmd2 := exec.Command("ffmpeg", "-i", demuxFileName, "-vf", "scale=-1:720", "-c:v", "libx264", "-crf", "18", "-preset", "ultrafast", "-c:a", "copy", "./assets/"+id.String()+"/random720p.mp4")
			stdout2, err := cmd2.Output()
			cmd3 := exec.Command("ffmpeg", "-i", demuxFileName, "-vf", "scale=-1:360", "-c:v", "libx264", "-crf", "18", "-preset", "ultrafast", "-c:a", "copy", "./assets/"+id.String()+"/random360p.mp4")
			stdout3, err := cmd3.Output()

			if err != nil {
				fmt.Println("Some issue with transcoding")
				log.Println(err)
				return
			}
			_ = stdout1
			_ = stdout2
			_ = stdout3

			stdout, err := exec.Command("ffprobe", "-i", demuxFileName, "-show_entries", "format=duration", "-v", "quiet", "-of", "csv=p=0").Output()
			if err != nil {
				log.Println(err)
				return
			}
			duration, e := strconv.ParseFloat(string(stdout)[:len(string(stdout))-2], 64)
			if e != nil {
				log.Println(e)
				return
			}

			// Fetch orchestrator stats from livepeer pricing tool:
			// GET https://livepeer-pricing-tool.com/orchestratorStats

			orchestratorStats := "https://livepeer-pricing-tool.com/orchestratorStats"

			resp, err := http.Get(orchestratorStats)

			if err != nil {
				log.Println(err)
			}

			defer resp.Body.Close()

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Println(err)
			}

			orchStats, err := util.GetOrchestratorStats([]byte(body))

			weightSum := big.NewInt(0)
			productSum := big.NewFloat(0)

			for _, i := range orchStats {
				stake := new(big.Float).SetInt(i.DelegatedStake)
				ppp := new(big.Float).SetFloat64(i.PricePerPixel)
				product := stake.Mul(stake, ppp)

				weightSum.Add(weightSum, i.DelegatedStake)
				productSum.Add(productSum, product)
			}

			// Calculate weighted average price per pixel (weighted by Delegated Stake)

			// weighted pricePerPixel
			pricePerPixel := new(big.Float).Quo(productSum, new(big.Float).SetInt(weightSum))

			pixels := util.GetTotalPixels(int(duration))

			// Calculate livepeer price for uploaded video

			livepeerPrice := new(big.Float).SetInt(big.NewInt(int64(1)))
			livepeerPrice = livepeerPrice.Mul(new(big.Float).SetInt(big.NewInt(int64(pixels))), pricePerPixel)

			fmt.Println("livepeerprice", livepeerPrice)
			transcodingCostWEI := livepeerPrice.String()
			// transcodingCostWEI := fmt.Sprintf("%s", livepeerPrice)
			fmt.Println("transcodingCostWEI", transcodingCostWEI)

			transcodingID := guuid.New()
			fmt.Printf("tidd: %s", transcodingID)
			dataservice.CreateTranscodingDeal(model.TranscodingDeal{
				TranscodingID:   transcodingID.String(),
				TranscodingCost: transcodingCostWEI,
				Directory:       id.String(),
				StorageStatus:   false,
			})

			allfiles, err := ioutil.ReadDir("./assets/" + id.String())
			if err != nil {
				log.Fatal(err)
			}

			mkdir1080p := exec.Command("mkdir", "./assets/"+id.String()+"/1080p")
			mkdir720p := exec.Command("mkdir", "./assets/"+id.String()+"/720p")
			mkdir360p := exec.Command("mkdir", "./assets/"+id.String()+"/360p")
			mkdir1080p.Output()
			mkdir720p.Output()
			mkdir360p.Output()

			for _, f := range allfiles {
				fname := f.Name()
				fmt.Println("fname", fname)
				nm := strings.Split(fname, "/")[len(strings.Split(fname, "/"))-1]
				name := strings.Split(nm, ".")[0]
				fmt.Println("name", name)
				if len(name) > 5 {
					if name[len(name)-5:] == "1080p" {
						// 1080p
						_, err := util.CreateSegments(fname, "1080p", id)
						if err != nil {
							log.Println(err)
							return
						}
					} else if name[len(name)-4:] == "720p" {
						// 720p
						_, err := util.CreateSegments(fname, "720p", id)
						if err != nil {
							log.Println(err)
							return
						}
					} else if name[len(name)-4:] == "360p" {
						// 360p
						_, err := util.CreateSegments(fname, "360p", id)
						if err != nil {
							log.Println(err)
							return
						}
					}
				} else if len(name) > 4 {
					if name[len(name)-4:] == "720p" {
						// 720p
						_, err := util.CreateSegments(fname, "720p", id)
						if err != nil {
							log.Println(err)
							return
						}
					} else if name[len(name)-4:] == "360p" {
						// 360p
						_, err := util.CreateSegments(fname, "360p", id)
						if err != nil {
							log.Println(err)
							return
						}
					}
				}
			}

			// // Janitor: delete original mp4 file
			// exec.Command("rm", "-rf", demuxFileName)

			// set AssetStatus to 2 (storing in ipfs+filecoin network)
			dataservice.UpdateAssetStatus(id.String(), 2)

			// store video in ipfs/filecoin network

			ctx := context.Background()

			var currcid cid.Cid
			var currfname string
			currcid, currfname, minername, storageprice, expiry, err := util.RunPow(ctx, setup, "./assets/"+id.String())
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println("currcid", currcid, "currfname", currfname)

			transcodingIDStr := fmt.Sprintf("%s", transcodingID)
			currcidStr := fmt.Sprintf("%s", currcid)

			dataservice.CreateStorageDeal(model.StorageDeal{
				CID:           currcidStr,
				Name:          currfname,
				AssetID:       id.String(),
				Miner:         minername,
				StorageCost:   float64(storageprice),
				Expiry:        uint32(expiry),
				TranscodingID: transcodingIDStr,
			})

			var IpfsHost = "http://0.0.0.0:8080/ipfs/"
			var IpfsHostWithCID = IpfsHost + currcidStr
			fmt.Println("ipfshostcid", IpfsHostWithCID)

			rootm3u8File, err := os.Create("./assets/" + id.String() + "/root.m3u8")

			w := bufio.NewWriter(rootm3u8File)

			n, err := w.WriteString(`#EXTM3U
#EXT-X-STREAM-INF:BANDWIDTH=6000000,RESOLUTION=1920x1080
` + IpfsHostWithCID + `/1080p/myvid.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=2000000,RESOLUTION=1280x720
` + IpfsHostWithCID + `/720p/myvid.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=500000,RESOLUTION=640x360
` + IpfsHostWithCID + `/360p/myvid.m3u8`)

			if err != nil {
				fmt.Println(err)
				return
			}

			fmt.Printf("wrote %d bytes\n", n)
			w.Flush()

			rootm3u8currcid, currfname, minername, rootm3u8storageprice, newexpiry, err := util.RunPow(ctx, setup, "./assets/"+id.String()+"/root.m3u8")
			if err != nil {
				fmt.Println(err)
				return
			}
			rootm3u8currcidStr := fmt.Sprintf("%s", rootm3u8currcid)
			fmt.Println("rootcurrcid", rootm3u8currcid, "rootcurrfname", currfname)
			totalstorageprice := rootm3u8storageprice + storageprice

			finalexpiry := expiry
			if newexpiry < expiry {
				finalexpiry = newexpiry
			}

			dataservice.UpdateStorageDeal(currcidStr, rootm3u8currcidStr, float64(totalstorageprice), uint32(finalexpiry))
			dataservice.UpdateAsset(id.String(), transcodingCostWEI, minername, float64(totalstorageprice), uint32(expiry))

			// set AssetStatus to 3 (completed storage process)
			dataservice.UpdateAssetStatus(id.String(), 3)
		}()

		w.WriteHeader(http.StatusOK)
		data := map[string]interface{}{
			"AssetID": id.String(),
		}
		json, err := json.MarshalIndent(data, "", "    ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Fprintln(w, string(json))
	}
}

// AssetsStatusHandler enables checking the status of an asset in its demux lifecycle.
func AssetsStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	if dataservice.IfAssetExists(vars["asset_id"]) {
		assetStatus := dataservice.GetAssetStatusIfExists(vars["asset_id"])
		// fmt.Println("assssssssss", assetStatus)
		w.WriteHeader(http.StatusOK)

		var data = make(map[string]interface{})
		data["AssetID"] = vars["asset_id"]
		data["AssetStatus"] = assetStatus

		if assetStatus == 3 {
			fmt.Println("assss 3")
			data["CID"], data["RootCID"] = dataservice.GetCIDsForAsset(vars["asset_id"])
			asset := dataservice.GetAsset(vars["asset_id"])
			data["TranscodingCost"] = asset.TranscodingCost
			data["Miner"] = asset.Miner
			data["StorageCost"] = asset.StorageCost
			data["Expiry"] = asset.Expiry
		}
		json, err := json.MarshalIndent(data, "", "    ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Fprintln(w, string(json))
	} else {
		w.WriteHeader(http.StatusNotFound)
		data := map[string]interface{}{
			"AssetID": nil,
			"Error":   "No such asset",
		}
		json, err := json.MarshalIndent(data, "", "    ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Fprintln(w, string(json))
	}
}
