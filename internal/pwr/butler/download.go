package butler

import (
	"HyLauncher/internal/env"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

func InstallButler() (string, error) {
	toolsDir := filepath.Join(env.GetDefaultAppDir(), "tools", "butler")
	zipPath := filepath.Join(toolsDir, "butler.zip")
	var butlerPath string
	if runtime.GOOS == "windows" {
		butlerPath = filepath.Join(toolsDir, "butler.exe")
	} else {
		butlerPath = filepath.Join(toolsDir, "butler")
	}

	// If binary already exists, skip
	if _, err := os.Stat(butlerPath); err == nil {
		return butlerPath, nil
	}

	// Determine download URL
	var url string
	switch runtime.GOOS {
	case "windows":
		url = "https://broth.itch.zone/butler/windows-amd64/LATEST/archive/default"
	case "darwin":
		url = "https://broth.itch.zone/butler/darwin-amd64/LATEST/archive/default"
	case "linux":
		url = "https://broth.itch.zone/butler/linux-amd64/LATEST/archive/default"
	default:
		return "", fmt.Errorf("unsupported OS")
	}

	fmt.Println("Downloading Butler...")
	if err := downloadFile(zipPath, url); err != nil {
		return "", err
	}

	fmt.Println("Extracting Butler...")
	if err := unzip(zipPath, toolsDir); err != nil {
		return "", err
	}

	// Make executable on unix
	if runtime.GOOS != "windows" {
		if err := os.Chmod(butlerPath, 0755); err != nil {
			return "", err
		}
	}

	// Cleanup zip
	_ = os.Remove(zipPath)
	return butlerPath, nil
}

// Download butler
func downloadFile(dest, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	total := resp.ContentLength
	var downloaded int64
	buf := make([]byte, 32*1024)
	start := time.Now()

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, wErr := out.Write(buf[:n]); wErr != nil {
				return wErr
			}
			downloaded += int64(n)
			if total > 0 {
				percent := float64(downloaded) / float64(total) * 100
				fmt.Printf("\r%.1f%% downloaded (%.2f MB/s)", percent, float64(downloaded)/1024/1024/time.Since(start).Seconds())
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
	}
	fmt.Println()
	return nil
}
