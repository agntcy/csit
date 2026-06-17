// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	// CGO looks under $GOPATH/.cgo-cache/slim-bindings/<tier>/ — tier is hardcoded in slim-bindings-go
	// slim_bindings.go #cgo LDFLAGS (currently v1.4.1), independent of go.mod pseudo-version.
	slimBindingsCGOCacheTier = "v1.4.1"

	// Native libs must land under $GOPATH/.cgo-cache/slim-bindings/<tier>/ where <tier> is the path
	// baked into slim-bindings-go's slim_bindings.go #cgo LDFLAGS (currently v1.4.1), NOT necessarily
	// the full module pseudo-version in go.mod. slim-bindings-setup chooses the cache dir from its own
	// build Version(); go run ...@v1.4.1-0.... would look for a non-existent GitHub release zip.
	slimBindingsSetupModule = "github.com/agntcy/slim-bindings-go/cmd/slim-bindings-setup@v1.4.1"

	// slimBindingsCacheTag must change whenever slimBindingsSetupModule / fixtures/go go.mod pin changes,
	// so cached probe/server binaries are not reused after a slim-bindings upgrade (stale CGO link → wire decode errors on the node).
	slimBindingsCacheTag = "v1.4.1-0b5d5177f2ae"

	// PyPI slima2a declares Python >=3.10 only; macOS /usr/bin/python3 is often 3.9.
	pythonMinMajor = 3
	pythonMinMinor = 10

	probeText = "Hello there!"

	// Probe scenario selectors (passed via --scenario). "echo" is the default
	// round-trip behavior; the others drive non-echo server responses. The
	// matching request-text sentinels live in the probe/server fixtures
	// (fixtures/{go,python}) since those are separate binaries/processes.
	scenarioEcho          = "echo"
	scenarioMessageOnly   = "message-only"
	scenarioTaskFailure   = "task-failure"
	scenarioInputRequired = "input-required"
	scenarioStreaming     = "streaming"
	scenarioTaskCancel    = "task-cancel"

	fixtureReadyTimeout = 90 * time.Second
	probeTimeout        = 3 * time.Minute
	buildTimeout        = 10 * time.Minute

	// slim-bindings / dataplane can log reconnect storms to stdout; unbounded capture makes
	// waitServerReady's periodic logs.String() scans effectively hang. Keep head (ready marker)
	// and recent tail for failures.
	maxFixtureLogBytes   = 512 * 1024
	logTruncateHeadBytes = 16 * 1024
	logTruncateTailBytes = 128 * 1024
)

type lockedBuffer struct {
	mu           sync.Mutex
	buf          bytes.Buffer
	streamPrefix []byte // first logTruncateHeadBytes of the stream (ready marker lives here)
}

func (lb *lockedBuffer) Write(p []byte) (int, error) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	if len(lb.streamPrefix) < logTruncateHeadBytes && len(p) > 0 {
		need := logTruncateHeadBytes - len(lb.streamPrefix)
		take := need
		if take > len(p) {
			take = len(p)
		}
		lb.streamPrefix = append(lb.streamPrefix, p[:take]...)
	}
	n, err := lb.buf.Write(p)
	lb.truncateLocked()
	return n, err
}

// truncateLocked shrinks the buffer when SLIM/slim-bindings flood logs (reconnect loops).
// Preserves the stream head (streamPrefix) and recent tail without copying the full oversized buffer.
func (lb *lockedBuffer) truncateLocked() {
	if lb.buf.Len() <= maxFixtureLogBytes {
		return
	}
	full := lb.buf.Bytes()
	if len(full) <= len(lb.streamPrefix)+logTruncateTailBytes {
		return
	}
	tailStart := len(full) - logTruncateTailBytes
	if tailStart < 0 {
		tailStart = 0
	}
	tail := append([]byte(nil), full[tailStart:]...)
	lb.buf.Reset()
	_, _ = lb.buf.Write(lb.streamPrefix)
	_, _ = lb.buf.WriteString("\n...[fixture log truncated: dataplane reconnect noise omitted]...\n")
	_, _ = lb.buf.Write(tail)
}

func (lb *lockedBuffer) String() string {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	return lb.buf.String()
}

func componentRoot() string {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("failed to determine test file path")
	}
	return filepath.Dir(filepath.Dir(currentFile))
}

