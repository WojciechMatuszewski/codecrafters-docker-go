package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func main() {
	command := os.Args[3]
	args := os.Args[4:len(os.Args)]

	cmd := exec.Command(command, args...)
	exitCode, err := run(cmd, os.Stdout, os.Stderr)
	if err != nil {
		panic(err)
	}

	os.Exit(exitCode)
}

func run(command *exec.Cmd, stdout io.Writer, stderr io.Writer) (int, error) {
	command.Stdout = stdout
	command.Stderr = stderr
	command.Stdin = os.Stdin

	// TODO: handle this better
	dirname := strings.Join([]string{"mydocker", fmt.Sprintf("%v", time.Now().UnixNano())}, "")
	tmpDir := path.Join(os.TempDir(), dirname)
	err := os.Mkdir(tmpDir, 0744)
	if err != nil {
		return 0, fmt.Errorf("failed to create temporary directory: %w", err)
	}

	err = exec.Command("mkdir", "-p", filepath.Join(tmpDir, filepath.Dir(command.Args[0]))).Run()
	if err != nil {
		return 0, fmt.Errorf("failed to create directory %s: %w", filepath.Join(tmpDir, filepath.Dir(command.Args[0])), err)
	}

	err = exec.Command("cp", command.Args[0], path.Join(tmpDir, command.Args[0])).Run()
	if err != nil {
		return 0, fmt.Errorf("failed to copy binary %s to chroot directory %s: %w", command.Args[0], tmpDir, err)
	}

	command.SysProcAttr = &syscall.SysProcAttr{
		Chroot: tmpDir,
		// Interestingly this import does not exist in my editor. Maybe I need to use the editor as sudo?
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID,
	}
	err = command.Run()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode(), nil
	}

	return command.ProcessState.ExitCode(), nil
}
