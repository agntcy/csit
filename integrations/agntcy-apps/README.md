# AGNTCY Apps Integration Tests

## Overview

This repository contains integration tests for the **AGNTCY Applications**. Currently, it focuses on testing the **Marketing Campaign Multi-Agent Software**, which demonstrates email composition and review capabilities using multiple coordinated AI agents.

## Prerequisites

- **Task**: Task runner for executing commands defined in `Taskfile.yml`
- **Poetry**: Python dependency management tool

## Running Tests

To execute the marketing campaign test:

```bash
task run-marketing-campaign
```

This command will:
- Generate the necessary configuration files using environment variables
- Download the Workflow Server Manager (WFSM) binary if needed
- Deploy the marketing campaign workflow
- Run tests for the email composer and reviewer components
- Verify correct workflow server operation

## Environment Variables

Set these environment variables before running the tests:

- **`AZURE_OPENAI_API_KEY`**: Your Azure OpenAI API key
- **`AZURE_OPENAI_ENDPOINT`**: Your Azure OpenAI endpoint
- **`SENDGRID_HOST`**: SendGrid host URL (default: `http://echo-server:80`)

## Repository Structure

- **`Taskfile.yml`**: Defines automation tasks
- **`tools/wfsm_runner.py`**: Validates environment files and launches the wfsm CLI
- **`marketing-campaign/run_marketing_campaign.py`**: Tests marketing campaign functionalities
- **`agentic-apps/`**: Submodule containing the application source code

> **Note**: Check `wfsm` binary compatibility with your system (`x86_64` or `arm64`)

## Additional Commands

Generate configuration files:
``` 
task get-marketing-campaign-cfgs
```

Download WFSM binary:
```
task download:wfsm-bin
```

Initialize submodules:
```
task init-submodules
```
