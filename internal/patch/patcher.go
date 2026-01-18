package patch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"HyLauncher/internal/env"
	"HyLauncher/internal/platform"
	"HyLauncher/internal/progress"
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

	cmd := platform.Command(
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
