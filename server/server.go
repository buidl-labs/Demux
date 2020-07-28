package server

import (
	// "encoding/json"
	"fmt"
	"net/http"

	// "strconv"

	// "github.com/buidl-labs/Demux/dataservice"
	"github.com/buidl-labs/Demux/model"
	"github.com/buidl-labs/Demux/server/routes"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Hello, world!\n%v\n", vars)
	// fmt.Fprintf(w, "Category: %v\n", vars["category"])
}

func StartServer(serverPort string) model.StorageDeal {
	router := mux.NewRouter()

	router.HandleFunc("/", HomeHandler).Methods("GET")
	router.HandleFunc("/assets", routes.AssetsHandler).Methods("POST")
	router.HandleFunc("/assets/{asset_id}", routes.AssetsHandler).Methods("GET")
	router.HandleFunc("/pricing", routes.PriceEstimateHandler).Methods("POST")

	log.Infoln("Starting server at PORT", serverPort)
	log.Fatalln("Error in starting server", http.ListenAndServe(serverPort, handlers.CORS()(router)))
	orch := model.StorageDeal{}
	return orch
}
