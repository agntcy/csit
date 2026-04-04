// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

use std::env;
use std::process;

use a2a::*;
use reqwest::header::ACCEPT;
use serde_json::Value;

const METHOD_SEND_MESSAGE: &str = "SendMessage";
const METHOD_SEND_STREAMING_MESSAGE: &str = "SendStreamingMessage";

struct Args {
    card_url: String,
    expect_text: String,
}

fn parse_args() -> Result<Args, String> {
    let mut args = env::args().skip(1);
    let mut card_url = None;
    let mut expect_text = None;

    while let Some(arg) = args.next() {
        match arg.as_str() {
            "--card-url" => {
                card_url = Some(
                    args.next()
                        .ok_or_else(|| "--card-url requires a value".to_string())?,
                );
            }
            "--expect-text" => {
                expect_text = Some(
                    args.next()
                        .ok_or_else(|| "--expect-text requires a value".to_string())?,
                );
            }
            other => {
                return Err(format!("unknown argument: {other}"));
            }
        }
    }

    Ok(Args {
        card_url: card_url.ok_or_else(|| "missing --card-url".to_string())?,
        expect_text: expect_text.ok_or_else(|| "missing --expect-text".to_string())?,
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

fn assert_event_text(response: StreamResponse, expected: &str) -> Result<(), String> {
    match response {
        StreamResponse::Message(message) => {
            let actual = first_text(&message)?;
            if actual == expected {
                Ok(())
            } else {
                Err(format!(
                    "unexpected unary response text: got {actual:?}, want {expected:?}"
                ))
            }
        }
        other => Err(format!("unexpected response event: {other:?}")),
    }
}

fn extract_jsonrpc_result(response: JsonRpcResponse) -> Result<Value, String> {
    if let Some(error) = response.error {
        return Err(format!("jsonrpc error {}: {}", error.code, error.message));
    }

    response
        .result
        .ok_or_else(|| "jsonrpc response missing result".to_string())
}

fn resolve_jsonrpc_url(card: &Value) -> Result<String, String> {
    let interfaces = card
        .get("supportedInterfaces")
        .and_then(Value::as_array)
        .ok_or_else(|| "agent card has no supportedInterfaces array".to_string())?;

    interfaces
        .iter()
        .find(|interface| {
            interface.get("protocolBinding").and_then(Value::as_str)
                == Some(TRANSPORT_PROTOCOL_JSONRPC)
        })
        .and_then(|interface| interface.get("url"))
        .and_then(Value::as_str)
        .map(ToString::to_string)
        .ok_or_else(|| "agent card has no JSON-RPC interface".to_string())
}

async fn resolve_agent_card(client: &reqwest::Client, card_url: &str) -> Result<Value, String> {
    let url = format!(
        "{}/.well-known/agent-card.json",
        card_url.trim_end_matches('/')
    );
    let response = client
        .get(url)
        .send()
        .await
        .map_err(|error| format!("agent card fetch failed: {error}"))?;

    if !response.status().is_success() {
        return Err(format!(
            "agent card fetch returned HTTP {}",
            response.status()
        ));
    }

    response
        .json::<Value>()
        .await
        .map_err(|error| format!("agent card parse failed: {error}"))
}

async fn send_unary(
    client: &reqwest::Client,
    endpoint: &str,
    request: &SendMessageRequest,
) -> Result<StreamResponse, String> {
    let rpc = JsonRpcRequest::new(
        JsonRpcId::String("interop-unary".to_string()),
        METHOD_SEND_MESSAGE,
        Some(serde_json::to_value(request).map_err(|error| error.to_string())?),
    );
    let response = client
        .post(endpoint)
        .json(&rpc)
        .send()
        .await
        .map_err(|error| format!("unary request failed: {error}"))?;

    if !response.status().is_success() {
        return Err(format!("unary request returned HTTP {}", response.status()));
    }

    let rpc_response = response
        .json::<JsonRpcResponse>()
        .await
        .map_err(|error| format!("failed to parse unary response: {error}"))?;
    serde_json::from_value(extract_jsonrpc_result(rpc_response)?)
        .map_err(|error| format!("failed to decode unary result: {error}"))
}

async fn send_streaming(
    client: &reqwest::Client,
    endpoint: &str,
    request: &SendMessageRequest,
) -> Result<StreamResponse, String> {
    let rpc = JsonRpcRequest::new(
        JsonRpcId::String("interop-stream".to_string()),
        METHOD_SEND_STREAMING_MESSAGE,
        Some(serde_json::to_value(request).map_err(|error| error.to_string())?),
    );
    let response = client
        .post(endpoint)
        .header(ACCEPT, "text/event-stream")
        .json(&rpc)
        .send()
        .await
        .map_err(|error| format!("streaming request failed: {error}"))?;

    if !response.status().is_success() {
        return Err(format!(
            "streaming request returned HTTP {}",
            response.status()
        ));
    }

    let body = response
        .text()
        .await
        .map_err(|error| format!("failed to read streaming response: {error}"))?;

    for line in body.lines() {
        let Some(payload) = line.strip_prefix("data:") else {
            continue;
        };

        let rpc_response = serde_json::from_str::<JsonRpcResponse>(payload.trim())
            .map_err(|error| format!("failed to parse SSE payload: {error}"))?;
        return serde_json::from_value(extract_jsonrpc_result(rpc_response)?)
            .map_err(|error| format!("failed to decode streaming result: {error}"));
    }

    Err("stream completed without a data event".to_string())
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
    let client = reqwest::Client::new();
    let card = resolve_agent_card(&client, &args.card_url)
        .await
        .map_err(|error| format!("agent card resolution failed: {error}"))?;
    let endpoint = resolve_jsonrpc_url(&card)?;

    let request = SendMessageRequest {
        message: Message::new(Role::User, vec![Part::text("ping")]),
        configuration: None,
        metadata: None,
        tenant: None,
    };

    let response = send_unary(&client, &endpoint, &request).await?;
    assert_event_text(response, &args.expect_text)?;

    let stream_event = send_streaming(&client, &endpoint, &request).await?;
    assert_event_text(stream_event, &args.expect_text)?;

    println!("validated {0} against {1}", args.expect_text, args.card_url);
    Ok(())
}
