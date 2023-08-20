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

	// Create a pipe to transfer data from the PTY to the "less" command.
	reader, writer := io.Pipe()

	// Copy the PTY master's output to the writer end of the pipe.
	go func() {
		defer writer.Close()
		io.Copy(writer, ptmx)
	}()

	// Create the "less -R" command.
	less := exec.Command("less", "-R")

	// Connect the "less" command's Stdin to the reader end of the pipe.
	less.Stdin = reader

	// Connect the "less" command's Stdout and Stderr to the terminal.
	less.Stdout = os.Stdout
	less.Stderr = os.Stderr

	// Run the "less" command.
	if err := less.Run(); err != nil {
		log.Fatalf("Error running 'less -R': %v\n", err)
	}

	// Wait for the command to complete.
	if err := c.Wait(); err != nil {
		log.Fatalf("Error waiting for command: %v\n", err)
	}
}

