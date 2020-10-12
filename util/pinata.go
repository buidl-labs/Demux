package util

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const endpoint = "https://api.pinata.cloud/pinning/pinFileToIPFS"

// PinFolder pins a folder
func PinFolder(folder string, name string) (string, error) {
	var pinataCID string

	// Build the request
	fu := NewFormUploader()

	// Scan for files and add them
	files := make([]string, 0)
	err := filepath.Walk(folder,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			// Add files only
			if info.Mode().IsRegular() {
				// Remove the folder from the beginning of the file's name
				path = strings.TrimPrefix(path, folder+"/")
				// Trim again an optional path separator
				path = strings.TrimPrefix(path, string(os.PathSeparator))
				files = append(files, path)
			}
			return nil
		})
	if err != nil {
		return pinataCID, err
	}
	fu.AddFiles("file", folder, files...)

	// Add the name
	keyValues := make(map[string]string)
	keyValues["PinnedBy"] = "Demux"
	pinataMetadata := struct {
		Name      string            `json:"name"`
		KeyValues map[string]string `json:"keyvalues"`
	}{
		Name:      name,
		KeyValues: keyValues,
	}
	pinataMetadataJSON, err := json.Marshal(pinataMetadata)
	if err != nil {
		return pinataCID, err
	}
	fu.AddField("pinataMetadata", string(pinataMetadataJSON))

	pinataOptions := struct {
		CidVersion int `json:"cidVersion"`
	}{
		CidVersion: 1,
	}
	pinataOptionsJSON, err := json.Marshal(pinataOptions)
	if err != nil {
		return pinataCID, err
	}
	fu.AddField("pinataOptions", string(pinataOptionsJSON))

	// Send the request
	client := &http.Client{
		// Do not set a timeout, as the files might be large
		Timeout: 0,
	}
	headers := make(map[string]string, 2)
	headers["pinata_api_key"] = os.Getenv("PINATA_API_KEY")
	headers["pinata_secret_api_key"] = os.Getenv("PINATA_SECRET_KEY")
	resp, err := fu.Post(client, endpoint, headers)
	if err != nil {
		return pinataCID, err
	}

	// Get the response
	res, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return pinataCID, err
	}

	// If status code isn't 2xx, we have an error
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return pinataCID, fmt.Errorf("Invalid response status code: %d", resp.StatusCode)
	}

	// Output the response (should be JSON)
	var j = []byte(string(res))

	// a map container to decode the JSON structure into
	c := make(map[string]json.RawMessage)
	// unmarschal JSON
	e := json.Unmarshal(j, &c)
	// panic on error
	if e != nil {
		return pinataCID, e
	}
	// a string slice to hold the keys
	k := make([]string, len(c))
	// iteration counter
	i := 0
	// copy c's keys into k
	for s, v := range c {
		k[i] = s
		if s == "IpfsHash" {
			pinataCID = strings.TrimPrefix(strings.TrimSuffix(string(v), "\""), "\"")
		}
		i++
	}

	return pinataCID, nil
}
