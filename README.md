# Intersight MCP Server

`intersight-mcp` is a local stdio MCP server for Cisco Intersight. It exposes three tools:

- `search` for exploring the generated discovery catalog, including resources, paths, metrics, and normalized schemas through `catalog`
- `query` for read-shaped SDK calls against the Intersight API and offline validation of write-shaped SDK calls without making API calls
- `mutate` for persistent write-shaped SDK calls against the Intersight API

The `search` discovery surface is resource-first: use `catalog.resources`, `catalog.resourceNames`, and `catalog.paths` to move from a resource family or REST path into the grouped operation set for that SDK stem. The public `search` view keeps `resource.operations` minimal: it is an array of supported verbs such as `["list", "get", "create", "update", "delete"]`. Operation defaults are documented once at the tool level instead of repeated on every resource: `create` requires a body, `delete` requires `path.Moid`, `get` requires `path.Moid` and supports standard get query parameters, `list` supports standard list query parameters, and `update` requires both `path.Moid` and a body. Its top-level `createFields` map is filtered to exclude read-only properties so the output stays focused on writable inputs. Use `resource.schema` with `catalog.schema(resource.schema)` for full normalized schema detail. When you need the fully-qualified SDK method from a public resource entry, derive it as `resourceKey + '.' + verb` where `resourceKey` is the parent map key and `verb` comes from `resource.operations`. When both POST-update and PATCH-update variants exist for the same resource, the public view also hides the redundant `post` alias and keeps `update`.

Telemetry is an exception: generated `telemetry.*` resources are intentionally excluded from the OpenAPI-derived SDK surface, and query mode exposes a custom `sdk.telemetry.query(...)` helper for Apache Druid `groupBy` queries. The helper accepts groupBy fields as top-level inputs and injects `queryType: "groupBy"` internally.

The binary is local-only, stdio-only, and configured exclusively through CLI flags plus environment variables. There is no config file support.

## Requirements

- Go 1.25.x
- Cisco Intersight OAuth client credentials
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

The tests include a local `httptest.Server` fake Intersight surface for auth bootstrap, collection reads, object reads, and mutate writes, plus a manual clock used by OAuth refresh/degraded-mode tests.

## Run

Set credentials in the environment when you need live reads or writes, then start the stdio server:

```bash
export INTERSIGHT_CLIENT_ID=your-client-id
export INTERSIGHT_CLIENT_SECRET=your-client-secret
./bin/intersight-mcp-$(go env GOOS)-$(go env GOARCH) serve
```

The process communicates only over stdin/stdout. It does not expose an HTTP listener.

Search execution reuses immutable embedded artifacts prepared at startup, but it does not pool QuickJS runtimes. Each `search` call executes in a fresh runtime.

## Configuration

Supported configuration comes from flags and matching environment variables. Flags take precedence over environment variables.

Credentials required for live `query` reads and `mutate` writes:

- `INTERSIGHT_CLIENT_ID`
- `INTERSIGHT_CLIENT_SECRET`

Optional settings most users might care about:

| Setting | Flag | Environment | Default |
|---|---|---|---|
| Endpoint origin | `--endpoint` | `INTERSIGHT_ENDPOINT` | `https://intersight.com` |
| Explicit outbound proxy URL | `--proxy` | `INTERSIGHT_PROXY_URL` | disabled |
| Max serialized tool payload | `--max-output` | `INTERSIGHT_MAX_OUTPUT` | `512KB` |

The server can still start without credentials so the offline `search` tool remains available. Write-shaped `query` validation also remains available because it runs locally.

`--max-output` applies to the serialized tool payload produced by sandbox execution, before MCP response wrapping. It does not count the duplicated MCP envelope fields on top of that payload.

Proxy configuration:

- The server does not inherit `HTTP_PROXY`, `HTTPS_PROXY`, or `NO_PROXY` from the host environment.
- Outbound OAuth and API traffic uses a proxy only when `--proxy` or `INTERSIGHT_PROXY_URL` is set explicitly.
- Supported proxy URL schemes are `http`, `https`, and `socks5`.

Endpoint validation rules:

- Accepts either a bare host like `intersight.example.com` or an origin-like value
- Bare hosts are normalized to `https://`
- If you provide a scheme explicitly, it must be `https://`
- Must not include user info, a query string, or a fragment
- Must be the origin only; path components are rejected
- OAuth and API base URLs are both derived from that HTTPS origin

Examples:

```bash
INTERSIGHT_CLIENT_ID=... \
INTERSIGHT_CLIENT_SECRET=... \
INTERSIGHT_ENDPOINT=intersight.com \
./bin/intersight-mcp serve
```

