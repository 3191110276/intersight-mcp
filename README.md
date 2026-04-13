# Intersight MCP Server

This is a local-only MCP server for Cisco Intersight which exposes the entire API.

Developer workflow is documented in [DEVELOPMENT.md](/Users/mimaurer/Documents/GitHub/intersight-mcp/DEVELOPMENT.md).

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

You can also pass settings as CLI flags in `args`:

```json
{
  "command": "/absolute/path/to/bin/intersight-mcp",
  "args": [
    "serve",
    "--endpoint", "intersight.example.com",
    "--read-only",
    "--max-output", "1MB"
  ],
  "env": {
    "INTERSIGHT_CLIENT_ID": "your-client-id",
    "INTERSIGHT_CLIENT_SECRET": "your-client-secret"
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

The server does not inherit `HTTP_PROXY`, `HTTPS_PROXY`, or `NO_PROXY`. Proxying is enabled only through `--proxy` or `INTERSIGHT_PROXY_URL`.

## Usage

By default the server registers three tools:

- `search` for exploring the generated discovery catalog through `catalog`
- `query` for read-shaped SDK calls and offline validation of write-shaped SDK calls
- `mutate` for persistent write-shaped SDK calls against the Intersight API

With `--read-only`, it registers only `search` and `query`.

The public execution surface is `sdk` for `query` and `mutate`. `search` exposes the `catalog` discovery object, including `catalog.schema(name)` for normalized schema drilldown.
