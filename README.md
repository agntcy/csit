# CSIT - Continuous System Integration Testing

- [CSIT - Continuous System Integration Testing](#csit---continuous-system-integration-testing)
  - [Architecture](#architecture)
- [Integration tests](#integration-tests)
  - [Directory structure](#directory-structure)
  - [Running tests](#running-tests)
  - [Running tests using GitHub actions](#running-tests-using-github-actions)
  - [Copyright Notice](#copyright-notice)

## Architecture

Agncty CSIT system design needs to meet continuously expanding requirements of
Agntcy projects including Agent Gateway Protocol, Agent Directory and many more.


# Integration tests

> Focuses on testing interactions between integrated components.

## Directory structure

Inside csit integrations directory contains the tasks that creating the test
environment, deploying the components that will be tested, and running the tests.

```
integrations
├── Taskfile.yaml                   # Task definitions
├── docs                            # Documentations
├── environment
│   └── kind                        # kind related manifests
├── agntcy-dir                      # Agent directory related tests, components, etc...
│   ├── components                  # the compontents charts
│   ├── examples                    # the examples that can be used for testing
│   ├── manifests                   # requred manifests for tests
│   └── tests                       # tests
├── agntcy-agp                      # Agent Gateway related tests, components, etc...
│   └── agentic-apps                # Agentic apps for gateway tests
│       ├── autogen_agent
│       └── langchain_agent
└── report                          # tools for reportning test results
```

## Running tests

We can launch tests using taskfile locally or in GitHub actions.
Running locally we need to create a test cluster and deploy the test env on
it before running the tests.

```
cd integrations
task kind:create
task test:env:directory:deploy
task test:directory
```

We can focus on specified tests:
```
task test:directory:compiler
```

After we finish the tests we can destroy the test cluster
```
task kind:destroy
```


## Running tests using GitHub actions

We can run integration test using Github actions using `gh` command line tool or using the GitHub web UI

```
gh workflow run test-integrations -f testenv=kind
```

If we want to run the tests on a specified branch

```
gh workflow run test-integrations --ref feat/integration/deploy-agent-directory -f testenv=kind
```

## Copyright Notice

[Copyright Notice and Licence](./LICENSE.md)

Copyright (c) 2025 Cisco and/or its affiliates.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.