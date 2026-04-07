// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

use std::env;
use std::process;
use std::sync::Arc;
use std::time::Duration;

use a2a::*;
use a2a_client::A2AClientFactory;
use a2a_client::agent_card::AgentCardResolver;
use a2a_grpc::GrpcTransportFactory;
use futures::StreamExt;
use futures::stream::BoxStream;
use serde_json::{Value, json};
use tokio::time::sleep;

const REQUEST_TEXT: &str = "ping";
const PENDING_REQUEST_TEXT: &str = "pending";
const MESSAGE_ONLY_REQUEST_TEXT: &str = "message-only";
const TASK_FAILURE_REQUEST_TEXT: &str = "task-failure";
const MULTI_TURN_START_REQUEST_TEXT: &str = "multi-turn start";
const MULTI_TURN_CONTINUE_REQUEST_TEXT: &str = "multi-turn continue";
const STREAMING_REQUEST_TEXT: &str = "streaming";
const LONG_RUNNING_REQUEST_TEXT: &str = "long-running";
const DATA_TYPES_REQUEST_TEXT: &str = "data-types";
const REQUEST_DATA_KIND: &str = "structured";
const REQUEST_DATA_SCOPE: &str = "interop";
const REQUEST_METADATA_KEY: &str = "csit";
const REQUEST_METADATA_VALUE: &str = "multipart";
const EXTENDED_CARD_SCHEME_ID: &str = "bearer_token";
const EXPECTED_SKILL_IDS: &[&str] = &[
    "message-only",
    "task-lifecycle",
    "task-failure",
    "task-cancel",
    "multi-turn",
    "streaming",
    "long-running",
    "data-types",
];

struct Args {
    card_url: String,
    server_prefix: String,
    expect_subscribe_unsupported: bool,
    expect_push_supported: bool,
    expect_push_unsupported: bool,
    relaxed_error_checks: bool,
    expected_push_error_code: i32,
}

fn parse_args() -> Result<Args, String> {
    let mut args = env::args().skip(1);
    let mut card_url = None;
    let mut server_prefix = None;
    let mut expect_subscribe_unsupported = false;
    let mut expect_push_supported = false;
    let mut expect_push_unsupported = false;
    let mut relaxed_error_checks = false;
    let mut expected_push_error_code = a2a::error_code::PUSH_NOTIFICATION_NOT_SUPPORTED;

    while let Some(arg) = args.next() {
        match arg.as_str() {
            "--card-url" => {
                card_url = Some(
                    args.next()
                        .ok_or_else(|| "--card-url requires a value".to_string())?,
                );
            }
            "--server-prefix" => {
                server_prefix = Some(
                    args.next()
                        .ok_or_else(|| "--server-prefix requires a value".to_string())?,
                );
            }
            "--expect-subscribe-unsupported" => {
                expect_subscribe_unsupported = true;
            }
            "--expect-push-supported" => {
                expect_push_supported = true;
            }
            "--expect-push-unsupported" => {
                expect_push_unsupported = true;
            }
            "--expected-push-error-code" => {
                let value = args
                    .next()
                    .ok_or_else(|| "--expected-push-error-code requires a value".to_string())?;
                expected_push_error_code = value
                    .parse::<i32>()
                    .map_err(|_| "--expected-push-error-code must be a valid i32".to_string())?;
            }
            "--relaxed-error-checks" => {
                relaxed_error_checks = true;
            }
            other => {
                return Err(format!("unknown argument: {other}"));
            }
        }
    }

    if expect_push_supported && expect_push_unsupported {
        return Err(
            "--expect-push-supported and --expect-push-unsupported are mutually exclusive"
                .to_string(),
        );
    }

    Ok(Args {
        card_url: card_url.ok_or_else(|| "missing --card-url".to_string())?,
        server_prefix: server_prefix.ok_or_else(|| "missing --server-prefix".to_string())?,
        expect_subscribe_unsupported,
        expect_push_supported,
        expect_push_unsupported,
        relaxed_error_checks,
        expected_push_error_code,
    })
}

fn first_text(message: &Message) -> Result<String, String> {
    message
        .parts
        .iter()
        .find_map(Part::as_text)
        .map(ToString::to_string)
        .ok_or_else(|| "message contained no text parts".to_string())
}

