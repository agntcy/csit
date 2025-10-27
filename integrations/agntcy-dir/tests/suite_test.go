// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"os"
	"runtime"
	"testing"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var (
	dirAPIHost = "127.0.0.1"
	dirAPIPort = 8888
)

func TestTests(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Tests Suite")
}

var _ = ginkgo.BeforeSuite(func() {
	if runtime.GOOS != "linux" && os.Getenv("RUNNER_TYPE") != "local" {
		dirAPIHost = "host.docker.internal"
	}
})
