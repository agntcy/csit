// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2aclient"
	"github.com/a2aproject/a2a-go/v2/a2aclient/agentcard"
	a2agrpc "github.com/a2aproject/a2a-go/v2/a2agrpc/v1"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	fixtureReadyTimeout  = 20 * time.Second
	probeTimeout         = 2 * time.Minute
	buildTimeout         = 3 * time.Minute
	stopTimeout          = 5 * time.Second
	requestText          = "ping"
	pendingRequestText   = "pending"
	requestDataKind      = "structured"
	requestDataScope     = "interop"
	requestMetadataKey   = "csit"
	requestMetadataValue = "multipart"
)

type transportProtocol string

const (
	transportJSONRPC transportProtocol = "jsonrpc"
	transportREST    transportProtocol = "rest"
	transportGRPC    transportProtocol = "grpc"
)

type rustProbeOptions struct {
	expectSubscribeUnsupported bool
	expectPushSupported        bool
	expectPushUnsupported      bool
	relaxedErrorChecks         bool
	expectedPushErrorCode      int
}

type fixtureBinaries struct {
	tempDir    string
	goServer   string
	rustServer string
	rustProbe  string
}

type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (buffer *lockedBuffer) Write(data []byte) (int, error) {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	return buffer.buf.Write(data)
}

func (buffer *lockedBuffer) String() string {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	return buffer.buf.String()
}

type fixtureProcess struct {
	name   string
	cmd    *exec.Cmd
	cancel context.CancelFunc
	done   chan error
	logs   *lockedBuffer
}

func (process *fixtureProcess) stop() error {
	process.cancel()

	select {
	case err := <-process.done:
		return normalizeStopError(err)
	case <-time.After(stopTimeout):
	}

	if process.cmd.Process != nil {
		_ = process.cmd.Process.Kill()
	}

	select {
	case err := <-process.done:
		return normalizeStopError(err)
	case <-time.After(stopTimeout):
		return fmt.Errorf("timed out stopping %s", process.name)
	}
}

func normalizeStopError(err error) error {
	if err == nil {
		return nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return nil
	}

	return err
}

func componentRoot() string {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("failed to determine test file path")
	}

	return filepath.Dir(filepath.Dir(currentFile))
}

func findFreePort() int {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	defer listener.Close()

	return listener.Addr().(*net.TCPAddr).Port
}

func waitForReady(url string, done <-chan error, logs *lockedBuffer) error {
	client := &http.Client{Timeout: 500 * time.Millisecond}
	deadline := time.Now().Add(fixtureReadyTimeout)

	for time.Now().Before(deadline) {
		select {
		case err := <-done:
			if err == nil {
				err = errors.New("process exited before becoming ready")
			}
			return fmt.Errorf("fixture exited early while waiting for %s: %w\n%s", url, err, logs.String())
		default:
		}

		response, err := client.Get(url)
		if err == nil {
			response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return nil
			}
		}

		time.Sleep(200 * time.Millisecond)
	}

	return fmt.Errorf("timed out waiting for fixture readiness at %s\n%s", url, logs.String())
}

func waitForTCPListener(address string, logs *lockedBuffer) error {
	deadline := time.Now().Add(fixtureReadyTimeout)

	for time.Now().Before(deadline) {
		connection, err := net.DialTimeout("tcp", address, 500*time.Millisecond)
		if err == nil {
			connection.Close()
			return nil
		}

		time.Sleep(200 * time.Millisecond)
	}

	return fmt.Errorf("timed out waiting for fixture listener at %s\n%s", address, logs.String())
}

func executableName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}

	return name
}

