package routes

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// PriceEstimateHandler handles the /pricing endpoint
func PriceEstimateHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	data := map[string]interface{}{
		"Pricing": "",
	}
	json, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Fprintln(w, string(json))
}
