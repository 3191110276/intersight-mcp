package tools

const searchDescription = `Search the Intersight discovery catalog for resources and metrics.

Only global: ` + "`catalog`" + `
- ` + "`resources`" + `: resource catalog
- ` + "`paths`" + `: REST path -> resource keys
- ` + "`metrics.groups`" + `: metrics by group
- ` + "`metrics.byName`" + `: metric by name
- ` + "`schema(name)`" + `: normalized schema lookup

Use ` + "`catalog.resources`" + ` first. Derive SDK methods as ` + "`resourceKey + '.' + verb`" + ` from ` + "`resource.operations`" + `.

Examples:
` + "`catalog.resources['vnic.ethIf']`" + `
` + "`catalog.paths['/vnic/FcNetworkPolicies']`" + `
` + "`catalog.metrics.byName['system.cpu.utilization_user']`" + ``

const queryDescription = `Query Intersight through ` + "`sdk`" + ` without persisting changes. Read-shaped SDK methods execute normally; write-shaped methods run offline validation only. Use ` + "`mutate`" + ` for persistent writes.

Default limit: 250 API calls/execution; override with ` + "`--max-api-calls`" + ` or ` + "`INTERSIGHT_MAX_API_CALLS`" + `. Per-call timeout: 15s.

Global: ` + "`sdk`" + `
- ` + "`await sdk.<namespace>.<resource>.<method>({ path?, query?, body?, ...headerArgs })`" + `
- ` + "`await sdk.telemetry.query({ dataSource, dimensions, granularity, intervals, render?, virtualColumns?, limitSpec?, having?, filter?, aggregations?, postAggregations?, subtotalsSpec?, context? })`" + `

Read queries preserve spec-defined query params, including OData (` + "`$filter`" + `, ` + "`$select`" + `, ` + "`$orderby`" + `, ` + "`$top`" + `, ` + "`$skip`" + `, ` + "`$expand`" + `) and operation-specific params. Write-shaped calls return validation reports with ` + "`valid`" + `, ` + "`issues`" + `, ` + "`layers`" + `.

` + "`sdk.telemetry.query(...)`" + ` accepts top-level Apache Druid groupBy fields, requires ` + "`dataSource`" + `, ` + "`dimensions`" + `, ` + "`granularity`" + `, ` + "`intervals`" + `, and issues a read-only telemetry POST with internal ` + "`queryType: 'groupBy'`" + `. Errors use the standard MCP error envelope.

Examples:
- ` + "`sdk.compute.rackUnit.list({ query: { '$top': 25, '$select': 'Name,Model' } })`" + `
- ` + "`sdk.telemetry.query({ dataSource: 'x', dimensions: ['host'], intervals: ['2026-04-01/2026-04-09'], granularity: 'hour' })`" + `
- ` + "`sdk.vnic.ethIf.create({ body: { Name: 'eth0' } })`" + `  // validates only`

const mutateDescription = `Persist write-shaped Intersight SDK operations through ` + "`sdk`" + `. Use ` + "`query`" + ` for the same validation without API calls.

Default limit: 250 API calls/execution; override with ` + "`--max-api-calls`" + ` or ` + "`INTERSIGHT_MAX_API_CALLS`" + `. Per-call timeout: 15s.

Global: ` + "`sdk`" + `
- ` + "`await sdk.<namespace>.<resource>.<method>({ path?, query?, body?, ...headerArgs })`" + `

If a request schema has exactly one valid discriminator, missing ` + "`ClassId`" + ` and ` + "`ObjectType`" + ` are auto-filled in the body and MoRef relationships; explicit values win.

Examples:
- ` + "`sdk.ntp.policy.create({ body: { Name: 'ntp-policy-01' } })`" + `
- ` + "`sdk.server.profile.update({ path: { Moid: '...' }, body: { Description: 'Updated' } })`" + `
- ` + "`sdk.ntp.policy.delete({ path: { Moid: '...' } })`" + ``
