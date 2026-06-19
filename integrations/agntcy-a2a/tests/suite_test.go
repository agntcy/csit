// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

func TestTests(t *testing.T) {
	gomega.RegisterFailHandler(Fail)
	RunSpecs(t, "A2A Interop Suite")
}
