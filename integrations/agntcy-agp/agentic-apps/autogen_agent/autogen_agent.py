# SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
# SPDX-License-Identifier: Apache-2.0

import asyncio
from simple_agentic_app.simple_agentic_app import simple_autogen_app

import argparse
import agp_bindings

gateway = agp_bindings.Gateway()

async def run_agent(address):
    agent = simple_autogen_app()

    local_organization = "cisco"
    local_namespace = "default"
    local_agent = "autogen"

    # Connect to the gateway server
    local_agent_id = await gateway.create_agent(
        local_organization, local_namespace, local_agent
    )

    # Connect to the service and subscribe for the local name
    _ = await gateway.connect(address, insecure=True)
    await gateway.subscribe(
        local_organization, local_namespace, local_agent, local_agent_id
    )

    while True:
        # receive messages
        src, msg = await gateway.receive()

        # handle received messages
        result = await agent.initate_chat(msg)
        print(result)

        # process response
        result.inner_messages
        weather_question = result.inner_messages[-1].content[-1].content.split(":")
        if weather_question[0] == "WEATHER":
            print("about to send back: ", weather_question[1])
            await gateway.publish_to(weather_question[1].encode(), src)

def main():
    parser = argparse.ArgumentParser(description="Command line client for message passing.")
    parser.add_argument("-g", "--gateway", type=str, help="Gateway address.", default="http://127.0.0.1:12345")
    args = parser.parse_args()
    asyncio.run(run_agent(args.gateway))

if __name__ == "__main__":
    main()