// integrationsDir is the integrations/ module root (parent of agntcy-a2a-slimrpc/).
func integrationsDir() string {
	return filepath.Clean(filepath.Join(componentRoot(), ".."))
}

// slimGoModCache returns GOMODCACHE to use for slim-bindings CGO: the upstream
// package embeds -L${SRCDIR}/../../../../../.cgo-cache/... which only resolves when
// the module lives under $HOME/go/pkg/mod (where slim-bindings-setup installs libs).
func slimGoModCache() string {
	home := os.Getenv("HOME")
	if home == "" {
		return ""
	}
	return filepath.Join(home, "go", "pkg", "mod")
}

func withSlimGoModCache(cmd *exec.Cmd) {
	dir := slimGoModCache()
	if dir == "" {
		return
	}
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return
	}
	if cmd.Env == nil {
		cmd.Env = os.Environ()
	}
	cmd.Env = appendEnvKV(cmd.Env, "GOMODCACHE", dir)
}

func appendEnvKV(env []string, key, val string) []string {
	prefix := key + "="
	var out []string
	for _, e := range env {
		if !strings.HasPrefix(e, prefix) {
			out = append(out, e)
		}
	}
	return append(out, prefix+val)
}

// ensureSlimBindingsSetup downloads/installs the slim-bindings native library required to link Go SLIMRPC fixtures.
// Set SKIP_SLIM_BINDINGS_SETUP=1 only if you have already run setup manually for this machine.
func ensureSlimBindingsSetup(ctx context.Context) error {
	if os.Getenv("SKIP_SLIM_BINDINGS_SETUP") != "1" {
		dir := integrationsDir()
		cmd := exec.CommandContext(ctx, "go", "run", slimBindingsSetupModule)
		cmd.Dir = dir
		withSlimGoModCache(cmd)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf(
				"slim-bindings-setup failed (needs network on first run): %w\n%s\n"+
					"fix: cd %q && go run %s",
				err, string(out), dir, slimBindingsSetupModule,
			)
		}
	}
	return installSlimNativeOverride(ctx)
}

// installSlimNativeOverride copies a locally built libslim_bindings_*.a into the CGO cache tier
// when CSIT_SLIM_NATIVE_LIB is set (file path, or directory containing the expected .a name).
// Use the same slim checkout / commit as your running slim node when slim-bindings-setup's
// v1.4.1 prebuild is too old (invalid wire type / session handshake failures against slim main).
func installSlimNativeOverride(ctx context.Context) error {
	raw := strings.TrimSpace(os.Getenv("CSIT_SLIM_NATIVE_LIB"))
	if raw == "" {
		return nil
	}
	expected := slimNativeStaticLibName()
	if expected == "" {
		return fmt.Errorf("CSIT_SLIM_NATIVE_LIB: unsupported GOOS/GOARCH %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	gopath, err := goEnvGOPATH(ctx)
	if err != nil {
		return fmt.Errorf("CSIT_SLIM_NATIVE_LIB: go env GOPATH: %w", err)
	}
	if gopath == "" {
		return fmt.Errorf("CSIT_SLIM_NATIVE_LIB: GOPATH is empty")
	}
	cacheDir := filepath.Join(gopath, ".cgo-cache", "slim-bindings", slimBindingsCGOCacheVersion())
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return fmt.Errorf("CSIT_SLIM_NATIVE_LIB: mkdir %q: %w", cacheDir, err)
	}

	srcFile, err := resolveSlimNativeLibSource(raw, expected)
	if err != nil {
		return fmt.Errorf("CSIT_SLIM_NATIVE_LIB: %w", err)
	}

	dst := filepath.Join(cacheDir, expected)
	in, err := os.Open(srcFile)
	if err != nil {
		return fmt.Errorf("CSIT_SLIM_NATIVE_LIB open %q: %w", srcFile, err)
	}
	defer in.Close()
	outf, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("CSIT_SLIM_NATIVE_LIB write %q: %w", dst, err)
	}
	defer outf.Close()
	if _, err := io.Copy(outf, in); err != nil {
		return fmt.Errorf("CSIT_SLIM_NATIVE_LIB copy to %q: %w", dst, err)
	}
	return nil
}

