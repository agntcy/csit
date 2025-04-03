// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package testutils

import (
	"os/exec"
	"strings"
)

type localRunner struct {
	envVars map[string]string
}

func (r *localRunner) Run(command string, args ...string) (string, error) {
	cmdBuilder := strings.Builder{}
	for key, value := range r.envVars {
		cmdBuilder.WriteString(key + "=" + value + " ")
	}
	cmdBuilder.WriteString(command)

	cmd := exec.Command(cmdBuilder.String(), args...)

	output, err := cmd.Output()

	return string(output), err
}

func (r *localRunner) GetDockerCommandAndArgs() string {
	return ""
}
