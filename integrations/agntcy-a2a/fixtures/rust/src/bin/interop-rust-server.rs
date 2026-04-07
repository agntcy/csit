// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

use std::env;
use std::collections::HashMap;
use std::sync::Arc;
use std::time::Duration;

use a2a::*;
use a2a_grpc::GrpcHandler;
use a2a_pb::proto::a2a_service_server::A2aServiceServer;
use a2a_server::{
    DefaultRequestHandler, InMemoryPushConfigStore, InMemoryTaskStore, StaticAgentCard,
};
use async_trait::async_trait;
use axum::Router;
use futures::StreamExt;
use futures::stream::{self, BoxStream};
use tokio::net::TcpListener;
use tokio::time::sleep;
use tonic::transport::Server;

const PENDING_REQUEST_TEXT: &str = "pending";
const MESSAGE_ONLY_REQUEST_TEXT: &str = "message-only";
const TASK_FAILURE_REQUEST_TEXT: &str = "task-failure";
const MULTI_TURN_START_REQUEST_TEXT: &str = "multi-turn start";
const MULTI_TURN_CONTINUE_REQUEST_TEXT: &str = "multi-turn continue";
const STREAMING_REQUEST_TEXT: &str = "streaming";
const LONG_RUNNING_REQUEST_TEXT: &str = "long-running";
const DATA_TYPES_REQUEST_TEXT: &str = "data-types";
const EXTENDED_CARD_SCHEME_ID: &str = "bearer_token";

#[derive(Clone, Copy)]
enum TransportProtocol {
    JsonRpc,
    Rest,
    Grpc,
}

impl TransportProtocol {
    fn from_str(value: &str) -> Self {
        match value {
            "jsonrpc" => Self::JsonRpc,
            "rest" => Self::Rest,
            "grpc" => Self::Grpc,
            other => panic!("unsupported protocol: {other}"),
        }
    }

    fn name(self) -> &'static str {
        match self {
            Self::JsonRpc => "jsonrpc",
            Self::Rest => "rest",
            Self::Grpc => "grpc",
        }
    }
}

struct InteropExecutor;

struct InteropHandler {
    inner: DefaultRequestHandler,
    extended_card: AgentCard,
}

fn first_text(message: Option<&Message>) -> String {
    message
        .and_then(|message| message.parts.iter().find_map(Part::as_text))
        .unwrap_or_default()
        .to_string()
}

fn build_agent_card(
    card_port: u16,
    protocol: TransportProtocol,
    grpc_port: Option<u16>,
    extended: bool,
) -> AgentCard {
    let base_url = format!("http://127.0.0.1:{card_port}");
    let (name, interface_url, binding) = match protocol {
        TransportProtocol::JsonRpc => (
            "CSIT Rust JSON-RPC Agent",
            format!("{base_url}/rpc"),
            TRANSPORT_PROTOCOL_JSONRPC,
        ),
        TransportProtocol::Rest => (
            "CSIT Rust REST Agent",
            base_url.clone(),
            TRANSPORT_PROTOCOL_HTTP_JSON,
        ),
        TransportProtocol::Grpc => (
            "CSIT Rust gRPC Agent",
            format!(
                "http://127.0.0.1:{}",
                grpc_port.expect("gRPC transport requires a dedicated port")
            ),
            TRANSPORT_PROTOCOL_GRPC,
        ),
    };

    AgentCard {
        name: name.to_string(),
        description: if extended {
            "Rust interoperability fixture for CSIT (extended)".to_string()
        } else {
            "Rust interoperability fixture for CSIT".to_string()
        },
        version: VERSION.to_string(),
        supported_interfaces: vec![AgentInterface::new(interface_url, binding)],
        capabilities: AgentCapabilities {
            streaming: Some(true),
            push_notifications: Some(true),
            extensions: None,
            extended_agent_card: Some(true),
        },
        default_input_modes: vec!["text/plain".to_string()],
        default_output_modes: vec!["text/plain".to_string()],
        skills: scenario_skills(),
        provider: None,
        documentation_url: None,
        icon_url: None,
        security_schemes: Some(HashMap::from([(
            EXTENDED_CARD_SCHEME_ID.to_string(),
            SecurityScheme::HttpAuth(HttpAuthSecurityScheme {
                scheme: "Bearer".to_string(),
                description: Some("Bearer token authentication".to_string()),
                bearer_format: Some("JWT".to_string()),
            }),
        )])),
        security_requirements: None,
        signatures: None,
    }
}

fn scenario_skills() -> Vec<AgentSkill> {
    vec![
        build_skill("message-only", "Returns a message response without creating a task."),
        build_skill("task-lifecycle", "Creates, lists, fetches, and cancels tasks."),
        build_skill("task-failure", "Returns a failed task response."),
        build_skill("task-cancel", "Creates a cancelable working task."),
        build_skill("multi-turn", "Requests more input before completing the task."),
        build_skill("streaming", "Streams task and artifact updates."),
        build_skill("long-running", "Returns early and completes asynchronously."),
        build_skill("data-types", "Produces text, structured data, and URL parts."),
    ]
}

