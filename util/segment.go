package util

import (
	guuid "github.com/google/uuid"
	"os/exec"
)

// CreateSegments creates segments of a transcoded video using ffmpeg
func CreateSegments(filename string, resolution string, id guuid.UUID) (bool, error) {
	success := false

	cmd := exec.Command(
		"ffmpeg", "-i", "./assets/"+id.String()+"/"+filename,
		"-profile:v", "baseline", "-level", "3.0", "-start_number", "0",
		"-hls_time", "10", "-hls_list_size", "0", "-f", "hls",
		"./assets/"+id.String()+"/"+resolution+"/myvid.m3u8")

	stdout, err := cmd.Output()
	if err == nil {
		success = true
	}
	_ = stdout
	if success == true {
		// cleanup
		exec.Command("rm", "-rf", "./assets/"+id.String()+"/"+filename).Output()
	}

	return success, err
}
