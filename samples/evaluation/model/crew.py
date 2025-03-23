# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

from langchain_openai import ChatOpenAI, AzureChatOpenAI
from crewai import Agent, Task, Crew
from crewai.project import crew
from datetime import date
from os import environ
from langchain_community.tools import DuckDuckGoSearchRun

### for observability
from opentelemetry.sdk.trace import TracerProvider

from openinference.instrumentation.crewai import CrewAIInstrumentor
from openinference.instrumentation.langchain import LangChainInstrumentor
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.trace.export import SimpleSpanProcessor

search_tool = DuckDuckGoSearchRun()

today = date.today()
current_date = today.strftime("%B %d, %Y")

telemetry_endpoint = environ.get('TELEMETRY_ENDPOINT')
if (telemetry_endpoint is not None):
  # defualt setting for local Agntcy Server
  # telemetry_endpoint="http://127.0.0.1:6006/v1/traces"
  trace_provider = TracerProvider()
  trace_provider.add_span_processor(SimpleSpanProcessor(OTLPSpanExporter(telemetry_endpoint)))
  CrewAIInstrumentor().instrument(tracer_provider=trace_provider)
  LangChainInstrumentor().instrument(tracer_provider=trace_provider)

azure_openai_api_key = environ.get("AZURE_OPENAI_API_KEY")
azure_openai_endpoint = environ.get("AZURE_OPENAI_ENDPOINT")
openai_api_version = environ.get("AZURE_OPENAI_API_VERSION", "2025-02-01-preview")
azure_deployment_name = environ.get("AZURE_DEPLOYMENT_NAME", "gpt-4o-mini")
azure_model_version = environ.get("AZURE_MODEL_VERSION", "gpt-4o-mini")

local_model_name = environ.get("LOCAL_MODEL_NAME", "llama3.1")
local_base_url = environ.get("LOCAL_MODEL_BASE_URL", "http://localhost:11434/v1/")

if (azure_openai_api_key is not None):
  llm = AzureChatOpenAI(
    azure_deployment=azure_deployment_name,
    model=azure_model_version,
    api_version=openai_api_version,
  )
else:
  llm = ChatOpenAI(
    model=local_model_name,
    base_url=local_base_url,
    api_key="NA",
  )

researcher = Agent(
  role="researcher",
  goal="Uncoverer very detailed information about new CVEs that came out today",
  backstory="You are a Top CyberSecurity researcher tasked with finding highly detailed information about new CVEs",
  verbose=True,
  allow_delegation=False,
  tools=[search_tool],
  llm=llm,
)

research_task = Task(
  description="Gather data about new Critical CVEs from todays date " + current_date,
  agent=researcher,
  expected_output="A bulleted list of todays CVEs",
  tools=[search_tool],
)

crew = Crew(
  agents=[researcher],
  tasks=[research_task],
  verbose=True,
)

def run_crew():
   return crew.kickoff()

if __name__ == "__main__":
  print(llm)
  result = run_crew()
  print("----")
  print(result.raw)
  print("----")
