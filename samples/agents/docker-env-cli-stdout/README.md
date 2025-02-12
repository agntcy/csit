# Docker Agent

This project demonstrates how to create a Docker container that runs a Python script capable of interacting with both Azure OpenAI and a local model. The script reads environment variables from a .env file and makes HTTP requests to the specified endpoints.

## Run

### Set the `.env` variable


```
AZURE_MODEL=gpt-4o-mini
AZURE_OPENAI_API_KEY=XXX
AZURE_OPENAI_API_VERSION=2025-02-01-preview
AZURE_OPENAI_ENDPOINT=https://your-azure-openai-endpont
AZURE_OPENAI_DEPLOYMENT_NAME=gpt-4o-mini
LOCAL_MODEL_NAME=llama3.1
LOCAL_MODEL_BASE_URL=http://localhost:11434
```

### Build

To build the Docker image, navigate to the agent folder and run:

```sh
docker build -t docker-agent .
```

### Run

To run the Docker container with the input string and environment variables, use the following command:

```sh
docker run --env-file .env docker-agent "Hello, World"
```