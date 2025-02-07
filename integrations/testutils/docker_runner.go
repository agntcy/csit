// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package testutils

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
)

type DockerRunner struct {
	runCmd   string
	baseArgs []string
}

func NewDockerRunner(dockerImage, mountString string, envVars map[string]string) *DockerRunner {
	baseArgs := []string{
		"run",
		"--rm",
	}

	if mountString != "" {
		baseArgs = append(baseArgs,
			"-v",
			mountString,
		)
	}

	if runtime.GOOS == "linux" {
		baseArgs = append(baseArgs,
			"--net=host",
		)
	}

	for k, v := range envVars {
		baseArgs = append(baseArgs, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	baseArgs = append(baseArgs, dockerImage)

	return &DockerRunner{runCmd: "docker",
		baseArgs: baseArgs,
	}
}

// example usage: runner.Run("push", "--from-file", "file.json")
func (r *DockerRunner) Run(args ...string) (bytes.Buffer, error) {
	var outputBuffer bytes.Buffer

	cmd := exec.Command(r.runCmd, append(r.baseArgs, args...)...)
	cmd.Stdout = &outputBuffer

	return outputBuffer, cmd.Run()
}
