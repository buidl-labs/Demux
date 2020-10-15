package routes

import (
	"crypto/sha256"
	"encoding/hex"
	"math/big"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/buidl-labs/Demux/dataservice"
	"github.com/buidl-labs/Demux/model"
	"github.com/buidl-labs/Demux/util"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// AssetHandler enables checking the status of an asset in its demux lifecycle.
func AssetHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")

	if r.Method == "POST" {
		// Verify auth tokens
		tokenID, tokenSecret, ok := r.BasicAuth()

		if ok {
			sha256Digest := sha256.Sum256([]byte(tokenID + ":" + tokenSecret))
			sha256DigestStr := hex.EncodeToString(sha256Digest[:])

			if dataservice.IfUserExists(sha256DigestStr) {
				log.Info("User auth successful")
				// Generate a new assetID.
				id := uuid.New()

				// Create asset directory.
				cmd := exec.Command("mkdir", "./assets/"+id.String())
				stdout, err := cmd.Output()
				if err != nil {
					log.Error(err)
					w.WriteHeader(http.StatusFailedDependency)
					data := map[string]interface{}{
						"error": "could not create asset",
					}
					util.WriteResponse(data, w)
					return
				}
				_ = stdout

				// Create a new asset.
				dataservice.InsertAsset(model.Asset{
					AssetID:         id.String(),
					AssetReady:      false,
					AssetStatusCode: 0,
					AssetStatus:     "asset created",
					AssetError:      false,
					CreatedAt:       time.Now().Unix(),
					Thumbnail:       "https://user-images.githubusercontent.com/24296199/94940994-e923d080-04f1-11eb-8c3d-5aad1f31e91f.png",
				})

				// Create a new upload.
				dataservice.InsertUpload(model.Upload{
					AssetID: id.String(),
					URL:     os.Getenv("DEMUX_URL") + "fileupload/" + id.String(),
					Status:  false,
				})

				dataservice.IncrementUserAssetCount(sha256DigestStr)

				w.WriteHeader(http.StatusOK)
				data := map[string]interface{}{
					"asset_id": id.String(),
					"url":      os.Getenv("DEMUX_URL") + "fileupload/" + id.String(),
				}
				util.WriteResponse(data, w)
			} else {
				w.WriteHeader(http.StatusUnauthorized)
				data := map[string]interface{}{
					"error": "please use a valid TOKEN_ID and TOKEN_SECRET",
				}
				util.WriteResponse(data, w)
				return
			}
		} else {
			w.WriteHeader(http.StatusUnauthorized)
			data := map[string]interface{}{
				"error": "please use a valid TOKEN_ID and TOKEN_SECRET",
			}
			util.WriteResponse(data, w)
			return
		}
	}
}

// AssetStatusHandler returns the asset details and status.
func AssetStatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method == "GET" {
		// Collect path parameters
		vars := mux.Vars(r)

		if dataservice.IfAssetExists(vars["asset_id"]) {
			asset := dataservice.GetAsset(vars["asset_id"])
			transcodingDeal := dataservice.GetTranscodingDeal(vars["asset_id"])
			storageDeal := dataservice.GetStorageDeal(vars["asset_id"])

			w.WriteHeader(http.StatusOK)
			data := map[string]interface{}{
				"asset_id":          asset.AssetID,
				"asset_ready":       asset.AssetReady,
				"asset_status_code": asset.AssetStatusCode,
				"asset_status":      asset.AssetStatus,
				"asset_error":       asset.AssetError,
				"stream_url":        asset.StreamURL,
				"thumbnail":         asset.Thumbnail,
				"created_at":        asset.CreatedAt,
			}
			storageCostBigInt := new(big.Int)
			storageCostBigInt, ok := storageCostBigInt.SetString(storageDeal.StorageCost, 10)
			if !ok {
				data["storage_cost"] = big.NewInt(0)
			} else {
				data["storage_cost"] = storageCostBigInt
			}

			storageCostEstimatedBigInt := new(big.Int)
			storageCostEstimatedBigInt, ok = storageCostEstimatedBigInt.SetString(storageDeal.StorageCostEstimated, 10)
			if !ok {
				data["storage_cost_estimated"] = big.NewInt(0)
			} else {
				data["storage_cost_estimated"] = storageCostEstimatedBigInt
			}

			transcodingCostBigInt := new(big.Int)
			transcodingCostBigInt, ok = transcodingCostBigInt.SetString(transcodingDeal.TranscodingCost, 10)
			if !ok {
				data["transcoding_cost"] = big.NewInt(0)
			} else {
				data["transcoding_cost"] = transcodingCostBigInt
			}

			transcodingCostEstimatedBigInt := new(big.Int)
			transcodingCostEstimatedBigInt, ok = transcodingCostEstimatedBigInt.SetString(transcodingDeal.TranscodingCostEstimated, 10)
			if !ok {
				data["transcoding_cost_estimated"] = big.NewInt(0)
			} else {
				data["transcoding_cost_estimated"] = transcodingCostEstimatedBigInt
			}

			util.WriteResponse(data, w)
		} else {
			w.WriteHeader(http.StatusNotFound)
			data := map[string]interface{}{
				"asset_id": nil,
				"error":    "no such asset",
			}
			util.WriteResponse(data, w)
		}
	}
}
