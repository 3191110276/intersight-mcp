# Intersight MCP Server

This is a local-only MCP server for Cisco Intersight which exposes the entire API.

Developer workflow is documented in [DEVELOPMENT.md](/Users/mimaurer/Documents/GitHub/intersight-mcp/DEVELOPMENT.md).

## Setup

To run the server on your machine, you need:

- a local `intersight-mcp` binary for your platform
- Cisco Intersight OAuth credentials if you want live reads or writes

Configure your MCP client to launch the binary as a local stdio command. For example:

```json
{
  "command": "/absolute/path/to/bin/intersight-mcp",
  "args": ["serve", "--max-output", "512KB"],
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
| Proxy URL | `--proxy` | `INTERSIGHT_PROXY_URL` | No | disabled |
| Max serialized tool payload | `--max-output` | `INTERSIGHT_MAX_OUTPUT` | No | `512KB` |
| Read-only mode | `--read-only` | — | No | `false` |

Note: The server does not inherit proxy settings from the host. Proxying is enabled only through `--proxy` or `INTERSIGHT_PROXY_URL`.

## Usage

By default the server registers three tools:

- `search` for exploring the generated discovery catalog through `catalog`
- `query` for read-shaped SDK calls and offline validation of write-shaped SDK calls
- `mutate` for persistent write-shaped SDK calls against the Intersight API

With `--read-only`, it registers only `search` and `query`.

The MCP client will thus usually call `search` first to understand the API, then either `query` or `mutate` to read or perform changes respectively.

`query` and `mutate` compact API objects by default to remove low-signal metadata fields from results. Pass `compact: false` in a tool call when raw API output is needed.
