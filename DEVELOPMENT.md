# Development

`DEVELOPMENT.md` is the right place in this repository for contributor and maintainer workflow. `CONTRIBUTING.md` is usually better when you want policy for external contributions, while this file is focused on building, verifying, and maintaining the project itself.

## Requirements

- Go 1.25.x
- Cisco Intersight OAuth client credentials if you want to exercise live reads or writes
- A local checkout of this repository with the pinned raw spec in `third_party/intersight/openapi/raw/openapi.json`

## Build

From the repository root:

```bash
make generate
make build
```

`make build` writes binaries for the default target set to `bin/`:

- `bin/intersight-mcp-darwin-amd64`
- `bin/intersight-mcp-darwin-arm64`
- `bin/intersight-mcp-linux-amd64`
- `bin/intersight-mcp-linux-arm64`
- `bin/intersight-mcp-windows-amd64.exe`

Override the target matrix when needed:

```bash
make build BUILD_TARGETS="linux/amd64 windows/amd64"
```

## Verify

Run the full local verification harness with:

```bash
make verify
```

`make verify` is a thin wrapper around:

```bash
make generate
GOCACHE=$PWD/.cache/go-build GOTMPDIR=$PWD/.tmp go test ./...
make build
```

Use `make generate` instead of bare `go generate ./...` when your environment cannot write to the default Go build cache path.

The tests include a local `httptest.Server` fake Intersight surface for auth bootstrap, collection reads, object reads, and mutate writes, plus a manual clock used by OAuth refresh and degraded-mode tests.

## Runtime Notes

Search execution reuses immutable embedded artifacts prepared at startup, but it does not pool QuickJS runtimes. Each `search` call executes in a fresh runtime.

Additional operational tuning flags exist beyond the end-user setup documented in [README.md](/Users/mimaurer/Documents/GitHub/intersight-mcp/README.md):

| Setting | Flag | Environment | Default |
|---|---|---|---|
| Global execution timeout | `--timeout` | `INTERSIGHT_TIMEOUT` | `40s` |
| Search execution timeout | `--search-timeout` | `INTERSIGHT_SEARCH_TIMEOUT` | `15s` |
| Per-call HTTP/bootstrap timeout | `--per-call-timeout` | `INTERSIGHT_PER_CALL_TIMEOUT` | `15s` |
| Max API calls per execution | `--max-api-calls` | `INTERSIGHT_MAX_API_CALLS` | `250` |
| Max concurrent tool executions | `--max-concurrent` | `INTERSIGHT_MAX_CONCURRENT` | `25` |
| Max submitted code size | `--max-code-size` | `INTERSIGHT_MAX_CODE_SIZE` | `100KB` |
| QuickJS memory limit | `--wasm-memory` | `INTERSIGHT_WASM_MEMORY` | `64MB` |
| Log level | `--log-level` | `INTERSIGHT_LOG_LEVEL` | `info` |
| Include submitted tool code in debug logs with best-effort redaction | `--unsafe-log-full-code` | `INTERSIGHT_UNSAFE_LOG_FULL_CODE` | `false` |
| Mirror structured content into text for legacy clients | `--legacy-content-mirror` | `INTERSIGHT_LEGACY_CONTENT_MIRROR` | `false` |

`--max-concurrent` is a shared process-wide limiter across `search`, `query`, and `mutate`. The default limit is `25` in-flight tool executions.

`--unsafe-log-full-code` and `INTERSIGHT_UNSAFE_LOG_FULL_CODE` are break-glass debugging options. When enabled together with `--log-level debug`, the server logs submitted tool code with best-effort redaction for bearer tokens, client secrets, and similar values, then emits a startup warning. Use this only temporarily on trusted machines; normal debug logging already captures code hashes, execution metadata, API call traces, and error details without storing submitted code.

## Spec Update Workflow

When the pinned Cisco OpenAPI input, the generator, or core dependencies change:

1. Replace `third_party/intersight/openapi/raw/openapi.json` with the new pinned raw spec.
2. Update `third_party/intersight/openapi/manifest.json` so the published version, source URL, SHA-256, and retrieval date match the new raw file.
3. Review `spec/filter.yaml` and adjust denylist entries only when there is an intentional routing change.
4. Regenerate the embedded normalized artifact with `make generate`.
5. Run the full verification harness with `make verify`.
6. Commit the updated raw spec, manifest, and regenerated `generated/spec_resolved.json`, `generated/sdk_catalog.json`, `generated/rules.json`, and `generated/search_catalog.json` together.

Do not fetch specs implicitly as part of build or generate. The workflow is local-only and reproducible from repository state.
