package routes

import (
	"fmt"
	"math/big"
	"net/http"

	"github.com/buidl-labs/Demux/dataservice"
	"github.com/buidl-labs/Demux/util"
	"github.com/gorilla/mux"
)

// AssetStatusHandler enables checking the status of an asset in its demux lifecycle.
func AssetStatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method == "GET" {

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
				"created_at":        asset.CreatedAt,
			}
			storageCostBigInt := new(big.Int)
			storageCostBigInt, ok := storageCostBigInt.SetString(storageDeal.StorageCost, 10)
			if !ok {
				fmt.Println("SetString: error", ok)
				data["storage_cost"] = storageDeal.StorageCost
			} else {
				fmt.Println(storageCostBigInt)
				data["storage_cost"] = storageCostBigInt
			}

			storageCostEstimatedBigInt := new(big.Int)
			storageCostEstimatedBigInt, ok = storageCostEstimatedBigInt.SetString(storageDeal.StorageCostEstimated, 10)
			if !ok {
				fmt.Println("SetString: error", ok)
				data["storage_cost_estimated"] = storageDeal.StorageCostEstimated
			} else {
				fmt.Println(storageCostBigInt)
				data["storage_cost_estimated"] = storageCostEstimatedBigInt
			}

			transcodingCostBigInt := new(big.Int)
			transcodingCostBigInt, ok = transcodingCostBigInt.SetString(transcodingDeal.TranscodingCost, 10)
			if !ok {
				fmt.Println("SetString: error", ok)
				data["transcoding_cost"] = transcodingDeal.TranscodingCost
			} else {
				fmt.Println(transcodingCostBigInt)
				data["transcoding_cost"] = transcodingCostBigInt
			}

			transcodingCostEstimatedBigInt := new(big.Int)
			transcodingCostEstimatedBigInt, ok = transcodingCostEstimatedBigInt.SetString(transcodingDeal.TranscodingCostEstimated, 10)
			if !ok {
				fmt.Println("SetString: error", ok)
				data["transcoding_cost_estimated"] = transcodingDeal.TranscodingCostEstimated
			} else {
				fmt.Println(storageCostBigInt)
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