fn assert_text(actual: String, expected: &str, kind: &str) -> Result<(), String> {
    if actual == expected {
        Ok(())
    } else {
        Err(format!(
            "unexpected {kind} response text: got {actual:?}, want {expected:?}"
        ))
    }
}

fn assert_state(actual: &TaskState, expected: TaskState, kind: &str) -> Result<(), String> {
    if *actual == expected {
        Ok(())
    } else {
        Err(format!(
            "unexpected {kind} task state: got {actual:?}, want {expected:?}"
        ))
    }
}

fn assert_error_code<T>(
    result: Result<T, A2AError>,
    expected_code: i32,
    kind: &str,
) -> Result<(), String> {
    match result {
        Ok(_) => Err(format!(
            "expected {kind} to fail with code {expected_code}, but it succeeded"
        )),
        Err(error) if error.code == expected_code => Ok(()),
        Err(error) => Err(format!(
            "unexpected {kind} error code: got {}, want {} ({error})",
            error.code, expected_code
        )),
    }
}

fn assert_failed<T>(result: Result<T, A2AError>, kind: &str) -> Result<(), String> {
    match result {
        Ok(_) => Err(format!("expected {kind} to fail, but it succeeded")),
        Err(_) => Ok(()),
    }
}

fn assert_push_config(
    actual: &TaskPushNotificationConfig,
    task_id: &str,
    expected: &PushNotificationConfig,
    kind: &str,
) -> Result<(), String> {
    if actual.task_id != task_id {
        return Err(format!(
            "unexpected {kind} task id: got {:?}, want {:?}",
            actual.task_id, task_id
        ));
    }

    if actual.config != *expected {
        return Err(format!(
            "unexpected {kind} push config: got {:?}, want {:?}",
            actual.config, expected
        ));
    }

    Ok(())
}

async fn assert_stream_error_code(
    result: Result<BoxStream<'static, Result<StreamResponse, A2AError>>, A2AError>,
    expected_code: i32,
    kind: &str,
) -> Result<(), String> {
    match result {
        Err(error) if error.code == expected_code => Ok(()),
        Err(error) => Err(format!(
            "unexpected {kind} error code: got {}, want {} ({error})",
            error.code, expected_code
        )),
        Ok(mut stream) => match stream.next().await {
            Some(Err(error)) if error.code == expected_code => Ok(()),
            Some(Err(error)) => Err(format!(
                "unexpected {kind} stream error code: got {}, want {} ({error})",
                error.code, expected_code
            )),
            Some(Ok(_)) => Err(format!(
                "expected {kind} to fail with code {expected_code}, but it yielded an event"
            )),
            None => Err(format!(
                "expected {kind} to fail with code {expected_code}, but the stream ended cleanly"
            )),
        },
    }
}

fn expected_response_text(server_prefix: &str, request_text: &str) -> String {
    format!("{server_prefix} server received: {request_text}")
}

fn expected_cancel_text(server_prefix: &str) -> String {
    format!("{server_prefix} server canceled task")
}

fn expected_scenario_text(server_prefix: &str, suffix: &str) -> String {
    format!("{server_prefix} server {suffix}")
}

fn request_with_payload(text: &str, return_immediately: bool) -> SendMessageRequest {
    request_with_payload_ids(text, return_immediately, None, None)
}

fn request_with_payload_ids(
    text: &str,
    return_immediately: bool,
    task_id: Option<String>,
    context_id: Option<String>,
) -> SendMessageRequest {
    let mut message = Message::new(
        Role::User,
        vec![
            Part::text(text),
            Part::data(json!({
                "kind": REQUEST_DATA_KIND,
                "scope": REQUEST_DATA_SCOPE,
            })),
        ],
    );
    message.task_id = task_id;
    message.context_id = context_id;
    message.metadata = Some(std::collections::HashMap::from([(
        REQUEST_METADATA_KEY.to_string(),
        json!(REQUEST_METADATA_VALUE),
    )]));

    SendMessageRequest {
        message,
        configuration: Some(SendMessageConfiguration {
            accepted_output_modes: None,
            push_notification_config: None,
            history_length: None,
            return_immediately: Some(return_immediately),
        }),
        metadata: None,
        tenant: None,
    }
}

fn task_text(task: &Task) -> Result<String, String> {
    task.status
        .message
        .as_ref()
        .ok_or_else(|| "task response contained no message".to_string())
        .and_then(first_text)
}

