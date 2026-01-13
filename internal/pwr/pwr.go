package pwr

import (
	"HyLauncher/internal/env"
	"HyLauncher/internal/pwr/butler"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func InstallGame(ctx context.Context, version, fileName string) error {
	gameLatest := filepath.Join(env.GetDefaultAppDir(), "release", "package", "game", "latest")

	// Check if Hytale client already exists
	gameClient := "HytaleClient"
	if os.PathSeparator == '\\' {
		gameClient += ".exe"
	}
	clientPath := filepath.Join(gameLatest, "Client", gameClient)
	if _, err := os.Stat(clientPath); err == nil {
		fmt.Println("Game already installed, skipping download.")
		return nil
	}

	// Download .pwr if needed
	pwrPath, err := DownloadPWR(version, fileName)
	if err != nil {
		return err
	}

	// Apply .pwr using Butler
	return ApplyPWR(ctx, pwrPath)
}

func ApplyPWR(ctx context.Context, pwrFile string) error {
	gameLatest := filepath.Join(env.GetDefaultAppDir(), "release", "package", "game", "latest")

	butlerPath, err := butler.InstallButler()
	if err != nil {
		return err
	}

	stagingDir := filepath.Join(gameLatest, "staging-temp")
	if err := os.MkdirAll(stagingDir, 0755); err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, butlerPath,
		"apply",
		"--staging-dir", stagingDir,
		pwrFile,
		gameLatest,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Println("Applying .pwr file...")
	if err := cmd.Run(); err != nil {
		return err
	}

	_ = os.RemoveAll(stagingDir)
	fmt.Println("Game extracted successfully")
	return nil
}
