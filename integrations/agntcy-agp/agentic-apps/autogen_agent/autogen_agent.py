# SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
# SPDX-License-Identifier: Apache-2.0

import asyncio
# import phoenix
from simple_agentic_app.simple_agentic_app import simple_autogen_app

import argparse
import gateway_bindings

phoenix = gateway_bindings.Phoenix()

async def run_agent(gateway):
    agent = simple_autogen_app()

    # register local agent
    await phoenix.create_agent("cisco", "default", "autogen")

    # connect to gateway server
    await phoenix.connect(gateway)

    while True:
        # receive messages
        src, msg = await phoenix.receive()

        # handle received messages
        result = await agent.initate_chat(msg)
        print(result)

        # process response
        result.inner_messages
        weather_question = result.inner_messages[-1].content[-1].content.split(":")
        if weather_question[0] == "WEATHER":
            print("about to send back: ", weather_question[1])
            await phoenix.publish_to(weather_question[1].encode(), src)

def main():
    parser = argparse.ArgumentParser(description="Command line client for message passing.")
    parser.add_argument("-g", "--gateway", type=str, help="Gateway address.", default="http://127.0.0.1:12345")
    args = parser.parse_args()
    asyncio.run(run_agent(args.gateway))

if __name__ == "__main__":
    main()