fn task_from_response(response: SendMessageResponse, kind: &str) -> Result<Task, String> {
    match response {
        SendMessageResponse::Task(task) => Ok(task),
        SendMessageResponse::Message(_) => Err(format!("unexpected {kind} response type: Message")),
    }
}

fn message_from_response(response: SendMessageResponse, kind: &str) -> Result<Message, String> {
    match response {
        SendMessageResponse::Task(_) => Err(format!("unexpected {kind} response type: Task")),
        SendMessageResponse::Message(message) => Ok(message),
    }
}

fn stream_response_text(response: StreamResponse) -> Result<Option<String>, String> {
    match response {
        StreamResponse::Message(message) => first_text(&message).map(Some),
        StreamResponse::Task(task) => task_text(&task).map(Some),
        StreamResponse::StatusUpdate(update) => {
            update.status.message.as_ref().map(first_text).transpose()
        }
        StreamResponse::ArtifactUpdate(_) => Ok(None),
    }
}

fn assert_message_payload(message: &Message, expected_text: &str, kind: &str) -> Result<(), String> {
    if message.parts.len() != 2 {
        return Err(format!(
            "{kind} message had {} parts, want 2",
            message.parts.len()
        ));
    }

    let actual = first_text(message)?;
    assert_text(actual, expected_text, kind)?;

    let part_data = match &message.parts[1].content {
        PartContent::Data(value) => value,
        _ => {
            return Err(format!(
                "{kind} message second part was not a structured data part"
            ));
        }
    };
    let kind_value = part_data
        .get("kind")
        .and_then(Value::as_str)
        .ok_or_else(|| format!("{kind} message data part was missing kind"))?;
    let scope_value = part_data
        .get("scope")
        .and_then(Value::as_str)
        .ok_or_else(|| format!("{kind} message data part was missing scope"))?;

    if kind_value != REQUEST_DATA_KIND || scope_value != REQUEST_DATA_SCOPE {
        return Err(format!(
            "{kind} message data part mismatch: got kind={kind_value:?} scope={scope_value:?}"
        ));
    }

    let metadata = message
        .metadata
        .as_ref()
        .ok_or_else(|| format!("{kind} message was missing metadata"))?;
    let metadata_value = metadata
        .get(REQUEST_METADATA_KEY)
        .and_then(Value::as_str)
        .ok_or_else(|| format!("{kind} message metadata was missing {REQUEST_METADATA_KEY}"))?;

    if metadata_value != REQUEST_METADATA_VALUE {
        return Err(format!(
            "{kind} message metadata mismatch: got {metadata_value:?}, want {REQUEST_METADATA_VALUE:?}"
        ));
    }

    Ok(())
}

fn assert_task_history(task: &Task, expected_text: &str, kind: &str) -> Result<(), String> {
    assert_task_history_entries(task, &[expected_text], kind)
}

fn assert_task_history_entries(task: &Task, expected_texts: &[&str], kind: &str) -> Result<(), String> {
    let history = task
        .history
        .as_ref()
        .ok_or_else(|| format!("{kind} task did not include history"))?;
    if history.len() != expected_texts.len() {
        return Err(format!(
            "{kind} task history length mismatch: got {}, want {}",
            history.len(),
            expected_texts.len()
        ));
    }

    for (message, expected_text) in history.iter().zip(expected_texts.iter()) {
        assert_message_payload(message, expected_text, kind)?;
    }

    Ok(())
}

fn first_artifact_text(artifact: &Artifact) -> Result<String, String> {
    artifact
        .parts
        .iter()
        .find_map(Part::as_text)
        .map(ToString::to_string)
        .ok_or_else(|| "artifact contained no text parts".to_string())
}

