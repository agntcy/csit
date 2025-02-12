# SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
# SPDX-License-Identifier: Apache-2.0

from os import environ

from langchain_community.tools import DuckDuckGoSearchRun
from langchain_core.tools import tool
from langchain_openai import AzureChatOpenAI
from langgraph.graph import END, START, MessagesState, StateGraph
from langgraph.prebuilt import ToolNode

azure_openai_api_key = environ.get("AZURE_OPENAI_API_KEY")
azure_openai_endpoint = environ.get("AZURE_OPENAI_ENDPOINT")
openai_api_version = environ.get(
    "AZURE_OPENAI_API_VERSION", "2025-02-01-preview"
)
azure_deployment_name = environ.get("AZURE_DEPLOYMENT_NAME", "gpt-4o-mini")
azure_model_version = environ.get("AZURE_MODEL_VERSION", "gpt-4o-mini")


llm = AzureChatOpenAI(
    azure_deployment=azure_deployment_name,
    model=azure_model_version,
    api_version=openai_api_version,
)

search = DuckDuckGoSearchRun()


@tool
def get_weather(location: str):
    """Call to get the current weather."""
    return search.invoke(f"What is the weather in {location}?")


@tool
def get_coolest_cities():
    """Get a list of coolest cities"""
    return "nyc, sf"


class SIMPLE_WEATHER_AGENT_WITH_TOOLS:
    def __init__(self):
        self.tools = [get_weather, get_coolest_cities]
        self.model_with_tools = llm.bind_tools(self.tools)
        self.tool_node = ToolNode(self.tools)

        self.workflow = StateGraph(MessagesState)

        # Define the two nodes we will cycle between
        self.workflow.add_node("agent", self.call_model)
        self.workflow.add_node("tools", self.tool_node)

        self.workflow.add_edge(START, "agent")
        self.workflow.add_conditional_edges(
            "agent", self.should_continue, ["tools", END]
        )
        self.workflow.add_edge("tools", "agent")

        self.app = self.workflow.compile()

    def should_continue(self, state: MessagesState):
        messages = state["messages"]
        last_message = messages[-1]
        if last_message.tool_calls:
            return "tools"
        return END

    def call_model(self, state: MessagesState):
        messages = state["messages"]
        response = self.model_with_tools.invoke(messages)
        return {"messages": [response]}

    def call(self, msg: str) -> None:
        messages = []
        for chunk in self.app.stream(
            {"messages": [("human", msg)]},
            stream_mode="values",
        ):
            messages.append(chunk["messages"][-1])
        last_message = messages[-1].content
        if len(last_message) > 0:
            return last_message
        else:
            return "Error: Could not obtain the weather."


# agent = SIMPLE_WEATHER_AGENT_WITH_TOOLS()
# while True:
#     try:
#         user_input = input("User: ")
#         if user_input.lower() in ["quit", "exit", "q"]:
#             print("Goodbye!")
#             break

#         print(agent.call(user_input))
#     except:
#         break
