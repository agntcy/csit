# SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
# SPDX-License-Identifier: Apache-2.0

import asyncio

from simple_weather_agent.simple_weather_agent import (
    SIMPLE_WEATHER_AGENT_WITH_TOOLS,
)

import argparse
import agp_bindings

gateway = agp_bindings.Gateway()

async def run_agent(message, address):

    agent = SIMPLE_WEATHER_AGENT_WITH_TOOLS()

    local_organization = "cisco"
    local_namespace = "default"
    local_agent = "langchain"

    # Connect to the gateway server
    local_agent_id = await gateway.create_agent(
        local_organization, local_namespace, local_agent
    )

    # Connect to the service and subscribe for the local name
    _ = await gateway.connect(address, insecure=True)
    await gateway.subscribe(
        local_organization, local_namespace, local_agent, local_agent_id
    )

    remote_organization = "cisco"
    remote_namespace = "default"
    remote_agent = "autogen"

    await gateway.set_route(remote_organization, remote_namespace, remote_agent)

    await gateway.publish(message.encode(), remote_organization, remote_namespace, remote_agent)
    print(f"sent: {str(message)}")

    try:
        # Wait to receive a message
        source, msg_rcv = await gateway.receive()
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