fn assert_data_types_task(task: &Task, kind: &str) -> Result<(), String> {
    let artifacts = task
        .artifacts
        .as_ref()
        .ok_or_else(|| format!("{kind} task did not include artifacts"))?;
    if artifacts.len() != 1 {
        return Err(format!(
            "{kind} task artifact count mismatch: got {}, want 1",
            artifacts.len()
        ));
    }

    let artifact = &artifacts[0];
    if artifact.parts.len() != 3 {
        return Err(format!(
            "{kind} artifact part count mismatch: got {}, want 3",
            artifact.parts.len()
        ));
    }

    assert_text(first_artifact_text(artifact)?, "structured summary", kind)?;

    match &artifact.parts[1].content {
        PartContent::Data(value) => {
            if value.get("kind").and_then(Value::as_str) != Some("report") {
                return Err(format!("{kind} data-types artifact was missing kind=report"));
            }
            let items_matches = value
                .get("items")
                .map(|items| {
                    items.as_i64() == Some(2)
                        || items.as_u64() == Some(2)
                        || items.as_f64() == Some(2.0)
                })
                .unwrap_or(false);
            if !items_matches {
                return Err(format!("{kind} data-types artifact was missing items=2"));
            }
        }
        _ => return Err(format!("{kind} data-types artifact second part was not data")),
    }

    match &artifact.parts[2].content {
        PartContent::Url(url) => {
            if url != "https://example.invalid/diagram.svg" {
                return Err(format!("{kind} data-types artifact URL mismatch: got {url:?}"));
            }
        }
        _ => return Err(format!("{kind} data-types artifact third part was not a URL")),
    }

    if artifact.parts[2].media_type.as_deref() != Some("image/svg+xml") {
        return Err(format!("{kind} data-types artifact media type mismatch"));
    }
    if artifact.parts[2].filename.as_deref() != Some("diagram.svg") {
        return Err(format!("{kind} data-types artifact filename mismatch"));
    }

    Ok(())
}

fn assert_extended_card_metadata(card: &AgentCard, kind: &str) -> Result<(), String> {
    if card.capabilities.extended_agent_card != Some(true) {
        return Err(format!("{kind} card did not advertise extendedAgentCard support"));
    }
    if !card.description.contains("(extended)") {
        return Err(format!("{kind} card did not include extended description metadata"));
    }

    if let Some(schemes) = card.security_schemes.as_ref() {
        let scheme = schemes
            .get(EXTENDED_CARD_SCHEME_ID)
            .ok_or_else(|| format!("{kind} card did not include {EXTENDED_CARD_SCHEME_ID}"))?;

        match scheme {
            SecurityScheme::HttpAuth(http) => {
                if http.scheme != "Bearer" {
                    return Err(format!("{kind} bearer scheme mismatch: got {:?}", http.scheme));
                }
            }
            _ => return Err(format!("{kind} card security scheme was not HTTP auth")),
        }
    }

    for expected_skill in EXPECTED_SKILL_IDS {
        if !card.skills.iter().any(|skill| skill.id == *expected_skill) {
            return Err(format!("{kind} card was missing skill {expected_skill}"));
        }
    }

    Ok(())
}

async fn wait_for_task_state(
    client: &a2a_client::client::A2AClient,
    task_id: &str,
    expected_state: TaskState,
    kind: &str,
) -> Result<Task, String> {
    for _ in 0..40 {
        let task = client
            .get_task(&GetTaskRequest {
                id: task_id.to_string(),
                history_length: None,
                tenant: None,
            })
            .await
            .map_err(|error| format!("{kind} get_task failed: {error}"))?;

        if task.status.state == expected_state {
            return Ok(task);
        }

        sleep(Duration::from_millis(50)).await;
    }

    Err(format!("timed out waiting for {kind} to reach state {expected_state:?}"))
}

#[tokio::main]
async fn main() {
    let args = match parse_args() {
        Ok(args) => args,
        Err(error) => {
            eprintln!("{error}");
            process::exit(2);
        }
    };

    if let Err(error) = run(args).await {
        eprintln!("{error}");
        process::exit(1);
    }
}

