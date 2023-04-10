package main

import (
	"io"
	"os"
	"os/exec"
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

	err := command.Run()
	if err != nil {
		return 0, err
	}

	return command.ProcessState.ExitCode(), nil
}
