// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

use std::env;

use a2a::*;
use axum::extract::State;
use axum::response::IntoResponse;
use axum::routing::{get, post};
use axum::{Json, Router};
use futures::stream::{self, BoxStream};
use serde_json::Value;
use tokio::net::TcpListener;

const METHOD_SEND_MESSAGE: &str = "SendMessage";
const METHOD_SEND_STREAMING_MESSAGE: &str = "SendStreamingMessage";
const LEGACY_METHOD_SEND_MESSAGE: &str = "message.send";
const LEGACY_METHOD_SEND_STREAMING_MESSAGE: &str = "message.stream";

#[derive(Clone)]
struct AppState {
    card: AgentCard,
}

fn first_text(message: Option<&Message>) -> String {
    message
        .and_then(|message| message.parts.iter().find_map(Part::as_text))
        .unwrap_or_default()
        .to_string()
}

fn build_agent_card(port: u16) -> AgentCard {
    let base_url = format!("http://127.0.0.1:{port}");

    AgentCard {
        name: "CSIT Rust JSON-RPC Agent".to_string(),
        description: "Rust interoperability fixture for CSIT".to_string(),
        version: VERSION.to_string(),
        supported_interfaces: vec![AgentInterface::new(
            format!("{base_url}/rpc"),
            TRANSPORT_PROTOCOL_JSONRPC,
        )],
        capabilities: AgentCapabilities {
            streaming: Some(true),
            push_notifications: Some(false),
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

fn build_response_message(request: &SendMessageRequest) -> Message {
    Message::new(
        Role::Agent,
        vec![Part::text(format!(
            "rust server received: {}",
            first_text(Some(&request.message))
        ))],
    )
}

fn error_response(id: JsonRpcId, error: A2AError) -> Json<JsonRpcResponse> {
    Json(JsonRpcResponse::error(id, error.to_jsonrpc_error()))
}

async fn handle_agent_card(State(state): State<AppState>) -> Json<AgentCard> {
    Json(state.card)
}

async fn handle_jsonrpc(
    State(_state): State<AppState>,
    Json(request): Json<JsonRpcRequest>,
) -> impl IntoResponse {
    let id = request.id.clone();
    let raw_params = request.params.unwrap_or(Value::Null);

    if request.jsonrpc != "2.0" {
        return error_response(id, A2AError::invalid_request("invalid jsonrpc version"))
            .into_response();
    }

    match request.method.as_str() {
        METHOD_SEND_MESSAGE | LEGACY_METHOD_SEND_MESSAGE => {
            match serde_json::from_value::<SendMessageRequest>(raw_params) {
                Ok(request) => {
                    let response = StreamResponse::Message(build_response_message(&request));
                    let value = serde_json::to_value(response)
                        .map_err(|error| A2AError::internal(error.to_string()));

                    match value {
                        Ok(value) => Json(JsonRpcResponse::success(id, value)).into_response(),
                        Err(error) => error_response(id, error).into_response(),
                    }
                }
                Err(error) => error_response(
                    id,
                    A2AError::invalid_request(format!("invalid params: {error}")),
                )
                .into_response(),
            }
        }
        METHOD_SEND_STREAMING_MESSAGE | LEGACY_METHOD_SEND_STREAMING_MESSAGE => {
            match serde_json::from_value::<SendMessageRequest>(raw_params) {
                Ok(request) => {
                    let response = StreamResponse::Message(build_response_message(&request));
                    let stream: BoxStream<'static, Result<StreamResponse, A2AError>> =
                        Box::pin(stream::once(async move { Ok(response) }));
                    a2a_server::sse::sse_jsonrpc_stream(id, stream).into_response()
                }
                Err(error) => error_response(
                    id,
                    A2AError::invalid_request(format!("invalid params: {error}")),
                )
                .into_response(),
            }
        }
        "" => error_response(id, A2AError::invalid_request("method is required")).into_response(),
        _ => error_response(id, A2AError::method_not_found(&request.method)).into_response(),
    }
}

fn parse_port() -> u16 {
    let mut args = env::args().skip(1);

    while let Some(arg) = args.next() {
        if arg == "--port" {
            let value = args.next().expect("--port requires a numeric argument");
            return value.parse::<u16>().expect("--port value must fit in u16");
        }
    }

    19092
}

#[tokio::main]
async fn main() {
    let port = parse_port();
    let state = AppState {
        card: build_agent_card(port),
    };

    let app = Router::new()
        .route("/rpc", post(handle_jsonrpc))
        .route("/jsonrpc", post(handle_jsonrpc))
        .route("/.well-known/agent-card.json", get(handle_agent_card))
        .with_state(state);

    let listener = TcpListener::bind(("127.0.0.1", port))
        .await
        .expect("listener should bind");

    println!("rust jsonrpc fixture listening on http://127.0.0.1:{port}");
    axum::serve(listener, app).await.expect("server should run");
}
