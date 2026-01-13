package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"HyLauncher/internal/env"
	"HyLauncher/internal/java"
	"HyLauncher/internal/pwr"
	"HyLauncher/internal/pwr/butler"

	"github.com/google/uuid"
)

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	env.CreateFolders()
}

func (a *App) DownloadAndLaunch(playerName string) error {
	if err := java.DownloadJRE(); err != nil {
		return err
	}
	javaBin := java.GetJavaExec()

	if _, err := butler.InstallButler(); err != nil {
		return err
	}

	version := "release"
	pwrFile := "1.pwr"

	gameLatest := filepath.Join(env.GetDefaultAppDir(), "release", "package", "game", "latest")
	gameClient := "HytaleClient"
	if os.PathSeparator == '\\' {
		gameClient += ".exe"
	}
	clientPath := filepath.Join(gameLatest, "Client", gameClient)

	if _, err := os.Stat(clientPath); os.IsNotExist(err) {
		// Only install game if client does not exist
		if err := pwr.InstallGame(a.ctx, version, pwrFile); err != nil {
			return err
		}
	} else {
		fmt.Println("Game already installed, skipping download.")
	}

	uuid := uuid.NewString()
	cmd := exec.Command(clientPath,
		"--app-dir", gameLatest,
		"--java-exec", javaBin,
		"--auth-mode", "offline",
		"--uuid", uuid,
		"--name", playerName,
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	fmt.Println("Launching game...")
	if err := cmd.Start(); err != nil {
		return err
	}

	return nil
}
