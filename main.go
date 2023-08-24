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

func main() {
	var loc string
	var commandIndex int

	if len(os.Args) > 2 && os.Args[1] == "--loc" {
		loc = os.Args[2]
		commandIndex = 3
	} else {
		loc = "lr"
		commandIndex = 1
	}

	if commandIndex >= len(os.Args) {
		log.Println("Usage: timer [--loc ur|ul|ll|lr] <command> [args ...]")
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

	// Handle terminal resizes.
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	go func() {
		for range ch {
			if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
				log.Fatalf("Error resizing pty: %v", err)
			}
			// Forward the signal to the child process.
			c.Process.Signal(syscall.SIGWINCH)
		}
	}()
	ch <- syscall.SIGWINCH // Trigger initial resize

	// Timer function to display in the specified corner.
	go func() {
		for {
			width, height, _ := terminal.GetSize(0)
			timerString := time.Now().Format("15:04:05")
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