async fn run(args: Args) -> Result<(), String> {
    let resolver = AgentCardResolver::new(None);
    let card = resolver
        .resolve(&args.card_url)
        .await
        .map_err(|error| format!("agent card resolution failed: {error}"))?;
    let client = A2AClientFactory::builder()
        .register(Arc::new(GrpcTransportFactory))
        .preferred_bindings(vec![
            TRANSPORT_PROTOCOL_GRPC.to_string(),
            TRANSPORT_PROTOCOL_JSONRPC.to_string(),
            TRANSPORT_PROTOCOL_HTTP_JSON.to_string(),
        ])
        .build()
        .create_from_card(&card)
        .await
        .map_err(|error| format!("client creation failed: {error}"))?;

    let expected_ping_text = expected_response_text(&args.server_prefix, REQUEST_TEXT);
    let expected_pending_text = expected_response_text(&args.server_prefix, PENDING_REQUEST_TEXT);
    let expected_cancel_text = expected_cancel_text(&args.server_prefix);
    let expected_message_only_text = expected_scenario_text(&args.server_prefix, "message-only response");
    let expected_failed_text = expected_scenario_text(&args.server_prefix, "failed task");
    let expected_input_required_text = expected_scenario_text(&args.server_prefix, "needs more input");
    let expected_multi_turn_complete_text = expected_scenario_text(&args.server_prefix, "multi-turn completed");
    let expected_streaming_start_text = expected_scenario_text(&args.server_prefix, "streaming started");
    let expected_streaming_complete_text = expected_scenario_text(&args.server_prefix, "streaming complete");
    let expected_long_running_start_text = expected_scenario_text(&args.server_prefix, "long-running started");
    let expected_long_running_complete_text = expected_scenario_text(&args.server_prefix, "long-running complete");
    let expected_data_types_text = expected_scenario_text(&args.server_prefix, "data-types ready");

    let request = request_with_payload(REQUEST_TEXT, false);

    let response = client
        .send_message(&request)
        .await
        .map_err(|error| format!("unary request failed: {error}"))?;
    let completed_task = task_from_response(response, "unary")?;
    assert_state(&completed_task.status.state, TaskState::Completed, "unary")?;
    assert_text(task_text(&completed_task)?, &expected_ping_text, "unary")?;
    assert_task_history(&completed_task, REQUEST_TEXT, "unary")?;

    let fetched_task = client
        .get_task(&GetTaskRequest {
            id: completed_task.id.clone(),
            history_length: Some(1),
            tenant: None,
        })
        .await
        .map_err(|error| format!("get_task failed: {error}"))?;
    assert_state(&fetched_task.status.state, TaskState::Completed, "get_task")?;
    assert_text(task_text(&fetched_task)?, &expected_ping_text, "get_task")?;
    assert_task_history(&fetched_task, REQUEST_TEXT, "get_task")?;

    let listed_tasks = client
        .list_tasks(&ListTasksRequest {
            context_id: Some(completed_task.context_id.clone()),
            status: None,
            page_size: None,
            page_token: None,
            history_length: None,
            status_timestamp_after: None,
            include_artifacts: None,
            tenant: None,
        })
        .await
        .map_err(|error| format!("list_tasks failed: {error}"))?;
    if !listed_tasks
        .tasks
        .iter()
        .any(|task| task.id == completed_task.id)
    {
        return Err(format!(
            "list_tasks did not include expected task {}",
            completed_task.id
        ));
    }

    let mut stream = client
        .send_streaming_message(&request)
        .await
        .map_err(|error| format!("streaming request failed: {error}"))?;

    let streaming_text = loop {
        match stream.next().await {
            Some(Ok(event)) => match stream_response_text(event)? {
                Some(text) => break text,
                None => continue,
            },
            Some(Err(error)) => {
                return Err(format!("streaming event failed: {error}"));
            }
            None => {
                return Err("stream completed without a terminal response event".to_string());
            }
        }
    };
    assert_text(streaming_text, &expected_ping_text, "streaming")?;

    let pending_task = task_from_response(
        client
            .send_message(&request_with_payload(PENDING_REQUEST_TEXT, true))
            .await
            .map_err(|error| format!("pending unary request failed: {error}"))?,
        "pending unary",
    )?;
    assert_state(
        &pending_task.status.state,
        TaskState::Working,
        "pending unary",
    )?;
    assert_text(
        task_text(&pending_task)?,
        &expected_pending_text,
        "pending unary",
    )?;

    let canceled_task = client
        .cancel_task(&CancelTaskRequest {
            id: pending_task.id.clone(),
            metadata: None,
            tenant: None,
        })
        .await
        .map_err(|error| format!("cancel_task failed: {error}"))?;
    assert_state(
        &canceled_task.status.state,
        TaskState::Canceled,
        "cancel_task",
    )?;
    assert_text(
        task_text(&canceled_task)?,
        &expected_cancel_text,
        "cancel_task",
    )?;

    let fetched_canceled_task = client
        .get_task(&GetTaskRequest {
            id: pending_task.id.clone(),
            history_length: None,
            tenant: None,
        })
        .await
        .map_err(|error| format!("get_task after cancel failed: {error}"))?;
    assert_state(
        &fetched_canceled_task.status.state,
        TaskState::Canceled,
        "get_task after cancel",
    )?;
    assert_text(
        task_text(&fetched_canceled_task)?,
        &expected_cancel_text,
        "get_task after cancel",
    )?;

    if args.relaxed_error_checks {
        assert_failed(
            client
                .get_task(&GetTaskRequest {
                    id: new_task_id(),
                    history_length: None,
                    tenant: None,
                })
                .await,
            "get missing task",
        )?;

        assert_failed(
            client
                .cancel_task(&CancelTaskRequest {
                    id: completed_task.id.clone(),
                    metadata: None,
                    tenant: None,
                })
                .await,
            "cancel completed task",
        )?;
    } else {
        assert_error_code(
            client
                .get_task(&GetTaskRequest {
                    id: new_task_id(),
                    history_length: None,
                    tenant: None,
                })
                .await,
            a2a::error_code::TASK_NOT_FOUND,
            "get missing task",
        )?;

        assert_error_code(
            client
                .cancel_task(&CancelTaskRequest {
                    id: completed_task.id.clone(),
                    metadata: None,
                    tenant: None,
                })
                .await,
            a2a::error_code::TASK_NOT_CANCELABLE,
            "cancel completed task",
        )?;
    }

    if args.expect_subscribe_unsupported {
        assert_stream_error_code(
            client
                .subscribe_to_task(&SubscribeToTaskRequest {
                    id: completed_task.id.clone(),
                    tenant: None,
                })
                .await,
            a2a::error_code::UNSUPPORTED_OPERATION,
            "subscribe_to_task",
        )
        .await?;
    }

    if args.expect_push_unsupported {
        let push_config = PushNotificationConfig {
            url: "https://example.invalid/webhook".to_string(),
            id: Some("interop-config".to_string()),
            token: None,
            authentication: None,
        };

        if args.relaxed_error_checks {
            assert_failed(
                client
                    .create_push_config(&CreateTaskPushNotificationConfigRequest {
                        task_id: completed_task.id.clone(),
                        config: push_config.clone(),
                        tenant: None,
                    })
                    .await,
                "create_push_config",
            )?;

            assert_failed(
                client
                    .get_push_config(&GetTaskPushNotificationConfigRequest {
                        task_id: completed_task.id.clone(),
                        id: "interop-config".to_string(),
                        tenant: None,
                    })
                    .await,
                "get_push_config",
            )?;

            assert_failed(
                client
                    .list_push_configs(&ListTaskPushNotificationConfigsRequest {
                        task_id: completed_task.id.clone(),
                        page_size: None,
                        page_token: None,
                        tenant: None,
                    })
                    .await,
                "list_push_configs",
            )?;

            assert_failed(
                client
                    .delete_push_config(&DeleteTaskPushNotificationConfigRequest {
                        task_id: completed_task.id.clone(),
                        id: "interop-config".to_string(),
                        tenant: None,
                    })
                    .await,
                "delete_push_config",
            )?;
        } else {
            assert_error_code(
                client
                    .create_push_config(&CreateTaskPushNotificationConfigRequest {
                        task_id: completed_task.id.clone(),
                        config: push_config.clone(),
                        tenant: None,
                    })
                    .await,
                args.expected_push_error_code,
                "create_push_config",
            )?;

            assert_error_code(
                client
                    .get_push_config(&GetTaskPushNotificationConfigRequest {
                        task_id: completed_task.id.clone(),
                        id: "interop-config".to_string(),
                        tenant: None,
                    })
                    .await,
                args.expected_push_error_code,
                "get_push_config",
            )?;

            assert_error_code(
                client
                    .list_push_configs(&ListTaskPushNotificationConfigsRequest {
                        task_id: completed_task.id.clone(),
                        page_size: None,
                        page_token: None,
                        tenant: None,
                    })
                    .await,
                args.expected_push_error_code,
                "list_push_configs",
            )?;

            assert_error_code(
                client
                    .delete_push_config(&DeleteTaskPushNotificationConfigRequest {
                        task_id: completed_task.id.clone(),
                        id: "interop-config".to_string(),
                        tenant: None,
                    })
                    .await,
                args.expected_push_error_code,
                "delete_push_config",
            )?;
        }
    } else if args.expect_push_supported {
        let push_config = PushNotificationConfig {
            url: "https://example.invalid/webhook".to_string(),
            id: Some("interop-config".to_string()),
            token: Some("interop-token".to_string()),
            authentication: Some(AuthenticationInfo {
                scheme: "Bearer".to_string(),
                credentials: Some("interop-credential".to_string()),
            }),
        };

        let created_push_config = client
            .create_push_config(&CreateTaskPushNotificationConfigRequest {
                task_id: completed_task.id.clone(),
                config: push_config.clone(),
                tenant: None,
            })
            .await
            .map_err(|error| format!("create_push_config failed: {error}"))?;
        assert_push_config(
            &created_push_config,
            &completed_task.id,
            &push_config,
            "create_push_config",
        )?;

        let fetched_push_config = client
            .get_push_config(&GetTaskPushNotificationConfigRequest {
                task_id: completed_task.id.clone(),
                id: "interop-config".to_string(),
                tenant: None,
            })
            .await
            .map_err(|error| format!("get_push_config failed: {error}"))?;
        assert_push_config(
            &fetched_push_config,
            &completed_task.id,
            &push_config,
            "get_push_config",
        )?;

        let listed_push_configs = client
            .list_push_configs(&ListTaskPushNotificationConfigsRequest {
                task_id: completed_task.id.clone(),
                page_size: None,
                page_token: None,
                tenant: None,
            })
            .await
            .map_err(|error| format!("list_push_configs failed: {error}"))?;
        if listed_push_configs.configs.len() != 1 {
            return Err(format!(
                "unexpected list_push_configs result count: got {}, want 1",
                listed_push_configs.configs.len()
            ));
        }
        assert_push_config(
            &listed_push_configs.configs[0],
            &completed_task.id,
            &push_config,
            "list_push_configs",
        )?;

        client
            .delete_push_config(&DeleteTaskPushNotificationConfigRequest {
                task_id: completed_task.id.clone(),
                id: "interop-config".to_string(),
                tenant: None,
            })
            .await
            .map_err(|error| format!("delete_push_config failed: {error}"))?;

        let listed_after_delete = client
            .list_push_configs(&ListTaskPushNotificationConfigsRequest {
                task_id: completed_task.id.clone(),
                page_size: None,
                page_token: None,
                tenant: None,
            })
            .await
            .map_err(|error| format!("list_push_configs after delete failed: {error}"))?;
        if !listed_after_delete.configs.is_empty() {
            return Err(format!(
                "expected list_push_configs after delete to be empty, got {:?}",
                listed_after_delete.configs
            ));
        }
    }

    let message_only = message_from_response(
        client
            .send_message(&request_with_payload(MESSAGE_ONLY_REQUEST_TEXT, false))
            .await
            .map_err(|error| format!("message-only request failed: {error}"))?,
        "message-only",
    )?;
    assert_text(
        first_text(&message_only)?,
        &expected_message_only_text,
        "message-only",
    )?;

    let failed_task = task_from_response(
        client
            .send_message(&request_with_payload(TASK_FAILURE_REQUEST_TEXT, false))
            .await
            .map_err(|error| format!("task-failure request failed: {error}"))?,
        "task-failure",
    )?;
    assert_state(&failed_task.status.state, TaskState::Failed, "task-failure")?;
    assert_text(task_text(&failed_task)?, &expected_failed_text, "task-failure")?;

    let input_required_task = task_from_response(
        client
            .send_message(&request_with_payload(MULTI_TURN_START_REQUEST_TEXT, false))
            .await
            .map_err(|error| format!("multi-turn start failed: {error}"))?,
        "multi-turn start",
    )?;
    assert_state(
        &input_required_task.status.state,
        TaskState::InputRequired,
        "multi-turn start",
    )?;
    assert_text(
        task_text(&input_required_task)?,
        &expected_input_required_text,
        "multi-turn start",
    )?;
    assert_task_history(&input_required_task, MULTI_TURN_START_REQUEST_TEXT, "multi-turn start")?;

    let multi_turn_completed = task_from_response(
        client
            .send_message(&request_with_payload_ids(
                MULTI_TURN_CONTINUE_REQUEST_TEXT,
                false,
                Some(input_required_task.id.clone()),
                Some(input_required_task.context_id.clone()),
            ))
            .await
            .map_err(|error| format!("multi-turn continuation failed: {error}"))?,
        "multi-turn continuation",
    )?;
    assert_state(
        &multi_turn_completed.status.state,
        TaskState::Completed,
        "multi-turn continuation",
    )?;
    assert_text(
        task_text(&multi_turn_completed)?,
        &expected_multi_turn_complete_text,
        "multi-turn continuation",
    )?;
    assert_task_history_entries(
        &multi_turn_completed,
        &[MULTI_TURN_START_REQUEST_TEXT, MULTI_TURN_CONTINUE_REQUEST_TEXT],
        "multi-turn continuation",
    )?;

    let mut scenario_stream = client
        .send_streaming_message(&request_with_payload(STREAMING_REQUEST_TEXT, false))
        .await
        .map_err(|error| format!("streaming scenario request failed: {error}"))?;
    let mut streaming_chunks = Vec::new();
    let mut saw_append = false;
    let mut saw_stream_start = false;
    let mut saw_stream_complete = false;
    while let Some(event) = scenario_stream.next().await {
        let event = event.map_err(|error| format!("streaming scenario event failed: {error}"))?;
        match event {
            StreamResponse::Task(task) => {
                saw_stream_start = true;
                assert_state(&task.status.state, TaskState::Working, "streaming scenario task")?;
                assert_text(task_text(&task)?, &expected_streaming_start_text, "streaming scenario task")?;
            }
            StreamResponse::ArtifactUpdate(update) => {
                streaming_chunks.push(first_artifact_text(&update.artifact)?);
                if update.append == Some(true) {
                    saw_append = true;
                }
            }
            StreamResponse::StatusUpdate(update) => {
                assert_state(&update.status.state, TaskState::Completed, "streaming scenario status")?;
                assert_text(
                    first_text(update.status.message.as_ref().ok_or_else(|| "streaming scenario completion was missing a message".to_string())?)?,
                    &expected_streaming_complete_text,
                    "streaming scenario status",
                )?;
                saw_stream_complete = true;
            }
            StreamResponse::Message(_) => return Err("streaming scenario yielded an unexpected message event".to_string()),
        }
    }
    if !saw_stream_start || !saw_stream_complete {
        return Err("streaming scenario did not emit the expected task/status events".to_string());
    }
    if streaming_chunks != vec!["streaming chunk 1".to_string(), "streaming chunk 2".to_string()] {
        return Err(format!("streaming scenario artifact chunks mismatch: got {streaming_chunks:?}"));
    }
    if !saw_append {
        return Err("streaming scenario did not emit an append artifact update".to_string());
    }

    let long_running_task = task_from_response(
        client
            .send_message(&request_with_payload(LONG_RUNNING_REQUEST_TEXT, true))
            .await
            .map_err(|error| format!("long-running request failed: {error}"))?,
        "long-running",
    )?;
    let long_running_completed = match long_running_task.status.state {
        TaskState::Working => {
            assert_text(
                task_text(&long_running_task)?,
                &expected_long_running_start_text,
                "long-running",
            )?;
            wait_for_task_state(
                &client,
                &long_running_task.id,
                TaskState::Completed,
                "long-running",
            )
            .await?
        }
        TaskState::Completed => long_running_task,
        ref other => {
            return Err(format!(
                "unexpected long-running task state: got {other:?}, want Working or Completed"
            ));
        }
    };
    assert_text(
        task_text(&long_running_completed)?,
        &expected_long_running_complete_text,
        "long-running completion",
    )?;

    let data_types_task = task_from_response(
        client
            .send_message(&request_with_payload(DATA_TYPES_REQUEST_TEXT, false))
            .await
            .map_err(|error| format!("data-types request failed: {error}"))?,
        "data-types",
    )?;
    assert_state(&data_types_task.status.state, TaskState::Completed, "data-types")?;
    assert_text(task_text(&data_types_task)?, &expected_data_types_text, "data-types")?;
    assert_data_types_task(&data_types_task, "data-types")?;

    let extended_card = client
        .get_extended_agent_card(&GetExtendedAgentCardRequest { tenant: None })
        .await
        .map_err(|error| format!("get_extended_agent_card failed: {error}"))?;
    assert_extended_card_metadata(&extended_card, "extended-card")?;

    let protocol = card
        .supported_interfaces
        .first()
        .map(|iface| iface.protocol_binding.clone())
        .unwrap_or_else(|| "unknown".to_string());
    println!(
        "validated {} {} lifecycle against {}",
        args.server_prefix, protocol, args.card_url
    );
    Ok(())
}