func goEnvGOPATH(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "go", "env", "GOPATH")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// slimRustTargetTriple matches slim-bindings-setup naming for static libs.
func slimRustTargetTriple() string {
	switch runtime.GOOS {
	case "darwin":
		if runtime.GOARCH == "arm64" {
			return "aarch64-apple-darwin"
		}
		return "x86_64-apple-darwin"
	case "linux":
		if runtime.GOARCH == "arm64" {
			return "aarch64-unknown-linux-gnu"
		}
		return "x86_64-unknown-linux-gnu"
	case "windows":
		return "x86_64-pc-windows-gnu"
	default:
		return ""
	}
}

// slimBindingsCGOCacheVersion is the .cgo-cache/slim-bindings/<version>/ tier used when copying CSIT_SLIM_NATIVE_LIB.
// slim `main` generated Go bindings default to "devel"; released slim-bindings-go uses v1.4.1.
func slimBindingsCGOCacheVersion() string {
	if v := strings.TrimSpace(os.Getenv("CSIT_SLIM_BINDINGS_CGO_VERSION")); v != "" {
		return v
	}
	if strings.TrimSpace(os.Getenv("CSIT_SLIM_BINDINGS_GO_REPLACE")) != "" {
		return "devel"
	}
	return slimBindingsCGOCacheTier
}

// slimNativeStaticLibName is the libslim_bindings_*.a basename expected in the CGO cache dir.
func slimNativeStaticLibName() string {
	t := slimRustTargetTriple()
	if t == "" {
		return ""
	}
	name := strings.ReplaceAll(t, "-unknown-", "_")
	name = strings.ReplaceAll(name, "-", "_")
	return fmt.Sprintf("libslim_bindings_%s.a", name)
}

// slimNativeLibSearchBasenames lists local build artifact names (cargo / slim repo) that map to the CGO cache name.
func slimNativeLibSearchBasenames() []string {
	primary := slimNativeStaticLibName()
	seen := map[string]struct{}{primary: {}}
	var out []string
	add := func(n string) {
		if n == "" {
			return
		}
		if _, ok := seen[n]; ok {
			return
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	add(primary)
	switch runtime.GOOS {
	case "darwin":
		if runtime.GOARCH == "arm64" {
			add("libslim_bindings_aarch64_darwin.a")
		} else {
			add("libslim_bindings_x86_64_darwin.a")
		}
	case "linux":
		if runtime.GOARCH == "arm64" {
			add("libslim_bindings_aarch64_linux_gnu.a")
		} else {
			add("libslim_bindings_x86_64_linux_gnu.a")
		}
	case "windows":
		add("libslim_bindings_x86_64_windows_gnu.a")
	}
	add("libslim_bindings.a")
	return out
}

func resolveSlimNativeLibSource(path, cacheBasename string) (string, error) {
	st, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("stat %q: %w", path, err)
	}
	if !st.IsDir() {
		return path, nil
	}
	for _, base := range slimNativeLibSearchBasenames() {
		candidate := filepath.Join(path, base)
		if st2, err2 := os.Stat(candidate); err2 == nil && !st2.IsDir() {
			return candidate, nil
		}
	}
	matches, _ := filepath.Glob(filepath.Join(path, "libslim_bindings*.a"))
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return pickPreferredSlimNativeLib(matches), nil
	}
	return "", fmt.Errorf(
		"directory %q has no libslim_bindings*.a (e.g. %q, libslim_bindings.a, or bindings/go/slim_bindings/libslim_bindings_aarch64_darwin.a)",
		path, cacheBasename,
	)
}

func pickPreferredSlimNativeLib(paths []string) string {
	best := paths[0]
	bestScore := slimNativeLibPathScore(best)
	for _, p := range paths[1:] {
		if s := slimNativeLibPathScore(p); s > bestScore {
			best = p
			bestScore = s
		}
	}
	return best
}

func slimNativeLibPathScore(p string) int {
	score := 0
	if strings.Contains(p, "release") {
		score += 4
	}
	if strings.Contains(p, "bindings/go") {
		score += 3
	}
	if strings.Contains(p, "deps"+string(filepath.Separator)) {
		score -= 2
	}
	if strings.Contains(p, "debug") {
		score -= 1
	}
	base := filepath.Base(p)
	if base == slimNativeStaticLibName() || base == "libslim_bindings_aarch64_darwin.a" {
		score += 2
	}
	return score
}

// slimGoOnly returns true when Python fixtures are skipped (slim-bindings 2.x on main vs PyPI slima2a~=1.1).
func slimGoOnly() bool {
	return os.Getenv("CSIT_SLIM_GO_ONLY") == "1"
}

