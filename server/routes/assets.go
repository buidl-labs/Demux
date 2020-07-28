package routes

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/buidl-labs/Demux/dataservice"
	"github.com/buidl-labs/Demux/model"

	guuid "github.com/google/uuid"
	"github.com/gorilla/mux"
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

		go func() {
			// set AssetStatus to 1 (transcoding)
			dataservice.UpdateAssetStatus(id.String(), 1)

			// Start transcoding
			cmd1 := exec.Command("./livepeerPull/livepeer", "-pull", demuxFileName,
				"-recordingDir", "./assets/"+id.String(), "-transcodingOptions",
				"./livepeerPull/configs/profiles.json", "-orchWebhookUrl",
				os.Getenv("ORCH_WEBHOOK_URL"), "-v", "99")
			stdout1, err := cmd1.Output()

			if err != nil {
				fmt.Println("Some issue with transcoding")
				log.Println(err)
				return
			}
			_ = stdout1

			transcodingID := guuid.New()
			dataservice.CreateTranscodingDeal(model.TranscodingDeal{
				TranscodingID:   transcodingID.String(),
				TranscodingCost: 31.5,
				Directory:       id.String(),
				StorageStatus:   false,
			})

			// set AssetStatus to 2 (storing in ipfs+filecoin network)
			dataservice.UpdateAssetStatus(id.String(), 2)

			// TODO: store video and create storage deal

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
		data := map[string]interface{}{
			"AssetID":     vars["asset_id"],
			"AssetStatus": assetStatus,
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
