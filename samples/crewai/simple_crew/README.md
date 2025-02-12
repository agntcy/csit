# Simple Crew

Simple Crew is the base use case in CrewAI, where a crew collaborates to write a report on a given topic. The crew consists of two agents:

- **Researcher Agent**: Responsible for gathering data and information relevant to the topic.
- **Writer Agent**: Tasked with compiling the gathered data into a coherent and comprehensive report.

The base use case has been extended to report metrics including token usage, task duration and task scores.

## Usage

### Set the `.env` variable

```
AZURE_MODEL=gpt-4o-mini
AZURE_OPENAI_API_KEY=XXX
AZURE_OPENAI_API_VERSION=2025-02-01-preview
AZURE_OPENAI_ENDPOINT=https://your-azure-openai-endpont
AZURE_OPENAI_DEPLOYMENT_NAME=gpt-4o-mini
```

### Run the app

```sh
task run:crew
```