fn build_skill(id: &str, description: &str) -> AgentSkill {
    AgentSkill {
        id: id.to_string(),
        name: id.to_string(),
        description: description.to_string(),
        tags: vec!["csit".to_string(), "scenario-parity".to_string()],
        examples: None,
        input_modes: None,
        output_modes: None,
        security_requirements: None,
    }
}

fn build_task_message(task_id: &str, context_id: &str, text: impl Into<String>) -> Message {
    let mut message = Message::new(Role::Agent, vec![Part::text(text)]);
    message.task_id = Some(task_id.to_string());
    message.context_id = Some(context_id.to_string());
    message
}

fn build_task(
    ctx: &a2a_server::ExecutorContext,
    state: TaskState,
    text: impl Into<String>,
) -> Task {
    let mut history = ctx
        .stored_task
        .as_ref()
        .and_then(|task| task.history.clone())
        .unwrap_or_default();
    if let Some(message) = ctx.message.clone() {
        let should_append = history
            .last()
            .map(|entry| entry.message_id != message.message_id)
            .unwrap_or(true);
        if should_append {
            history.push(message);
        }
    }

    let mut task = Task {
        id: ctx.task_id.clone(),
        context_id: ctx.context_id.clone(),
        status: TaskStatus {
            state,
            message: None,
            timestamp: None,
        },
        artifacts: None,
        history: (!history.is_empty()).then_some(history),
        metadata: None,
    };
    task.status.message = Some(build_task_message(&task.id, &task.context_id, text));
    task
}

fn build_status_update(
    ctx: &a2a_server::ExecutorContext,
    state: TaskState,
    text: impl Into<String>,
) -> TaskStatusUpdateEvent {
    TaskStatusUpdateEvent {
        task_id: ctx.task_id.clone(),
        context_id: ctx.context_id.clone(),
        status: TaskStatus {
            state,
            message: Some(build_task_message(&ctx.task_id, &ctx.context_id, text)),
            timestamp: None,
        },
        metadata: None,
    }
}

fn build_artifact_update(
    ctx: &a2a_server::ExecutorContext,
    artifact_id: &str,
    parts: Vec<Part>,
    append: bool,
    last_chunk: bool,
) -> TaskArtifactUpdateEvent {
    TaskArtifactUpdateEvent {
        task_id: ctx.task_id.clone(),
        context_id: ctx.context_id.clone(),
        artifact: Artifact {
            artifact_id: artifact_id.to_string(),
            name: Some(artifact_id.to_string()),
            description: None,
            parts,
            metadata: None,
            extensions: None,
        },
        append: Some(append),
        last_chunk: Some(last_chunk),
        metadata: None,
    }
}

fn build_data_types_artifact(task_id: &str) -> Artifact {
    Artifact {
        artifact_id: format!("{task_id}-artifact"),
        name: Some("data-types-artifact".to_string()),
        description: Some("Mixed content artifact for scenario parity".to_string()),
        parts: vec![
            Part::text("structured summary"),
            Part::data(serde_json::json!({
                "kind": "report",
                "items": 2,
            })),
            Part::url("https://example.invalid/diagram.svg")
                .with_media_type("image/svg+xml")
                .with_filename("diagram.svg"),
        ],
        metadata: None,
        extensions: None,
    }
}

fn message_text(ctx: &a2a_server::ExecutorContext) -> String {
    first_text(ctx.message.as_ref())
}

impl InteropHandler {
    fn new(inner: DefaultRequestHandler, extended_card: AgentCard) -> Self {
        Self {
            inner,
            extended_card,
        }
    }
}

#[async_trait]
impl a2a_server::RequestHandler for InteropHandler {
    async fn send_message(
        &self,
        params: &a2a_server::ServiceParams,
        req: SendMessageRequest,
    ) -> Result<SendMessageResponse, A2AError> {
        self.inner.send_message(params, req).await
    }

