package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/kilip/opus/server/internal/config"
	"github.com/kilip/opus/server/internal/container"
	"github.com/kilip/opus/server/internal/shared/logger"
	"github.com/spf13/cobra"
)

var daemonBool bool

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the Opus server",
	Long:  `Starts the background queue workers and the Fiber HTTP web server.`,
	RunE:  runStart,
}

func runStart(cmd *cobra.Command, args []string) error {
	pid, err := readPID()
	if err == nil && isProcessRunning(pid) {
		return fmt.Errorf("opus server is already running with PID %d", pid)
	}
	if daemonBool {
		// Run in background (daemon mode)
		var daemonArgs []string
		for _, arg := range os.Args[1:] {
			// Exclude the daemon flags to prevent recursive startup loops
			if arg != "-d" && arg != "--daemon" && arg != "-d=true" && arg != "--daemon=true" {
				daemonArgs = append(daemonArgs, arg)
			}
		}

		cmdPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to find executable path: %w", err)
		}

		logPath := getLogFilePath()
		logDir := filepath.Dir(logPath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return fmt.Errorf("failed to create log directory: %w", err)
		}

		logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("failed to open log file for daemon: %w", err)
		}
		defer func() {
			if cerr := logFile.Close(); cerr != nil {
				fmt.Fprintf(os.Stderr, "failed to close log file: %v\n", cerr)
			}
		}()

		processCmd := exec.Command(cmdPath, daemonArgs...)
		processCmd.Stdout = logFile
		processCmd.Stderr = logFile
		processCmd.SysProcAttr = &syscall.SysProcAttr{
			Setsid: true, // Decouple child process from parent terminal session
		}

		if err := processCmd.Start(); err != nil {
			return fmt.Errorf("failed to start background process: %w", err)
		}
		// Sleep briefly to check if background process exited immediately (e.g. port collision)
		time.Sleep(250 * time.Millisecond)
		if !isProcessRunning(processCmd.Process.Pid) {
			return fmt.Errorf("background process started but exited immediately. Please check logs at: %s", logPath)
		}

		cmd.Printf("Opus server started in background with PID %d.\n", processCmd.Process.Pid)
		return nil
	}

	// Foreground normal startup
	if err := writePID(); err != nil {
		return fmt.Errorf("main.runStart: failed to write PID file: %w", err)
	}
	defer func() {
		_ = removePID()
	}()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("main.runStart: failed to load config: %w", err)
	}

	container.Bootstrap(*cfg)

	ctx := context.Background()
	log := container.GetLogger()

	log.Info("start opus queue worker")
	if err := container.GetQueue().Start(ctx); err != nil {
		return fmt.Errorf("main.runStart: failed to start queue: %w", err)
	}

	errCh := make(chan error, 2)

	// API Server
	go func() {
		address := cfg.Server.Address
		if address == "" {
			address = ":8080"
		}
		log.Info("start opus api server", logger.String("address", address))
		if err := container.GetFiber().Listen(address); err != nil {
			errCh <- fmt.Errorf("api server: %w", err)
		}
	}()

	// Dash Server
	go func() {
		address := cfg.Dash.Address
		if address == "" {
			address = ":8081"
		}
		log.Info("start opus dash server", logger.String("address", address))
		if err := container.GetDash().Listen(address); err != nil {
			errCh <- fmt.Errorf("dash server: %w", err)
		}
	}()

	return <-errCh
	}
func init() {
	startCmd.Flags().BoolVarP(&daemonBool, "daemon", "d", false, "Run Opus server in background")
	rootCmd.AddCommand(startCmd)
	defaultRunFunc = runStart
}
