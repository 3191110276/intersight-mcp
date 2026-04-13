package tools

const searchDescription = `Search the Intersight discovery catalog for resource and metrics discovery.

Your code runs as the body of an async function. Use ` + "`return`" + ` to send results back.
The return value is JSON-serialized.
console.log() output appears as a separate text section after the result.

Available globals:
  catalog.resources — Record<string, SearchResource>
  catalog.paths — Record<string, string[]>
  catalog.metrics.groups — Record<string, SearchMetricsGroup>
  catalog.metrics.byName — Record<string, SearchMetric>
  catalog.schema(name) — Search for one normalized schema by name

Use ` + "`catalog.resources`" + ` as the primary discovery surface.
Use ` + "`catalog.paths`" + ` to map a REST path to resource keys.
Use ` + "`catalog.schema(name)`" + ` to drill into the normalized schema for a discovered resource via ` + "`resource.schema`" + `.
For a resource entry, derive the SDK method as ` + "`resourceKey + '.' + verb`" + ` using ` + "`resource.operations`" + `.
Only the ` + "`catalog`" + ` global is exposed in ` + "`search`" + `.

Examples:

  // Get one resource entry
  return catalog.resources['vnic.ethIf'] || null;

  // Drill into the normalized schema for a discovered resource
  const resource = catalog.resources['ntp.policy'];
  return resource ? catalog.schema(resource.schema) : null;

  // Find resource keys from a REST path
  return catalog.paths['/vnic/FcNetworkPolicies'] || [];

  // Look up one metric by name
  return catalog.metrics.byName['system.cpu.utilization_user'] || null;

  // List metrics in a metrics group
  return catalog.metrics.groups['system.cpu'] || null;`

const queryDescription = `Query Intersight with the generated ` + "`sdk`" + ` object. This tool is non-mutating: read-shaped SDK methods execute normally, while write-shaped SDK methods run offline validation and return a validation report without making API calls. Use ` + "`mutate`" + ` for persistent writes.

Your code runs as the body of an async function. Use ` + "`return`" + ` to send results back.
The return value is JSON-serialized.
console.log() output appears as a separate text section after the result.
You can make up to 250 API calls per execution by default. Override this with ` + "`--max-api-calls`" + ` or ` + "`INTERSIGHT_MAX_API_CALLS`" + `. Each call has a 15-second timeout.

Available global: sdk
  await sdk.<namespace>.<resource>.<method>({
    path?: object,
    query?: object,
    body?: object,
    ...headerArgs
  }): object
  await sdk.telemetry.query({
    dataSource: string,
    dimensions: array,
    granularity: string | object,
    intervals: array,
    render?: 'off',
    virtualColumns?: array,
    limitSpec?: object,
    having?: object,
    filter?: object,
    aggregations?: array,
    postAggregations?: array,
    subtotalsSpec?: array,
    context?: object
  }): object

Read queries preserve spec-defined query parameters, including OData fields such as ` + "`$filter`" + `, ` + "`$select`" + `, ` + "`$orderby`" + `, ` + "`$top`" + `, ` + "`$skip`" + `, ` + "`$expand`" + `, and non-OData operation-specific query parameters.
Write-shaped SDK calls return the same offline validation report shape used for local request checking, including ` + "`valid`" + `, ` + "`issues`" + `, and ` + "`layers`" + `.
The custom ` + "`sdk.telemetry.query(...)`" + ` method is also available in ` + "`query`" + `: it accepts Apache Druid groupBy query fields as top-level inputs, validates the required ` + "`dataSource`" + `, ` + "`dimensions`" + `, ` + "`granularity`" + `, and ` + "`intervals`" + ` arguments, and emits a read-only telemetry POST with ` + "`queryType: 'groupBy'`" + ` internally. Use the optional ` + "`render`" + ` field to declare whether chart presentation should be ` + "`'off'`" + `.
Inside the JS runtime, SDK calls can surface auth, HTTP, network, timeout, limit, validation, and reference errors through the standard MCP error envelope.

Examples:

  // List rack units with OData query parameters
  const page = await sdk.compute.rackUnit.list({
    query: {
      '$select': 'Name,Model,Serial,ManagementIp',
      '$top': 25,
      '$orderby': 'Name asc'
    }
  });
  return page.Results;

  // Fetch a single object by Moid
  return await sdk.compute.rackUnit.get({
    path: { Moid: '60b1f2a36972652d30e4b2c1' },
    query: { '$select': 'Name,Model,Serial' }
  });

  // Paginate through a collection
  let all = [];
  let skip = 0;
  while (true) {
    const page = await sdk.network.elementSummary.list({
      query: { '$top': 100, '$skip': skip, '$select': 'Name,Dn,Model,ManagementIp' }
    });
    all = all.concat(page.Results || []);
    if (!page.Results || page.Results.length < 100) break;
    skip += 100;
  }
  return all;

  // Execute a telemetry groupBy query
  return await sdk.telemetry.query({
    dataSource: 'example_datasource',
    dimensions: ['host'],
    intervals: ['2026-04-01/2026-04-09'],
    granularity: 'hour',
    render: 'off',
    aggregations: [
      { type: 'longSum', name: 'total', fieldName: 'value' }
    ]
  });

  // Validate a create without persisting it
  return await sdk.vnic.ethIf.create({
    body: {
      Name: 'eth0',
      LanConnectivityPolicy: { Moid: '...' },
      EthAdapterPolicy: { Moid: '...' }
    }
  });`

const mutateDescription = `Modify Intersight with the generated ` + "`sdk`" + ` object. Use this tool for persistent write-shaped SDK operations only. ` + "`query`" + ` runs the same mandatory local validation without making API calls.

Your code runs as the body of an async function. Use ` + "`return`" + ` to send results back.
The return value is JSON-serialized.
console.log() output appears as a separate text section after the result.
You can make up to 250 API calls per execution by default. Override this with ` + "`--max-api-calls`" + ` or ` + "`INTERSIGHT_MAX_API_CALLS`" + `. Each call has a 15-second timeout.

Available global: sdk
  await sdk.<namespace>.<resource>.<method>({
    path?: object,
    query?: object,
    body?: object,
    ...headerArgs
  }): object

When the request schema has a single valid discriminator, the runtime auto-fills missing ` + "`ClassId`" + ` and ` + "`ObjectType`" + ` values in the request body and MoRef relationships. If you provide them explicitly, your values are preserved.

Examples:

  // Create an NTP policy
  return await sdk.ntp.policy.create({
    body: {
      Name: 'ntp-policy-01',
      Enabled: true,
      Timezone: 'UTC',
      NtpServers: ['pool.ntp.org', 'time.google.com'],
      Organization: { Moid: '5ddf1d456972652d30bc0a10' }
    }
  });

  // Update a server profile description
  return await sdk.server.profile.update({
    path: { Moid: '60b1f2a36972652d30e4b2c1' },
    body: { Description: 'Updated by MCP server' }
  });

  // Delete a policy
  return await sdk.ntp.policy.delete({
    path: { Moid: '60b1f2a36972652d30e4b2c1' }
  });`
