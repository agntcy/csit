// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Thin A2A probe adapter: executes a single A2A operation and writes the raw
// JSON result to stdout.  On A2A error, writes {"code":…,"message":"…"} to
// stderr and exits 1.  No assertions or orchestration live here — all test
// logic is in the Go/Ginkgo spec tree.

use std::env;
use std::process;
use std::sync::Arc;

use a2a::*;
use a2a_client::A2AClientFactory;
use a2a_client::agent_card::AgentCardResolver;
use a2a_grpc::GrpcTransportFactory;
use futures::StreamExt;
use serde_json::{Value, json};

// ── error helpers ──────────────────────────────────────────────────────────

fn exit_error(code: i32, message: impl std::fmt::Display) -> ! {
    eprintln!("{}", json!({"code": code, "message": message.to_string()}));
    process::exit(1);
}

fn fail_a2a(error: A2AError) -> ! {
    eprintln!("{}", json!({"code": error.code, "message": error.to_string()}));
    process::exit(1);
}

fn fail(message: impl std::fmt::Display) -> ! {
    exit_error(-32000, message);
}

// ── argument parsing ───────────────────────────────────────────────────────

struct ParsedArgs {
    card_url: String,
    subcommand: String,
    rest: Vec<String>,
}

fn parse_args() -> ParsedArgs {
    let all: Vec<String> = env::args().skip(1).collect();
    let mut card_url: Option<String> = None;
    let mut i = 0;
    while i < all.len() {
        match all[i].as_str() {
            "--card-url" => {
                i += 1;
                if i >= all.len() {
                    fail("--card-url requires a value");
                }
                card_url = Some(all[i].clone());
                i += 1;
            }
            _ => break,
        }
    }
    let card_url = card_url.unwrap_or_else(|| fail("missing --card-url"));
    if i >= all.len() {
        fail("missing subcommand");
    }
    let subcommand = all[i].clone();
    let rest = all[(i + 1)..].to_vec();
    ParsedArgs { card_url, subcommand, rest }
}

fn flag_value(args: &[String], flag: &str) -> Option<String> {
    let mut it = args.iter();
    while let Some(arg) = it.next() {
        if arg.as_str() == flag {
            return it.next().cloned();
        }
    }
    None
}

fn require_flag(args: &[String], flag: &str) -> String {
    flag_value(args, flag).unwrap_or_else(|| fail(format!("missing {flag}")))
}

// ── push config helpers ────────────────────────────────────────────────────

// Serialize a TaskPushNotificationConfig to the JSON format expected by the
// Go probe client (camelCase field names, "authentication" for the auth block).
fn push_config_to_json(cfg: &TaskPushNotificationConfig) -> Value {
    let mut obj = json!({
        "url": cfg.url,
        "taskId": cfg.task_id,
    });
    if let Some(id) = &cfg.id {
        obj["id"] = json!(id);
    }
    if let Some(token) = &cfg.token {
        obj["token"] = json!(token);
    }
    if let Some(auth) = &cfg.authentication {
        let mut auth_obj = json!({"scheme": auth.scheme});
        if let Some(creds) = &auth.credentials {
            auth_obj["credentials"] = json!(creds);
        }
        obj["authentication"] = auth_obj;
    }
    obj
}

// Parse a PushConfig JSON (as produced by the Go SDK) into a Rust
// TaskPushNotificationConfig.
fn push_config_from_json(val: &Value) -> TaskPushNotificationConfig {
    let authentication = if val["authentication"].is_object() {
        Some(AuthenticationInfo {
            scheme: val["authentication"]["scheme"]
                .as_str()
                .unwrap_or("")
                .to_string(),
            credentials: val["authentication"]["credentials"]
                .as_str()
                .map(|s| s.to_string()),
        })
    } else {
        None
    };
    TaskPushNotificationConfig {
        task_id: val["taskId"].as_str().unwrap_or("").to_string(),
        id: val["id"].as_str().map(|s| s.to_string()),
        url: val["url"].as_str().unwrap_or("").to_string(),
        token: val["token"].as_str().map(|s| s.to_string()),
        authentication,
        tenant: None,
    }
}

