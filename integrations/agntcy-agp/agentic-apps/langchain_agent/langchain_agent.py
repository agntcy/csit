# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

import asyncio

from simple_weather_agent.simple_weather_agent import (
    SIMPLE_WEATHER_AGENT_WITH_TOOLS,
)

import argparse
import agp_bindings


async def run_agent(message, address, iterations):
    agent = SIMPLE_WEATHER_AGENT_WITH_TOOLS()

    local_organization = "cisco"
    local_namespace = "default"
    local_agent = "langchain"

    remote_organization = "cisco"
    remote_namespace = "default"
    remote_agent = "autogen"

    # create new gateway object
    gateway = await agp_bindings.Gateway.new(local_organization, local_namespace, local_agent)

    # Connect to remote gateway server
    print(f"connecting to: {address}")
    _ = await gateway.connect({"endpoint": address, "tls": {"insecure": True}})

    # Get the local agent instance from env
    instance = "langchain_instance"

    async with gateway:
        if message:
            # Create a route to the remote ID
            await gateway.set_route(remote_organization, remote_namespace, remote_agent)

            # create a session
            session = await gateway.create_session(
                agp_bindings.PySessionConfiguration.FireAndForget()
            )

            for i in range(0, iterations):
                try:
                    # Send the message
                    await gateway.publish(
                        session,
                        message.encode(),
                        remote_organization,
                        remote_namespace,
                        remote_agent,
                    )
                    print(f"{instance} sent:", message)

                    # Wait for a reply
                    session_info, msg = await gateway.receive(session=session.id)
                    print(
                        f"{instance.capitalize()} received (from session {session_info.id}):",
                        f"{msg.decode()}",
                    )
                except Exception as e:
                    print("received error: ", e)

                # handle received messages
                result = agent.call(msg)
                print(result)

                await asyncio.sleep(1)
        else:
            # Wait for a message and reply in a loop
            while True:
                session_info, _ = await gateway.receive()
                print(
                    f"{instance.capitalize()} received a new session:",
                    f"{session_info.id}",
                )

                async def background_task(session_id):
                    while True:
                        # Receive the message from the session
                        session, msg = await gateway.receive(session=session_id)
                        print(
                            f"{instance.capitalize()} received (from session {session_id}):",
                            f"{msg.decode()}",
                        )

                asyncio.create_task(background_task(session_info.id))


async def main():
    parser = argparse.ArgumentParser(description="Command line client for message passing.")
    parser.add_argument("-m", "--message", type=str, help="Message to send.")
    parser.add_argument("-g", "--gateway", type=str, help="Gateway address.", default="http://127.0.0.1:12345")
    parser.add_argument("-i", "--iterations",type=int,help="Number of messages to send, one per second.", default=1)
    args = parser.parse_args()
    await run_agent(args.message, args.gateway, args.iterations)

if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        print("Program terminated by user.")