```bash
INTERSIGHT_CLIENT_ID=... \
INTERSIGHT_CLIENT_SECRET=... \
INTERSIGHT_PROXY_URL=http://proxy.example.com:8080 \
./bin/intersight-mcp serve
```

```bash
INTERSIGHT_CLIENT_ID=... \
INTERSIGHT_CLIENT_SECRET=... \
./bin/intersight-mcp serve --max-output 1MB
```

### Advanced Tuning

These settings are available, but they are operational tuning knobs rather than normal setup requirements:

| Setting | Flag | Environment | Default |
|---|---|---|---|
| Global execution timeout | `--timeout` | `INTERSIGHT_TIMEOUT` | `40s` |
| Search execution timeout | `--search-timeout` | `INTERSIGHT_SEARCH_TIMEOUT` | `15s` |
| Per-call HTTP/bootstrap timeout | `--per-call-timeout` | `INTERSIGHT_PER_CALL_TIMEOUT` | `15s` |
| Max API calls per execution | `--max-api-calls` | `INTERSIGHT_MAX_API_CALLS` | `250` |
| Max concurrent tool executions | `--max-concurrent` | `INTERSIGHT_MAX_CONCURRENT` | `25` |
| Read-only mode | `--read-only` | — | `false` |
| Max submitted code size | `--max-code-size` | `INTERSIGHT_MAX_CODE_SIZE` | `100KB` |
| QuickJS memory limit | `--wasm-memory` | `INTERSIGHT_WASM_MEMORY` | `64MB` |
| Log level | `--log-level` | `INTERSIGHT_LOG_LEVEL` | `info` |
| Include submitted tool code in debug logs with best-effort redaction | `--unsafe-log-full-code` | `INTERSIGHT_UNSAFE_LOG_FULL_CODE` | `false` |
| Mirror structured content into text for legacy clients | `--legacy-content-mirror` | `INTERSIGHT_LEGACY_CONTENT_MIRROR` | `false` |

`--max-concurrent` is a shared process-wide limiter across `search`, `query`, and `mutate`. The default limit is `25` in-flight tool executions.

When `--read-only` is set, the server omits the `mutate` tool entirely and exposes only `search` and `query`. This is the recommended mode when you want discovery and read access without allowing persistent writes.

## MCP Client Setup

Configure your MCP client to launch the binary as a local stdio command. Example shape:

```json
{
  "command": "/absolute/path/to/implementation/bin/intersight-mcp",
  "args": ["serve"],
  "env": {
    "INTERSIGHT_CLIENT_ID": "your-client-id",
    "INTERSIGHT_CLIENT_SECRET": "your-client-secret",
    "INTERSIGHT_ENDPOINT": "intersight.com"
  }
}
```

By default the server registers three tools: `search`, `query`, and `mutate`. With `--read-only`, it registers only `search` and `query`.
The public execution surface is `sdk` only for `query` and `mutate`. `search` exposes only the `catalog` discovery object, including `catalog.schema(name)` for normalized schema drilldown.

If credentials are missing or initial OAuth bootstrap fails, the server still starts so `search` remains usable. Live `query` reads and `mutate` writes then return auth errors until credentials work again.

`--unsafe-log-full-code` and `INTERSIGHT_UNSAFE_LOG_FULL_CODE` are break-glass debugging options. When enabled together with `--log-level debug`, the server logs submitted tool code with best-effort redaction for bearer tokens, client secrets, and similar values, then emits a startup warning. Use this only temporarily on trusted machines; normal debug logging already captures code hashes, execution metadata, API call traces, and error details without storing submitted code.

Example reverse lookup in `search`:

```js
const keys = catalog.paths['/vnic/FcNetworkPolicies'] || [];
return keys.map(key => catalog.resources[key]);
```

Example schema drilldown in `search`:

```js
const resource = catalog.resources['ntp.policy'];
return resource ? catalog.schema(resource.schema) : null;
```

## Spec Update Workflow

When the pinned Cisco OpenAPI input, the generator, or core dependencies change:

1. Replace `third_party/intersight/openapi/raw/openapi.json` with the new pinned raw spec.
2. Update `third_party/intersight/openapi/manifest.json` so the published version, source URL, SHA-256, and retrieval date match the new raw file.
3. Review `spec/filter.yaml` and adjust denylist entries only when there is an intentional routing change.
4. Regenerate the embedded normalized artifact with `make generate`.
5. Run the full verification harness with `make verify`.
6. Commit the updated raw spec, manifest, and regenerated `generated/spec_resolved.json`, `generated/sdk_catalog.json`, `generated/rules.json`, and `generated/search_catalog.json` together.

Do not fetch specs implicitly as part of build or generate. The workflow is local-only and reproducible from repository state.
