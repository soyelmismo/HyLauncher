package pwr

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"HyLauncher/internal/env"
)

func DownloadPWR(version, fileName string) (string, error) {
	cacheDir := filepath.Join(env.GetDefaultAppDir(), "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", err
	}

	osName := runtime.GOOS
	arch := runtime.GOARCH

	url := fmt.Sprintf("https://game-patches.hytale.com/patches/%s/%s/%s/0/%s",
		osName, arch, version, fileName)

	dest := filepath.Join(cacheDir, fileName)

	// Skip if already downloaded
	if _, err := os.Stat(dest); err == nil {
		fmt.Println("PWR file already exists:", dest)
		return dest, nil
	}

	fmt.Println("Downloading PWR file:", url)
	if err := downloadFile(dest, url); err != nil {
		return "", err
	}

	fmt.Println("PWR downloaded to:", dest)
	return dest, nil
}

// downloadFile with simple progress
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
