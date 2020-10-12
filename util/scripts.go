package util

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

// WriteResponse writes some response for a given http request.
func WriteResponse(data map[string]interface{}, w http.ResponseWriter) {
	jsonData, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		fmt.Fprintln(w, data)
		log.Warn(err)
		return
	}
	fmt.Fprintln(w, string(jsonData))
	return
}

// DirSize returns the size of a directory.
func DirSize(path string) (uint64, error) {
	var size uint64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += uint64(info.Size())
		}
		return err
	})
	return size, err
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
