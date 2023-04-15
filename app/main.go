package main

import (
	"context"
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
	dirname := strings.Join([]string{"mydocker", fmt.Sprintf("%v", time.Now().UnixNano())}, "")
	chroot := path.Join(os.TempDir(), dirname)
	err := os.Mkdir(chroot, 0744)
	if err != nil {
		panic(err)
	}

	registry := NewRegistry("alpine:latest", chroot)
	err = registry.Pull(context.Background())
	if err != nil {
		panic(err)
	}

	command := os.Args[3]
	args := os.Args[4:len(os.Args)]

	cmd := exec.Command(command, args...)
	exitCode, err := run(cmd, os.Stdout, os.Stderr, chroot)
	if err != nil {
		panic(err)
	}

	os.Exit(exitCode)
}

func run(command *exec.Cmd, stdout io.Writer, stderr io.Writer, chroot string) (int, error) {
	command.Stdout = stdout
	command.Stderr = stderr
	command.Stdin = os.Stdin

	err := exec.Command("mkdir", "-p", filepath.Join(chroot, filepath.Dir(command.Args[0]))).Run()
	if err != nil {
		return 0, fmt.Errorf("failed to create directory %s: %w", filepath.Join(chroot, filepath.Dir(command.Args[0])), err)
	}

	err = exec.Command("cp", command.Args[0], path.Join(chroot, command.Args[0])).Run()
	if err != nil {
		return 0, fmt.Errorf("failed to copy binary %s to chroot directory %s: %w", command.Args[0], chroot, err)
	}

	command.SysProcAttr = &syscall.SysProcAttr{
		Chroot:     chroot,
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID,
	}
	err = command.Run()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode(), nil
	}

	return command.ProcessState.ExitCode(), nil
}
