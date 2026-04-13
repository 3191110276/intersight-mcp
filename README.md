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

Configuration variables:

| Setting | Flag | Environment | Required | Default |
|---|---|---|---|---|
| Client ID | — | `INTERSIGHT_CLIENT_ID` | Yes | none |
| Client secret | — | `INTERSIGHT_CLIENT_SECRET` | Yes | none |
| Endpoint origin | `--endpoint` | `INTERSIGHT_ENDPOINT` | No | `https://intersight.com` |
| Explicit outbound proxy URL | `--proxy` | `INTERSIGHT_PROXY_URL` | No | disabled |
| Max serialized tool payload | `--max-output` | `INTERSIGHT_MAX_OUTPUT` | No | `512KB` |
| Read-only mode | `--read-only` | — | No | `false` |

Outbound OAuth and API traffic uses a proxy only when `--proxy` or `INTERSIGHT_PROXY_URL` is set explicitly

## Usage

By default the server registers three tools:

- `search` for exploring the generated discovery catalog through `catalog`
- `query` for read-shaped SDK calls and offline validation of write-shaped SDK calls
- `mutate` for persistent write-shaped SDK calls against the Intersight API

With `--read-only`, it registers only `search` and `query`.

The public execution surface is `sdk` for `query` and `mutate`. `search` exposes the `catalog` discovery object, including `catalog.schema(name)` for normalized schema drilldown.
