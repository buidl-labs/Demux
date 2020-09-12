package util

import (
	"encoding/json"
	"fmt"
	golog "log"
	"net/http"
	"os"
	"path/filepath"
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

// Upload is a helper function
func Upload(w http.ResponseWriter, r *http.Request) {}

// RemoveContents removes unnecessary files from an asset directory.
func RemoveContents(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		if name != "random1080p.mp4" && name != "random720p.mp4" && name != "random360p.mp4" {
			err = os.RemoveAll(filepath.Join(dir, name))
			if err != nil {
				return err
			}
		}
	}
	return nil
}
