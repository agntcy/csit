# CSIT - Continuous System Integration Testing

- [CSIT - Continuous System Integration Testing](#csit---continuous-system-integration-testing)
  - [Architecture](#architecture)
  - [Tasks](#tasks)
- [Integration tests](#integration-tests)
  - [Directory structure](#directory-structure)
  - [Running tests](#running-tests)
  - [A2A interoperability smoke tests](#a2a-interoperability-smoke-tests)
  - [Running tests using GitHub actions](#running-tests-using-github-actions)
  - [How to extend tests with your own test](#how-to-extend-tests-with-your-own-test)
  - [Updating the agntcy/dir testdata](#updating-the-agntcydir-testdata)
  - [Copyright Notice](#copyright-notice)

## Architecture

Agncty CSIT system design needs to meet continuously expanding requirements of
Agntcy projects including Slim Protocol, Agent Directory and many more.

The directory structure of the CSIT:

```
csit
├── benchmarks                                    # Benchmark tests
│   ├── agntcy-slim                                # Benchmark tests for Slim
│   │   ├── Taskfile.yml                          # Tasks for Slim benchmark tests
│   │   └── tests
│   ├── agntcy-dir                                # Benchmark tests for ADS
│   │   ├── Taskfile.yml                          # Tasks for ADS benchmark tests
│   │   └── tests
│   ├── go.mod
│   ├── go.sum
│   └── Taskfile.yml
├── integrations                                  # Integration tests
│   ├── agntcy-a2a                                 # Integration tests for Rust/Go A2A interoperability
│   │   ├── fixtures
│   │   ├── Taskfile.yml                          # Tasks for A2A interoperability tests
│   │   └── tests
│   ├── agntcy-slim                                # Integration tests for [agntcy/slim](https://github.com/agntcy/slim)
│   │   ├── agentic-apps
│   │   ├── Taskfile.yml                          # Tasks for Slim integration tests
│   │   └── tests
│   ├── agntcy-apps                               # Integration tests for ([agntcy/agentic-apps](https://github.com/agntcy/agentic-apps))
│   │   ├── agentic-apps
│   │   ├── Taskfile.yml                          # Tasks for agentic-apps integration tests
│   │   └──  tools
│   ├── agntcy-dir                                # Integration tests for [agntcy/dir](https://github.com/agntcy/dir)
│   │   ├── components
│   │   ├── examples
│   │   ├── manifests
│   │   ├── Taskfile.yml                          # Tasks for ADS integration tests
│   │   └── tests
│   ├── environment                               # Test environment helpers
│   │   └── kind
│   ├── Taskfile.yml                              # Tasks for integration tests
│   └── testutils                                 # Go test utils
└── Taskfile.yml                                  # Repository level task definitions
```

In the Taskfiles, all required tasks and steps are defined in a structured manner. Each CSIT component contains its necessary tasks within dedicated Taskfiles, with higher-level Taskfiles incorporating lower-level ones to efficiently leverage their defined tasks.

## Tasks

You can list all the task defined in the Taskfiles using the `task -l` or simply run `task`.
The following tasks are defined:

```bash
task: Available tasks for this project:
* benchmarks:directory:test:                              All ADS benchmark test
* benchmarks:slim:test:                                All Slim benchmark test
* integrations:a2a:test:                                 All A2A interoperability tests
* integrations:a2a:test:go-dotnet:                       Go and C# interoperability tests
* integrations:a2a:test:python-go:                       Python and Go interoperability tests
* integrations:a2a:test:python-dotnet:                   Python and C# interoperability tests
* integrations:a2a:test:rust-python:                     Rust and Python interoperability tests
* integrations:a2a:test:rust-go:jsonrpc:                 Rust and Go JSON-RPC interoperability smoke test
* integrations:a2a:test:rust-go:jsonrpc:go-go:           Go client to Go server JSON-RPC interoperability test
* integrations:a2a:test:rust-go:jsonrpc:go-rust:         Go client to Rust server JSON-RPC interoperability test
* integrations:a2a:test:rust-go:jsonrpc:rust-go:         Rust client to Go server JSON-RPC interoperability test
* integrations:a2a:test:rust-go:jsonrpc:rust-rust:       Rust client to Rust server JSON-RPC interoperability test
* integrations:apps:download:wfsm-bin:                    Get wfsm binary from GitHub
* integrations:apps:get-marketing-campaign-cfgs:          Populate marketing campaign config file
* integrations:apps:init-submodules:                      Initialize submodules
* integrations:apps:run-marketing-campaign:               Run marketing campaign
* integrations:directory:download:dirctl-bin:             Get dirctl binary from GitHub
* integrations:directory:test:                            All directory test
* integrations:directory:test-env:bootstrap:deploy:       Deploy Directory network peers
* integrations:directory:test-env:cleanup:                Remove agntcy directory test env
* integrations:directory:test-env:deploy:                 Deploy Agntcy directory test env
* integrations:directory:test-env:network:cleanup:        Remove Directory network peers
* integrations:directory:test-env:network:deploy:         Deploy Directory network peers
* integrations:directory:test:compiler:                   Agntcy compiler test
* integrations:directory:test:delete:                     Directory agent delete test
* integrations:directory:test:list:                       Directory agent list test
* integrations:directory:test:networking:                 Directory agent networking test
* integrations:directory:test:push:                       Directory agent push test
* integrations:slim:build:agentic-apps:                   Build agentic containers
* integrations:slim:cert-manager:deploy:                  Deploy cert-manager
* integrations:slim:cert-manager:remove:                  Remove cert-manager
* integrations:slim:certificates:create:                  Create certificates
* integrations:slim:spire:deploy:                         Deploy SPIRE server
* integrations:slim:spire:remove:                         Remove SPIRE server
* integrations:slim:test-env:cleanup:                     Remove agent slim test env
* integrations:slim:test-env:cleanup:contoroller:         Remove slim controller test env
* integrations:slim:test-env:cleanup:generated:           Undeploy agntcy slim test env for each values file in config/.generated
* integrations:slim:test-env:deploy:                      Deploy agntcy slim test env
* integrations:slim:test-env:deploy:controller:           Deploy slim controller
* integrations:slim:test-env:deploy:generated:            Deploy agntcy slim test env for each values file in config/.generated
* integrations:slim:test-env:generate:configs:            Generates test environment configuration(s) for agntcy slim based on the provided test-setup descriptor
* integrations:slim:test:mcp-server:                      Test MCP over Slim
* integrations:slim:test:mcp-server:mcp-proxy:            Test MCP server via MCP proxy
* integrations:slim:test:mcp-server:slim-native:          Test Slim native MCP server
* integrations:slim:test:sanity:                          Sanity slim test
* integrations:slim:test:slimctl-download:                Download slimctl executable for current OS and architecture
* integrations:slim:test:topology:                        Slim topology test
* integrations:kind:create:                               Create kind cluster
* integrations:kind:destroy:                              Destroy kind cluster
* integrations:version:                                   Get version
```

# Integration tests

> Focuses on testing interactions between integrated components.

## Directory structure

Inside csit integrations directory contains the tasks that creating the test
environment, deploying the components that will be tested, and running the tests.

```
├── agntcy-slim                                # Integration tests for [agntcy/slim](https://github.com/agntcy/slim)
│   ├── agentic-apps
│   ├── Taskfile.yml                          # Tasks for Slim integration tests
│   └── tests
├── agntcy-apps                               # Integration tests for ([agntcy/agentic-apps](https://github.com/agntcy/agentic-apps))
│   ├── agentic-apps
│   ├── Taskfile.yml                          # Tasks for agentic-apps integration tests
│   └──  tools
├── agntcy-dir                                # Integration tests for [agntcy/dir](https://github.com/agntcy/dir)
│   ├── components
│   ├── examples
│   ├── manifests
│   ├── Taskfile.yml                          # Tasks for ADS integration tests
│   └── tests
├── environment                               # Test environment helpers
│   └── kind
├── Taskfile.yml                              # Tasks for integration tests
└── testutils                                 # Go test utils
```

## Running tests

We can launch tests using taskfile locally or in GitHub actions.
Some suites are self-contained and run directly on the host, while others need
a Kubernetes-based test environment.

Suites that deploy components on Kubernetes require creating a test cluster and
deploying the test environment before running the tests.
They require the following tools to be installed on the local machine:
  - [Taskfile](https://taskfile.dev/installation/)
  - [Go](https://go.dev/doc/install)
  - [Docker](https://docs.docker.com/get-started/get-docker/)
  - [Kind](https://kind.sigs.k8s.io/docs/user/quick-start#installation)
  - [Kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl)
  - [Helm](https://helm.sh/docs/intro/install/)

```bash
task integrations:kind:create
task integrations:directory:test-env:deploy
task integrations:directory:test
```

We can focus on specified tests:
```bash
task integrations:directory:test:compiler
```

After we finish the tests we can destroy the test cluster
```bash
task integratons:kind:destroy
```

## A2A interoperability smoke tests

The `integrations/agntcy-a2a` suite is self-contained and does not require a
Kind cluster, Helm, or repository sibling checkouts. It builds and runs small
Go, Rust, .NET, and Python fixtures locally across these suite slices:

| SDK pair | JSON-RPC | HTTP+JSON | gRPC | Root task |
| --- | --- | --- | --- | --- |
| Rust/Go | Yes | Yes | Yes | `task integrations:a2a:test:rust-go` |
| Rust/.NET | Yes | Yes | No | `task integrations:a2a:test:rust-dotnet` |
| Go/.NET | Yes | Yes | No | `task integrations:a2a:test:go-dotnet` |
| Python/Go | Yes | Yes | Yes | `task integrations:a2a:test:python-go` |
| Rust/Python | Yes | Yes | Yes | `task integrations:a2a:test:rust-python` |
| Python/.NET | Yes | Yes | No | `task integrations:a2a:test:python-dotnet` |

To run it from the repository root you need:

- [Taskfile](https://taskfile.dev/installation/)
- [Rust and Cargo](https://www.rust-lang.org/tools/install)
- [Go](https://go.dev/doc/install) for the Rust/Go, Python/Go, and Go/.NET slices
- Python 3.10+ for the Python/Go, Rust/Python, and Python/.NET slices
- .NET 8 SDK for the Rust/.NET, Go/.NET, and Python/.NET slices

```bash
task integrations:a2a:test
task integrations:a2a:test:rust-go:jsonrpc
task integrations:a2a:test:go-dotnet
task integrations:a2a:test:python-go:grpc
task integrations:a2a:test:python-go
task integrations:a2a:test:rust-python
task integrations:a2a:test:python-dotnet
task integrations:a2a:test:rust-python:jsonrpc:python-rust
```

The suite writes Ginkgo JSON and JUnit reports under `integrations/agntcy-a2a/reports/`.


## Running tests using GitHub actions

We can run integration test using Github actions using `gh` command line tool or using the GitHub web UI

```bash
gh workflow run test-integrations -f testenv=kind
```

If we want to run the tests on a specified branch

```bash
gh workflow run test-integrations --ref feat/integration/deploy-agent-directory -f testenv=kind
```


## How to extend tests with your own test

Contributing your own tests to our project is a great way to improve the robustness and coverage of our testing suite. Follow these steps to add your tests.

1. Fork and Clone the Repository

Fork the repository to your GitHub account.
Clone your fork to your local machine.

```bash
git clone https://github.com/your-username/repository.git
cd repository
```

2. Create a New Branch

Create a new branch for your test additions to keep your changes organized and separate from the main codebase.


```bash
git checkout -b add-new-test
```

3. Navigate to the Integrations Directory

Locate the integrations directory where the test components are organized.

```bash
cd integrations
```

4. Add Your Test

Create a new sub-directory for your test if necessary, following the existing structure. For example, integrations/new-component.
Add all necessary test files, such as scripts, manifests, and configuration files.

5. Update Taskfile

Modify the Taskfile.yaml to include tasks for deploying and running your new test.

```yaml
tasks:
  test:env:new-component:deploy:
    desc: Desription of deployig new component elements
    cmds:
      - # Command for deploying your components if needed

  test:env:new-component:cleanup:
    desc: Desription of cleaning up component elements
    cmds:
      - # Command for cleaning up your components if needed

  test:new-component:
    desc: Desription of the test
    cmds:
      - # Commands to set up and run your test
```

6. Test Locally

Before pushing your changes, test them locally to ensure everything works as expected.

```bash
task integrations:kind:create
task integrations:new-componet:test-env:deploy
task integrations:new-component:test
task integrations:new-componet:test-env:cleanup
task integrations:kind:destroy
```

7. Document Your Test

Update the documentation in the docs folder to include details about your new test. Explain the purpose of the test, any special setup instructions, and how it fits into the overall testing strategy.

8. Commit and Push Your Changes

Commit your changes with a descriptive message and push them to your fork.

```bash
git add .
git commit -m "feat: add new test for component X"
git push origin add-new-test
```

9. Submit a Pull Request

Go to the original repository on GitHub and submit a pull request from your branch.
Provide a detailed description of what your test covers and any additional context needed for reviewers.

## Updating the agntcy/dir testdata

If we want to update the `integrations/agntcy-dir/examples/dir/e2e/testdata` directory we will need to add `agntcy/dir` as a remote and create a patch for it by diffing with the `agntcy/dir` repo

```bash
# add agntcy/dir as remote
git remote add -f dir https://github.com/agntcy/dir.git
# fetch dir
git fetch dir
# example of updating the integrations/agntcy-dir/examples/dir/e2e/testdata directory to the agntcy/dir main
git diff --binary HEAD:integrations/agntcy-dir/examples/dir/e2e/testdata dir/main:e2e/testdata | git apply --directory=integrations/agntcy-dir/examples/dir/e2e/testdata
```

## Copyright Notice

[Copyright Notice and License](./LICENSE.md)

Distributed under Apache 2.0 License. See LICENSE for more information.
Copyright AGNTCY Contributors (https://github.com/agntcy)
