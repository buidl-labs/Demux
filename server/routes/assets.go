package routes

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/buidl-labs/Demux/dataservice"
	"github.com/buidl-labs/Demux/model"

	guuid "github.com/google/uuid"
	"github.com/gorilla/mux"
)

// Asset not decided yet whether it should be stored in db or somewhere else (or just remain in memory)
type Asset struct {
	AssetID string
	Status  int
}

// Transcode a video in the livepeer network
func Transcode(assetID string, demuxFileName string) <-chan model.TranscodingDeal {
	// TODO: Make livepeer do its job
	r := make(chan model.TranscodingDeal)

	go func() {
		defer close(r)

		cmd1 := exec.Command("./livepeerPull/livepeer", "-pull", demuxFileName,
			"-recordingDir", "./assets/"+assetID, "-transcodingOptions",
			"./livepeerPull/configs/profiles.json", "-orchWebhookUrl",
			os.Getenv("ORCH_WEBHOOK_URL"), "-v", "99")
		stdout1, err := cmd1.Output()

		if err != nil {
			fmt.Println("Some issue with transcoding")
			log.Println(err)
			return
		}
		_ = stdout1

		// Simulate a workload.
		time.Sleep(time.Second * 3)
		r <- model.TranscodingDeal{56, 21.5, "/randomuuiddirectory", false}
	}()

	return r
}

// Store transcoded video in the ipfs+filecoin network
func Store() <-chan string {
	// TODO: IPFS+Filecoin (powergate)
	r := make(chan string)

	go func() {
		defer close(r)

		// Simulate a workload.
		time.Sleep(time.Second * 3)
		r <- "somestring"
	}()

	return r
}

// AssetsHandler handles the asset uploads and status
func AssetsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// TODO: handle /assets/:asset_id endpoint
		// check video status
	} else if r.Method == "POST" {
		// upload a video

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
		dataservice.CreateAsset(model.Asset{id.String(), handler.Filename, 0})

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
		// fmt.Println("stdout", stdout)
		_ = stdout
		f, err := os.OpenFile("./assets/"+id.String()+"/"+handler.Filename, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			log.Println(err)
			return
		}
		fmt.Println(f.Name())
		demuxFileName := f.Name()
		defer f.Close()
		io.Copy(f, clientFile)

		// r := <-Transcode(id.String(), demuxFileName)

		// ch1 := make(chan string)
		// ch2 := make(chan string)
		// go doStuff(ch1, ch2)
		// ch1 <- id.String()
		// ch2 <- demuxFileName

		// _ = r
		// return

		go func() {
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

			// TODO: insert transcodingDeal in db
		}()

		/*
			assetID := guuid.New()
			fmt.Println(assetID)

			result := <-Transcode()
			fmt.Println(result)
			// TODO: now if trancoding is successful, store it
			result2 := <-Store()
			fmt.Println(result2)
		*/
	}
	vars := mux.Vars(r)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Hello, world!\n%v\n", vars)
}

func doStuff(assetID chan string, demuxFileName chan string) {
	cmd1 := exec.Command("./livepeerPull/livepeer", "-pull", <-demuxFileName,
		"-recordingDir", "./assets/"+<-assetID, "-transcodingOptions",
		"./livepeerPull/configs/profiles.json", "-orchWebhookUrl",
		os.Getenv("ORCH_WEBHOOK_URL"), "-v", "99")
	stdout1, err := cmd1.Output()

	if err != nil {
		fmt.Println("Some issue with transcoding")
		log.Println(err)
		return
	}
	_ = stdout1
}
