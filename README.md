# Intersight MCP Server

This is an MCP server for Cisco Intersight which exposes the entire API.

It exposes three tools:

- `search` for exploring the generated discovery catalog, including resources, paths, metrics, and normalized schemas through `catalog`
- `query` for read-shaped SDK calls against the Intersight API and offline validation of write-shaped SDK calls without making API calls
- `mutate` for persistent write-shaped SDK calls against the Intersight API

The server is local-only and stdio-only. It does not expose an HTTP listener or use a config file.

Developer workflow, source builds, and spec maintenance live in [DEVELOPMENT.md](/Users/mimaurer/Documents/GitHub/intersight-mcp/DEVELOPMENT.md).

## Requirements

- Cisco Intersight OAuth client credentials for live reads and writes
- A local binary for your platform, or a source checkout if you want to build it yourself

The server can still start without credentials so the offline `search` tool remains available. Write-shaped `query` validation also remains available because it runs locally.

## Configuration

Supported configuration comes from flags and matching environment variables. Flags take precedence over environment variables.

Credentials required for live `query` reads and `mutate` writes:

- `INTERSIGHT_CLIENT_ID`
- `INTERSIGHT_CLIENT_SECRET`

Common settings:

| Setting | Flag | Environment | Default |
|---|---|---|---|
| Endpoint origin | `--endpoint` | `INTERSIGHT_ENDPOINT` | `https://intersight.com` |
| Explicit outbound proxy URL | `--proxy` | `INTERSIGHT_PROXY_URL` | disabled |
| Max serialized tool payload | `--max-output` | `INTERSIGHT_MAX_OUTPUT` | `512KB` |
| Read-only mode | `--read-only` | — | `false` |

When `--read-only` is set, the server omits the `mutate` tool entirely and exposes only `search` and `query`.

`--max-output` applies to the serialized tool payload produced by sandbox execution, before MCP response wrapping. It does not count the duplicated MCP envelope fields on top of that payload.

### Endpoint Rules

- Accepts either a bare host like `intersight.example.com` or an origin-like value
- Bare hosts are normalized to `https://`
- If you provide a scheme explicitly, it must be `https://`
- Must not include user info, a query string, or a fragment
- Must be the origin only; path components are rejected
- OAuth and API base URLs are both derived from that HTTPS origin

### Proxy Rules

- The server does not inherit `HTTP_PROXY`, `HTTPS_PROXY`, or `NO_PROXY` from the host environment
- Outbound OAuth and API traffic uses a proxy only when `--proxy` or `INTERSIGHT_PROXY_URL` is set explicitly
- Supported proxy URL schemes are `http`, `https`, and `socks5`

### Examples

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

## MCP Client Setup

Configure your MCP client to launch the binary as a local stdio command:

```json
{
  "command": "/absolute/path/to/bin/intersight-mcp",
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
