package routes

import (
	"io"
	"math/big"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/buidl-labs/Demux/dataservice"
	"github.com/buidl-labs/Demux/model"
	"github.com/buidl-labs/Demux/util"

	guuid "github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

// PriceEstimateHandler handles the /pricing endpoint
func PriceEstimateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {

		var responded = false

		// TODO: handle the case when a remote file is sent
		// example: https://file-examples-com.github.io/uploads/2017/04/file_example_MP4_1280_10MG.mp4
		r.Body = http.MaxBytesReader(w, r.Body, 30*1024*1024)
		clientFile, handler, err := r.FormFile("inputfile")
		if err != nil {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			data := map[string]interface{}{
				"Error": "Please upload a file of size less than 30MB",
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
				"Error": "Please upload an mp4 file",
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
		})

		transcodingCostWEI, err := util.CalculateTranscodingCost(demuxFileName)
		if err != nil {
			// TODO: handle this case
			log.Println(err)
			transcodingCostWEI = big.NewInt(0)
			// w.WriteHeader(http.StatusFailedDependency)
			// data := map[string]interface{}{
			// 	"Error": "could not estimate transcoding cost",
			// }
			// util.WriteResponse(data, w)
			// responded = true
			// return
		}

		// TODO: Calculate powergate (filecoin) storage price

		// ctx, cancel := context.WithCancel(context.Background())
		// defer cancel()
		// pgClient, _ := powc.NewClient(util.InitialPowergateSetup.PowergateAddr)
		// defer func() {
		// 	if err := pgClient.Close(); err != nil {
		// 		log.Errorf("closing powergate client: %s", err)
		// 	}
		// }()

		// index, err := pgClient.Asks.Get(ctx)
		// if err != nil {
		// 	log.Errorf("getting asks: %s", err)
		// }
		// if len(index.Storage) > 0 {
		// 	log.Printf("Storage median price: %v\n", index.StorageMedianPrice)
		// 	log.Printf("Last updated: %v\n", index.LastUpdated.Format("01/02/06 15:04 MST"))
		// }

		// TODO: Convert total price to USD and return

		if responded == false {
			w.WriteHeader(http.StatusOK)
			data := map[string]interface{}{
				"TranscodingCostEstimated": transcodingCostWEI,
			}
			util.WriteResponse(data, w)
		}
	}
}
