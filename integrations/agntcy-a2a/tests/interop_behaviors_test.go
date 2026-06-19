// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

// This file contains the full explicit Ginkgo spec tree for all A2A interop behaviors,
// along with the assertion helpers shared across the native Go path and the external
// probe path. Every spec is a readable When/Context/It block — no matrix loops.
//
// registerBehaviors registers all specs against a probeClient factory and a server
// target. The containing Context must be Ordered so the outer BeforeAll (which
// creates the client) runs before any It block.

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/a2aproject/a2a-go/v2/a2a"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// ─── helpers ─────────────────────────────────────────────────────────────────

func newInteropRequest(text string, returnImmediately bool) *a2a.SendMessageRequest {
	return newInteropRequestWithIDs(text, returnImmediately, "", "")
}

func newInteropRequestWithIDs(text string, returnImmediately bool, taskID a2a.TaskID, contextID string) *a2a.SendMessageRequest {
	message := a2a.NewMessage(
		a2a.MessageRoleUser,
		a2a.NewTextPart(text),
		a2a.NewDataPart(map[string]any{
			"kind":  requestDataKind,
			"scope": requestDataScope,
		}),
	)
	if taskID != "" {
		message.TaskID = taskID
	}
	if contextID != "" {
		message.ContextID = contextID
	}
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

func firstMessageText(message *a2a.Message) (string, error) {
	for _, part := range message.Parts {
		if text := part.Text(); text != "" {
			return text, nil
		}
	}
	return "", errors.New("message did not include a text part")
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

func firstArtifactText(artifact *a2a.Artifact) (string, error) {
	if artifact == nil {
		return "", errors.New("artifact was nil")
	}
	for _, part := range artifact.Parts {
		if text := part.Text(); text != "" {
			return text, nil
		}
	}
	return "", errors.New("artifact did not include a text part")
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
	assertTaskHistoryPayloads(task, []string{expectedText}, kind)
}

func assertTaskHistoryPayloads(task *a2a.Task, expectedTexts []string, kind string) {
	gomega.Expect(task.History).To(gomega.HaveLen(len(expectedTexts)), kind)
	for index, expectedText := range expectedTexts {
		assertMessageInteropPayload(task.History[index], expectedText, kind)
	}
}

func assertTaskPushConfig(config *a2a.PushConfig, taskID a2a.TaskID, expected *a2a.PushConfig, kind string) {
	gomega.Expect(config).NotTo(gomega.BeNil(), kind)
	gomega.Expect(config.TaskID).To(gomega.Equal(taskID), kind)
	gomega.Expect(config.ID).To(gomega.Equal(expected.ID), kind)
	gomega.Expect(config.URL).To(gomega.Equal(expected.URL), kind)
	gomega.Expect(config.Token).To(gomega.Equal(expected.Token), kind)
	gomega.Expect(config.Auth).To(gomega.Equal(expected.Auth), kind)
}

func assertTaskNotFoundError(err error, kind string) {
	gomega.Expect(err).To(gomega.HaveOccurred(), kind)
	gomega.Expect(strings.ToLower(err.Error())).To(gomega.Or(
		gomega.ContainSubstring("task not found"),
		gomega.ContainSubstring("not found"),
	), kind)
}

func assertTaskNotCancelableError(err error, kind string) {
	gomega.Expect(err).To(gomega.HaveOccurred(), kind)
	gomega.Expect(strings.ToLower(err.Error())).To(gomega.Or(
		gomega.ContainSubstring("cancel"),
		gomega.ContainSubstring("terminal state"),
		gomega.ContainSubstring("not cancelable"),
	), kind)
}

func assertPushUnsupportedError(err error, kind string) {
	gomega.Expect(err).To(gomega.HaveOccurred(), kind)
	gomega.Expect(strings.ToLower(err.Error())).To(gomega.Or(
		gomega.ContainSubstring("push notification not supported"),
		gomega.ContainSubstring("push_notification_not_supported"),
		gomega.ContainSubstring("not supported"),
		gomega.ContainSubstring("unsupported"),
		gomega.ContainSubstring("unimplemented"),
		gomega.ContainSubstring("server error"),
	), kind)
}

func assertDataTypesTask(task *a2a.Task, kind string) {
	gomega.Expect(task.Artifacts).To(gomega.HaveLen(1), kind)
	artifact := task.Artifacts[0]
	gomega.Expect(artifact.Parts).To(gomega.HaveLen(3), kind)
	gomega.Expect(artifact.Parts[0].Text()).To(gomega.Equal("structured summary"), kind)
	dataPart, ok := artifact.Parts[1].Data().(map[string]any)
	gomega.Expect(ok).To(gomega.BeTrue(), kind)
	gomega.Expect(dataPart).To(gomega.HaveKeyWithValue("kind", "report"), kind)
	gomega.Expect(dataPart).To(gomega.HaveKeyWithValue("items", float64(2)), kind)
	gomega.Expect(string(artifact.Parts[2].URL())).To(gomega.Equal("https://example.invalid/diagram.svg"), kind)
	gomega.Expect(artifact.Parts[2].MediaType).To(gomega.Equal("image/svg+xml"), kind)
}

func assertExtendedCardMetadata(card *a2a.AgentCard, kind string) {
	gomega.Expect(card).NotTo(gomega.BeNil(), kind)
	gomega.Expect(card.Capabilities.ExtendedAgentCard).To(gomega.BeTrue(), kind)
	gomega.Expect(card.Description).To(gomega.ContainSubstring("(extended)"), kind)

	if len(card.SecuritySchemes) > 0 {
		scheme, ok := card.SecuritySchemes[extendedCardSchemeID]
		gomega.Expect(ok).To(gomega.BeTrue(), kind)
		httpScheme, ok := scheme.(a2a.HTTPAuthSecurityScheme)
		gomega.Expect(ok).To(gomega.BeTrue(), kind)
		gomega.Expect(httpScheme.Scheme).To(gomega.Equal("Bearer"), kind)
	}

	for _, expectedSkill := range expectedSkillIDs {
		gomega.Expect(card.Skills).To(gomega.ContainElement(gomega.HaveField("ID", expectedSkill)), kind)
	}
}

func waitForProbeTaskState(ctx context.Context, client probeClient, taskID a2a.TaskID, expectedState a2a.TaskState) (*a2a.Task, error) {
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		task, err := client.GetTask(ctx, &a2a.GetTaskRequest{ID: taskID})
		if err != nil {
			return nil, err
		}
		if task.Status.State == expectedState {
			return task, nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return nil, fmt.Errorf("timed out waiting for task %s to reach state %s", taskID, expectedState)
}

// ─── registerBehaviors ───────────────────────────────────────────────────────

// registerBehaviors registers the full interop spec tree against the given client factory
// and server starter. A fresh server and client are created for every spec via
// JustBeforeEach and torn down in AfterEach, preventing any state crosstalk between specs.
//
// enablePushSupport defaults to false in the outer BeforeEach; the push-enabled
// Context overrides it to true in its own BeforeEach so that JustBeforeEach
// starts the server with push support enabled for those specs.
func registerBehaviors(
	newClient    newClientFn,
	startServer  serverStarter,
	protocol     transportProtocol,
	serverPrefix string,
) {
	var (
		process           *fixtureProcess
		client            probeClient
		enablePushSupport bool
	)

	BeforeEach(func() {
		enablePushSupport = false
	})

	JustBeforeEach(func() {
		var srvURL string
		var err error
		process, srvURL, err = startServer(protocol, enablePushSupport)
		if errors.Is(err, ErrUnsupportedConfig) {
			Skip("transport is not supported by this server")
		}
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		client, err = newClient(srvURL, protocol)
		if errors.Is(err, ErrUnsupportedConfig) {
			Skip("transport is not supported by this client")
		}
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	AfterEach(func() { stopFixtureIfRunning(process) })

	// ── send message (unary) ─────────────────────────────────────────────────

	When("the client sends a message", func() {
		Context("and returnImmediately=false", func() {
			Context("and the server creates a task", func() {
				It("receives the expected completed Task", func(ctx SpecContext) {
					result, err := client.SendMessage(ctx, newInteropRequest(requestText, false))
					gomega.Expect(err).NotTo(gomega.HaveOccurred())
					task, ok := result.(*a2a.Task)
					gomega.Expect(ok).To(gomega.BeTrue(), "expected Task result from SendMessage")

					By("task status is completed")
					gomega.Expect(task.Status.State).To(gomega.Equal(a2a.TaskStateCompleted))

					By("task history contains the request payload")
					assertTaskHistoryPayload(task, requestText, "task history")

					By("task status message contains the server response text")
					text, err := taskStatusText(task)
					gomega.Expect(err).NotTo(gomega.HaveOccurred())
					gomega.Expect(text).To(gomega.Equal(expectedServerText(serverPrefix, requestText)))
				})
			})

			Context("and the server returns a message-only response", func() {
				It("the client receives a Message result", func(ctx SpecContext) {
					result, err := client.SendMessage(ctx, newInteropRequest(messageOnlyRequestText, false))
					gomega.Expect(err).NotTo(gomega.HaveOccurred())
					message, ok := result.(*a2a.Message)
					gomega.Expect(ok).To(gomega.BeTrue(), "expected Message result from SendMessage")
					text, err := firstMessageText(message)
					gomega.Expect(err).NotTo(gomega.HaveOccurred())
					gomega.Expect(text).To(gomega.Equal(fmt.Sprintf("%s server message-only response", serverPrefix)))
				})
			})

			Context("and the server fails the task", func() {
				It("the client receives a failed task", func(ctx SpecContext) {
					result, err := client.SendMessage(ctx, newInteropRequest(taskFailureRequestText, false))
					gomega.Expect(err).NotTo(gomega.HaveOccurred())
					task, ok := result.(*a2a.Task)
					gomega.Expect(ok).To(gomega.BeTrue())
					gomega.Expect(task.Status.State).To(gomega.Equal(a2a.TaskStateFailed))
					text, err := taskStatusText(task)
					gomega.Expect(err).NotTo(gomega.HaveOccurred())
					gomega.Expect(text).To(gomega.Equal(fmt.Sprintf("%s server failed task", serverPrefix)))
				})
			})

			Context("and the server returns rich data types", func() {
				It("the completed task includes structured artifact data", func(ctx SpecContext) {
					result, err := client.SendMessage(ctx, newInteropRequest(dataTypesRequestText, false))
					gomega.Expect(err).NotTo(gomega.HaveOccurred())
					task, ok := result.(*a2a.Task)
					gomega.Expect(ok).To(gomega.BeTrue())
					gomega.Expect(task.Status.State).To(gomega.Equal(a2a.TaskStateCompleted))
					text, err := taskStatusText(task)
					gomega.Expect(err).NotTo(gomega.HaveOccurred())
					gomega.Expect(text).To(gomega.Equal(fmt.Sprintf("%s server data-types ready", serverPrefix)))
					assertDataTypesTask(task, "data-types")
				})
			})

			Context("and the agent requires multi-turn input", func() {
				It("completes a multi-turn conversation", func(ctx SpecContext) {
					By("first turn returns input-required state")
					result, err := client.SendMessage(ctx, newInteropRequest(multiTurnStartRequestText, false))
					gomega.Expect(err).NotTo(gomega.HaveOccurred())
					inputRequiredTask, ok := result.(*a2a.Task)
					gomega.Expect(ok).To(gomega.BeTrue())
					gomega.Expect(inputRequiredTask.Status.State).To(gomega.Equal(a2a.TaskStateInputRequired))
					text, err := taskStatusText(inputRequiredTask)
					gomega.Expect(err).NotTo(gomega.HaveOccurred())
					gomega.Expect(text).To(gomega.Equal(fmt.Sprintf("%s server needs more input", serverPrefix)))
					assertTaskHistoryPayload(inputRequiredTask, multiTurnStartRequestText, "multi-turn start")

					By("second turn completes the task")
					continueResult, err := client.SendMessage(ctx, newInteropRequestWithIDs(
						multiTurnContinueRequestText, false,
						inputRequiredTask.ID, inputRequiredTask.ContextID,
					))
					gomega.Expect(err).NotTo(gomega.HaveOccurred())
					continuedTask, ok := continueResult.(*a2a.Task)
					gomega.Expect(ok).To(gomega.BeTrue())
					gomega.Expect(continuedTask.Status.State).To(gomega.Equal(a2a.TaskStateCompleted))
					text, err = taskStatusText(continuedTask)
					gomega.Expect(err).NotTo(gomega.HaveOccurred())
					gomega.Expect(text).To(gomega.Equal(fmt.Sprintf("%s server multi-turn completed", serverPrefix)))
					assertTaskHistoryPayloads(continuedTask,
						[]string{multiTurnStartRequestText, multiTurnContinueRequestText},
						"multi-turn continuation",
					)
				})
			})
		})

		Context("and returnImmediately=true", func() {
			Context("and the server creates a task", func() {
				It("receives a working task", func(ctx SpecContext) {
					result, err := client.SendMessage(ctx, newInteropRequest(pendingRequestText, true))
					gomega.Expect(err).NotTo(gomega.HaveOccurred())
					task, ok := result.(*a2a.Task)
					gomega.Expect(ok).To(gomega.BeTrue(), "expected Task result from SendMessage")
					gomega.Expect(task.Status.State).To(gomega.Equal(a2a.TaskStateWorking))
				})
			})

			Context("and the server handles a long-running request", func() {
				It("returns a completed long-running Task", func(ctx SpecContext) {
					result, err := client.SendMessage(ctx, newInteropRequest(longRunningRequestText, true))
					gomega.Expect(err).NotTo(gomega.HaveOccurred())
					task, ok := result.(*a2a.Task)
					gomega.Expect(ok).To(gomega.BeTrue())

					var completedTask *a2a.Task
					switch task.Status.State {
					case a2a.TaskStateWorking:
						By("polling until the task reaches completed state")
						startText, startTextErr := taskStatusText(task)
						gomega.Expect(startTextErr).NotTo(gomega.HaveOccurred())
						gomega.Expect(startText).To(gomega.Equal(fmt.Sprintf("%s server long-running started", serverPrefix)))
						completedTask, err = waitForProbeTaskState(ctx, client, task.ID, a2a.TaskStateCompleted)
						gomega.Expect(err).NotTo(gomega.HaveOccurred())
					case a2a.TaskStateCompleted:
						completedTask = task
					default:
						Fail(fmt.Sprintf("unexpected long-running task initial state: %s", task.Status.State))
					}

					text, err := taskStatusText(completedTask)
					gomega.Expect(err).NotTo(gomega.HaveOccurred())
					gomega.Expect(text).To(gomega.Equal(fmt.Sprintf("%s server long-running complete", serverPrefix)))
				})
			})
		})
	})

	// ── streaming send message ───────────────────────────────────────────────

	When("the client streams a message", func() {
		Context("and the server creates a task", func() {
			It("receives at least one event with the server response text", func(ctx SpecContext) {
				events, err := client.SendStreamingMessage(ctx, newInteropRequest(requestText, false))
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Expect(events).NotTo(gomega.BeEmpty())
				var texts []string
				for _, event := range events {
					text, hasText, err := eventText(event)
					gomega.Expect(err).NotTo(gomega.HaveOccurred())
					if hasText {
						texts = append(texts, text)
					}
				}
				gomega.Expect(texts).To(gomega.ContainElement(
					expectedServerText(serverPrefix, requestText),
				))
			})
		})

		Context("and the server streams artifacts", func() {
			It("emits the complete streaming sequence", func(ctx SpecContext) {
				streamingEvents, err := client.SendStreamingMessage(ctx, newInteropRequest(streamingRequestText, false))
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				By("a working task event at the start")
				sawStart := false
				for _, event := range streamingEvents {
					if task, ok := event.(*a2a.Task); ok {
						gomega.Expect(task.Status.State).To(gomega.Equal(a2a.TaskStateWorking))
						text, err := taskStatusText(task)
						gomega.Expect(err).NotTo(gomega.HaveOccurred())
						gomega.Expect(text).To(gomega.Equal(fmt.Sprintf("%s server streaming started", serverPrefix)))
						sawStart = true
						break
					}
				}
				gomega.Expect(sawStart).To(gomega.BeTrue(), "expected a working Task streaming-start event")

				By("artifact chunks emitted in order with at least one append")
				var chunks []string
				sawAppend := false
				for _, event := range streamingEvents {
					if update, ok := event.(*a2a.TaskArtifactUpdateEvent); ok {
						text, err := firstArtifactText(update.Artifact)
						gomega.Expect(err).NotTo(gomega.HaveOccurred())
						chunks = append(chunks, text)
						if update.Append {
							sawAppend = true
						}
					}
				}
				gomega.Expect(sawAppend).To(gomega.BeTrue(), "expected an append artifact update")
				gomega.Expect(chunks).To(gomega.Equal([]string{"streaming chunk 1", "streaming chunk 2"}))

				By("a completed status update at the end")
				sawComplete := false
				for _, event := range streamingEvents {
					if update, ok := event.(*a2a.TaskStatusUpdateEvent); ok {
						gomega.Expect(update.Status.State).To(gomega.Equal(a2a.TaskStateCompleted))
						text, err := firstMessageText(update.Status.Message)
						gomega.Expect(err).NotTo(gomega.HaveOccurred())
						gomega.Expect(text).To(gomega.Equal(fmt.Sprintf("%s server streaming complete", serverPrefix)))
						sawComplete = true
					}
				}
				gomega.Expect(sawComplete).To(gomega.BeTrue(), "expected a completed TaskStatusUpdateEvent")
			})
		})
	})

	// ── get task ─────────────────────────────────────────────────────────────

	When("the client gets a task by ID", func() {
		Context("and the task exists", func() {
			It("returns the completed task with full response details", func(ctx SpecContext) {
				result, err := client.SendMessage(ctx, newInteropRequest(requestText, false))
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				completedTask, ok := result.(*a2a.Task)
				gomega.Expect(ok).To(gomega.BeTrue())

				fetchedTask, err := client.GetTask(ctx, &a2a.GetTaskRequest{ID: completedTask.ID})
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				By("task status is completed")
				gomega.Expect(fetchedTask.Status.State).To(gomega.Equal(a2a.TaskStateCompleted))

				By("response text matches the server echo")
				text, err := taskStatusText(fetchedTask)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Expect(text).To(gomega.Equal(expectedServerText(serverPrefix, requestText)))

				By("history contains the request payload")
				assertTaskHistoryPayload(fetchedTask, requestText, "fetched task history")
			})
		})

		Context("and the task does not exist", func() {
			It("returns a not-found error", func(ctx SpecContext) {
				_, err := client.GetTask(ctx, &a2a.GetTaskRequest{ID: a2a.NewTaskID()})
				assertTaskNotFoundError(err, "missing task")
			})
		})
	})

	// ── list tasks ───────────────────────────────────────────────────────────

	When("the client lists tasks by context ID", func() {
		It("includes the sent task in the list", func(ctx SpecContext) {
			result, err := client.SendMessage(ctx, newInteropRequest(requestText, false))
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			completedTask, ok := result.(*a2a.Task)
			gomega.Expect(ok).To(gomega.BeTrue())

			listedTasks, err := client.ListTasks(ctx, &a2a.ListTasksRequest{ContextID: completedTask.ContextID})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(listedTasks.Tasks).NotTo(gomega.BeEmpty())
			gomega.Expect(listedTasks.Tasks).To(gomega.ContainElement(gomega.HaveField("ID", completedTask.ID)))
		})
	})

	// ── cancel task ──────────────────────────────────────────────────────────

	When("the client cancels a task", func() {
		Context("and the task is working", func() {
			It("cancels the working task and the canceled state persists", func(ctx SpecContext) {
				result, err := client.SendMessage(ctx, newInteropRequest(pendingRequestText, true))
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				workingTask, ok := result.(*a2a.Task)
				gomega.Expect(ok).To(gomega.BeTrue())
				gomega.Expect(workingTask.Status.State).To(gomega.Equal(a2a.TaskStateWorking))

				canceledTask, err := client.CancelTask(ctx, &a2a.CancelTaskRequest{ID: workingTask.ID})
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				fetchedCanceledTask, err := client.GetTask(ctx, &a2a.GetTaskRequest{ID: workingTask.ID})
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				By("cancel response returns a canceled task")
				gomega.Expect(canceledTask.Status.State).To(gomega.Equal(a2a.TaskStateCanceled))

				By("cancel response includes the server cancel text")
				text, err := taskStatusText(canceledTask)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Expect(text).To(gomega.Equal(expectedCancelText(serverPrefix)))

				By("fetched task confirms the canceled state persists")
				gomega.Expect(fetchedCanceledTask.Status.State).To(gomega.Equal(a2a.TaskStateCanceled))
			})
		})

		Context("and the task is already completed", func() {
			It("returns an error when canceling a completed task", func(ctx SpecContext) {
				result, err := client.SendMessage(ctx, newInteropRequest(requestText, false))
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				completedTask, ok := result.(*a2a.Task)
				gomega.Expect(ok).To(gomega.BeTrue())

				_, err = client.CancelTask(ctx, &a2a.CancelTaskRequest{ID: completedTask.ID})
				assertTaskNotCancelableError(err, "canceling a terminal task should fail")
			})
		})
	})

	// ── push notification config ─────────────────────────────────────────────

	When("the client manages push notification config", func() {
		// Push-enabled context: JustBeforeEach starts the server with push enabled.
		// If startServer returns ErrUnsupportedConfig (e.g. .NET), the spec is skipped.
		Context("and the server supports push notifications", func() {
			BeforeEach(func() {
				enablePushSupport = true
			})

			It("manages the push config lifecycle", func(ctx SpecContext) {
				result, err := client.SendMessage(ctx, newInteropRequest(requestText, false))
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				completedTask, ok := result.(*a2a.Task)
				gomega.Expect(ok).To(gomega.BeTrue())
				gomega.Expect(completedTask.Status.State).To(gomega.Equal(a2a.TaskStateCompleted))

				cfg := newInteropPushConfig()
				cfg.TaskID = completedTask.ID

				By("creating a push config")
				createdConfig, err := client.CreatePushConfig(ctx, cfg)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				assertTaskPushConfig(createdConfig, completedTask.ID, newInteropPushConfig(), "created push config")

				By("fetching the push config by ID")
				fetched, err := client.GetPushConfig(ctx, &a2a.GetTaskPushConfigRequest{
					TaskID: completedTask.ID,
					ID:     createdConfig.ID,
				})
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				assertTaskPushConfig(fetched, completedTask.ID, newInteropPushConfig(), "fetched push config")

				By("listing exactly one push config for the task")
				listed, err := client.ListPushConfigs(ctx, &a2a.ListTaskPushConfigRequest{TaskID: completedTask.ID})
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Expect(listed).To(gomega.HaveLen(1))
				assertTaskPushConfig(listed[0], completedTask.ID, newInteropPushConfig(), "listed push config")

				By("deleting the push config and confirming the list is empty")
				err = client.DeletePushConfig(ctx, &a2a.DeleteTaskPushConfigRequest{
					TaskID: completedTask.ID,
					ID:     createdConfig.ID,
				})
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				listed, err = client.ListPushConfigs(ctx, &a2a.ListTaskPushConfigRequest{TaskID: completedTask.ID})
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Expect(listed).To(gomega.BeEmpty())
			})
		})

		// Push-disabled context: server started without push support (works for all 4 languages).
		Context("and the server does not support push notifications", func() {
			BeforeEach(func() {
				enablePushSupport = false
			})

			It("rejects all push config operations", func(ctx SpecContext) {
				result, err := client.SendMessage(ctx, newInteropRequest(requestText, false))
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				completedTask, ok := result.(*a2a.Task)
				gomega.Expect(ok).To(gomega.BeTrue())

				By("CreatePushConfig returns an unsupported error")
				cfg := newInteropPushConfig()
				cfg.TaskID = completedTask.ID
				_, err = client.CreatePushConfig(ctx, cfg)
				assertPushUnsupportedError(err, "create push config unsupported")

				By("GetPushConfig returns an unsupported error")
				_, err = client.GetPushConfig(ctx, &a2a.GetTaskPushConfigRequest{
					TaskID: completedTask.ID,
					ID:     "interop-config",
				})
				assertPushUnsupportedError(err, "get push config unsupported")

				By("ListPushConfigs returns an unsupported error")
				_, err = client.ListPushConfigs(ctx, &a2a.ListTaskPushConfigRequest{TaskID: completedTask.ID})
				assertPushUnsupportedError(err, "list push config unsupported")

				By("DeletePushConfig returns an unsupported error")
				err = client.DeletePushConfig(ctx, &a2a.DeleteTaskPushConfigRequest{
					TaskID: completedTask.ID,
					ID:     "interop-config",
				})
				assertPushUnsupportedError(err, "delete push config unsupported")
			})
		})
	})

	// ── extended agent card ──────────────────────────────────────────────────

	When("the client fetches the extended agent card", func() {
		It("returns a valid extended card", func(ctx SpecContext) {
			card, err := client.GetExtendedCard(ctx, &a2a.GetExtendedAgentCardRequest{})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			assertExtendedCardMetadata(card, "extended-card")
		})
	})
}
