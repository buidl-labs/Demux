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

			// livepeerContext, cancel := context.WithTimeout(context.Background(), 30)
			// if cancel != nil {
			// 	fmt.Println(cancel)
			// }
			// Transcode(livepeerContext)

			// Start transcoding
			fmt.Println("start transcoding", demuxFileName)
			// cmd1 := exec.Command("./livepeerPull/livepeer", "-pull", demuxFileName,
			// 	"-recordingDir", "./assets/"+id.String(), "-transcodingOptions",
			// 	"./livepeerPull/configs/profiles.json", "-orchWebhookUrl",
			// 	os.Getenv("ORCH_WEBHOOK_URL"), "-v", "99")
			// stdout1, err := cmd1.Output()

			// transcoding using ffmpeg for now
			cmd1 := exec.Command("ffmpeg", "-i", demuxFileName, "-vf", "scale=-1:1080", "-c:v", "libx264", "-crf", "18", "-preset", "medium", "-c:a", "copy", "./assets/"+id.String()+"/random1080p.mp4")
			stdout1, err := cmd1.Output()
			cmd2 := exec.Command("ffmpeg", "-i", demuxFileName, "-vf", "scale=-1:720", "-c:v", "libx264", "-crf", "18", "-preset", "medium", "-c:a", "copy", "./assets/"+id.String()+"/random720p.mp4")
			stdout2, err := cmd2.Output()
			cmd3 := exec.Command("ffmpeg", "-i", demuxFileName, "-vf", "scale=-1:360", "-c:v", "libx264", "-crf", "18", "-preset", "medium", "-c:a", "copy", "./assets/"+id.String()+"/random360p.mp4")
			stdout3, err := cmd3.Output()

			if err != nil {
				fmt.Println("Some issue with transcoding")
				log.Println(err)
				return
			}
			// fmt.Println("livepeerstdout", stdout1)
			_ = stdout1
			_ = stdout2
			_ = stdout3

			transcodingID := guuid.New()
			fmt.Printf("tidd: %s\n", transcodingID)
			dataservice.CreateTranscodingDeal(model.TranscodingDeal{
				TranscodingID:   transcodingID.String(),
				TranscodingCost: 31.5,
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
						// 1080
						_, err := util.CreateSegments(fname, "1080p", id)
						if err != nil {
							log.Println(err)
							return
						}
					} else if name[len(name)-4:] == "720p" {
						// 720
						_, err := util.CreateSegments(fname, "720p", id)
						if err != nil {
							log.Println(err)
							return
						}
					} else if name[len(name)-4:] == "360p" {
						// 360
						_, err := util.CreateSegments(fname, "360p", id)
						if err != nil {
							log.Println(err)
							return
						}
					}
				} else if len(name) > 4 {
					if name[len(name)-4:] == "720p" {
						// 720
						_, err := util.CreateSegments(fname, "720p", id)
						if err != nil {
							log.Println(err)
							return
						}
					} else if name[len(name)-4:] == "360p" {
						// 360
						_, err := util.CreateSegments(fname, "360p", id)
						if err != nil {
							log.Println(err)
							return
						}
					}
				}
			}

			// fmt.Println("./assets/" + id.String() + "/" + demuxFileName)
			// cmd := exec.Command(
			// 	"ffmpeg", "-i", demuxFileName,
			// 	"-profile:v", "baseline", "-level", "3.0", "-start_number", "0",
			// 	"-hls_time", "10", "-hls_list_size", "0", "-f", "hls",
			// 	"./assets/"+id.String()+"/myvid.m3u8")
			// stdout, err := cmd.Output()
			// if err != nil {
			// 	log.Println(err)
			// 	return
			// }
			// _ = stdout

			// files, err := ioutil.ReadDir("./assets/" + id.String())
			// if err != nil {
			// 	log.Fatal(err)
			// }

			// var segments []string
			// fmt.Println("files are:")
			// for _, f := range files {
			// 	fname := f.Name()
			// 	if strings.Split(fname, ".")[1] == "ts" {
			// 		fmt.Println("./assets/" + id.String() + "/" + fname)
			// 		segments = append(segments, "./assets/"+id.String()+"/"+fname)
			// 	}
			// }

			files1080p, err := ioutil.ReadDir("./assets/" + id.String() + "/1080p")
			if err != nil {
				log.Fatal(err)
			}
			var segments1080p []string
			fmt.Println("1080p files:")
			for _, f := range files1080p {
				fname := f.Name()
				if strings.Split(fname, ".")[1] == "ts" {
					fmt.Println("./assets/" + id.String() + "/1080p/" + fname)
					segments1080p = append(segments1080p, "./assets/"+id.String()+"/1080p/"+fname)
				}
			}

			files720p, err := ioutil.ReadDir("./assets/" + id.String() + "/720p")
			if err != nil {
				log.Fatal(err)
			}
			var segments720p []string
			fmt.Println("720p files:")
			for _, f := range files720p {
				fname := f.Name()
				if strings.Split(fname, ".")[1] == "ts" {
					fmt.Println("./assets/" + id.String() + "/720p/" + fname)
					segments720p = append(segments720p, "./assets/"+id.String()+"/720p/"+fname)
				}
			}

			files360p, err := ioutil.ReadDir("./assets/" + id.String() + "/360p")
			if err != nil {
				log.Fatal(err)
			}
			var segments360p []string
			fmt.Println("360p files:")
			for _, f := range files360p {
				fname := f.Name()
				if strings.Split(fname, ".")[1] == "ts" {
					fmt.Println("./assets/" + id.String() + "/360p/" + fname)
					segments360p = append(segments360p, "./assets/"+id.String()+"/360p/"+fname)
				}
			}

			// set AssetStatus to 2 (storing in ipfs+filecoin network)
			dataservice.UpdateAssetStatus(id.String(), 2)

			// store video in ipfs/filecoin network

			ctx := context.Background()

			var currcid cid.Cid
			var currfname string
			var fileCidMap = make(map[string]string)

			var resolutions [][]string
			resolutions = append(resolutions, segments1080p)
			resolutions = append(resolutions, segments720p)
			resolutions = append(resolutions, segments360p)

			var wg sync.WaitGroup
			// wg.Add(len(segments))
			wg.Add(len(segments1080p) * 3)
			for _, res := range resolutions {
				go func(segments []string) {
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
				}(res)
			}

			// for _, segment := range segments {
			// 	go func(segment string) {
			// 		currcid, currfname, err = util.RunPow(ctx, setup, segment)
			// 		if err != nil {
			// 			fmt.Println(err)
			// 			return
			// 		}
			// 		fileCidMap[currfname] = fmt.Sprintf("%s", currcid)

			// 		wg.Done()
			// 	}(segment)
			// }

			wg.Wait()

			// lines, err := util.ReadLines("./assets/" + id.String() + "/myvid.m3u8")
			lines1080p, err := util.ReadLines("./assets/" + id.String() + "/1080p/myvid.m3u8")
			if err != nil {
				fmt.Println(err)
				return
			}
			lines720p, err := util.ReadLines("./assets/" + id.String() + "/720p/myvid.m3u8")
			if err != nil {
				fmt.Println(err)
				return
			}
			lines360p, err := util.ReadLines("./assets/" + id.String() + "/360p/myvid.m3u8")
			if err != nil {
				fmt.Println(err)
				return
			}

			// TODO: (i think it's done now) lines.extend(lines1080p) # (like in python)
			lines := []string{}
			lines = append(lines, lines1080p...)
			lines = append(lines, lines720p...)
			lines = append(lines, lines360p...)

			var IpfsHost = "http://0.0.0.0:8080/ipfs/"
			// for i, line := range lines {
			// 	fmt.Printf("line %d: %s\n", i, line)
			// 	if strings.HasPrefix(line, "myvid") {
			// 		if strings.Split(line, ".")[1] == "ts" {
			// 			// TODO: change path
			// 			fmt.Println("this", fileCidMap["./assets/"+id.String()+"/"+line])
			// 			lines[i] = IpfsHost + fileCidMap["./assets/"+id.String()+"/"+line]
			// 		}
			// 	}
			// }
			fmt.Println("now:", lines)

			// if err := util.WriteLines(lines, "./assets/"+id.String()+"/myvid.m3u8"); err != nil {
			// 	log.Fatalf("writeLines: %s", err)
			// 	return
			// }

			for i, line := range lines1080p {
				fmt.Printf("line %d: %s\n", i, line)
				if strings.HasPrefix(line, "myvid") {
					if strings.Split(line, ".")[1] == "ts" {
						// TODO: change path
						fmt.Println("this", fileCidMap["./assets/"+id.String()+"/1080p/"+line])
						lines1080p[i] = IpfsHost + fileCidMap["./assets/"+id.String()+"/1080p/"+line]
					}
				}
			}
			if err := util.WriteLines(lines1080p, "./assets/"+id.String()+"/1080p/myvid.m3u8"); err != nil {
				log.Fatalf("writeLines: %s", err)
				return
			}

			for i, line := range lines720p {
				fmt.Printf("line %d: %s\n", i, line)
				if strings.HasPrefix(line, "myvid") {
					if strings.Split(line, ".")[1] == "ts" {
						// TODO: change path
						fmt.Println("this", fileCidMap["./assets/"+id.String()+"/720p/"+line])
						lines720p[i] = IpfsHost + fileCidMap["./assets/"+id.String()+"/720p/"+line]
					}
				}
			}
			if err := util.WriteLines(lines720p, "./assets/"+id.String()+"/720p/myvid.m3u8"); err != nil {
				log.Fatalf("writeLines: %s", err)
				return
			}

			for i, line := range lines360p {
				fmt.Printf("line %d: %s\n", i, line)
				if strings.HasPrefix(line, "myvid") {
					if strings.Split(line, ".")[1] == "ts" {
						// TODO: change path
						fmt.Println("this", fileCidMap["./assets/"+id.String()+"/360p/"+line])
						lines360p[i] = IpfsHost + fileCidMap["./assets/"+id.String()+"/360p/"+line]
					}
				}
			}
			if err := util.WriteLines(lines360p, "./assets/"+id.String()+"/360p/myvid.m3u8"); err != nil {
				log.Fatalf("writeLines: %s", err)
				return
			}

			resos := [3]string{"1080p", "720p", "360p"}
			var m3u8CidMap = make(map[string]string)

			var wgm3u8 sync.WaitGroup
			wgm3u8.Add(len(resos))
			for _, res := range resos {
				go func(res string) {
					currm3u8cid, currfname, err := util.RunPow(ctx, setup, "./assets/"+id.String()+"/"+res+"/myvid.m3u8")
					if err != nil {
						fmt.Println(err)
						return
					}
					m3u8CidMap[currfname] = fmt.Sprintf("%s", currm3u8cid)

					wgm3u8.Done()
				}(res)
			}
			wgm3u8.Wait()

			fmt.Println("msu8cidmap", m3u8CidMap)

			// m3u8cid, m3u8fname, err := util.RunPow(ctx, setup, "./assets/"+id.String()+"/myvid.m3u8")
			// if err != nil {
			// 	fmt.Println(err)
			// 	return
			// }

			// m3u8cidStr := fmt.Sprintf("%s", m3u8cid)
			transcodingIDStr := fmt.Sprintf("%s", transcodingID)

			dataservice.CreateStorageDeal(model.StorageDeal{
				DealID:        guuid.New().String(),
				CID1080p:      m3u8CidMap["./assets/"+id.String()+"/1080p/myvid.m3u8"],
				CID720p:       m3u8CidMap["./assets/"+id.String()+"/720p/myvid.m3u8"],
				CID360p:       m3u8CidMap["./assets/"+id.String()+"/360p/myvid.m3u8"],
				AssetID:       id.String(),
				Miner:         "t01000",   // fake miner
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

		deal, err := dataservice.GetDealForAsset(vars["asset_id"])
		if err != nil {
			fmt.Println(err)
		}
		if assetStatus == 3 {
			data["CID1080p"] = deal.CID1080p
			data["CID720p"] = deal.CID720p
			data["CID360p"] = deal.CID360p
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
