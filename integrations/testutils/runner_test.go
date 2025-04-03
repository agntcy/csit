package testutils

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRunner(t *testing.T) {
	local, err := NewRunner(
		RunnerTypeLocal,
	)
	if err != nil {
		t.Errorf("Error creating local runner: %v", err)
		os.Exit(1)
	}

	docker, err := NewRunner(
		RunnerTypeDocker,
		WithDockerCmd("docker"),
		WithDockerArgs([]string{"run", "--rm"}),
		WithDockerImage("busybox:latest"),
	)
	if err != nil {
		t.Errorf("Error creating docker runner: %v", err)
		os.Exit(1)
	}

	output, err := local.Run("echo", "-n", "hello world")
	require.NoError(t, err)

	assert.Equal(t, "hello world", output)

	output, err = docker.Run("echo", "-n", "hello world")
	require.NoError(t, err)

	assert.Equal(t, "hello world", output)
}
