package routes

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	"github.com/google/uuid"
	"github.com/ipfs/go-cid"
)

// UploadsHandler handles the asset uploads
func UploadsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {

		var responded = false

		tokenID, tokenSecret, ok := r.BasicAuth()
		fmt.Println("sent", "tokenID:", tokenID, "\ntokenSecret:", tokenSecret)
		fmt.Println("ok", ok)
		if ok {
			sha256Digest := sha256.Sum256([]byte(tokenID + ":" + tokenSecret))
			sha256DigestStr := hex.EncodeToString(sha256Digest[:])
			fmt.Println("now", sha256DigestStr)
			if dataservice.IfUserExists(sha256DigestStr) {
				fmt.Println("AssetCount++")
				dataservice.IncrementUserAssetCount(sha256DigestStr)
			} else {
				w.WriteHeader(http.StatusUnauthorized)
				data := map[string]interface{}{
					"error": "please use a valid TOKEN_ID and TOKEN_SECRET",
				}
				util.WriteResponse(data, w)
				responded = true
				return
			}
		} else {
			fmt.Println("not ok")
			w.WriteHeader(http.StatusUnauthorized)
			data := map[string]interface{}{
				"error": "please use a valid TOKEN_ID and TOKEN_SECRET",
			}
			util.WriteResponse(data, w)
			responded = true
			return
		}

		// TODO: handle the case when a remote file is sent
		// example: https://file-examples-com.github.io/uploads/2017/04/file_example_MP4_1280_10MG.mp4
		r.Body = http.MaxBytesReader(w, r.Body, 30*1024*1024)
		clientFile, handler, err := r.FormFile("inputfile")
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			data := map[string]interface{}{
				"error": fmt.Sprintf("please upload a file of size less than 30MB: %s", err),
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
				"error": "please upload an mp4 file",
			}
			util.WriteResponse(data, w)
			responded = true
			return
		}

		// Generate a new assetID.
		id := uuid.New()

		cmd := exec.Command("mkdir", "./assets/"+id.String())
		stdout, err := cmd.Output()
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusFailedDependency)
			data := map[string]interface{}{
				"error": "could not create asset",
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
				"error": "could not create asset",
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
				"error": "could not create asset",
			}
			util.WriteResponse(data, w)
			responded = true
			return
		}

		// Create a new asset.
		dataservice.CreateAsset(model.Asset{
			AssetID:         id.String(),
			AssetReady:      false,
			AssetStatusCode: 0,
			AssetStatus:     "video uploaded successfully",
			AssetError:      false,
			CreatedAt:       time.Now().Unix(),
		})

		go func() {

			livepeerPullCompleted := false

			livepeerAPIKey, livepeerAPIKeyExists := os.LookupEnv("LIVEPEER_COM_API_KEY")
			if !livepeerAPIKeyExists {
				log.Println("please provide the environment variable `LIVEPEER_COM_API_KEY`")
				dataservice.UpdateAssetStatus(id.String(), 1, "processing in livepeer", true)
				return
			}

			// Set AssetStatus to 1 (processing in livepeer)
			dataservice.UpdateAssetStatus(id.String(), 1, "processing in livepeer", false)

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
				dataservice.UpdateAssetStatus(id.String(), 1, "processing in livepeer", true)
				// TODO: this is temporary:
				livepeerPullCompleted = true
				// return
			case err := <-done:
				if err != nil {
					// Set AssetError to true
					dataservice.UpdateAssetStatus(id.String(), 1, "processing in livepeer", true)
					return
				}
				livepeerPullCompleted = true
				log.Println("Done running livepeer cmd")
			}

			// Transcode using ffmpeg if livepeer pull fails or times out
			if livepeerPullCompleted == false {
				// Set AssetError to true
				dataservice.UpdateAssetStatus(id.String(), 1, "processing in livepeer", true)
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

			items, err := ioutil.ReadDir("./assets/" + id.String())
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
							segments, err := ioutil.ReadDir("./assets/" + id.String() + "/" + f.Name() + "/" + res)
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

									stdout, err := exec.Command("ffprobe", "-i", "./assets/"+id.String()+"/"+f.Name()+"/"+res+"/"+segName, "-show_entries", "format=duration", "-v", "quiet", "-of", "csv=p=0").Output()
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

								m3u8strFile, err := os.Create("./assets/" + id.String() + "/" + f.Name() + "/" + res + ".m3u8")
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
			// pattern := "./assets/" + id.String() + "/*"
			// matches, err := filepath.Glob(pattern)
			// if err != nil {
			// 	log.Println(err)
			// 	// Set AssetError to true
			// 	dataservice.UpdateAssetStatus(id.String(), 2, "attempting to pin to ipfs", true)
			// 	return
			// }
			// if len(matches) == 2 {
			// 	fmt.Println("matchesm3u8", matches)
			// 	renameCmd := exec.Command("cp", matches[0], "./assets/"+id.String()+"/root.m3u8")
			// 	stdout, err := renameCmd.Output()
			// 	if err != nil {
			// 		log.Println(err)
			// 		// Set AssetError to true
			// 		dataservice.UpdateAssetStatus(id.String(), 2, "attempting to pin to ipfs", true)
			// 		return
			// 	}
			// 	_ = stdout
			// }

			// stdout, err := exec.Command("ffprobe", "-i", "./assets/"+id.String(), "-show_entries", "format=duration", "-v", "quiet", "-of", "csv=p=0").Output()
			// if err != nil {
			// 	return transcodingCostEstimated, fmt.Errorf("finding video duration: %s", err)
			// }
			// duration, err := strconv.ParseFloat(string(stdout)[:len(string(stdout))-2], 64)
			// if err != nil {
			// 	return transcodingCostEstimated, fmt.Errorf("finding video duration: %s", err)
			// }

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

			// Set AssetStatus to 2 (attempting to pin to ipfs)
			dataservice.UpdateAssetStatus(id.String(), 2, "attempting to pin to ipfs", false)

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
					dataservice.UpdateAssetStatus(id.String(), 2, "attempting to pin to ipfs", true)
					return
				}

				pattern := "./assets/" + id.String() + "/*.mp4"
				matches, err := filepath.Glob(pattern)
				if err != nil {
					log.Println(err)
					// Set AssetError to true
					dataservice.UpdateAssetStatus(id.String(), 2, "attempting to pin to ipfs", true)
					return
				}
				fmt.Println("mpfmatches", matches)
				for _, match := range matches {
					rmcmd = exec.Command("rm", "-rf", match)
					_, err := rmcmd.Output()
					if err != nil {
						log.Println(err)
						// Set AssetError to true
						dataservice.UpdateAssetStatus(id.String(), 2, "attempting to pin to ipfs", true)
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
					dataservice.UpdateAssetStatus(id.String(), 2, "attempting to pin to ipfs", true)
					return
				}
				if len(matches) == 1 {
					fmt.Println("matchesm3u8", matches)
					renameCmd := exec.Command("cp", matches[0], "./assets/"+id.String()+"/root.m3u8")
					stdout, err := renameCmd.Output()
					if err != nil {
						log.Println(err)
						// Set AssetError to true
						dataservice.UpdateAssetStatus(id.String(), 2, "attempting to pin to ipfs", true)
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
						StorageStatus:        "pinned to ipfs, attempting to store in filecoin",
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

					// Set AssetStatus to 3 (pinned to ipfs, attempting to store in filecoin)
					dataservice.UpdateAssetStatus(id.String(), 3, "pinned to ipfs, attempting to store in filecoin", false)
				} else {
					// Set AssetError to true
					dataservice.UpdateAssetStatus(id.String(), 2, "attempting to pin to ipfs", true)
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
				StorageStatus:        "pinned to ipfs, attempting to store in filecoin",
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

			// Set AssetStatus to 3 (pinned to ipfs, attempting to store in filecoin)
			dataservice.UpdateAssetStatus(id.String(), 3, "pinned to ipfs, attempting to store in filecoin", false)

			// Set AssetReady to true
			dataservice.UpdateAssetReady(id.String(), true)
		}()

		if responded == false {
			w.WriteHeader(http.StatusOK)
			data := map[string]interface{}{
				"asset_id": id.String(),
			}
			util.WriteResponse(data, w)
			responded = true
		}
	}
}
