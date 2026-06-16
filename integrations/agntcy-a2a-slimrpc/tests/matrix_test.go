// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var (
	sharedAssets *fixtureAssets
	skipAll      bool
)

var _ = ginkgo.BeforeSuite(func() {
	if os.Getenv("SKIP_SLIM_A2A") == "1" {
		skipAll = true
		return
	}

	root := componentRoot()
	cache := filepath.Join(root, ".cache")
	if err := os.MkdirAll(cache, 0o755); err != nil {
		ginkgo.Fail(fmt.Sprintf("mkdir cache: %v", err))
	}

	buildCtx, cancel := context.WithTimeout(context.Background(), buildTimeout)
	defer cancel()

	if err := ensureSlimBindingsSetup(buildCtx); err != nil {
		ginkgo.Fail(err.Error())
	}

	goSrv := filepath.Join(cache, "csit-slim-go-server-"+slimBindingsCacheTag)
	goPrb := filepath.Join(cache, "csit-slim-go-probe-"+slimBindingsCacheTag)
	if err := buildGoFixture(buildCtx, root, goSrv, goPrb); err != nil {
		ginkgo.Fail(err.Error())
	}

	pyDir := filepath.Join(root, "fixtures", "python")
	assets := &fixtureAssets{
		goServerBin:      goSrv,
		goProbeBin:       goPrb,
		pythonFixtureDir: pyDir,
	}
	if !slimGoOnly() {
		venv := filepath.Join(cache, "csit-slim-venv")
		pyBin, err := ensurePythonVenv(buildCtx, root, venv)
		if err != nil {
			ginkgo.Fail(err.Error())
		}
		assets.pythonBin = pyBin
	}
	sharedAssets = assets
})

var _ = ginkgo.Describe("A2A SLIMRPC interoperability", ginkgo.Ordered, ginkgo.Label("suite-slim-a2a"), func() {
	ginkgo.BeforeEach(func() {
		if skipAll {
			ginkgo.Skip("SKIP_SLIM_A2A=1")
		}
		if !slimReachable(slimServerURL()) {
			ginkgo.Skip("SLIM node not reachable at " + slimServerURL() + " (set SLIM_SERVER or start slimctl)")
		}
		gomega.Expect(sharedAssets).NotTo(gomega.BeNil())
	})

	langs := interopLanguages()
	for _, srv := range langs {
		for _, cli := range langs {
			srv := srv
			cli := cli

			ginkgo.It(
				fmt.Sprintf("SLIMRPC client %s calls server %s (%s)", cli, srv, probeText),
				ginkgo.Label("behavior-core", "pair-"+cli+"-"+srv),
				func() {
					out, err := runInteropProbe(srv, cli)
					gomega.Expect(err).NotTo(gomega.HaveOccurred(), fmt.Sprintf("probe output:\n%s", out))
					gomega.Expect(out).To(gomega.ContainSubstring(probeText))
				},
			)

			ginkgo.It(
				fmt.Sprintf("SLIMRPC client %s observes a completed task with an echoed artifact from server %s", cli, srv),
				ginkgo.Label("behavior-lifecycle", "pair-"+cli+"-"+srv),
				func() {
					out, err := runInteropProbe(srv, cli)
					gomega.Expect(err).NotTo(gomega.HaveOccurred(), fmt.Sprintf("probe output:\n%s", out))

					state, ok := parseProbeField(out, "CSIT_SLIM_TASK_STATE")
					gomega.Expect(ok).To(gomega.BeTrue(), fmt.Sprintf("probe output missing CSIT_SLIM_TASK_STATE:\n%s", out))
					gomega.Expect(state).To(gomega.Equal("TASK_STATE_COMPLETED"), fmt.Sprintf("probe output:\n%s", out))

					present, ok := parseProbeField(out, "CSIT_SLIM_ARTIFACT_PRESENT")
					gomega.Expect(ok).To(gomega.BeTrue(), fmt.Sprintf("probe output missing CSIT_SLIM_ARTIFACT_PRESENT:\n%s", out))
					gomega.Expect(present).To(gomega.Equal("true"), fmt.Sprintf("probe output:\n%s", out))

					artifactText, ok := parseProbeField(out, "CSIT_SLIM_ARTIFACT_TEXT")
					gomega.Expect(ok).To(gomega.BeTrue(), fmt.Sprintf("probe output missing CSIT_SLIM_ARTIFACT_TEXT:\n%s", out))
					gomega.Expect(artifactText).To(gomega.ContainSubstring(probeText), fmt.Sprintf("probe output:\n%s", out))
				},
			)
		}
	}
})

// runInteropProbe starts the server fixture for srv, waits for its ready marker, runs the
// cli probe against it, and returns the probe's combined output. Cleanup of the server
// process is registered on the calling spec via ginkgo.DeferCleanup.
func runInteropProbe(srv, cli string) (string, error) {
	ctx := context.Background()
	logs := &lockedBuffer{}
	u := slimServerURL()
	sec := slimSharedSecret()

	cmd, err := startServer(ctx, srv, u, sec, sharedAssets, logs)
	if err != nil {
		return "", err
	}
	ginkgo.DeferCleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Signal(syscall.SIGTERM)
			_, _ = cmd.Process.Wait()
		}
	})

	if err := waitServerReady(cmd, logs, readyMarkerForServer(srv)); err != nil {
		return "", err
	}

	remote := serverIdentity(srv)
	return runProbe(ctx, cli, u, sec, remote, sharedAssets)
}

// parseProbeField extracts the value of a `KEY=value` line emitted by the probe fixtures.
func parseProbeField(out, key string) (string, bool) {
	prefix := key + "="
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.HasPrefix(line, prefix) {
			return strings.TrimPrefix(line, prefix), true
		}
	}
	return "", false
}