// interopLanguages is the server/client language matrix for this run.
func interopLanguages() []string {
	if slimGoOnly() {
		return []string{"go"}
	}
	return []string{"go", "python"}
}

func slimServerURL() string {
	if v := os.Getenv("SLIM_SERVER"); v != "" {
		return v
	}
	return "http://127.0.0.1:46357"
}

func slimSharedSecret() string {
	if v := os.Getenv("SLIM_SHARED_SECRET"); v != "" {
		return v
	}
	return "my_shared_secret_for_testing_purposes_only"
}

func slimReachable(endpoint string) bool {
	u, err := url.Parse(endpoint)
	if err != nil {
		return false
	}
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		switch u.Scheme {
		case "https":
			port = "443"
		default:
			port = "80"
		}
	}
	addr := net.JoinHostPort(host, port)
	c, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return false
	}
	_ = c.Close()
	return true
}

func pythonMeetsSlimA2A(py string) bool {
	cmd := exec.Command(py, "-c", fmt.Sprintf(
		"import sys; sys.exit(0 if sys.version_info[:2]>=(%d,%d) else 1)",
		pythonMinMajor, pythonMinMinor))
	return cmd.Run() == nil
}

func resolvePythonCommand() (string, error) {
	seen := map[string]struct{}{}
	var candidates []string
	add := func(p string) {
		if p == "" {
			return
		}
		if _, ok := seen[p]; ok {
			return
		}
		seen[p] = struct{}{}
		candidates = append(candidates, p)
	}
	if p := os.Getenv("PYTHON"); p != "" {
		add(p)
	}
	for _, name := range []string{"python3.13", "python3.12", "python3.11", "python3.10", "python3", "python"} {
		if path, err := exec.LookPath(name); err == nil {
			add(path)
		}
	}
	for _, p := range candidates {
		if pythonMeetsSlimA2A(p) {
			return p, nil
		}
	}
	return "", fmt.Errorf(
		"no Python >=%d.%d on PATH (PyPI slima2a requires it); install python3.10+ (e.g. brew install python@3.12) or set PYTHON=/path/to/python3.10",
		pythonMinMajor, pythonMinMinor,
	)
}

func buildGoFixture(ctx context.Context, root, outServer, outProbe string) error {
	dir := filepath.Join(root, "fixtures", "go")
	modFile, err := slimFixtureGoModFile(ctx, dir)
	if err != nil {
		return err
	}
	sumFile := strings.TrimSuffix(modFile, ".mod") + ".sum"

	stale := func(bin string) bool {
		mi, err := os.Stat(modFile)
		if err != nil {
			return true
		}
		bi, err := os.Stat(bin)
		if err != nil {
			return true
		}
		if bi.ModTime().Before(mi.ModTime()) {
			return true
		}
		if si, err := os.Stat(sumFile); err == nil && bi.ModTime().Before(si.ModTime()) {
			return true
		}
		return false
	}

	if _, err := os.Stat(outServer); err == nil {
		if _, err2 := os.Stat(outProbe); err2 == nil {
			if !stale(outServer) && !stale(outProbe) {
				return nil
			}
		}
	}
	build := func(out, pkg string) error {
		args := []string{"build", "-modfile", modFile, "-o", out, pkg}
		cmd := exec.CommandContext(ctx, "go", args...)
		cmd.Dir = dir
		withSlimGoModCache(cmd)
		outBytes, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("go build %s: %w\n%s", pkg, err, string(outBytes))
		}
		return nil
	}
	if err := build(outServer, "./cmd/server"); err != nil {
		return err
	}
	if err := build(outProbe, "./cmd/probe"); err != nil {
		return err
	}
	return nil
}

