package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/creack/pty"
	"golang.org/x/crypto/ssh/terminal"
)

const APP_NAME = "cless"

func main() {
	var loc, style string
	var commandIndex int

	if len(os.Args) > 2 {
		if os.Args[1] == "--loc" {
			loc = os.Args[2]
			commandIndex += 2
		}
		if os.Args[commandIndex+1] == "--style" {
			style = os.Args[commandIndex+2]
			commandIndex += 2
		}
	}

	if loc == "" {
		loc = "lr"
	}

	if style == "" {
		style = "stopwatch"
	}

	commandIndex += 1

	if commandIndex >= len(os.Args) || os.Args[commandIndex] == "--help" || os.Args[commandIndex] == "-h" {
		fmt.Fprintf(os.Stderr, "Usage: %s [--loc ur|ul|ll|lr] [--style clock|stopwatch] <command> [args ...]\n", APP_NAME)
		return
	}

	// Create the command.
	c := exec.Command(os.Args[commandIndex], os.Args[commandIndex+1:]...)

	// Create a PTY.
	ptmx, tty, err := pty.Open()
	if err != nil {
		log.Fatalf("Error opening pty: %v\n", err)
	}
	defer ptmx.Close()

	// Add the PTY to ExtraFiles and set Ctty to 3.
	c.ExtraFiles = []*os.File{tty}
	c.SysProcAttr = &syscall.SysProcAttr{
		Ctty: 3,
	}

	// Redirect the command's Stdin, Stdout, and Stderr to the PTY slave.
	c.Stdin = tty
	c.Stdout = tty
	c.Stderr = tty

	// Start the command with the PTY.
	if err := c.Start(); err != nil {
		log.Fatalf("Error starting command: %v\n", err)
	}

	// Close the PTY slave to allow the PTY master to detect EOF.
	tty.Close()

	// Set terminal to raw mode
	oldState, err := terminal.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		log.Fatalf("Error setting terminal to raw mode: %v\n", err)
	}
	defer terminal.Restore(int(os.Stdin.Fd()), oldState)

	// Handle terminal resizes and interrupt signal.
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH, syscall.SIGINT)
	go func() {
		for sig := range ch {
			if sig == syscall.SIGWINCH {
				if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
					log.Fatalf("Error resizing pty: %v", err)
				}
				// Forward the signal to the child process.
				c.Process.Signal(syscall.SIGWINCH)
			} else if sig == syscall.SIGINT {
				c.Process.Signal(syscall.SIGINT)
			}
		}
	}()
	ch <- syscall.SIGWINCH // Trigger initial resize

	startTime := time.Now()

	// Timer function to display in the specified corner.
	go func() {
		for {
			width, height, _ := terminal.GetSize(0)
			var timerString string
			if style == "clock" {
				timerString = time.Now().Format("15:04:05")
			} else { // "stopwatch"
				elapsedTime := time.Since(startTime)
				timerString = fmt.Sprintf("%02d:%02d:%02d", int(elapsedTime.Hours()), int(elapsedTime.Minutes())%60, int(elapsedTime.Seconds())%60)
			}
			row, col := getTimerPosition(loc, width, height, len(timerString))
			fmt.Printf("\033[s\033[%d;%dH\033[7m%v\033[m\033[u", row, col, timerString)
			time.Sleep(1 * time.Second)
		}
	}()

	// Copy the PTY master's output to stdout.
	go func() {
		io.Copy(os.Stdout, ptmx)
	}()

	// Copy stdin to the PTY master's input.
	go func() {
		io.Copy(ptmx, os.Stdin)
	}()

	// Wait for the command to complete.
	if err := c.Wait(); err != nil {
		log.Fatalf("Error waiting for command: %v\n", err)
	}
}

func getTimerPosition(loc string, width, height, timerLength int) (int, int) {
	switch loc {
	case "ur":
		return 1, width - timerLength - 1
	case "ul":
		return 1, 1
	case "ll":
		return height, 1
	default: // "lr"
		return height, width - timerLength - 1
	}
}
