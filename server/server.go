package server

import (
	"fmt"
	"net/http"

	"github.com/buidl-labs/Demux/server/routes"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// HomeHandler handles the home route
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Hello, world!")
}

// StartServer starts the web server
func StartServer(serverPort string) {
	router := mux.NewRouter()

	router.HandleFunc("/", HomeHandler).Methods("GET", http.MethodOptions)
	router.HandleFunc("/asset/{asset_id}", routes.AssetStatusHandler).Methods("GET", http.MethodOptions)
	router.HandleFunc("/pricing", routes.PriceEstimateHandler).Methods("POST", http.MethodOptions)
	router.HandleFunc("/upload", routes.UploadsHandler).Methods("POST", http.MethodOptions)

	log.Infoln("Starting server at PORT", serverPort)
	log.Fatalln("Error in starting server", http.ListenAndServe(serverPort, router))
}
