# Intersight MCP Server

This is an MCP server for Cisco Intersight which exposes the entire API.

The server is local-only and stdio-only. It does not expose an HTTP listener or use a config file.

Developer workflow, source builds, and spec maintenance live in [DEVELOPMENT.md](/Users/mimaurer/Documents/GitHub/intersight-mcp/DEVELOPMENT.md).

## Setup

To run the server on your machine, you need:

- a local `intersight-mcp` binary for your platform
- Cisco Intersight OAuth credentials if you want live reads or writes

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

Required environment variables for live `query` reads and `mutate` writes:

- `INTERSIGHT_CLIENT_ID`
- `INTERSIGHT_CLIENT_SECRET`

Common optional settings:

| Setting | Flag | Environment | Default |
|---|---|---|---|
| Endpoint origin | `--endpoint` | `INTERSIGHT_ENDPOINT` | `https://intersight.com` |
| Explicit outbound proxy URL | `--proxy` | `INTERSIGHT_PROXY_URL` | disabled |
| Max serialized tool payload | `--max-output` | `INTERSIGHT_MAX_OUTPUT` | `512KB` |
| Read-only mode | `--read-only` | — | `false` |

Endpoint rules:

- Accepts either a bare host like `intersight.example.com` or an origin-like value
- Bare hosts are normalized to `https://`
- If you provide a scheme explicitly, it must be `https://`
- Must not include user info, a query string, or a fragment
- Must be the origin only; path components are rejected
- OAuth and API base URLs are both derived from that HTTPS origin

Proxy rules:

- The server does not inherit `HTTP_PROXY`, `HTTPS_PROXY`, or `NO_PROXY` from the host environment
- Outbound OAuth and API traffic uses a proxy only when `--proxy` or `INTERSIGHT_PROXY_URL` is set explicitly
- Supported proxy URL schemes are `http`, `https`, and `socks5`

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

The server can still start without credentials so the offline `search` tool remains available. Live `query` reads and `mutate` writes will return auth errors until credentials work again.

## Usage

By default the server registers three tools:

- `search` for exploring the generated discovery catalog through `catalog`
- `query` for read-shaped SDK calls and offline validation of write-shaped SDK calls
- `mutate` for persistent write-shaped SDK calls against the Intersight API

With `--read-only`, it registers only `search` and `query`.

The public execution surface is `sdk` for `query` and `mutate`. `search` exposes the `catalog` discovery object, including `catalog.schema(name)` for normalized schema drilldown.