    async fn send_streaming_message(
        &self,
        params: &a2a_server::ServiceParams,
        req: SendMessageRequest,
    ) -> Result<BoxStream<'static, Result<StreamResponse, A2AError>>, A2AError> {
        self.inner.send_streaming_message(params, req).await
    }

    async fn get_task(
        &self,
        params: &a2a_server::ServiceParams,
        req: GetTaskRequest,
    ) -> Result<Task, A2AError> {
        self.inner.get_task(params, req).await
    }

    async fn list_tasks(
        &self,
        params: &a2a_server::ServiceParams,
        req: ListTasksRequest,
    ) -> Result<ListTasksResponse, A2AError> {
        self.inner.list_tasks(params, req).await
    }

    async fn cancel_task(
        &self,
        params: &a2a_server::ServiceParams,
        req: CancelTaskRequest,
    ) -> Result<Task, A2AError> {
        self.inner.cancel_task(params, req).await
    }

    async fn subscribe_to_task(
        &self,
        params: &a2a_server::ServiceParams,
        req: SubscribeToTaskRequest,
    ) -> Result<BoxStream<'static, Result<StreamResponse, A2AError>>, A2AError> {
        self.inner.subscribe_to_task(params, req).await
    }

    async fn create_push_config(
        &self,
        params: &a2a_server::ServiceParams,
        req: CreateTaskPushNotificationConfigRequest,
    ) -> Result<TaskPushNotificationConfig, A2AError> {
        self.inner.create_push_config(params, req).await
    }

    async fn get_push_config(
        &self,
        params: &a2a_server::ServiceParams,
        req: GetTaskPushNotificationConfigRequest,
    ) -> Result<TaskPushNotificationConfig, A2AError> {
        self.inner.get_push_config(params, req).await
    }

    async fn list_push_configs(
        &self,
        params: &a2a_server::ServiceParams,
        req: ListTaskPushNotificationConfigsRequest,
    ) -> Result<ListTaskPushNotificationConfigsResponse, A2AError> {
        self.inner.list_push_configs(params, req).await
    }

    async fn delete_push_config(
        &self,
        params: &a2a_server::ServiceParams,
        req: DeleteTaskPushNotificationConfigRequest,
    ) -> Result<(), A2AError> {
        self.inner.delete_push_config(params, req).await
    }

    async fn get_extended_agent_card(
        &self,
        _params: &a2a_server::ServiceParams,
        _req: GetExtendedAgentCardRequest,
    ) -> Result<AgentCard, A2AError> {
        Ok(self.extended_card.clone())
    }
}

impl a2a_server::AgentExecutor for InteropExecutor {
    fn execute(
        &self,
        ctx: a2a_server::ExecutorContext,
    ) -> BoxStream<'static, Result<StreamResponse, A2AError>> {
        let request_text = message_text(&ctx);

        match request_text.as_str() {
            MESSAGE_ONLY_REQUEST_TEXT => {
                let response = StreamResponse::Message(Message::new(
                    Role::Agent,
                    vec![Part::text("rust server message-only response")],
                ));
                Box::pin(stream::once(async move { Ok(response) }))
            }
            TASK_FAILURE_REQUEST_TEXT => {
                let response = StreamResponse::Task(build_task(
                    &ctx,
                    TaskState::Failed,
                    "rust server failed task",
                ));
                Box::pin(stream::once(async move { Ok(response) }))
            }
            MULTI_TURN_START_REQUEST_TEXT => {
                let response = StreamResponse::Task(build_task(
                    &ctx,
                    TaskState::InputRequired,
                    "rust server needs more input",
                ));
                Box::pin(stream::once(async move { Ok(response) }))
            }
            MULTI_TURN_CONTINUE_REQUEST_TEXT => {
                let response = StreamResponse::Task(build_task(
                    &ctx,
                    TaskState::Completed,
                    "rust server multi-turn completed",
                ));
                Box::pin(stream::once(async move { Ok(response) }))
            }
            STREAMING_REQUEST_TEXT => {
                let working = StreamResponse::Task(build_task(
                    &ctx,
                    TaskState::Working,
                    "rust server streaming started",
                ));
                let artifact_id = format!("{}-stream", ctx.task_id);
                let first_artifact = StreamResponse::ArtifactUpdate(build_artifact_update(
                    &ctx,
                    &artifact_id,
                    vec![Part::text("streaming chunk 1")],
                    false,
                    false,
                ));
                let second_artifact = StreamResponse::ArtifactUpdate(build_artifact_update(
                    &ctx,
                    &artifact_id,
                    vec![Part::text("streaming chunk 2")],
                    true,
                    true,
                ));
                let completed = StreamResponse::StatusUpdate(build_status_update(
                    &ctx,
                    TaskState::Completed,
                    "rust server streaming complete",
                ));
                Box::pin(stream::iter(vec![
                    Ok(working),
                    Ok(first_artifact),
                    Ok(second_artifact),
                    Ok(completed),
                ]))
            }
            LONG_RUNNING_REQUEST_TEXT => {
                let started = StreamResponse::Task(build_task(
                    &ctx,
                    TaskState::Working,
                    "rust server long-running started",
                ));
                let progress = StreamResponse::StatusUpdate(build_status_update(
                    &ctx,
                    TaskState::Working,
                    "rust server long-running progress",
                ));
                let completed = StreamResponse::StatusUpdate(build_status_update(
                    &ctx,
                    TaskState::Completed,
                    "rust server long-running complete",
                ));
                Box::pin(
                    stream::once(async move { Ok(started) })
                        .chain(stream::once(async move {
                            sleep(Duration::from_millis(150)).await;
                            Ok(progress)
                        }))
                        .chain(stream::once(async move {
                            sleep(Duration::from_millis(150)).await;
                            Ok(completed)
                        })),
                )
            }
            DATA_TYPES_REQUEST_TEXT => {
                let mut task = build_task(&ctx, TaskState::Completed, "rust server data-types ready");
                task.artifacts = Some(vec![build_data_types_artifact(&ctx.task_id)]);
                let response = StreamResponse::Task(task);
                Box::pin(stream::once(async move { Ok(response) }))
            }
            _ => {
                let response_text = format!("rust server received: {request_text}");
                let state = if request_text == PENDING_REQUEST_TEXT {
                    TaskState::Working
                } else {
                    TaskState::Completed
                };
                let response = StreamResponse::Task(build_task(&ctx, state, response_text));
                Box::pin(stream::once(async move { Ok(response) }))
            }
        }
    }

    fn cancel(
        &self,
        ctx: a2a_server::ExecutorContext,
    ) -> BoxStream<'static, Result<StreamResponse, A2AError>> {
        let response = StreamResponse::StatusUpdate(build_status_update(
            &ctx,
            TaskState::Canceled,
            "rust server canceled task",
        ));
        Box::pin(stream::once(async move { Ok(response) }))
    }
}

