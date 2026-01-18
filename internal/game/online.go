package game

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"HyLauncher/internal/progress"
	"HyLauncher/pkg/download"
	"HyLauncher/pkg/extract"
	"HyLauncher/pkg/fileutil"
)

const (
	onlineFixAssetName = "online-fix.zip"
)

func ApplyOnlineFixWindows(ctx context.Context, gameDir string, reporter *progress.Reporter) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("online fix is only for Windows")
	}

	cacheDir := filepath.Join(gameDir, ".cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	zipPath := filepath.Join(cacheDir, onlineFixAssetName)

	reporter.Report(progress.StageOnlineFix, 0, "Downloading online-fix...")

	scaler := progress.NewScaler(reporter, progress.StageOnlineFix, 0, 70)

	if err := download.DownloadLatestReleaseAsset(ctx, onlineFixAssetName, zipPath, func(stage string, prog float64, message string, currentFile string, speed string, downloaded, total int64) {
		scaler.ReportDownload(progress.StageOnlineFix, prog, message, onlineFixAssetName, speed, downloaded, total)
	}); err != nil {
		_ = os.Remove(zipPath)
		return fmt.Errorf("failed to download online-fix: %w", err)
	}

	reporter.Report(progress.StageOnlineFix, 70, "Extracting online-fix...")

	if err := extractAndApplyFix(zipPath, gameDir, cacheDir); err != nil {
		return err
	}

	_ = os.Remove(zipPath)

	reporter.Report(progress.StageOnlineFix, 100, "Online-fix applied successfully")

	return nil
}

func extractAndApplyFix(zipPath, gameDir, cacheDir string) error {
	tempDir := filepath.Join(cacheDir, "temp_extract")

	if err := os.RemoveAll(tempDir); err != nil {
		return fmt.Errorf("failed to clean temp directory: %w", err)
	}
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	if err := extract.ExtractZip(zipPath, tempDir); err != nil {
		return fmt.Errorf("failed to extract ZIP: %w", err)
	}

	clientSrc := filepath.Join(tempDir, "Client", "HytaleClient.exe")
	clientDst := filepath.Join(gameDir, "Client", "HytaleClient.exe")

	if err := os.MkdirAll(filepath.Dir(clientDst), 0755); err != nil {
		return fmt.Errorf("failed to create client directory: %w", err)
	}
	if err := fileutil.CopyFile(clientSrc, clientDst); err != nil {
		return fmt.Errorf("failed to copy client executable: %w", err)
	}

	serverDir := filepath.Join(gameDir, "Server")
	if err := os.MkdirAll(serverDir, 0755); err != nil {
		return fmt.Errorf("failed to create server directory: %w", err)
	}

	serverJarSrc := filepath.Join(tempDir, "Server", "HytaleServer.jar")
	serverJarDst := filepath.Join(serverDir, "HytaleServer.jar")
	if err := fileutil.CopyFile(serverJarSrc, serverJarDst); err != nil {
		return fmt.Errorf("failed to copy HytaleServer.jar: %w", err)
	}

	startBatSrc := filepath.Join(tempDir, "Server", "start-server.bat")
	startBatDst := filepath.Join(serverDir, "start-server.bat")
	if err := fileutil.CopyFile(startBatSrc, startBatDst); err != nil {
		return fmt.Errorf("failed to copy start-server.bat: %w", err)
	}

	return nil
}

func EnsureServerAndClientFix(ctx context.Context, gameDir string, reporter *progress.Reporter) error {
	if runtime.GOOS != "windows" {
		return nil
	}

	serverBat := filepath.Join(gameDir, "Server", "start-server.bat")
	if _, err := os.Stat(serverBat); err == nil {
		return nil
	}

	reporter.Report(progress.StageOnlineFix, 0, "Applying online fix for server...")

	if err := ApplyOnlineFixWindows(ctx, gameDir, reporter); err != nil {
		return fmt.Errorf("failed to apply online fix: %w", err)
	}

	return nil
}
