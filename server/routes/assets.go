package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"

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
			// cmd1 := exec.Command("./livepeerPull/livepeer", "-pull", demuxFileName,
			// 	"-recordingDir", "./assets/"+id.String(), "-transcodingOptions",
			// 	"./livepeerPull/configs/profiles.json", "-orchWebhookUrl",
			// 	os.Getenv("ORCH_WEBHOOK_URL"), "-v", "99")
			// cmd1 := exec.Command("ffmpeg", "-i", demuxFileName, "-vf", "scale=-1:1080", "-c:v", "libx264", "-crf", "18", "-preset", "veryfast", "-c:a", "copy", "./assets/"+id.String()+"/random1080p.mp4")
			// stdout1, err := cmd1.Output()

			// if err != nil {
			// 	fmt.Println("Some issue with transcoding")
			// 	log.Println(err)
			// 	return
			// }
			// _ = stdout1

			transcodingID := guuid.New()
			fmt.Printf("tidd: %s", transcodingID)
			dataservice.CreateTranscodingDeal(model.TranscodingDeal{
				TranscodingID:   transcodingID.String(),
				TranscodingCost: 31.5,
				Directory:       id.String(),
				StorageStatus:   false,
			})

			fmt.Println("./assets/" + id.String() + "/" + demuxFileName)
			cmd := exec.Command(
				"ffmpeg", "-i", demuxFileName,
				"-profile:v", "baseline", "-level", "3.0", "-start_number", "0",
				"-hls_time", "10", "-hls_list_size", "0", "-f", "hls",
				"./assets/"+id.String()+"/myvid.m3u8")
			stdout, err := cmd.Output()
			if err != nil {
				log.Println(err)
				return
			}
			_ = stdout

			files, err := ioutil.ReadDir("./assets/" + id.String())
			if err != nil {
				log.Fatal(err)
			}

			var segments []string
			fmt.Println("files are:")
			for _, f := range files {
				fname := f.Name()
				if strings.Split(fname, ".")[1] == "ts" {
					fmt.Println("./assets/" + id.String() + "/" + fname)
					segments = append(segments, "./assets/"+id.String()+"/"+fname)
				}
			}

			// set AssetStatus to 2 (storing in ipfs+filecoin network)
			dataservice.UpdateAssetStatus(id.String(), 2)

			// store video in ipfs/filecoin network

			ctx := context.Background()

			var currcid cid.Cid
			var currfname string
			var fileCidMap = make(map[string]string)

			var wg sync.WaitGroup
			wg.Add(len(segments))

			for _, segment := range segments {
				go func(segment string) {
					currcid, currfname, err = util.RunPow(ctx, setup, segment)
					if err != nil {
						fmt.Println(err)
						return
					}
					fileCidMap[currfname] = fmt.Sprintf("%s", currcid)

					wg.Done()
				}(segment)
			}

			wg.Wait()

			lines, err := util.ReadLines("./assets/" + id.String() + "/myvid.m3u8")
			if err != nil {
				fmt.Println(err)
				return
			}
			var IpfsHost = "http://0.0.0.0:8080/ipfs/"
			for i, line := range lines {
				fmt.Printf("line %d: %s\n", i, line)
				if strings.HasPrefix(line, "myvid") {
					if strings.Split(line, ".")[1] == "ts" {
						fmt.Println("this", fileCidMap["./assets/"+id.String()+"/"+line])
						lines[i] = IpfsHost + fileCidMap["./assets/"+id.String()+"/"+line]
					}
				}
			}
			fmt.Println("now:", lines)

			if err := util.WriteLines(lines, "./assets/"+id.String()+"/myvid.m3u8"); err != nil {
				log.Fatalf("writeLines: %s", err)
				return
			}

			m3u8cid, m3u8fname, err := util.RunPow(ctx, setup, "./assets/"+id.String()+"/myvid.m3u8")
			if err != nil {
				fmt.Println(err)
				return
			}

			m3u8cidStr := fmt.Sprintf("%s", m3u8cid)
			transcodingIDStr := fmt.Sprintf("%s", transcodingID)

			dataservice.CreateStorageDeal(model.StorageDeal{
				CID:           m3u8cidStr,
				Name:          m3u8fname,
				AssetID:       id.String(),
				StorageCost:   5.0,        // fake cost
				Expiry:        1609459200, // fake timestamp
				TranscodingID: transcodingIDStr,
			})

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
		w.WriteHeader(http.StatusOK)

		var data = make(map[string]interface{})
		data["AssetID"] = vars["asset_id"]
		data["AssetStatus"] = assetStatus

		if assetStatus == 3 {
			data["CID"] = dataservice.GetCIDForAsset(vars["asset_id"])
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
