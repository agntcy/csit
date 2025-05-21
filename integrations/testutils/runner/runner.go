// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package runner

import (
	"fmt"
)

type Runner interface {
	Run(command string, args ...string) (string, error)
	GetDockerCommandAndArgs() string
}

type RunnerType int

const (
	RunnerTypeLocal RunnerType = iota
	RunnerTypeDocker
)

type RunnerOption interface {
	applyOption(runner Runner) Runner
}

type withEnvVarsOption struct {
	envVars map[string]string
}

func (o *withEnvVarsOption) applyOption(runner Runner) Runner {
	if localRunner, ok := runner.(*localRunner); ok {
		localRunner.envVars = o.envVars
	} else if dockerRunner, ok := runner.(*dockerRunner); ok {
		dockerRunner.envVars = o.envVars
	}

	return runner
}

func WithEnvVars(envVars map[string]string) RunnerOption {
	return &withEnvVarsOption{envVars: envVars}
}

func NewRunner(variant RunnerType, options ...RunnerOption) (Runner, error) {
	var runner Runner

	switch variant {
	case RunnerTypeLocal:
		runner = &localRunner{}
	case RunnerTypeDocker:
		runner = &dockerRunner{}
	}

	for _, option := range options {
		if option != nil {
			runner = option.applyOption(runner)
		}
	}

	switch typedRunner := runner.(type) {
	case *dockerRunner:
		if typedRunner.dockerCmd == "" {
			return nil, fmt.Errorf("docker cmd is required")
		}
		if typedRunner.dockerImage == "" {
			return nil, fmt.Errorf("docker image is required")
		}
	}

	return runner, nil
}
