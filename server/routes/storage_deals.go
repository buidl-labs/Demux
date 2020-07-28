package routes

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// StorageDealsHandler handles the /pricing endpoint
func StorageDealsHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	data := map[string]interface{}{
		"StorageDeals": "",
	}
	json, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Fprintln(w, string(json))
}
