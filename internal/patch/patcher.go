package patch

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"HyLauncher/internal/env"
	"HyLauncher/internal/platform"
	"HyLauncher/internal/progress"
	"HyLauncher/pkg/download"
)

// ApplyPWR - Original version from upstream
func ApplyPWR(ctx context.Context, pwrFile string, reporter *progress.Reporter) error {
	return ApplyPWRWithOptions(ctx, "release", pwrFile, "latest", reporter)
}

// ApplyPWRWithOptions - New function with additional options
func ApplyPWRWithOptions(ctx context.Context, channel string, pwrFile string, installDirName string, reporter *progress.Reporter) error {
	gameInstallDir := filepath.Join(env.GetDefaultAppDir(), channel, "package", "game", installDirName)
	stagingDir := filepath.Join(env.GetDefaultAppDir(), channel, "package", "game", "staging-temp")

	// Create parent directory
	_ = os.MkdirAll(filepath.Dir(gameInstallDir), 0755)

	// Create target directory explicitly (butler requires it to exist)
	if err := os.MkdirAll(gameInstallDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Clean up any previous staging directory
	_ = os.RemoveAll(stagingDir)
	_ = os.MkdirAll(stagingDir, 0755)

	butlerPath := filepath.Join(env.GetDefaultAppDir(), "tools", "butler", "butler")
	if runtime.GOOS == "windows" {
		butlerPath += ".exe"
	}

	// Ensure butler is executable on non-Windows
	if runtime.GOOS != "windows" {
		_ = os.Chmod(butlerPath, 0755)
	}

	cmd := exec.Command(
		butlerPath,
		"apply",
		"--staging-dir", stagingDir,
		pwrFile,
		gameInstallDir,
	)

	platform.HideConsoleWindow(cmd)

	// Open log file for this operation
	logDir := filepath.Join(env.GetDefaultAppDir(), "logs")
	_ = os.MkdirAll(logDir, 0755)
	logFile, err := os.Create(filepath.Join(logDir, "butler_apply.log"))
	if err == nil {
		defer logFile.Close()
		cmd.Stdout = logFile
		cmd.Stderr = logFile
		fmt.Fprintf(logFile, "Starting butler apply for %s to %s\n", pwrFile, gameInstallDir)
	} else {
		// Fallback if log file fails
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	reporter.Report(progress.StagePatch, 60, "Applying game patch...")

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("butler apply failed (check logs/butler_apply.log): %w", err)
	}

	_ = os.RemoveAll(stagingDir)

	reporter.Report(progress.StagePatch, 100, "Game patched!")

	return nil
}

func DownloadPWR(ctx context.Context, versionType string, prevVer int, targetVer int, reporter *progress.Reporter) (string, error) {
	cacheDir := filepath.Join(env.GetDefaultAppDir(), "cache")
	_ = os.MkdirAll(cacheDir, 0755)
	osName := runtime.GOOS
	arch := runtime.GOARCH
	fileName := fmt.Sprintf("%d.pwr", targetVer)
	dest := filepath.Join(cacheDir, fileName)
	tempDest := dest + ".tmp"

	_ = os.Remove(tempDest)

	if _, err := os.Stat(dest); err == nil {
		reporter.Report(progress.StagePWR, 100, "PWR file cached")
		return dest, nil
	}

	// Create a scaler for the download portion (0-100%)
	scaler := progress.NewScaler(reporter, progress.StagePWR, 0, 100)

	url := fmt.Sprintf("https://game-patches.hytale.com/patches/%s/%s/%s/%d/%s", osName, arch, versionType, prevVer, fileName)
	if err := download.DownloadWithReporter(dest, url, fileName, reporter, progress.StagePWR, scaler); err != nil {
		_ = os.Remove(tempDest)
		return "", err
	}

	reporter.Report(progress.StagePWR, 100, "PWR file downloaded")

	return dest, nil
}
