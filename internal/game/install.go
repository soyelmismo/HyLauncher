package game

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"

	"HyLauncher/internal/env"
	"HyLauncher/internal/java"
	"HyLauncher/internal/patch"
	"HyLauncher/internal/progress"
)

var (
	installMutex sync.Mutex
	isInstalling bool
)

// EnsureInstalled - Original version from upstream (for compatibility)
func EnsureInstalled(ctx context.Context, reporter *progress.Reporter) error {
	return EnsureInstalledWithOptions(ctx, "release", 0, true, reporter)
}

// EnsureInstalledWithOptions - New function with additional options
func EnsureInstalledWithOptions(ctx context.Context, channel string, targetVersion int, enableOnlineFix bool, reporter *progress.Reporter) error {
	// Prevent multiple simultaneous installations
	installMutex.Lock()
	if isInstalling {
		installMutex.Unlock()
		return fmt.Errorf("installation already in progress")
	}
	isInstalling = true
	installMutex.Unlock()

	defer func() {
		installMutex.Lock()
		isInstalling = false
		installMutex.Unlock()
	}()

	// Download JRE
	if err := java.DownloadJRE(ctx, reporter); err != nil {
		return fmt.Errorf("failed to download Java Runtime: %w", err)
	}

	// Install Butler
	if _, err := patch.InstallButler(ctx, reporter); err != nil {
		return fmt.Errorf("failed to install Butler tool: %w", err)
	}

	// Find latest version with details
	if reporter != nil {
		reporter.Report(progress.StageVerify, 0, "Checking for game updates")
	}

	// Run version check (will use cache if available)
	result := patch.FindLatestVersionWithDetails(channel)

	if result.Error != nil {
		return fmt.Errorf(
			"cannot find game versions on server\n\n"+
				"Platform: %s %s\n"+
				"Error: %v\n\n"+
				"Troubleshooting:\n"+
				"• Ensure your system is supported (Windows/Linux/macOS)\n"+
				"• Check if game is available for your architecture\n"+
				"• Verify firewall allows connections to game-patches.hytale.com\n"+
				"• Try disabling VPN temporarily\n\n"+
				"Checked URLs: %d\n"+
				"Sample: %s",
			runtime.GOOS,
			runtime.GOARCH,
			result.Error,
			len(result.CheckedURLs),
			getFirstURL(result.CheckedURLs),
		)
	}

	if result.LatestVersion == 0 {
		return fmt.Errorf(
			"no game versions found for your platform\n\n"+
				"Platform: %s/%s\n"+
				"Version type: %s\n\n"+
				"This usually means:\n"+
				"• The game is not yet available for your platform\n"+
				"• Your system architecture is not supported\n"+
				"• Server configuration has changed\n\n"+
				"Please check the official Hytale website for platform availability.",
			runtime.GOOS,
			runtime.GOARCH,
			channel,
		)
	}

	// If targetVersion is 0, use the latest version, otherwise use the specified one
	installVersion := result.LatestVersion
	installDirName := "latest"

	if targetVersion > 0 {
		// Verify if the requested version exists
		if err := patch.VerifyVersionExists(channel, targetVersion); err != nil {
			return fmt.Errorf("requested version %d is not available: %w", targetVersion, err)
		}
		installVersion = targetVersion
		installDirName = strconv.Itoa(targetVersion)
	}

	if reporter != nil {
		reporter.Report(progress.StageVerify, 100, "Checking complete")
		reporter.Report(progress.StageComplete, 0, fmt.Sprintf("Found version %d", installVersion))
	}

	fmt.Printf("Target version: %d\n", installVersion)
	if result.SuccessURL != "" {
		fmt.Printf("Success URL: %s\n", result.SuccessURL)
	}

	return InstallGame(ctx, channel, installVersion, installDirName, enableOnlineFix, reporter)
}

func InstallGame(ctx context.Context, versionType string, remoteVer int, installDirName string, enableOnlineFix bool, reporter *progress.Reporter) error {
	localStr := patch.GetLocalVersion(versionType)
	local, _ := strconv.Atoi(localStr)

	gameInstallDir := filepath.Join(env.GetDefaultAppDir(), versionType, "package", "game", installDirName)

	// Adjust game client executable to operating system
	gameClient := "HytaleClient"
	if runtime.GOOS == "windows" {
		gameClient += ".exe"
	}
	clientPath := filepath.Join(gameInstallDir, "Client", gameClient)
	_, clientErr := os.Stat(clientPath)

	// Check if our game version is same as latest
	if local == remoteVer && clientErr == nil {
		if reporter != nil {
			reporter.Report(progress.StageComplete, 100, "Game is up to date")
		}
		return nil
	}

	prevVer := local
	if clientErr != nil {
		prevVer = 0
		if reporter != nil {
			reporter.Report(progress.StagePWR, 0, fmt.Sprintf("Installing game version %d...", remoteVer))
		}
	} else {
		if reporter != nil {
			reporter.Report(progress.StagePWR, 0, fmt.Sprintf("Updating from version %d to %d...", local, remoteVer))
		}
	}

	// Download the patch file
	pwrPath, err := patch.DownloadPWR(ctx, versionType, prevVer, remoteVer, reporter)
	if err != nil {
		return fmt.Errorf("failed to download game patch: %w", err)
	}

	// Verify the patch file exists and is readable
	info, err := os.Stat(pwrPath)
	if err != nil {
		return fmt.Errorf("patch file not accessible: %w", err)
	}
	fmt.Printf("Patch file size: %d bytes\n", info.Size())

	// Apply the patch
	if reporter != nil {
		reporter.Report(progress.StagePatch, 0, "Applying game patch...")
	}

	if err := patch.ApplyPWRWithOptions(ctx, versionType, pwrPath, installDirName, reporter); err != nil {
		return fmt.Errorf("failed to apply game patch: %w", err)
	}

	// Verify installation
	if _, err := os.Stat(clientPath); err != nil {
		return fmt.Errorf("game installation incomplete: client executable not found at %s", clientPath)
	}

	// Save the new version
	if err := patch.SaveLocalVersion(versionType, remoteVer); err != nil {
		fmt.Printf("Warning: failed to save version info: %v\n", err)
	}

	// Apply online fix only on windows if enabled
	if runtime.GOOS == "windows" && enableOnlineFix {
		if reporter != nil {
			reporter.Report(progress.StageOnlineFix, 0, "Applying online fix...")
		}

		if err := ApplyOnlineFixWindows(ctx, gameInstallDir, reporter); err != nil {
			return fmt.Errorf("failed to apply online fix: %w", err)
		}

		if reporter != nil {
			reporter.Report(progress.StageOnlineFix, 100, "Online fix applied")
		}
	}

	if reporter != nil {
		reporter.Report(progress.StageComplete, 100, "Game installed successfully")
	}

	return nil
}

func getFirstURL(urls []string) string {
	if len(urls) == 0 {
		return "none"
	}
	return urls[0]
}