func buildFixtureBinaries() (fixtureBinaries, error) {
	root := componentRoot()
	tempDir, err := os.MkdirTemp("", "agntcy-a2a-binaries-")
	if err != nil {
		return fixtureBinaries{}, fmt.Errorf("create temp dir: %w", err)
	}

	binaries := fixtureBinaries{
		tempDir:    tempDir,
		goServer:   filepath.Join(tempDir, executableName("go-jsonrpc-server")),
		rustServer: filepath.Join(tempDir, "cargo-target", "debug", executableName("interop-rust-server")),
		rustProbe:  filepath.Join(tempDir, "cargo-target", "debug", executableName("interop-rust-probe")),
	}

	buildCtx, cancel := context.WithTimeout(context.Background(), buildTimeout)
	defer cancel()

	goBuild := exec.CommandContext(buildCtx, "go", "build", "-o", binaries.goServer, "./fixtures/go-jsonrpc-server")
	goBuild.Dir = root
	if output, err := goBuild.CombinedOutput(); err != nil {
		_ = os.RemoveAll(tempDir)
		return fixtureBinaries{}, fmt.Errorf("build go fixture: %w\n%s", err, string(output))
	}

	rustBuild := exec.CommandContext(
		buildCtx,
		"cargo",
		"build",
		"--manifest-path",
		filepath.Join(root, "fixtures", "rust", "Cargo.toml"),
		"--bins",
		"--target-dir",
		filepath.Join(tempDir, "cargo-target"),
	)
	rustBuild.Dir = root
	if output, err := rustBuild.CombinedOutput(); err != nil {
		_ = os.RemoveAll(tempDir)
		return fixtureBinaries{}, fmt.Errorf("build rust fixtures: %w\n%s", err, string(output))
	}

	return binaries, nil
}

func startFixtureProcess(name string, dir string, readyURL string, command string, args ...string) (*fixtureProcess, error) {
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = dir

	logs := &lockedBuffer{}
	cmd.Stdout = logs
	cmd.Stderr = logs

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("start %s: %w", name, err)
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	if err := waitForReady(readyURL, done, logs); err != nil {
		cancel()
		<-done
		return nil, fmt.Errorf("wait for %s readiness: %w", name, err)
	}

	return &fixtureProcess{name: name, cmd: cmd, cancel: cancel, done: done, logs: logs}, nil
}

func startGoFixture(binaries fixtureBinaries, port int, protocol transportProtocol) (*fixtureProcess, string, error) {
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	args := []string{
		"--port",
		fmt.Sprintf("%d", port),
		"--protocol",
		string(protocol),
	}
	grpcAddress := ""
	if protocol == transportGRPC {
		grpcPort := findFreePort()
		grpcAddress = fmt.Sprintf("127.0.0.1:%d", grpcPort)
		args = append(args, "--grpc-port", fmt.Sprintf("%d", grpcPort))
	}

	process, err := startFixtureProcess(
		fmt.Sprintf("go-%s-server", protocol),
		componentRoot(),
		baseURL+"/.well-known/agent-card.json",
		binaries.goServer,
		args...,
	)
	if err != nil {
		return nil, "", err
	}
	if grpcAddress != "" {
		if err := waitForTCPListener(grpcAddress, process.logs); err != nil {
			_ = process.stop()
			return nil, "", fmt.Errorf("wait for go gRPC fixture listener: %w", err)
		}
	}
	return process, baseURL, err
}

func startRustFixture(binaries fixtureBinaries, port int, protocol transportProtocol) (*fixtureProcess, string, error) {
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	args := []string{
		"--port",
		fmt.Sprintf("%d", port),
		"--protocol",
		string(protocol),
	}
	grpcAddress := ""
	if protocol == transportGRPC {
		grpcPort := findFreePort()
		grpcAddress = fmt.Sprintf("127.0.0.1:%d", grpcPort)
		args = append(args, "--grpc-port", fmt.Sprintf("%d", grpcPort))
	}

	process, err := startFixtureProcess(
		fmt.Sprintf("rust-%s-server", protocol),
		componentRoot(),
		baseURL+"/.well-known/agent-card.json",
		binaries.rustServer,
		args...,
	)
	if err != nil {
		return nil, "", err
	}
	if grpcAddress != "" {
		if err := waitForTCPListener(grpcAddress, process.logs); err != nil {
			_ = process.stop()
			return nil, "", fmt.Errorf("wait for rust gRPC fixture listener: %w", err)
		}
	}
	return process, baseURL, err
}

