package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var (
	followBool bool
	linesCount int
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View Opus server logs",
	Long:  `View and tail execution logs of the server.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logPath := getLogFilePath()
		file, err := os.Open(logPath)
		if err != nil {
			if os.IsNotExist(err) {
				cmd.Printf("No logs found at %s. Start the server first.\n", logPath)
				return nil
			}
			return fmt.Errorf("failed to open log file: %w", err)
		}
		defer func() {
			if cerr := file.Close(); cerr != nil {
				cmd.Printf("failed to close log file: %v\n", cerr)
			}
		}()
		// If tail lines count is specified, print them
		if linesCount > 0 {
			lines, err := tailFile(file, linesCount)
			if err != nil {
				return fmt.Errorf("failed to tail log file: %w", err)
			}
			for _, line := range lines {
				cmd.Println(line)
			}
		} else if linesCount < 0 || !followBool {
			// Print whole file if negative tail value (meaning whole file) or not following
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				cmd.Println(scanner.Text())
			}
		}

		if followBool {
			// Seek to the end of the file if we printed last N lines or want only new ones
			if linesCount >= 0 {
				_, err = file.Seek(0, io.SeekEnd)
				if err != nil {
					return fmt.Errorf("failed to seek to end of file: %w", err)
				}
			}

			// Stream new contents using direct un-buffered reads to bypass EOF caching
			var buf [4096]byte
			for {
				select {
				case <-cmd.Context().Done():
					return nil
				default:
				}

				n, err := file.Read(buf[:])
				if n > 0 {
					cmd.Print(string(buf[:n]))
				}
				if err != nil {
					if err == io.EOF {
						// Sleep briefly and wait for new logs to be written
						time.Sleep(100 * time.Millisecond)
						continue
					}
					return fmt.Errorf("error reading logs: %w", err)
				}
			}
		}

		return nil
	},
}

// tailFile returns the last N lines of a file.
func tailFile(file *os.File, n int) ([]string, error) {
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(lines) <= n {
		return lines, nil
	}
	return lines[len(lines)-n:], nil
}

func init() {
	logsCmd.Flags().BoolVarP(&followBool, "follow", "f", false, "Stream logs in real-time")
	logsCmd.Flags().IntVarP(&linesCount, "tail", "n", 100, "Output the last N lines")
	rootCmd.AddCommand(logsCmd)
}
