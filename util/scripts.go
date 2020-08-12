package util

import (
	"encoding/json"
	"fmt"
	golog "log"
	"net/http"
)

// WriteResponse writes some response for a given http request.
func WriteResponse(data map[string]interface{}, w http.ResponseWriter) {
	jsonData, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		fmt.Fprintln(w, data)
		golog.Println(err)
		return
	}
	fmt.Fprintln(w, string(jsonData))
	return
}

func Upload(w http.ResponseWriter, r *http.Request) {}