func newGoClient(ctx context.Context, baseURL string) (*a2aclient.Client, error) {
	card, err := agentcard.DefaultResolver.Resolve(ctx, baseURL)
	if err != nil {
		return nil, err
	}

	return a2aclient.NewFromCard(
		ctx,
		card,
		a2agrpc.WithGRPCTransport(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
}

func newInteropRequest(text string, returnImmediately bool) *a2a.SendMessageRequest {
	message := a2a.NewMessage(
		a2a.MessageRoleUser,
		a2a.NewTextPart(text),
		a2a.NewDataPart(map[string]any{
			"kind":  requestDataKind,
			"scope": requestDataScope,
		}),
	)
	message.Metadata = map[string]any{
		requestMetadataKey: requestMetadataValue,
	}

	return &a2a.SendMessageRequest{
		Message: message,
		Config: &a2a.SendMessageConfig{
			ReturnImmediately: returnImmediately,
		},
	}
}

func assertMessageInteropPayload(message *a2a.Message, expectedText string, kind string) {
	gomega.Expect(message).NotTo(gomega.BeNil(), kind)

	text, err := firstMessageText(message)
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), kind)
	gomega.Expect(text).To(gomega.Equal(expectedText), kind)
	gomega.Expect(message.Parts).To(gomega.HaveLen(2), kind)

	dataPart, ok := message.Parts[1].Data().(map[string]any)
	gomega.Expect(ok).To(gomega.BeTrue(), kind)
	gomega.Expect(dataPart).To(gomega.HaveKeyWithValue("kind", requestDataKind), kind)
	gomega.Expect(dataPart).To(gomega.HaveKeyWithValue("scope", requestDataScope), kind)
	gomega.Expect(message.Metadata).To(gomega.HaveKeyWithValue(requestMetadataKey, requestMetadataValue), kind)
}

func assertTaskHistoryPayload(task *a2a.Task, expectedText string, kind string) {
	gomega.Expect(task.History).To(gomega.HaveLen(1), kind)
	assertMessageInteropPayload(task.History[0], expectedText, kind)
}

func firstMessageText(message *a2a.Message) (string, error) {
	for _, part := range message.Parts {
		if text := part.Text(); text != "" {
			return text, nil
		}
	}

	return "", errors.New("message did not include a text part")
}

func expectedServerText(serverPrefix string, text string) string {
	return fmt.Sprintf("%s server received: %s", serverPrefix, text)
}

func expectedCancelText(serverPrefix string) string {
	return fmt.Sprintf("%s server canceled task", serverPrefix)
}

func newInteropPushConfig() *a2a.PushConfig {
	return &a2a.PushConfig{
		ID:    "interop-config",
		URL:   "https://example.invalid/webhook",
		Token: "interop-token",
		Auth: &a2a.PushAuthInfo{
			Scheme:      "Bearer",
			Credentials: "interop-credential",
		},
	}
}

func assertTaskPushConfig(config *a2a.TaskPushConfig, taskID a2a.TaskID, expected *a2a.PushConfig, kind string) {
	gomega.Expect(config).NotTo(gomega.BeNil(), kind)
	gomega.Expect(config.TaskID).To(gomega.Equal(taskID), kind)
	gomega.Expect(config.Config).To(gomega.Equal(*expected), kind)
}

func goClientAssertPushLifecycle(ctx context.Context, client *a2aclient.Client, taskID a2a.TaskID) {
	pushConfig := newInteropPushConfig()

	createdConfig, err := client.CreateTaskPushConfig(ctx, &a2a.CreateTaskPushConfigRequest{
		TaskID: taskID,
		Config: *pushConfig,
	})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	assertTaskPushConfig(createdConfig, taskID, pushConfig, "created push config")

	fetchedConfig, err := client.GetTaskPushConfig(ctx, &a2a.GetTaskPushConfigRequest{
		TaskID: taskID,
		ID:     pushConfig.ID,
	})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	assertTaskPushConfig(fetchedConfig, taskID, pushConfig, "fetched push config")

	listedConfigs, err := client.ListTaskPushConfigs(ctx, &a2a.ListTaskPushConfigRequest{TaskID: taskID})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(listedConfigs).To(gomega.HaveLen(1))
	assertTaskPushConfig(listedConfigs[0], taskID, pushConfig, "listed push config")

	err = client.DeleteTaskPushConfig(ctx, &a2a.DeleteTaskPushConfigRequest{
		TaskID: taskID,
		ID:     pushConfig.ID,
	})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	listedConfigs, err = client.ListTaskPushConfigs(ctx, &a2a.ListTaskPushConfigRequest{TaskID: taskID})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(listedConfigs).To(gomega.BeEmpty())
}

