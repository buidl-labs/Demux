package server

import (
	"fmt"
	"net/http"

	"github.com/buidl-labs/Demux/dataservice"
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
func StartServer(serverPort string, db dataservice.DatabaseHelper) {
	assetDB := dataservice.NewAssetDatabase(db)
	uploadDB := dataservice.NewUploadDatabase(db)
	transcodingDealDB := dataservice.NewTranscodingDealDatabase(db)
	storageDealDB := dataservice.NewStorageDealDatabase(db)
	userDB := dataservice.NewUserDatabase(db)
	sizeRatioDB := dataservice.NewSizeRatioDatabase(db)
	meanSizeRatioDB := dataservice.NewMeanSizeRatioDatabase(db)

	router := mux.NewRouter()

	router.HandleFunc("/", HomeHandler).Methods("GET", http.MethodOptions)
	router.HandleFunc("/asset", func(w http.ResponseWriter, r *http.Request) {
		routes.AssetHandler(w, r, userDB, assetDB, uploadDB)
	}).Methods("POST", http.MethodOptions)
	router.HandleFunc("/asset/{asset_id}", func(w http.ResponseWriter, r *http.Request) {
		routes.AssetStatusHandler(w, r, assetDB, transcodingDealDB, storageDealDB)
	}).Methods("GET", http.MethodOptions)
	router.HandleFunc("/pricing", func(w http.ResponseWriter, r *http.Request) {
		routes.PriceEstimateHandler(w, r, meanSizeRatioDB)
	}).Methods("POST", http.MethodOptions)
	router.HandleFunc("/fileupload/{asset_id}", func(w http.ResponseWriter, r *http.Request) {
		routes.FileUploadHandler(w, r, assetDB, uploadDB, transcodingDealDB, storageDealDB, sizeRatioDB, meanSizeRatioDB)
	}).Methods("GET", "POST", "PATCH", "HEAD", "OPTIONS", "PUT")
	router.HandleFunc("/upload/{asset_id}", func(w http.ResponseWriter, r *http.Request) {
		routes.UploadStatusHandler(w, r, uploadDB)
	}).Methods("GET", http.MethodOptions)

	log.Info("Starting server at PORT", serverPort)
	log.Fatal("Error in starting server", http.ListenAndServe(serverPort, router))
}
