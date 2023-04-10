package main_test

import (
	"context"
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestMain(t *testing.T) {
	ctx := context.Background()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dockerfilePath := path.Join(wd, "..")
	fmt.Println("path", dockerfilePath)

	t.Run("Forwards the stdout", func(t *testing.T) {
		testLog := "TEST_LOG"
		req := testcontainers.ContainerRequest{
			FromDockerfile: testcontainers.FromDockerfile{
				Context:       dockerfilePath,
				PrintBuildLog: true,
			},
			Cmd: []string{"ubuntu:latest", "/usr/local/bin/docker-explorer", "echo", testLog},
			WaitingFor: wait.ForAll(
				wait.ForLog(testLog),
			),
		}

		myC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		})
		if err != nil {
			t.Fatal(err)
		}

		defer func() {
			err := myC.Terminate(ctx)
			if err != nil {
				t.Fatal(err)
			}
		}()

	})
}