func taskStatusText(task *a2a.Task) (string, error) {
	if task == nil || task.Status.Message == nil {
		return "", errors.New("task status did not include a message")
	}

	return firstMessageText(task.Status.Message)
}

func eventText(event a2a.Event) (string, bool, error) {
	switch value := event.(type) {
	case *a2a.Message:
		text, err := firstMessageText(value)
		return text, true, err
	case *a2a.Task:
		text, err := taskStatusText(value)
		return text, true, err
	case *a2a.TaskStatusUpdateEvent:
		if value.Status.Message == nil {
			return "", false, nil
		}
		text, err := firstMessageText(value.Status.Message)
		return text, true, err
	default:
		return "", false, nil
	}
}

func goClientSendTask(ctx context.Context, client *a2aclient.Client, text string, returnImmediately bool) (*a2a.Task, error) {
	result, err := client.SendMessage(ctx, newInteropRequest(text, returnImmediately))
	if err != nil {
		return nil, err
	}

	task, ok := result.(*a2a.Task)
	if !ok {
		return nil, fmt.Errorf("unexpected unary response type %T", result)
	}

	return task, nil
}

func goClientUnaryText(ctx context.Context, client *a2aclient.Client) (string, error) {
	task, err := goClientSendTask(ctx, client, requestText, false)
	if err != nil {
		return "", err
	}

	return taskStatusText(task)
}

func goClientStreamingText(ctx context.Context, client *a2aclient.Client) (string, error) {
	request := newInteropRequest(requestText, false)

	for event, err := range client.SendStreamingMessage(ctx, request) {
		if err != nil {
			return "", err
		}

		text, hasText, textErr := eventText(event)
		if textErr != nil {
			return "", textErr
		}
		if !hasText {
			continue
		}

		return text, nil
	}

	return "", errors.New("stream completed without a message event")
}

func goClientAssertLifecycle(ctx context.Context, client *a2aclient.Client, serverPrefix string, expectPushSupported bool) {
	completedTask, err := goClientSendTask(ctx, client, requestText, false)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(completedTask.Status.State).To(gomega.Equal(a2a.TaskStateCompleted))

	completedText, err := taskStatusText(completedTask)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(completedText).To(gomega.Equal(expectedServerText(serverPrefix, requestText)))
	assertTaskHistoryPayload(completedTask, requestText, "completed task history")

	fetchedTask, err := client.GetTask(ctx, &a2a.GetTaskRequest{ID: completedTask.ID})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(fetchedTask.Status.State).To(gomega.Equal(a2a.TaskStateCompleted))

	fetchedText, err := taskStatusText(fetchedTask)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(fetchedText).To(gomega.Equal(expectedServerText(serverPrefix, requestText)))
	assertTaskHistoryPayload(fetchedTask, requestText, "fetched task history")

	listedTasks, err := client.ListTasks(ctx, &a2a.ListTasksRequest{ContextID: completedTask.ContextID})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(listedTasks.Tasks).NotTo(gomega.BeEmpty())
	gomega.Expect(listedTasks.Tasks).To(gomega.ContainElement(gomega.HaveField("ID", completedTask.ID)))

	if expectPushSupported {
		goClientAssertPushLifecycle(ctx, client, completedTask.ID)
	}

	pendingTask, err := goClientSendTask(ctx, client, pendingRequestText, true)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(pendingTask.Status.State).To(gomega.Equal(a2a.TaskStateWorking))

	pendingText, err := taskStatusText(pendingTask)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(pendingText).To(gomega.Equal(expectedServerText(serverPrefix, pendingRequestText)))

	canceledTask, err := client.CancelTask(ctx, &a2a.CancelTaskRequest{ID: pendingTask.ID})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(canceledTask.Status.State).To(gomega.Equal(a2a.TaskStateCanceled))

	canceledText, err := taskStatusText(canceledTask)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(canceledText).To(gomega.Equal(expectedCancelText(serverPrefix)))

	fetchedCanceledTask, err := client.GetTask(ctx, &a2a.GetTaskRequest{ID: pendingTask.ID})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(fetchedCanceledTask.Status.State).To(gomega.Equal(a2a.TaskStateCanceled))

	fetchedCanceledText, err := taskStatusText(fetchedCanceledTask)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(fetchedCanceledText).To(gomega.Equal(expectedCancelText(serverPrefix)))

	_, err = client.GetTask(ctx, &a2a.GetTaskRequest{ID: a2a.NewTaskID()})
	gomega.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("task not found")))

	_, err = client.CancelTask(ctx, &a2a.CancelTaskRequest{ID: completedTask.ID})
	gomega.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("cancel")))
}

