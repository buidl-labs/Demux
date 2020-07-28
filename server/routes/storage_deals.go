package routes

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

// StorageDealsHandler handles the /pricing endpoint
func StorageDealsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Hello, world!\n%v\n", vars)
	// fmt.Fprintf(w, "Category: %v\n", vars["category"])
}
