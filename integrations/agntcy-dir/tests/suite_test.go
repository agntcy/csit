package tests

import (
	"testing"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

func TestTests(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Tests Suite")
}
