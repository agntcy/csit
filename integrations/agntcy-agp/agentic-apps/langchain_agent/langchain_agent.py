# SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
# SPDX-License-Identifier: Apache-2.0

import asyncio

from simple_weather_agent.simple_weather_agent import (
    SIMPLE_WEATHER_AGENT_WITH_TOOLS,
)

import argparse
import gateway_bindings

phoenix = gateway_bindings.Phoenix()

async def run_agent(message, gateway):

    agent = SIMPLE_WEATHER_AGENT_WITH_TOOLS()

    # register local agent
    await phoenix.create_agent("cisco", "default", "langchain")

    # connect to gateway server
    conn_id = await phoenix.connect(gateway)

    remote_organization = "cisco"
    remote_namespace = "default"
    remote_agent = "autogen"

    await phoenix.set_route(remote_organization, remote_namespace, remote_agent)

    await phoenix.publish(message.encode(), remote_organization, remote_namespace, remote_agent)
    print(f"sent: {str(message)}")

    try:
        # Wait to receive a message
        source, msg_rcv = await phoenix.receive()
        msg = msg_rcv.decode()
        print(f"received: {str(msg)}")
    except asyncio.CancelledError:
        print(f"stopped.")

    result = agent.call(msg)
    print(result)

def main():
    parser = argparse.ArgumentParser(description="Command line client for message passing.")
    parser.add_argument("-m", "--message", type=str, help="Message to send.")
    parser.add_argument("-g", "--gateway", type=str, help="Gateway address.", default="http://127.0.0.1:12345")
    args = parser.parse_args()
    asyncio.run(run_agent(args.message, args.gateway))

if __name__ == "__main__":
    main()
