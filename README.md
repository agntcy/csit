# CSIT: Continuous System Integration Testing

CSIT is the continuous system integration testing repository of the
[AGNTCY](https://github.com/agntcy) projects, including SLIM, Directory and A2A
interoperability. It runs interoperability, integration and conformance suites
and benchmarks, locally and in GitHub Actions, through shared
[Taskfile](https://taskfile.dev/) tasks.

## Documentation

All CSIT documentation lives in **[docs/csit.md](docs/csit.md)**. It is the
single source of truth for:

- the repository architecture and layout
- the task system and how to list and run tasks
- every test suite, its prerequisites and how to run it
- reports, dashboards and GitHub Actions workflows (including manual dispatch)
- how to add your own test

## Quick start

Install [Taskfile](https://taskfile.dev/installation/) and
[Go](https://go.dev/doc/install), then:

```bash
task -l                       # list available tasks
task integrations:a2a:test    # example: run the A2A interoperability suite
```

See [docs/csit.md](docs/csit.md) for the per-suite requirements.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) and the
[Adding a test](docs/csit.md#adding-a-test) section of the documentation.

## Copyright Notice

[Copyright Notice and License](./LICENSE.md)

Distributed under Apache 2.0 License. See LICENSE for more information.
Copyright AGNTCY Contributors (https://github.com/agntcy)