fn parse_args() -> (u16, TransportProtocol, Option<u16>) {
    let mut args = env::args().skip(1);
    let mut port = 19092;
    let mut protocol = TransportProtocol::JsonRpc;
    let mut grpc_port = None;

    while let Some(arg) = args.next() {
        match arg.as_str() {
            "--port" => {
                let value = args.next().expect("--port requires a numeric argument");
                port = value.parse::<u16>().expect("--port value must fit in u16");
            }
            "--grpc-port" => {
                let value = args
                    .next()
                    .expect("--grpc-port requires a numeric argument");
                grpc_port = Some(
                    value
                        .parse::<u16>()
                        .expect("--grpc-port value must fit in u16"),
                );
            }
            "--protocol" => {
                let value = args
                    .next()
                    .expect("--protocol requires either 'jsonrpc', 'rest', or 'grpc'");
                protocol = TransportProtocol::from_str(&value);
            }
            other => panic!("unknown argument: {other}"),
        }
    }

    (port, protocol, grpc_port)
}

#[tokio::main]
async fn main() {
    let (port, protocol, grpc_port) = parse_args();
    let public_card = build_agent_card(port, protocol, grpc_port, false);
    let extended_card = build_agent_card(port, protocol, grpc_port, true);
    let handler = Arc::new(InteropHandler::new(
        DefaultRequestHandler::new(InteropExecutor, InMemoryTaskStore::new())
            .with_push_config_store(InMemoryPushConfigStore::new()),
        extended_card,
    ));
    let card_producer = Arc::new(StaticAgentCard::new(public_card));

    let app = match protocol {
        TransportProtocol::JsonRpc => Router::new()
            .nest("/rpc", a2a_server::jsonrpc::jsonrpc_router(handler))
            .merge(a2a_server::agent_card::agent_card_router(card_producer)),
        TransportProtocol::Rest => Router::new()
            .merge(a2a_server::rest::rest_router(handler))
            .merge(a2a_server::agent_card::agent_card_router(card_producer)),
        TransportProtocol::Grpc => {
            let grpc_port = grpc_port.expect("gRPC transport requires --grpc-port");
            let card_listener = TcpListener::bind(("127.0.0.1", port))
                .await
                .expect("agent card listener should bind");
            let card_app =
                Router::new().merge(a2a_server::agent_card::agent_card_router(card_producer));
            let grpc_addr = format!("127.0.0.1:{grpc_port}")
                .parse()
                .expect("gRPC address should parse");
            let grpc_service = A2aServiceServer::new(GrpcHandler::new(handler));

            println!(
                "rust grpc fixture listening on http://127.0.0.1:{grpc_port} with card http://127.0.0.1:{port}"
            );

            let card_server = tokio::spawn(async move {
                axum::serve(card_listener, card_app)
                    .await
                    .expect("agent card server should run");
            });

            Server::builder()
                .add_service(grpc_service)
                .serve(grpc_addr)
                .await
                .expect("gRPC server should run");

            let _ = card_server.await;

            return;
        }
    };

    let listener = TcpListener::bind(("127.0.0.1", port))
        .await
        .expect("listener should bind");

    println!(
        "rust {} fixture listening on http://127.0.0.1:{port}",
        protocol.name()
    );
    axum::serve(listener, app).await.expect("server should run");
}
