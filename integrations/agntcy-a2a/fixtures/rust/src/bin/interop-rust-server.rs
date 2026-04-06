// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

use std::env;
use std::sync::Arc;

use a2a::*;
use a2a_grpc::GrpcHandler;
use a2a_pb::proto::a2a_service_server::A2aServiceServer;
use a2a_server::{
    DefaultRequestHandler, InMemoryPushConfigStore, InMemoryTaskStore, StaticAgentCard,
};
use axum::Router;
use futures::stream::{self, BoxStream};
use tokio::net::TcpListener;
use tonic::transport::Server;

const PENDING_REQUEST_TEXT: &str = "pending";

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
        description: "Rust interoperability fixture for CSIT".to_string(),
        version: VERSION.to_string(),
        supported_interfaces: vec![AgentInterface::new(interface_url, binding)],
        capabilities: AgentCapabilities {
            streaming: Some(true),
            push_notifications: Some(true),
            extensions: None,
            extended_agent_card: None,
        },
        default_input_modes: vec!["text/plain".to_string()],
        default_output_modes: vec!["text/plain".to_string()],
        skills: vec![],
        provider: None,
        documentation_url: None,
        icon_url: None,
        security_schemes: None,
        security_requirements: None,
        signatures: None,
    }
}

fn build_response_message(request: Option<&Message>) -> Message {
    Message::new(
        Role::Agent,
        vec![Part::text(format!(
            "rust server received: {}",
            first_text(request)
        ))],
    )
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
    let mut task = Task {
        id: ctx.task_id.clone(),
        context_id: ctx.context_id.clone(),
        status: TaskStatus {
            state,
            message: None,
            timestamp: None,
        },
        artifacts: None,
        history: ctx.message.clone().map(|message| vec![message]),
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

impl a2a_server::AgentExecutor for InteropExecutor {
    fn execute(
        &self,
        ctx: a2a_server::ExecutorContext,
    ) -> BoxStream<'static, Result<StreamResponse, A2AError>> {
        let response_text = format!("rust server received: {}", first_text(ctx.message.as_ref()));
        let state = if first_text(ctx.message.as_ref()) == PENDING_REQUEST_TEXT {
            TaskState::Working
        } else {
            TaskState::Completed
        };
        let response = StreamResponse::Task(build_task(&ctx, state, response_text));
        Box::pin(stream::once(async move { Ok(response) }))
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
    let handler = Arc::new(
        DefaultRequestHandler::new(InteropExecutor, InMemoryTaskStore::new())
            .with_push_config_store(InMemoryPushConfigStore::new()),
    );
    let card_producer = Arc::new(StaticAgentCard::new(build_agent_card(
        port, protocol, grpc_port,
    )));

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