// slimFixtureGoModFile returns go.mod or a generated go.csit.mod when CSIT_SLIM_BINDINGS_GO_REPLACE is set.
func slimFixtureGoModFile(ctx context.Context, dir string) (string, error) {
	replace := strings.TrimSpace(os.Getenv("CSIT_SLIM_BINDINGS_GO_REPLACE"))
	if replace == "" {
		return filepath.Join(dir, "go.mod"), nil
	}
	abs, err := filepath.Abs(replace)
	if err != nil {
		return "", fmt.Errorf("CSIT_SLIM_BINDINGS_GO_REPLACE: %w", err)
	}
	if st, err := os.Stat(abs); err != nil || !st.IsDir() {
		return "", fmt.Errorf("CSIT_SLIM_BINDINGS_GO_REPLACE %q is not a directory", abs)
	}
	goFile := filepath.Join(abs, "slim_bindings.go")
	if st, err := os.Stat(goFile); err != nil || st.IsDir() {
		return "", fmt.Errorf(
			"CSIT_SLIM_BINDINGS_GO_REPLACE %q missing slim_bindings.go — run in slim: cd data-plane/bindings/go && task generate PROFILE=release",
			abs,
		)
	}
	localMod := filepath.Join(dir, "go.csit.mod")
	base, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return "", err
	}
	lines := strings.Split(strings.TrimRight(string(base), "\n"), "\n")
	var out []string
	const replaceLine = "replace github.com/agntcy/slim-bindings-go => "
	hasReplace := false
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "replace github.com/agntcy/slim-bindings-go") {
			out = append(out, replaceLine+abs)
			hasReplace = true
			continue
		}
		out = append(out, line)
	}
	if !hasReplace {
		out = append(out, "", replaceLine+abs)
	}
	if err := os.WriteFile(localMod, []byte(strings.Join(out, "\n")+"\n"), 0o644); err != nil {
		return "", err
	}
	tidy := exec.CommandContext(ctx, "go", "mod", "tidy", "-modfile", localMod)
	tidy.Dir = dir
	withSlimGoModCache(tidy)
	if out, err := tidy.CombinedOutput(); err != nil {
		return "", fmt.Errorf("go mod tidy -modfile go.csit.mod: %w\n%s", err, string(out))
	}
	return localMod, nil
}

func venvPythonBin(venvDir string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(venvDir, "Scripts", "python.exe")
	}
	return filepath.Join(venvDir, "bin", "python3")
}

func venvPipBin(venvDir string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(venvDir, "Scripts", "pip.exe")
	}
	return filepath.Join(venvDir, "bin", "pip")
}

// venvProbeScript must stay in sync with fixtures/python/requirements*.txt (slim-a2a-python / slima2a + slim-bindings).
// Upper bound <2 matches slima2a's slim-bindings~=1.x constraint; allows 1.5+ from main-line wheels when they appear.
const venvProbeScript = "import importlib.metadata as m\n" +
	"from packaging.version import Version\n" +
	"v = Version(m.version('slim-bindings'))\n" +
	"assert Version('1.4.1') <= v < Version('2.0.0'), m.version('slim-bindings')\n" +
	"v2 = Version(m.version('slima2a'))\n" +
	"assert v2 >= Version('0.6.0'), m.version('slima2a')\n" +
	"import slima2a\n" +
	"from a2a.client import ClientFactory\n"

func venvHasSlimA2A(ctx context.Context, venvPython string) bool {
	cmd := exec.CommandContext(ctx, venvPython, "-c", venvProbeScript)
	return cmd.Run() == nil
}

