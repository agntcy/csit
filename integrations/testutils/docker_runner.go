// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package testutils

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

type dockerRunner struct {
	dockerCmd   string
	dockerArgs  []string
	dockerImage string
	envVars     map[string]string
}

func (r *dockerRunner) Run(command string, args ...string) (string, error) {
	combinedArgs := make([]string, 0, len(r.dockerArgs)+1+len(args))
	combinedArgs = append(combinedArgs, r.dockerArgs...)

	combinedArgs = append(combinedArgs, "--entrypoint", command)

	for key, value := range r.envVars {
		combinedArgs = append(combinedArgs, "-e", key+"="+value)
	}

	if runtime.GOOS == "linux" {
		combinedArgs = append(combinedArgs,
			"--net=host",
		)
	}

	combinedArgs = append(combinedArgs, r.dockerImage)
	combinedArgs = append(combinedArgs, args...)

	cmd := exec.Command(r.dockerCmd, combinedArgs...)

	output, err := cmd.Output()

	return string(output), err
}

type withDockerCmdOption struct {
	dockerCmd string
}

func (o *withDockerCmdOption) applyOption(runner Runner) Runner {
	if dockerRunner, ok := runner.(*dockerRunner); ok {
		dockerRunner.dockerCmd = o.dockerCmd
	}

	return runner
}

func WithDockerCmd(dockerCmd string) RunnerOption {
	return &withDockerCmdOption{dockerCmd: dockerCmd}
}

type withDockerArgsOption struct {
	dockerArgs []string
}

func (o *withDockerArgsOption) applyOption(runner Runner) Runner {
	if dockerRunner, ok := runner.(*dockerRunner); ok {
		dockerRunner.dockerArgs = o.dockerArgs
	}

	return runner
}

func WithDockerArgs(dockerArgs []string) RunnerOption {
	return &withDockerArgsOption{dockerArgs: dockerArgs}
}

type withDockerImageOption struct {
	dockerImage string
}

func (o *withDockerImageOption) applyOption(runner Runner) Runner {
	if dockerRunner, ok := runner.(*dockerRunner); ok {
		dockerRunner.dockerImage = o.dockerImage
	}

	return runner
}

func WithDockerImage(dockerImage string) RunnerOption {
	return &withDockerImageOption{dockerImage: dockerImage}
}

func (r *dockerRunner) GetDockerCommandAndArgs() string {
	return fmt.Sprintf("%s %s %s", r.dockerCmd, strings.Join(r.dockerArgs, " "), r.dockerImage)
}