func runRustProbe(
	ctx context.Context,
	binaries fixtureBinaries,
	baseURL string,
	serverPrefix string,
	options rustProbeOptions,
) (string, error) {
	args := []string{
		"--card-url",
		baseURL,
		"--server-prefix",
		serverPrefix,
	}
	if options.expectSubscribeUnsupported {
		args = append(args, "--expect-subscribe-unsupported")
	}
	if options.expectPushSupported {
		args = append(args, "--expect-push-supported")
	}
	if options.expectPushUnsupported {
		args = append(args, "--expect-push-unsupported")
		if options.expectedPushErrorCode != 0 {
			args = append(args, "--expected-push-error-code", fmt.Sprintf("%d", options.expectedPushErrorCode))
		}
	}
	if options.relaxedErrorChecks {
		args = append(args, "--relaxed-error-checks")
	}

	cmd := exec.CommandContext(
		ctx,
		binaries.rustProbe,
		args...,
	)
	cmd.Dir = componentRoot()

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("rust probe failed: %w\n%s", err, string(output))
	}

	return string(output), nil
}

var _ = ginkgo.Describe("A2A Rust and Go interoperability", ginkgo.Ordered, func() {
	var (
		binaries              fixtureBinaries
		goJSONRPCFixture      *fixtureProcess
		rustJSONRPCFixture    *fixtureProcess
		goRESTFixture         *fixtureProcess
		rustRESTFixture       *fixtureProcess
		goGRPCFixture         *fixtureProcess
		rustGRPCFixture       *fixtureProcess
		goJSONRPCFixtureURL   string
		rustJSONRPCFixtureURL string
		goRESTFixtureURL      string
		rustRESTFixtureURL    string
		goGRPCFixtureURL      string
		rustGRPCFixtureURL    string
	)

	ginkgo.BeforeAll(func() {
		var err error

		binaries, err = buildFixtureBinaries()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		goJSONRPCFixture, goJSONRPCFixtureURL, err = startGoFixture(binaries, findFreePort(), transportJSONRPC)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		rustJSONRPCFixture, rustJSONRPCFixtureURL, err = startRustFixture(binaries, findFreePort(), transportJSONRPC)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		goRESTFixture, goRESTFixtureURL, err = startGoFixture(binaries, findFreePort(), transportREST)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		rustRESTFixture, rustRESTFixtureURL, err = startRustFixture(binaries, findFreePort(), transportREST)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		goGRPCFixture, goGRPCFixtureURL, err = startGoFixture(binaries, findFreePort(), transportGRPC)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		rustGRPCFixture, rustGRPCFixtureURL, err = startRustFixture(binaries, findFreePort(), transportGRPC)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.AfterAll(func() {
		if rustGRPCFixture != nil {
			gomega.Expect(rustGRPCFixture.stop()).To(gomega.Succeed(), rustGRPCFixture.logs.String())
		}
		if goGRPCFixture != nil {
			gomega.Expect(goGRPCFixture.stop()).To(gomega.Succeed(), goGRPCFixture.logs.String())
		}
		if rustRESTFixture != nil {
			gomega.Expect(rustRESTFixture.stop()).To(gomega.Succeed(), rustRESTFixture.logs.String())
		}
		if goRESTFixture != nil {
			gomega.Expect(goRESTFixture.stop()).To(gomega.Succeed(), goRESTFixture.logs.String())
		}
		if rustJSONRPCFixture != nil {
			gomega.Expect(rustJSONRPCFixture.stop()).To(gomega.Succeed(), rustJSONRPCFixture.logs.String())
		}
		if goJSONRPCFixture != nil {
			gomega.Expect(goJSONRPCFixture.stop()).To(gomega.Succeed(), goJSONRPCFixture.logs.String())
		}
		if binaries.tempDir != "" {
			gomega.Expect(os.RemoveAll(binaries.tempDir)).To(gomega.Succeed())
		}
	})

	ginkgo.Context("JSON-RPC transport", func() {
		ginkgo.It("lets the Go client call the Go fixture", ginkgo.Label("jsonrpc", "go-go"), func(ctx ginkgo.SpecContext) {
			requestCtx, cancel := context.WithTimeout(ctx, probeTimeout)
			defer cancel()

			client, err := newGoClient(requestCtx, goJSONRPCFixtureURL)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			unaryText, err := goClientUnaryText(requestCtx, client)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(unaryText).To(gomega.Equal(expectedServerText("go", requestText)))

			streamText, err := goClientStreamingText(requestCtx, client)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(streamText).To(gomega.Equal(expectedServerText("go", requestText)))

			goClientAssertLifecycle(requestCtx, client, "go", true)
		})

		ginkgo.It("lets the Go client call the Rust fixture", ginkgo.Label("jsonrpc", "go-rust"), func(ctx ginkgo.SpecContext) {
			requestCtx, cancel := context.WithTimeout(ctx, probeTimeout)
			defer cancel()

			client, err := newGoClient(requestCtx, rustJSONRPCFixtureURL)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			unaryText, err := goClientUnaryText(requestCtx, client)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(unaryText).To(gomega.Equal(expectedServerText("rust", requestText)))

			streamText, err := goClientStreamingText(requestCtx, client)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(streamText).To(gomega.Equal(expectedServerText("rust", requestText)))

			goClientAssertLifecycle(requestCtx, client, "rust", true)
		})

		ginkgo.It("lets the Rust client call the Go fixture", ginkgo.Label("jsonrpc", "rust-go"), func(ctx ginkgo.SpecContext) {
			requestCtx, cancel := context.WithTimeout(ctx, probeTimeout)
			defer cancel()

			output, err := runRustProbe(requestCtx, binaries, goJSONRPCFixtureURL, "go", rustProbeOptions{
				expectPushSupported: true,
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), output)
		})

		ginkgo.It("lets the Rust client call the Rust fixture", ginkgo.Label("jsonrpc", "rust-rust"), func(ctx ginkgo.SpecContext) {
			requestCtx, cancel := context.WithTimeout(ctx, probeTimeout)
			defer cancel()

			output, err := runRustProbe(requestCtx, binaries, rustJSONRPCFixtureURL, "rust", rustProbeOptions{
				expectPushSupported: true,
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), output)
		})
	})

	ginkgo.Context("HTTP+JSON transport", func() {
		ginkgo.It("lets the Go client call the Go fixture over REST", ginkgo.Label("rest", "go-go"), func(ctx ginkgo.SpecContext) {
			requestCtx, cancel := context.WithTimeout(ctx, probeTimeout)
			defer cancel()

			client, err := newGoClient(requestCtx, goRESTFixtureURL)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			unaryText, err := goClientUnaryText(requestCtx, client)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(unaryText).To(gomega.Equal(expectedServerText("go", requestText)))

			streamText, err := goClientStreamingText(requestCtx, client)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(streamText).To(gomega.Equal(expectedServerText("go", requestText)))

			goClientAssertLifecycle(requestCtx, client, "go", true)
		})

		ginkgo.It("lets the Go client call the Rust fixture over REST", ginkgo.Label("rest", "go-rust"), func(ctx ginkgo.SpecContext) {
			requestCtx, cancel := context.WithTimeout(ctx, probeTimeout)
			defer cancel()

			client, err := newGoClient(requestCtx, rustRESTFixtureURL)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			unaryText, err := goClientUnaryText(requestCtx, client)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(unaryText).To(gomega.Equal(expectedServerText("rust", requestText)))

			streamText, err := goClientStreamingText(requestCtx, client)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(streamText).To(gomega.Equal(expectedServerText("rust", requestText)))

			goClientAssertLifecycle(requestCtx, client, "rust", true)
		})

		ginkgo.It("lets the Rust client call the Go fixture over REST", ginkgo.Label("rest", "rust-go"), func(ctx ginkgo.SpecContext) {
			requestCtx, cancel := context.WithTimeout(ctx, probeTimeout)
			defer cancel()

			output, err := runRustProbe(requestCtx, binaries, goRESTFixtureURL, "go", rustProbeOptions{
				expectPushSupported: true,
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), output)
		})

		ginkgo.It("lets the Rust client call the Rust fixture over REST", ginkgo.Label("rest", "rust-rust"), func(ctx ginkgo.SpecContext) {
			requestCtx, cancel := context.WithTimeout(ctx, probeTimeout)
			defer cancel()

			output, err := runRustProbe(requestCtx, binaries, rustRESTFixtureURL, "rust", rustProbeOptions{
				expectPushSupported: true,
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), output)
		})
	})

	ginkgo.Context("gRPC transport", func() {
		ginkgo.It("lets the Go client call the Go fixture over gRPC", ginkgo.Label("grpc", "go-go"), func(ctx ginkgo.SpecContext) {
			requestCtx, cancel := context.WithTimeout(ctx, probeTimeout)
			defer cancel()

			client, err := newGoClient(requestCtx, goGRPCFixtureURL)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			unaryText, err := goClientUnaryText(requestCtx, client)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(unaryText).To(gomega.Equal(expectedServerText("go", requestText)))

			streamText, err := goClientStreamingText(requestCtx, client)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(streamText).To(gomega.Equal(expectedServerText("go", requestText)))

			goClientAssertLifecycle(requestCtx, client, "go", true)
		})

		ginkgo.It("lets the Go client call the Rust fixture over gRPC", ginkgo.Label("grpc", "go-rust"), func(ctx ginkgo.SpecContext) {
			requestCtx, cancel := context.WithTimeout(ctx, probeTimeout)
			defer cancel()

			client, err := newGoClient(requestCtx, rustGRPCFixtureURL)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			unaryText, err := goClientUnaryText(requestCtx, client)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(unaryText).To(gomega.Equal(expectedServerText("rust", requestText)))

			streamText, err := goClientStreamingText(requestCtx, client)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(streamText).To(gomega.Equal(expectedServerText("rust", requestText)))

			goClientAssertLifecycle(requestCtx, client, "rust", true)
		})

		ginkgo.It("lets the Rust client call the Go fixture over gRPC", ginkgo.Label("grpc", "rust-go"), func(ctx ginkgo.SpecContext) {
			requestCtx, cancel := context.WithTimeout(ctx, probeTimeout)
			defer cancel()

			output, err := runRustProbe(requestCtx, binaries, goGRPCFixtureURL, "go", rustProbeOptions{
				expectPushSupported: true,
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), output)
		})

		ginkgo.It("lets the Rust client call the Rust fixture over gRPC", ginkgo.Label("grpc", "rust-rust"), func(ctx ginkgo.SpecContext) {
			requestCtx, cancel := context.WithTimeout(ctx, probeTimeout)
			defer cancel()

			output, err := runRustProbe(requestCtx, binaries, rustGRPCFixtureURL, "rust", rustProbeOptions{
				expectPushSupported: true,
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), output)
		})
	})
})
