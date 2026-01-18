package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"HyLauncher/internal/config"
	"HyLauncher/internal/env"
	"HyLauncher/internal/game"
	"HyLauncher/internal/patch"
	"HyLauncher/internal/progress"
	"HyLauncher/pkg/hyerrors"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

var AppVersion string = config.Default().Version

type App struct {
	ctx      context.Context
	cfg      *config.Config
	gameCmd  *exec.Cmd
	progress *progress.Reporter
}

type GameVersions struct {
	Current string `json:"current"`
	Latest  string `json:"latest"`
}

func NewApp() *App {
	cfg, _ := config.Load()
	return &App{
		cfg: cfg,
	}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.progress = progress.New(ctx)

	fmt.Println("Application starting up...")
	fmt.Printf("Current launcher version: %s\n", AppVersion)

	go func() {
		fmt.Println("Creating folders...")
		env.CreateFolders()
	}()

	go func() {
		fmt.Println("Starting background update check...")
		a.checkUpdateSilently()
	}()

	go func() {
		fmt.Println("Starting cleanup")
		env.CleanupLauncher()
	}()
}

func (a *App) handleError(errType hyerrors.ErrorType, userMsg string, err error) error {
	appErr := hyerrors.NewAppError(errType, userMsg, err)
	a.emitError(appErr)
	return appErr
}

func (a *App) emitError(err error) {
	if appErr, ok := err.(*hyerrors.AppError); ok {
		runtime.EventsEmit(a.ctx, "error", appErr)
	} else {
		runtime.EventsEmit(a.ctx, "error", hyerrors.NewAppError(
			hyerrors.ErrorTypeUnknown,
			err.Error(),
			err,
		))
	}
}

func (a *App) GetVersions() GameVersions {
	channel := a.cfg.Settings.Channel
	if channel == "" {
		channel = "release"
	}
	current := patch.GetLocalVersion(channel)
	latest := patch.FindLatestVersion(channel)
	fmt.Printf("GetVersions: Channel=%s, Current=%s, Latest=%d\n", channel, current, latest)
	return GameVersions{
		Current: current,
		Latest:  strconv.Itoa(latest),
	}
}

func (a *App) DownloadAndLaunch(playerName string) error {
	if len(playerName) == 0 {
		return a.handleError(
			hyerrors.ErrorTypeValidation,
			"Please enter a nickname",
			nil,
		)
	}

	if len(playerName) > 16 {
		return a.handleError(
			hyerrors.ErrorTypeValidation,
			"Nickname is too long (max 16 characters)",
			nil,
		)
	}

	channel := a.cfg.Settings.Channel
	if channel == "" {
		channel = "release"
	}
	targetVersion := a.cfg.Settings.GameVersion

	if err := game.EnsureInstalledWithOptions(a.ctx, channel, targetVersion, a.cfg.Settings.OnlineFix, a.progress); err != nil {
		wrappedErr := hyerrors.NewAppError(hyerrors.ErrorTypeGame, "Failed to install or update game", err)
		a.emitError(wrappedErr)
		return wrappedErr
	}

	a.progress.Report(progress.StageLaunch, 100, "Launching game...")

	playerUUID := a.cfg.CurrentProfile

	versionStr := "latest"
	if a.cfg.Settings.GameVersion != 0 {
		versionStr = strconv.Itoa(a.cfg.Settings.GameVersion)
	}

	cmd, err := game.Launch(playerName, channel, playerUUID, versionStr, a.cfg.Settings.OnlineFix)
	if err != nil {
		wrappedErr := hyerrors.NewAppError(hyerrors.ErrorTypeGame, "Failed to launch game", err)
		a.emitError(wrappedErr)
		return wrappedErr
	}

	a.gameCmd = cmd
	runtime.EventsEmit(a.ctx, "game-launched", nil)

	go func() {
		if err := cmd.Wait(); err != nil {
			fmt.Printf("Game process exited with error: %v\n", err)
		}
		a.gameCmd = nil
		runtime.EventsEmit(a.ctx, "game-closed", nil)
	}()

	return nil
}

func (a *App) StopGame() {
	if a.gameCmd != nil && a.gameCmd.Process != nil {
		if err := a.gameCmd.Process.Kill(); err != nil {
			fmt.Printf("Failed to kill game process: %v\n", err)
		}
		a.gameCmd = nil
		runtime.EventsEmit(a.ctx, "game-closed", nil)
	}
}

func (a *App) GetLogs() (string, error) {
	logFile := filepath.Join(env.GetDefaultAppDir(), "logs", "errors.log")
	data, err := os.ReadFile(logFile)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
