// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

use std::env;
use std::sync::Arc;

use a2a::*;
use a2a_server::{DefaultRequestHandler, InMemoryTaskStore, StaticAgentCard};
use axum::Router;
use futures::stream::{self, BoxStream};
use tokio::net::TcpListener;

struct InteropExecutor;

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

fn build_response_message(request: Option<&Message>) -> Message {
    Message::new(
        Role::Agent,
        vec![Part::text(format!(
            "rust server received: {}",
            first_text(request)
        ))],
    )
}

impl a2a_server::AgentExecutor for InteropExecutor {
    fn execute(
        &self,
        ctx: a2a_server::ExecutorContext,
    ) -> BoxStream<'static, Result<StreamResponse, A2AError>> {
        let response = StreamResponse::Message(build_response_message(ctx.message.as_ref()));
        Box::pin(stream::once(async move { Ok(response) }))
    }

    fn cancel(
        &self,
        _ctx: a2a_server::ExecutorContext,
    ) -> BoxStream<'static, Result<StreamResponse, A2AError>> {
        Box::pin(stream::empty())
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
    let handler = Arc::new(DefaultRequestHandler::new(
        InteropExecutor,
        InMemoryTaskStore::new(),
    ));
    let card_producer = Arc::new(StaticAgentCard::new(build_agent_card(port)));

    let app = Router::new()
        .nest("/rpc", a2a_server::jsonrpc::jsonrpc_router(handler))
        .merge(a2a_server::agent_card::agent_card_router(card_producer));

    let listener = TcpListener::bind(("127.0.0.1", port))
        .await
        .expect("listener should bind");

    println!("rust jsonrpc fixture listening on http://127.0.0.1:{port}");
    axum::serve(listener, app).await.expect("server should run");
}
