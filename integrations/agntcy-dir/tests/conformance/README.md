<!--
Copyright AGNTCY Contributors (https://github.com/agntcy)
SPDX-License-Identifier: Apache-2.0
-->

# Directory conformance — configuration matrix

This suite runs the upstream Directory e2e **client** test package
(`github.com/agntcy/dir/tests/e2e/client`, pinned by version) against many **deployment
configurations** of `dir`. Each configuration is a self-contained folder under
[`server/<name>/`](./server) that stands up a differently-configured `dir-ctl daemon`.

There are two matrices, both driven from [`Taskfile.yml`](./Taskfile.yml):

| Matrix | Var | Purpose |
| --- | --- | --- |
| Cross-version | `CONFORMANCE_SERVER_VERSIONS` × `CONFORMANCE_CLIENT_VERSIONS` | Pinned client suite against released server images (packaging / backward-compat). |
| Configuration | `CONFORMANCE_SERVER_CONFIGS` (client `CONFORMANCE_CONFIG_CLIENT_VERSION`) | One client version against many deployment configurations of `dir`. |

Run everything and build `summary.html`:

```bash
task integrations:directory:tests:client-server:test:all
```

Run only the configuration matrix, or a single profile:

```bash
task integrations:directory:tests:client-server:configs:all
task integrations:directory:tests:client-server:test SERVER_VERSION=postgres-zot CLIENT_VERSION=v1.6.1
```

## Reporting & result collection

Each cell writes a Ginkgo `Server-<server>-Client-<client>.{json,xml}` report into
[`reports/`](./reports); `report_dashboard.go` renders them into a single `summary.html` with one
column per cell.

- **Locally**, `test:all` / `configs:all` run the cells and build the dashboard in one shot.
- **In CI** ([`test-directory-conformance.yaml`](../../../../../.github/workflows/test-directory-conformance.yaml)),
  the cells are fanned out into a **parallel job matrix** (source of truth: the `matrix:list` task):
  1. `matrix` — emits the cell list as JSON;
  2. `run` — one job per cell (`fail-fast: false`), each uploading its own report artifact
     (`directory-conformance-report-<server>-<client>`);
  3. `collect-report` — merges every cell's report, runs the `report` task to build `summary.html`,
     and uploads it as `directory-conformance-test-result`;
  4. `publish-pages` — publishes the combined dashboard to GitHub Pages (on `main`).

Rebuild the dashboard from existing reports without re-running tests:

```bash
task integrations:directory:tests:client-server:report
```

## Configuration profiles (base version v1.6.1)

| Profile | DB | OCI store | AuthN | Notes |
| --- | --- | --- | --- | --- |
| `v1.6.1` | sqlite | embedded local-fs | insecure | Baseline (default deployment shape). |
| `sqlite-zot` | sqlite | external Zot | insecure | `store.oci.registry_address` → Zot sidecar, no `local_dir`. |
| `postgres-oci` | postgres | embedded local-fs | insecure | Postgres sidecar. |
| `postgres-zot` | postgres | external Zot | insecure | Postgres + Zot sidecars. |
| `x509-spire` | sqlite | embedded local-fs | **x509-SVID** | SPIRE server/agent; client authenticates with a SPIFFE SVID. |

> **Note on DB defaults:** at v1.6.x the default `database.type` is `postgres`, so sqlite profiles
> set `database.type: sqlite` explicitly.

### Coverage note vs. `dir`'s own e2e

`dir`'s own suite tests code at HEAD; this suite tests **released artifacts across versions**.
The storage/authN profiles above overlap with `dir`'s own e2e (which is intentional — released-image
coverage), except the postgres-**embedded-OCI** pairing, which `dir` e2e never runs.

**Signing is deliberately not covered here.** The client suite this harness runs has no signing
tests; signing lives in `dir`'s `local` suite (key-based only), and OIDC-keyless / KMS (incl.
`awskms://`) are tested nowhere upstream. Signing coverage — including the AWS secret-store path —
is tracked as an **upstream contribution to the `dir` repo**, where the sign harness lives.

## Adding a new configuration profile

1. Create `server/<name>/` with:
   - `docker-compose.yaml` — the `dir-ctl daemon` service (image `ghcr.io/agntcy/dir-ctl:v1.6.1`)
     plus any sidecars (postgres, zot, ...). Template from an existing profile.
   - `dir-daemon-config.yaml` — authored against the **v1.6.x** schema (see
     `dir`'s `tests/e2e/*/testenv/*/dir-daemon-config.yaml` and `install/charts/dir/apiserver/values.yaml`).
   - `Taskfile.yml` — the standard `up`/`down` (copy from another compose-only profile).
   - `client-config.yaml` — client-side knobs for this profile (auth mode, SPIFFE socket, ...).
     Resolved automatically in preference to the version-keyed `client/<version>/test-config.yaml`.
   - *(optional)* an empty `needs-overlay` marker file — only for profiles that require the
     legacy `common.go` source overlay (`patches/e2e-shared-utils-common.go`). v1.6.1+ profiles
     support SPIFFE natively via `client-config.yaml` and must **not** carry this marker.
2. Register it in the `includes:` block of [`Taskfile.yml`](./Taskfile.yml).
3. Add its name to `CONFORMANCE_SERVER_CONFIGS`.
