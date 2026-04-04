// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

use std::env;
use std::process;

use a2a::*;
use a2a_client::A2AClientFactory;
use a2a_client::agent_card::AgentCardResolver;
use futures::StreamExt;

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

fn assert_text(actual: String, expected: &str, kind: &str) -> Result<(), String> {
    if actual == expected {
        Ok(())
    } else {
        Err(format!(
            "unexpected {kind} response text: got {actual:?}, want {expected:?}"
        ))
    }
}

fn unary_text(response: SendMessageResponse) -> Result<String, String> {
    match response {
        SendMessageResponse::Message(message) => first_text(&message),
        SendMessageResponse::Task(task) => task
            .status
            .message
            .as_ref()
            .ok_or_else(|| "task response contained no message".to_string())
            .and_then(first_text),
    }
}

fn streaming_text(response: StreamResponse) -> Result<String, String> {
    match response {
        StreamResponse::Message(message) => first_text(&message),
        StreamResponse::Task(task) => task
            .status
            .message
            .as_ref()
            .ok_or_else(|| "stream task event contained no message".to_string())
            .and_then(first_text),
        other => Err(format!("unexpected streaming event: {other:?}")),
    }
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
        .build()
        .create_from_card(&card)
        .await
        .map_err(|error| format!("client creation failed: {error}"))?;

    let request = SendMessageRequest {
        message: Message::new(Role::User, vec![Part::text("ping")]),
        configuration: None,
        metadata: None,
        tenant: None,
    };

    let response = client
        .send_message(&request)
        .await
        .map_err(|error| format!("unary request failed: {error}"))?;
    assert_text(unary_text(response)?, &args.expect_text, "unary")?;

    let mut stream = client
        .send_streaming_message(&request)
        .await
        .map_err(|error| format!("streaming request failed: {error}"))?;

    let stream_event = loop {
        match stream.next().await {
            Some(Ok(StreamResponse::StatusUpdate(_))) => continue,
            Some(Ok(StreamResponse::ArtifactUpdate(_))) => continue,
            Some(Ok(event)) => break event,
            Some(Err(error)) => {
                return Err(format!("streaming event failed: {error}"));
            }
            None => {
                return Err("stream completed without a terminal response event".to_string());
            }
        }
    };
    assert_text(
        streaming_text(stream_event)?,
        &args.expect_text,
        "streaming",
    )?;

    println!("validated {0} against {1}", args.expect_text, args.card_url);
    Ok(())
}
