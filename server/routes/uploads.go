package routes

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"

	"github.com/gorilla/mux"
)

func UploadsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	assetID := vars["asset_id"]
	if _, err := os.Stat("./assets/" + assetID); os.IsNotExist(err) {
		// path/to/whatever does not exist
		fmt.Println("doesn't exist")
		return
	}
	exec.Command("mkdir", "./assets/"+assetID+"/temp").Output()
	tempFolder := "./assets/" + assetID + "/temp/"

	switch r.Method {
	case "GET":
		resumableIdentifier, _ := r.URL.Query()["resumableIdentifier"]
		resumableChunkNumber, _ := r.URL.Query()["resumableChunkNumber"]
		path := fmt.Sprintf("%s%s", tempFolder, resumableIdentifier[0])
		relativeChunk := fmt.Sprintf("%s%s%s%s", path, "/", "part", resumableChunkNumber[0])

		if _, err := os.Stat(path); os.IsNotExist(err) {
			os.Mkdir(path, os.ModePerm)
		}

		if _, err := os.Stat(relativeChunk); os.IsNotExist(err) {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusMethodNotAllowed)
		} else {
			http.Error(w, "Chunk already exist", http.StatusCreated)
		}

	default:
		r.ParseMultipartForm(10 << 20)
		file, _, err := r.FormFile("file")
		if err != nil {
			print(err.Error())
			return
		}
		defer file.Close()
		resumableIdentifier, _ := r.URL.Query()["resumableIdentifier"]
		resumableChunkNumber, _ := r.URL.Query()["resumableChunkNumber"]
		path := fmt.Sprintf("%s%s", tempFolder, resumableIdentifier[0])
		relativeChunk := fmt.Sprintf("%s%s%s%s", path, "/", "part", resumableChunkNumber[0])

		if _, err := os.Stat(path); os.IsNotExist(err) {
			os.Mkdir(path, os.ModePerm)
		}

		f, err := os.OpenFile(relativeChunk, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			print(err.Error())
		}
		defer f.Close()
		io.Copy(f, file)

		/*
			If it is the last chunk, trigger the recombination of chunks
		*/
		resumableTotalChunks, _ := r.URL.Query()["resumableTotalChunks"]

		current, err := strconv.Atoi(resumableChunkNumber[0])
		total, err := strconv.Atoi(resumableTotalChunks[0])
		if current == total {
			print("Combining chunks into one file")

		}
	}

	// if r.Method == "GET" {

	// } else if r.Method == "POST" {
	// 	vars := mux.Vars(r)
	// 	assetID := vars["asset_id"]
	// 	fmt.Println(assetID)
	// 	if _, err := os.Stat("./assets/" + assetID); os.IsNotExist(err) {
	// 		// path/to/whatever does not exist
	// 		fmt.Println("doesn't exist")
	// 		return
	// 	}
	// 	tempFolder := "./assets/" + assetID + "/temp/"

	// 	r.ParseMultipartForm(10 << 20)
	// 	file, _, err := r.FormFile("file")
	// 	if err != nil {
	// 		print(err)
	// 		return
	// 	}
	// 	defer file.Close()
	// 	fmt.Println(tempFolder)
	// 	// cr := r.Header.Get("Content-Range")
	// 	// fmt.Println(cr)
	// 	resumableIdentifier, _ := r.URL.Query()["resumableIdentifier"]
	// 	resumableChunkNumber, _ := r.URL.Query()["resumableChunkNumber"]
	// 	path := fmt.Sprintf("%s%s", tempFolder, resumableIdentifier[0])
	// 	relativeChunk := fmt.Sprintf("%s%s%s%s", path, "/", "part", resumableChunkNumber[0])

	// }
}
