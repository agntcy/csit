# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

import asyncio
from simple_agentic_app.simple_agentic_app import simple_autogen_app

import argparse
import slim_bindings


async def run_agent(message, address, iterations):
    agent = simple_autogen_app()

    local_organization = "cisco"
    local_namespace = "default"
    local_agent = "autogen"

    remote_organization = "cisco"
    remote_namespace = "default"
    remote_agent = "langchain"

    # create new participant object
    participant = await slim_bindings.Slim.new(local_organization, local_namespace, local_agent)

    # Connect to remote slim server
    print(f"connecting to: {address}")
    _ = await participant.connect({"endpoint": address, "tls": {"insecure": True}})

    # Get the local agent instance from env
    instance = "autogen_instance"

    async with participant:
        if message:
            # Create a route to the remote ID
            await participant.set_route(remote_organization, remote_namespace, remote_agent)

            # create a session
            session = await participant.create_session(
                slim_bindings.PySessionConfiguration.RequestResponse()
            )

            for i in range(0, iterations):
                try:
                    # Send the message
                    await participant.publish(
                        session,
                        message.encode(),
                        remote_organization,
                        remote_namespace,
                        remote_agent,
                    )
                    print(f"{instance} sent:", message)

                    # Wait for a reply
                    session_info, msg = await participant.receive(session=session.id)
                    print(
                        f"{instance.capitalize()} received (from session {session_info.id}):",
                        f"{msg.decode()}",
                    )
                except Exception as e:
                    print("received error: ", e)

                await asyncio.sleep(1)
        else:
            # Wait for a message and reply in a loop
            while True:
                session_info, _ = await participant.receive()
                print(
                    f"{instance.capitalize()} received a new session:",
                    f"{session_info.id}",
                )

                async def background_task(session_id):
                    while True:
                        # Receive the message from the session
                        session, msg = await participant.receive(session=session_id)
                        print(
                            f"{instance.capitalize()} received (from session {session_id}):",
                            f"{msg.decode()}",
                        )

                        # handle received messages
                        result = await agent.initate_chat(msg)
                        print(result)

                        # process response
                        result.inner_messages
                        weather_question = result.inner_messages[-1].content[-1].content.split(":")
                        if weather_question[0] == "WEATHER":
                            await participant.publish_to(session, weather_question[1].encode())
                            print(f"{instance.capitalize()} replies:", weather_question[1])

                asyncio.create_task(background_task(session_info.id))


async def main():
    parser = argparse.ArgumentParser(description="Command line client for message passing.")
    parser.add_argument("-s", "--slim", type=str, help="Slim address.", default="http://127.0.0.1:12345")
    parser.add_argument("-m", "--message", type=str, help="Message to send.")
    parser.add_argument("-i", "--iterations",type=int,help="Number of messages to send, one per second.", default=1)
    args = parser.parse_args()
    await run_agent(args.message, args.slim, args.iterations)


if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        print("Program terminated by user.")