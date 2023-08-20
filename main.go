package main

import (
	"io"
	"log"
	"os"
	"os/exec"
	"syscall"

	"github.com/creack/pty"
)

func main() {
	if len(os.Args) < 2 {
		log.Println("Usage: cless <command> [args ...]")
		return
	}

	// Create the command.
	c := exec.Command(os.Args[1], os.Args[2:]...)

	// Create a PTY.
	ptmx, tty, err := pty.Open()
	if err != nil {
		log.Fatalf("Error opening pty: %v\n", err)
	}
	defer ptmx.Close()
	defer tty.Close()

	// Set the PTY as Stdin, Stdout, and Stderr.
	c.Stdin = tty
	c.Stdout = tty
	c.Stderr = tty

	// Add the PTY to ExtraFiles and set Ctty to 3.
	c.ExtraFiles = []*os.File{tty}
	c.SysProcAttr = &syscall.SysProcAttr{
		Ctty: 3,
	}

	// Start the command with the PTY.
	if err := c.Start(); err != nil {
		log.Fatalf("Error starting command: %v\n", err)
	}

	// Copy the PTY master's output to the standard output.
	if _, err := io.Copy(os.Stdout, ptmx); err != nil {
		log.Fatalf("Error copying output to stdout: %v\n", err)
	}

	// Wait for the command to complete.
	if err := c.Wait(); err != nil {
		log.Fatalf("Error waiting for command: %v\n", err)
	}
}