func ensurePythonVenv(ctx context.Context, root, venvDir string) (python string, err error) {
	py, err := resolvePythonCommand()
	if err != nil {
		return "", err
	}
	python = venvPythonBin(venvDir)
	pip := venvPipBin(venvDir)
	if _, err := os.Stat(python); err == nil {
		if venvHasSlimA2A(ctx, python) {
			return python, nil
		}
		if err := os.RemoveAll(venvDir); err != nil {
			return "", fmt.Errorf("remove stale/incomplete venv %s: %w", venvDir, err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(venvDir), 0o755); err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, py, "-m", "venv", venvDir)
	if out, e := cmd.CombinedOutput(); e != nil {
		return "", fmt.Errorf("python -m venv: %w\n%s", e, string(out))
	}
	// Old venvs ship pip that mishandles metadata; slima2a needs a current resolver.
	cmd = exec.CommandContext(ctx, python, "-m", "pip", "install", "-U", "pip")
	if out, e := cmd.CombinedOutput(); e != nil {
		return "", fmt.Errorf("venv pip upgrade: %w\n%s", e, string(out))
	}
	req := filepath.Join(root, "fixtures", "python", "requirements.txt")
	if p := os.Getenv("CSIT_SLIM_PYTHON_REQUIREMENTS"); p != "" {
		if filepath.IsAbs(p) {
			req = p
		} else {
			req = filepath.Join(root, "fixtures", "python", p)
		}
	}
	cmd = exec.CommandContext(ctx, pip, "install", "-r", req)
	if out, e := cmd.CombinedOutput(); e != nil {
		return "", fmt.Errorf("pip install: %w\n%s", e, string(out))
	}
	return python, nil
}

func serverIdentity(lang string) string {
	return fmt.Sprintf("agntcy/a2a_csit_slim/server_%s", lang)
}

func clientIdentity(lang string) string {
	return fmt.Sprintf("agntcy/a2a_csit_slim/client_%s", lang)
}

func readyMarkerForServer(lang string) string {
	switch lang {
	case "go", "python":
		return "CSIT_SLIM_SERVER_READY"
	default:
		return "CSIT_SLIM_SERVER_READY"
	}
}

func fixtureServerWriters(logs *lockedBuffer) (stdout, stderr io.Writer) {
	// Teeing to os.Stdout/stderr unbounded will freeze or crash the IDE when the dataplane
	// logs reconnect storms; logs are still captured in logs and printed on failure.
	if os.Getenv("CSIT_SLIM_STREAM_SERVER_LOGS") == "1" {
		return io.MultiWriter(os.Stdout, logs), io.MultiWriter(os.Stderr, logs)
	}
	return logs, logs
}

func startServer(ctx context.Context, lang, slimURL, secret string, assets *fixtureAssets, logs *lockedBuffer) (*exec.Cmd, error) {
	srvID := serverIdentity(lang)
	outW, errW := fixtureServerWriters(logs)
	switch lang {
	case "go":
		cmd := exec.CommandContext(ctx, assets.goServerBin,
			"--slim-endpoint", slimURL,
			"--identity", srvID,
			"--secret", secret,
		)
		cmd.Stdout = outW
		cmd.Stderr = errW
		if err := cmd.Start(); err != nil {
			return nil, err
		}
		return cmd, nil
	case "python":
		cmd := exec.CommandContext(ctx, assets.pythonBin,
			filepath.Join(assets.pythonFixtureDir, "csit_server.py"),
			"--slim-url", slimURL,
			"--identity", srvID,
			"--secret", secret,
		)
		cmd.Dir = assets.pythonFixtureDir
		cmd.Stdout = outW
		cmd.Stderr = errW
		if err := cmd.Start(); err != nil {
			return nil, err
		}
		return cmd, nil
	default:
		return nil, fmt.Errorf("unknown server language %q", lang)
	}
}

func runProbe(ctx context.Context, clientLang, slimURL, secret, remoteServerID, scenario, text string, assets *fixtureAssets) (string, error) {
	local := clientIdentity(clientLang)
	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	switch clientLang {
	case "go":
		cmd := exec.CommandContext(ctx, assets.goProbeBin,
			"--slim-endpoint", slimURL,
			"--local", local,
			"--remote", remoteServerID,
			"--secret", secret,
			"--scenario", scenario,
			"--text", text,
		)
		out, err := cmd.CombinedOutput()
		return string(out), err
	case "python":
		cmd := exec.CommandContext(ctx, assets.pythonBin,
			filepath.Join(assets.pythonFixtureDir, "csit_probe.py"),
			"--slim-url", slimURL,
			"--local", local,
			"--remote", remoteServerID,
			"--secret", secret,
			"--scenario", scenario,
			"--text", text,
		)
		cmd.Dir = assets.pythonFixtureDir
		out, err := cmd.CombinedOutput()
		return string(out), err
	default:
		return "", fmt.Errorf("unknown client language %q", clientLang)
	}
}

type fixtureAssets struct {
	goServerBin      string
	goProbeBin       string
	pythonBin        string
	pythonFixtureDir string
}

var errStoppedEarly = errors.New("server exited before ready")

func waitServerReady(cmd *exec.Cmd, logs *lockedBuffer, marker string) error {
	deadline := time.Now().Add(fixtureReadyTimeout)
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	for time.Now().Before(deadline) {
		select {
		case err := <-done:
			if err == nil {
				err = errStoppedEarly
			}
			return fmt.Errorf("server exited: %w\n%s", err, logs.String())
		default:
		}
		logStr := logs.String()
		if strings.Contains(logStr, marker) {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for server ready marker %q:\n%s", marker, logs.String())
}
