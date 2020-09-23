package server

import (
	"fmt"
	"net/http"

	"github.com/buidl-labs/Demux/server/routes"

	"github.com/gorilla/handlers"
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

	router.HandleFunc("/", HomeHandler).Methods("GET")
	router.HandleFunc("/asset", routes.AssetsHandler).Methods("POST")
	router.HandleFunc("/asset/{asset_id}", routes.AssetsStatusHandler).Methods("GET")
	router.HandleFunc("/pricing", routes.PriceEstimateHandler).Methods("POST")

	log.Infoln("Starting server at PORT", serverPort)
	log.Fatalln("Error in starting server", http.ListenAndServe(serverPort, handlers.CORS()(router)))
}