// ── main ───────────────────────────────────────────────────────────────────

#[tokio::main]
async fn main() {
    let args = parse_args();

    let resolver = AgentCardResolver::new(None);
    let card = resolver
        .resolve(&args.card_url)
        .await
        .unwrap_or_else(|e| exit_error(-32000, format!("agent card resolution failed: {e}")));

    let client = A2AClientFactory::builder()
        .register(Arc::new(GrpcTransportFactory {}))
        .preferred_bindings(vec![
            TRANSPORT_PROTOCOL_GRPC.to_string(),
            TRANSPORT_PROTOCOL_JSONRPC.to_string(),
            TRANSPORT_PROTOCOL_HTTP_JSON.to_string(),
        ])
        .build()
        .create_from_card(&card)
        .await
        .unwrap_or_else(|e| exit_error(-32000, format!("client creation failed: {e}")));

    match args.subcommand.as_str() {
        // ── send-message ────────────────────────────────────────────────────
        "send-message" => {
            let msg_json = require_flag(&args.rest, "--message-json");
            let req: SendMessageRequest = serde_json::from_str(&msg_json)
                .unwrap_or_else(|e| fail(format!("parse send-message request: {e}")));

            match client.send_message(&req).await {
                Err(e) => fail_a2a(e),
                Ok(SendMessageResponse::Task(task)) => {
                    let data = serde_json::to_value(&task)
                        .unwrap_or_else(|e| fail(format!("serialize task: {e}")));
                    println!("{}", json!({"type": "task", "data": data}));
                }
                Ok(SendMessageResponse::Message(message)) => {
                    let data = serde_json::to_value(&message)
                        .unwrap_or_else(|e| fail(format!("serialize message: {e}")));
                    println!("{}", json!({"type": "message", "data": data}));
                }
            }
        }

        // ── send-streaming-message ──────────────────────────────────────────
        "send-streaming-message" => {
            let msg_json = require_flag(&args.rest, "--message-json");
            let req: SendMessageRequest = serde_json::from_str(&msg_json)
                .unwrap_or_else(|e| fail(format!("parse send-streaming-message request: {e}")));

            let mut stream = match client.send_streaming_message(&req).await {
                Ok(s) => s,
                Err(e) => fail_a2a(e),
            };

            while let Some(event) = stream.next().await {
                let (event_type, data) = match event {
                    Err(e) => fail_a2a(e),
                    Ok(StreamResponse::Task(task)) => {
                        let data = serde_json::to_value(&task)
                            .unwrap_or_else(|e| fail(format!("serialize task event: {e}")));
                        ("task", data)
                    }
                    Ok(StreamResponse::StatusUpdate(update)) => {
                        let data = serde_json::to_value(&update)
                            .unwrap_or_else(|e| fail(format!("serialize status-update event: {e}")));
                        ("task-status-update", data)
                    }
                    Ok(StreamResponse::ArtifactUpdate(update)) => {
                        let data = serde_json::to_value(&update)
                            .unwrap_or_else(|e| fail(format!("serialize artifact-update event: {e}")));
                        ("task-artifact-update", data)
                    }
                    Ok(StreamResponse::Message(message)) => {
                        let data = serde_json::to_value(&message)
                            .unwrap_or_else(|e| fail(format!("serialize message event: {e}")));
                        ("message", data)
                    }
                };
                println!("{}", json!({"type": event_type, "data": data}));
            }
        }

        // ── get-task ────────────────────────────────────────────────────────
        "get-task" => {
            let task_id = require_flag(&args.rest, "--task-id");
            match client
                .get_task(&GetTaskRequest {
                    id: task_id,
                    history_length: None,
                    tenant: None,
                })
                .await
            {
                Err(e) => fail_a2a(e),
                Ok(task) => {
                    println!(
                        "{}",
                        serde_json::to_string(&task)
                            .unwrap_or_else(|e| fail(format!("serialize get-task response: {e}")))
                    );
                }
            }
        }

        // ── cancel-task ─────────────────────────────────────────────────────
        "cancel-task" => {
            let task_id = require_flag(&args.rest, "--task-id");
            match client
                .cancel_task(&CancelTaskRequest {
                    id: task_id,
                    metadata: None,
                    tenant: None,
                })
                .await
            {
                Err(e) => fail_a2a(e),
                Ok(task) => {
                    println!(
                        "{}",
                        serde_json::to_string(&task)
                            .unwrap_or_else(|e| fail(format!("serialize cancel-task response: {e}")))
                    );
                }
            }
        }

        // ── list-tasks ──────────────────────────────────────────────────────
        "list-tasks" => {
            let context_id = require_flag(&args.rest, "--context-id");
            match client
                .list_tasks(&ListTasksRequest {
                    context_id: Some(context_id),
                    status: None,
                    page_size: None,
                    page_token: None,
                    history_length: None,
                    status_timestamp_after: None,
                    include_artifacts: None,
                    tenant: None,
                })
                .await
            {
                Err(e) => fail_a2a(e),
                Ok(resp) => {
                    println!(
                        "{}",
                        serde_json::to_string(&resp)
                            .unwrap_or_else(|e| fail(format!("serialize list-tasks response: {e}")))
                    );
                }
            }
        }

        // ── create-push-config ──────────────────────────────────────────────
        "create-push-config" => {
            let config_json = require_flag(&args.rest, "--config-json");
            let val: Value = serde_json::from_str(&config_json)
                .unwrap_or_else(|e| fail(format!("parse push config: {e}")));
            let config = push_config_from_json(&val);
            match client.create_push_config(&config).await {
                Err(e) => fail_a2a(e),
                Ok(result) => println!("{}", push_config_to_json(&result)),
            }
        }

        // ── get-push-config ─────────────────────────────────────────────────
        "get-push-config" => {
            let task_id = require_flag(&args.rest, "--task-id");
            let config_id = require_flag(&args.rest, "--config-id");
            match client
                .get_push_config(&GetTaskPushNotificationConfigRequest {
                    task_id,
                    id: config_id,
                    tenant: None,
                })
                .await
            {
                Err(e) => fail_a2a(e),
                Ok(result) => println!("{}", push_config_to_json(&result)),
            }
        }

        // ── list-push-configs ───────────────────────────────────────────────
        "list-push-configs" => {
            let task_id = require_flag(&args.rest, "--task-id");
            match client
                .list_push_configs(&ListTaskPushNotificationConfigsRequest {
                    task_id,
                    page_size: None,
                    page_token: None,
                    tenant: None,
                })
                .await
            {
                Err(e) => fail_a2a(e),
                Ok(resp) => {
                    let configs: Vec<Value> =
                        resp.configs.iter().map(push_config_to_json).collect();
                    println!("{}", json!({"configs": configs}));
                }
            }
        }

        // ── delete-push-config ──────────────────────────────────────────────
        "delete-push-config" => {
            let task_id = require_flag(&args.rest, "--task-id");
            let config_id = require_flag(&args.rest, "--config-id");
            match client
                .delete_push_config(&DeleteTaskPushNotificationConfigRequest {
                    task_id,
                    id: config_id,
                    tenant: None,
                })
                .await
            {
                Err(e) => fail_a2a(e),
                Ok(()) => {}
            }
        }

        // ── get-extended-card ───────────────────────────────────────────────
        "get-extended-card" => {
            match client
                .get_extended_agent_card(&GetExtendedAgentCardRequest { tenant: None })
                .await
            {
                Err(e) => fail_a2a(e),
                Ok(card) => {
                    println!(
                        "{}",
                        serde_json::to_string(&card)
                            .unwrap_or_else(|e| fail(format!("serialize agent card: {e}")))
                    );
                }
            }
        }

        other => fail(format!("unknown subcommand: {other}")),
    }
}
