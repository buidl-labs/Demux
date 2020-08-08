package routes

import (
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
)

// PriceEstimateHandler handles the /pricing endpoint
func PriceEstimateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		// Upload video to demux and perform checks

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

		stdout1, err := exec.Command("ffprobe", "-i", demuxFileName, "-show_entries", "format=duration", "-v", "quiet", "-of", "csv=p=0").Output()
		if err != nil {
			log.Println(err)
			return
		}
		duration, e := strconv.ParseFloat(string(stdout1)[:len(string(stdout1))-2], 64)
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

		// Calculate powergate (filecoin) storage price

		// Convert total price to USD and return

		data := map[string]interface{}{
			"PricePerPixel": pricePerPixel,
			"LivepeerPrice": livepeerPrice,
		}
		json, err := json.MarshalIndent(data, "", "    ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Fprintln(w, string(json))
	}
}
