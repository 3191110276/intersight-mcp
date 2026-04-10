package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/mimaurer/intersight-mcp/internal/contracts"
	"github.com/pb33f/libopenapi"
	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	openapiorderedmap "github.com/pb33f/libopenapi/orderedmap"
	"go.yaml.in/yaml/v4"
)

const resolvedSpecSoftLimit = 30 * 1024 * 1024

type generator struct {
	inPath      string
	filterPath  string
	metricsPath string
	outPath     string
	stdout      io.Writer
	stderr      io.Writer
}

func newGenerator(inPath, filterPath, metricsPath, outPath string, stdout, stderr io.Writer) *generator {
	return &generator{
		inPath:      inPath,
		filterPath:  filterPath,
		metricsPath: metricsPath,
		outPath:     outPath,
		stdout:      stdout,
		stderr:      stderr,
	}
}

type manifest struct {
	PublishedVersion string `json:"published_version"`
	SourceURL        string `json:"source_url"`
	SHA256           string `json:"sha256"`
	RetrievalDate    string `json:"retrieval_date"`
}

type filterPolicy struct {
	Denylist denylistPolicy `yaml:"denylist"`
}

type denylistPolicy struct {
	Namespaces   []denyRule `yaml:"namespaces"`
	PathPrefixes []denyRule `yaml:"pathPrefixes"`
	OperationIDs []denyRule `yaml:"operationIds"`
}

type denyRule struct {
	Name      string `yaml:"name"`
	Prefix    string `yaml:"prefix"`
	ID        string `yaml:"id"`
	Rationale string `yaml:"rationale"`
}

type normalizedSpec = contracts.NormalizedSpec
type normalizedTag = contracts.NormalizedTag
type normalizedOperation = contracts.NormalizedOperation
type normalizedParameter = contracts.NormalizedParameter
type normalizedRequestBody = contracts.NormalizedRequestBody
type normalizedResponse = contracts.NormalizedResponse
type normalizedMediaContent = contracts.NormalizedMediaContent
type normalizedSchema = contracts.NormalizedSchema

type normalizationContext struct {
	rootName        string
	stack           []string
	collapseRootRef bool
}

type generationSummary struct {
	PublishedVersion    string                `json:"publishedVersion"`
	SourceURL           string                `json:"sourceURL"`
	RetrievalDate       string                `json:"retrievalDate"`
	KeptOperations      int                   `json:"keptOperations"`
	DroppedOperations   int                   `json:"droppedOperations"`
	KeptSchemas         int                   `json:"keptSchemas"`
	DroppedSchemas      int                   `json:"droppedSchemas"`
	ActiveDenylist      []generationDenyEntry `json:"activeDenylist"`
	FinalJSONSizeBytes  int                   `json:"finalJSONSizeBytes"`
	ResolvedParseTimeMS int64                 `json:"resolvedParseTimeMs"`
	StartupAllocBytes   uint64                `json:"startupAllocBytes"`
	Warnings            []string              `json:"warnings,omitempty"`
}

type generationDenyEntry struct {
	Kind      string `json:"kind"`
	Value     string `json:"value"`
	Rationale string `json:"rationale"`
}

func (g *generator) Run() error {
	g.logStep("reading pinned spec")
	rawBytes, err := os.ReadFile(g.inPath)
	if err != nil {
		return fmt.Errorf("read input spec: %w", err)
	}

	manifestPath := filepath.Join(filepath.Dir(filepath.Dir(g.inPath)), "manifest.json")
	g.logStep("validating manifest")
	mf, err := loadManifest(manifestPath)
	if err != nil {
		return err
	}
	if err := validateManifest(rawBytes, mf); err != nil {
		return err
	}

	g.logStep("loading filter policy")
	policy, err := loadFilterPolicy(g.filterPath)
	if err != nil {
		return err
	}

	g.logStep("loading metrics catalog")
	metricsCatalog, err := loadMetricsCatalog(g.metricsPath)
	if err != nil {
		return err
	}

	g.logStep("building OpenAPI model")
	doc, err := libopenapi.NewDocument(rawBytes)
	if err != nil {
		return fmt.Errorf("parse OpenAPI document: %w", err)
	}
	defer doc.Release()

	model, err := doc.BuildV3Model()
	if err != nil {
		return fmt.Errorf("build OpenAPI v3 model: %w", err)
	}

	g.logStep("normalizing retained routes and schemas")
	spec, keptOps, droppedOps, reachableSchemas, err := normalizeDocument(model.Model, policy)
	if err != nil {
		return err
	}
	spec.Metadata = contracts.ArtifactSourceMetadata{
		PublishedVersion: mf.PublishedVersion,
		SourceURL:        mf.SourceURL,
		SHA256:           mf.SHA256,
		RetrievalDate:    mf.RetrievalDate,
	}

	g.logStep("building sdk catalog")
	catalog, err := contracts.BuildSDKCatalog(spec)
	if err != nil {
		return err
	}
	if err := contracts.ValidateSDKCatalogAgainstSpec(spec, catalog); err != nil {
		return err
	}

	g.logStep("building rule metadata")
	rules, err := contracts.BuildRuleCatalog(spec, catalog)
	if err != nil {
		return err
	}
	if err := contracts.ValidateRuleCatalogAgainstArtifacts(spec, catalog, rules); err != nil {
		return err
	}

	g.logStep("building search catalog")
	searchCatalog, err := contracts.BuildSearchCatalog(spec, catalog, rules, metricsCatalog)
	if err != nil {
		return err
	}
	if err := contracts.ValidateSearchCatalogAgainstArtifacts(spec, catalog, rules, searchCatalog); err != nil {
		return err
	}

	g.logStep("writing generated artifacts")
	if err := os.MkdirAll(filepath.Dir(g.outPath), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	rendered, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal normalized spec: %w", err)
	}
	rendered = append(rendered, '\n')

	if err := os.WriteFile(g.outPath, rendered, 0o644); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	catalogRendered, err := json.MarshalIndent(catalog, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal sdk catalog: %w", err)
	}
	catalogRendered = append(catalogRendered, '\n')
	catalogPath := filepath.Join(filepath.Dir(g.outPath), "sdk_catalog.json")
	if err := os.WriteFile(catalogPath, catalogRendered, 0o644); err != nil {
		return fmt.Errorf("write sdk catalog: %w", err)
	}

	rulesRendered, err := json.MarshalIndent(rules, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal rules: %w", err)
	}
	rulesRendered = append(rulesRendered, '\n')
	rulesPath := filepath.Join(filepath.Dir(g.outPath), "rules.json")
	if err := os.WriteFile(rulesPath, rulesRendered, 0o644); err != nil {
		return fmt.Errorf("write rules: %w", err)
	}

	searchCatalogRendered, err := json.MarshalIndent(searchCatalog, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal search catalog: %w", err)
	}
	searchCatalogRendered = append(searchCatalogRendered, '\n')
	searchCatalogPath := filepath.Join(filepath.Dir(g.outPath), "search_catalog.json")
	if err := os.WriteFile(searchCatalogPath, searchCatalogRendered, 0o644); err != nil {
		return fmt.Errorf("write search catalog: %w", err)
	}

	g.logStep("measuring output and emitting summaries")
	summary := buildSummary(mf, policy, model.Model, spec, keptOps, droppedOps, reachableSchemas, rendered)
	if err := emitSummary(g.stdout, g.stderr, summary); err != nil {
		return fmt.Errorf("emit summary: %w", err)
	}

	return nil
}

func (g *generator) logStep(message string) {
	if g.stderr == nil {
		return
	}
	fmt.Fprintf(g.stderr, "[generate] %s\n", message)
}

func loadManifest(path string) (manifest, error) {
	var mf manifest
	data, err := os.ReadFile(path)
	if err != nil {
		return mf, fmt.Errorf("read manifest: %w", err)
	}
	if err := json.Unmarshal(data, &mf); err != nil {
		return mf, fmt.Errorf("parse manifest: %w", err)
	}
	return mf, nil
}

func loadMetricsCatalog(path string) (contracts.SearchMetricsCatalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return contracts.SearchMetricsCatalog{}, fmt.Errorf("read metrics catalog: %w", err)
	}
	catalog, err := contracts.LoadSearchMetricsCatalogJSON(data)
	if err != nil {
		return contracts.SearchMetricsCatalog{}, err
	}
	return catalog, nil
}

func validateManifest(raw []byte, mf manifest) error {
	sum := sha256.Sum256(raw)
	if got := hex.EncodeToString(sum[:]); !strings.EqualFold(got, mf.SHA256) {
		return fmt.Errorf("manifest mismatch: sha256 %s does not match manifest %s", got, mf.SHA256)
	}

	var doc struct {
		Info struct {
			Version string `yaml:"version"`
		} `yaml:"info"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("parse raw spec for manifest validation: %w", err)
	}
	if doc.Info.Version != mf.PublishedVersion {
		return fmt.Errorf("manifest mismatch: spec version %q does not match manifest %q", doc.Info.Version, mf.PublishedVersion)
	}
	return nil
}

func loadFilterPolicy(path string) (filterPolicy, error) {
	var policy filterPolicy
	data, err := os.ReadFile(path)
	if err != nil {
		return policy, fmt.Errorf("read filter policy: %w", err)
	}
	if err := yaml.Unmarshal(data, &policy); err != nil {
		return policy, fmt.Errorf("parse filter policy: %w", err)
	}
	for _, rule := range policy.Denylist.Namespaces {
		if rule.Name == "" || rule.Rationale == "" {
			return policy, fmt.Errorf("invalid namespace denylist entry: every entry requires name and rationale")
		}
	}
	for _, rule := range policy.Denylist.PathPrefixes {
		if rule.Prefix == "" || rule.Rationale == "" {
			return policy, fmt.Errorf("invalid path prefix denylist entry: every entry requires prefix and rationale")
		}
	}
	for _, rule := range policy.Denylist.OperationIDs {
		if rule.ID == "" || rule.Rationale == "" {
			return policy, fmt.Errorf("invalid operationId denylist entry: every entry requires id and rationale")
		}
	}
	return policy, nil
}

func normalizeDocument(doc v3.Document, policy filterPolicy) (normalizedSpec, int, int, map[string]struct{}, error) {
	result := normalizedSpec{
		Paths:   make(map[string]map[string]normalizedOperation),
		Schemas: make(map[string]normalizedSchema),
		Tags:    normalizeTags(doc.Tags),
	}

	components := doc.Components
	if components == nil || components.Schemas == nil {
		return result, 0, 0, nil, fmt.Errorf("OpenAPI document missing components.schemas")
	}

	keptOps := 0
	droppedOps := 0
	reachableSchemas := make(map[string]struct{})

	if doc.Paths == nil || doc.Paths.PathItems == nil {
		return result, 0, 0, nil, fmt.Errorf("OpenAPI document missing paths")
	}

	pathCount := 0
	for path, pathItem := range doc.Paths.PathItems.FromOldest() {
		if !strings.HasPrefix(path, "/api/v1/") {
			continue
		}
		pathCount++

		methods := make(map[string]normalizedOperation)
		for method, op := range pathItem.GetOperations().FromOldest() {
			if shouldDropOperation(path, op, policy) {
				droppedOps++
				continue
			}

			combinedParams := combineParameters(pathItem.Parameters, op.Parameters)
			for _, param := range combinedParams {
				collectOperationSchemaRefsFromParameter(param, reachableSchemas)
			}
			collectOperationSchemaRefsFromRequestBody(op.RequestBody, reachableSchemas)
			collectOperationSchemaRefsFromResponses(op.Responses, reachableSchemas)

			methods[strings.ToLower(method)] = normalizedOperation{
				Summary:     op.Summary,
				OperationID: op.OperationId,
				Tags:        cloneStrings(op.Tags),
				Parameters:  normalizeParameters(combinedParams),
				RequestBody: normalizeRequestBody(op.RequestBody),
				Responses:   normalizeResponses(op.Responses),
			}
			keptOps++
		}

		if len(methods) > 0 {
			result.Paths[path] = methods
		}
	}

	reachableSchemas = expandReachableSchemas(components.Schemas, reachableSchemas)

	schemaNames := make([]string, 0, len(reachableSchemas))
	for name := range reachableSchemas {
		schemaNames = append(schemaNames, name)
	}
	slices.Sort(schemaNames)
	for _, name := range schemaNames {
		proxy := components.Schemas.GetOrZero(name)
		if proxy == nil {
			continue
		}
		schema := normalizeSchemaProxy(proxy, normalizationContext{rootName: name})
		if schema != nil {
			result.Schemas[name] = *schema
		}
	}

	return result, keptOps, droppedOps, reachableSchemas, nil
}

func normalizeTags(tags []*highbase.Tag) []normalizedTag {
	out := make([]normalizedTag, 0, len(tags))
	for _, tag := range tags {
		if tag == nil || tag.Name == "" {
			continue
		}
		out = append(out, normalizedTag{
			Name:        tag.Name,
			Description: tag.Description,
		})
	}
	return out
}

func shouldDropOperation(path string, op *v3.Operation, policy filterPolicy) bool {
	namespace := namespaceForPath(path)
	for _, rule := range policy.Denylist.Namespaces {
		if namespace == rule.Name {
			return true
		}
	}
	for _, rule := range policy.Denylist.PathPrefixes {
		if strings.HasPrefix(path, rule.Prefix) {
			return true
		}
	}
	for _, rule := range policy.Denylist.OperationIDs {
		if op != nil && op.OperationId == rule.ID {
			return true
		}
	}
	return false
}

func namespaceForPath(path string) string {
	trimmed := strings.TrimPrefix(path, "/api/v1/")
	if trimmed == path || trimmed == "" {
		return ""
	}
	if idx := strings.IndexByte(trimmed, '/'); idx >= 0 {
		return trimmed[:idx]
	}
	return trimmed
}

func combineParameters(pathParams, opParams []*v3.Parameter) []*v3.Parameter {
	index := make(map[string]int)
	out := make([]*v3.Parameter, 0, len(pathParams)+len(opParams))
	for _, param := range pathParams {
		key := parameterKey(param)
		index[key] = len(out)
		out = append(out, param)
	}
	for _, param := range opParams {
		key := parameterKey(param)
		if i, ok := index[key]; ok {
			out[i] = param
			continue
		}
		index[key] = len(out)
		out = append(out, param)
	}
	return out
}

func parameterKey(param *v3.Parameter) string {
	if param == nil {
		return ""
	}
	return param.In + "\x00" + param.Name
}

func normalizeParameters(params []*v3.Parameter) []normalizedParameter {
	out := make([]normalizedParameter, 0, len(params))
	for _, param := range params {
		if param == nil {
			continue
		}
		var required bool
		if param.Required != nil {
			required = *param.Required
		}

		var schema *normalizedSchema
		if param.Schema != nil {
			schema = normalizeOperationSchemaProxy(param.Schema)
		} else if media := jsonMediaType(param.Content); media != nil && media.Schema != nil {
			schema = normalizeOperationSchemaProxy(media.Schema)
		}

		out = append(out, normalizedParameter{
			Name:     param.Name,
			In:       param.In,
			Required: required,
			Schema:   schema,
		})
	}
	return out
}

func normalizeRequestBody(body *v3.RequestBody) *normalizedRequestBody {
	if body == nil {
		return nil
	}
	media := jsonMediaType(body.Content)
	if media == nil || media.Schema == nil {
		return nil
	}

	required := false
	if body.Required != nil {
		required = *body.Required
	}
	return &normalizedRequestBody{
		Required: required,
		Content: map[string]normalizedMediaContent{
			"application/json": {Schema: normalizeOperationSchemaProxy(media.Schema)},
		},
	}
}

func normalizeResponses(responses *v3.Responses) map[string]normalizedResponse {
	if responses == nil {
		return nil
	}
	out := make(map[string]normalizedResponse)
	for code, resp := range responses.Codes.FromOldest() {
		out[code] = normalizeResponse(resp)
	}
	if responses.Default != nil {
		out["default"] = normalizeResponse(responses.Default)
	}
	return out
}

func normalizeResponse(resp *v3.Response) normalizedResponse {
	if resp == nil {
		return normalizedResponse{}
	}
	out := normalizedResponse{Description: resp.Description}
	if media := jsonMediaType(resp.Content); media != nil && media.Schema != nil {
		out.Content = map[string]normalizedMediaContent{
			"application/json": {Schema: normalizeOperationSchemaProxy(media.Schema)},
		}
	}
	return out
}

func jsonMediaType(content interface{ GetOrZero(string) *v3.MediaType }) *v3.MediaType {
	if content == nil {
		return nil
	}
	if typed, ok := any(content).(*openapiorderedmap.Map[string, *v3.MediaType]); ok && typed == nil {
		return nil
	}
	return content.GetOrZero("application/json")
}

func normalizeSchemaProxy(proxy *highbase.SchemaProxy, ctx normalizationContext) *normalizedSchema {
	if proxy == nil {
		return nil
	}
	if ctx.collapseRootRef && len(ctx.stack) == 0 {
		if refName := refToSchemaName(proxy.GetReference()); refName != "" {
			return &normalizedSchema{Circular: refName}
		}
	}
	schema := proxy.Schema()
	if schema == nil {
		return nil
	}

	fallbackRoot := ""
	if len(ctx.stack) == 0 {
		fallbackRoot = ctx.rootName
	}
	identity, sentinel := schemaIdentity(proxy, schema, fallbackRoot)
	if identity != "" && slices.Contains(ctx.stack, identity) {
		return &normalizedSchema{Circular: sentinel}
	}

	nextCtx := ctx
	if identity != "" {
		nextCtx.stack = append(cloneStrings(ctx.stack), identity)
	}
	nextCtx.collapseRootRef = false

	result := &normalizedSchema{}
	for _, item := range schema.AllOf {
		child := normalizeSchemaProxy(item, nextCtx)
		mergeNormalizedSchema(result, child)
	}

	mergeNormalizedSchema(result, normalizeSchemaBody(schema, nextCtx))

	if result.Circular == "" {
		if expandTarget := deriveExpandTarget(proxy, schema); expandTarget != "" {
			applyRelationshipSchema(result, expandTarget)
		}
	}

	if len(result.Required) > 0 {
		result.Required = uniqueSorted(result.Required)
	}
	if len(result.Properties) == 0 {
		result.Properties = nil
	}
	if len(result.OneOf) == 0 {
		result.OneOf = nil
	}
	if len(result.AnyOf) == 0 {
		result.AnyOf = nil
	}

	return result
}

func normalizeOperationSchemaProxy(proxy *highbase.SchemaProxy) *normalizedSchema {
	return normalizeSchemaProxy(proxy, normalizationContext{collapseRootRef: true})
}

func normalizeSchemaBody(schema *highbase.Schema, ctx normalizationContext) *normalizedSchema {
	if schema == nil {
		return nil
	}
	out := &normalizedSchema{}
	if len(schema.Type) > 0 {
		out.Type = schema.Type[0]
	}
	if schema.Format != "" {
		out.Format = schema.Format
	}
	if schema.Nullable != nil {
		out.Nullable = *schema.Nullable
	}
	if schema.ReadOnly != nil {
		out.ReadOnly = *schema.ReadOnly
	}
	if len(schema.Required) > 0 {
		out.Required = append(out.Required, schema.Required...)
	}
	if len(schema.Enum) > 0 {
		out.Enum = make([]any, len(schema.Enum))
		for i, value := range schema.Enum {
			out.Enum[i] = normalizeEnumValue(value)
		}
	}
	if schema.Properties != nil {
		out.Properties = make(map[string]*normalizedSchema)
		for name, prop := range schema.Properties.FromOldest() {
			out.Properties[name] = normalizeNestedSchemaProxy(prop, ctx)
		}
	}
	if schema.Items != nil && schema.Items.IsA() && schema.Items.A != nil {
		out.Items = normalizeNestedSchemaProxy(schema.Items.A, ctx)
	}
	if schema.AdditionalProperties != nil {
		if schema.AdditionalProperties.IsA() && schema.AdditionalProperties.A != nil {
			out.AdditionalProperties = normalizeNestedSchemaProxy(schema.AdditionalProperties.A, ctx)
		} else if schema.AdditionalProperties.IsB() {
			out.AdditionalProperties = schema.AdditionalProperties.B
		}
	}
	if len(schema.OneOf) > 0 {
		out.OneOf = make([]*normalizedSchema, 0, len(schema.OneOf))
		for _, item := range schema.OneOf {
			out.OneOf = append(out.OneOf, normalizeNestedSchemaProxy(item, ctx))
		}
	}
	if len(schema.AnyOf) > 0 {
		out.AnyOf = make([]*normalizedSchema, 0, len(schema.AnyOf))
		for _, item := range schema.AnyOf {
			out.AnyOf = append(out.AnyOf, normalizeNestedSchemaProxy(item, ctx))
		}
	}
	return out
}

func normalizeNestedSchemaProxy(proxy *highbase.SchemaProxy, ctx normalizationContext) *normalizedSchema {
	if proxy == nil {
		return nil
	}
	if refName := refToSchemaName(proxy.GetReference()); refName != "" {
		schema := proxy.Schema()
		if schema == nil {
			return &normalizedSchema{Circular: refName}
		}
		out := &normalizedSchema{}
		if len(schema.Type) > 0 {
			out.Type = schema.Type[0]
		}
		if schema.Format != "" {
			out.Format = schema.Format
		}
		if schema.Nullable != nil {
			out.Nullable = *schema.Nullable
		}
		if schema.ReadOnly != nil {
			out.ReadOnly = *schema.ReadOnly
		}
		if expandTarget := deriveExpandTarget(proxy, schema); expandTarget != "" {
			if out.Type == "" {
				out.Type = "object"
			}
			applyRelationshipSchema(out, expandTarget)
			return out
		}
		if out.Type == "" && out.Format == "" {
			out.Circular = refName
		}
		return out
	}
	return normalizeSchemaProxy(proxy, ctx)
}

func normalizeEnumValue(value any) any {
	switch typed := value.(type) {
	case *yaml.Node:
		var out any
		if err := typed.Decode(&out); err == nil {
			return out
		}
	case yaml.Node:
		var out any
		if err := typed.Decode(&out); err == nil {
			return out
		}
	}
	return value
}

func schemaIdentity(proxy *highbase.SchemaProxy, schema *highbase.Schema, fallback string) (string, string) {
	if proxy != nil {
		if name := refToSchemaName(proxy.GetReference()); name != "" {
			return name, name
		}
	}
	if schema != nil && schema.ParentProxy != nil {
		if name := refToSchemaName(schema.ParentProxy.GetReference()); name != "" {
			return name, name
		}
	}
	if fallback != "" {
		return fallback, fallback
	}
	return "", ""
}

func mergeNormalizedSchema(dst, src *normalizedSchema) {
	if dst == nil || src == nil {
		return
	}
	if src.Circular != "" {
		*dst = *src
		return
	}
	if dst.Circular != "" {
		return
	}
	if dst.Type == "" {
		dst.Type = src.Type
	}
	if dst.Format == "" {
		dst.Format = src.Format
	}
	if !dst.Nullable && src.Nullable {
		dst.Nullable = true
	}
	if !dst.ReadOnly && src.ReadOnly {
		dst.ReadOnly = true
	}
	if dst.ExpandTarget == "" {
		dst.ExpandTarget = src.ExpandTarget
	}
	if !dst.Relationship && src.Relationship {
		dst.Relationship = true
	}
	if dst.RelationshipTarget == "" {
		dst.RelationshipTarget = src.RelationshipTarget
	}
	if len(dst.RelationshipWriteForms) == 0 && len(src.RelationshipWriteForms) > 0 {
		dst.RelationshipWriteForms = append([]string(nil), src.RelationshipWriteForms...)
	}
	if len(dst.Enum) == 0 && len(src.Enum) > 0 {
		dst.Enum = append([]any(nil), src.Enum...)
	}
	if src.Properties != nil {
		if dst.Properties == nil {
			dst.Properties = make(map[string]*normalizedSchema, len(src.Properties))
		}
		for name, prop := range src.Properties {
			dst.Properties[name] = prop
		}
	}
	if len(src.Required) > 0 {
		dst.Required = append(dst.Required, src.Required...)
	}
	if dst.Items == nil {
		dst.Items = src.Items
	}
	if dst.AdditionalProperties == nil {
		dst.AdditionalProperties = src.AdditionalProperties
	}
	if len(dst.OneOf) == 0 && len(src.OneOf) > 0 {
		dst.OneOf = src.OneOf
	}
	if len(dst.AnyOf) == 0 && len(src.AnyOf) > 0 {
		dst.AnyOf = src.AnyOf
	}
}

func applyRelationshipSchema(dst *normalizedSchema, target string) {
	if dst == nil {
		return
	}
	dst.Type = "object"
	dst.ExpandTarget = target
	dst.Relationship = true
	dst.RelationshipTarget = target
	dst.RelationshipWriteForms = []string{"moidRef", "typedMoRef"}
	dst.Properties = map[string]*normalizedSchema{
		"Moid": {
			Type: "string",
		},
		"ObjectType": {
			Type: "string",
			Enum: []any{target},
		},
		"ClassId": {
			Type: "string",
			Enum: []any{"mo.MoRef"},
		},
	}
	dst.OneOf = []*normalizedSchema{
		{
			Type: "object",
			Properties: map[string]*normalizedSchema{
				"Moid": {
					Type: "string",
				},
			},
			Required: []string{"Moid"},
		},
		{
			Type: "object",
			Properties: map[string]*normalizedSchema{
				"Moid": {
					Type: "string",
				},
				"ObjectType": {
					Type: "string",
					Enum: []any{target},
				},
				"ClassId": {
					Type: "string",
					Enum: []any{"mo.MoRef"},
				},
			},
			Required: []string{"Moid", "ObjectType", "ClassId"},
		},
	}
}

func collectOperationSchemaRefsFromParameter(param *v3.Parameter, out map[string]struct{}) {
	if param == nil {
		return
	}
	if param.Schema != nil {
		collectDirectSchemaRefsFromProxy(param.Schema, out, map[string]struct{}{})
		return
	}
	if media := jsonMediaType(param.Content); media != nil && media.Schema != nil {
		collectDirectSchemaRefsFromProxy(media.Schema, out, map[string]struct{}{})
	}
}

func collectOperationSchemaRefsFromRequestBody(body *v3.RequestBody, out map[string]struct{}) {
	if body == nil {
		return
	}
	if media := jsonMediaType(body.Content); media != nil && media.Schema != nil {
		collectDirectSchemaRefsFromProxy(media.Schema, out, map[string]struct{}{})
	}
}

func collectOperationSchemaRefsFromResponses(responses *v3.Responses, out map[string]struct{}) {
	if responses == nil {
		return
	}
	for _, resp := range responses.Codes.FromOldest() {
		collectOperationSchemaRefsFromResponse(resp, out)
	}
	if responses.Default != nil {
		collectOperationSchemaRefsFromResponse(responses.Default, out)
	}
}

func collectOperationSchemaRefsFromResponse(resp *v3.Response, out map[string]struct{}) {
	if resp == nil {
		return
	}
	if media := jsonMediaType(resp.Content); media != nil && media.Schema != nil {
		collectDirectSchemaRefsFromProxy(media.Schema, out, map[string]struct{}{})
	}
}

func collectDirectSchemaRefsFromProxy(proxy *highbase.SchemaProxy, out, seen map[string]struct{}) {
	if proxy == nil {
		return
	}
	if name := refToSchemaName(proxy.GetReference()); name != "" {
		out[name] = struct{}{}
		return
	}
	schema := proxy.Schema()
	if schema == nil {
		return
	}
	collectDirectSchemaRefsFromSchema(schema, out, seen)
}

func collectDirectSchemaRefsFromSchema(schema *highbase.Schema, out, seen map[string]struct{}) {
	if schema == nil {
		return
	}
	key := fmt.Sprintf("schema:%p", schema)
	if _, ok := seen[key]; ok {
		return
	}
	seen[key] = struct{}{}

	for _, item := range schema.AllOf {
		collectDirectSchemaRefsFromProxy(item, out, seen)
	}
	for _, item := range schema.OneOf {
		collectDirectSchemaRefsFromProxy(item, out, seen)
	}
	for _, item := range schema.AnyOf {
		collectDirectSchemaRefsFromProxy(item, out, seen)
	}
	if schema.Properties != nil {
		for _, prop := range schema.Properties.FromOldest() {
			collectDirectSchemaRefsFromProxy(prop, out, seen)
		}
	}
	if schema.Items != nil && schema.Items.IsA() && schema.Items.A != nil {
		collectDirectSchemaRefsFromProxy(schema.Items.A, out, seen)
	}
	if schema.AdditionalProperties != nil && schema.AdditionalProperties.IsA() && schema.AdditionalProperties.A != nil {
		collectDirectSchemaRefsFromProxy(schema.AdditionalProperties.A, out, seen)
	}
}

func expandReachableSchemas(components interface {
	GetOrZero(string) *highbase.SchemaProxy
}, seeds map[string]struct{}) map[string]struct{} {
	reachable := make(map[string]struct{}, len(seeds))
	queue := make([]string, 0, len(seeds))
	for name := range seeds {
		queue = append(queue, name)
	}
	slices.Sort(queue)

	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		if _, ok := reachable[name]; ok {
			continue
		}
		reachable[name] = struct{}{}

		proxy := components.GetOrZero(name)
		if proxy == nil {
			continue
		}
		refs := make(map[string]struct{})
		collectDirectSchemaRefsFromProxy(proxy, refs, map[string]struct{}{})
		for refName := range refs {
			if _, ok := reachable[refName]; ok {
				continue
			}
			queue = append(queue, refName)
		}
	}

	return reachable
}

func deriveExpandTarget(proxy *highbase.SchemaProxy, schema *highbase.Schema) string {
	if schema == nil || len(schema.AllOf) == 0 {
		return ""
	}
	if !hasProperty(schema, "Moid") || !hasProperty(schema, "ObjectType") {
		return ""
	}

	target := ""
	foundMoRef := false
	for _, item := range schema.AllOf {
		refName := refToSchemaName(item.GetReference())
		if refName == "" {
			continue
		}
		if refName == "mo.MoRef" || schemaInheritsFromRef(item, "mo.MoRef", map[string]struct{}{}) {
			foundMoRef = true
			continue
		}
		if target != "" {
			return ""
		}
		target = refName
	}
	if !foundMoRef || target == "" {
		return ""
	}
	if proxy != nil && refToSchemaName(proxy.GetReference()) == target {
		return ""
	}
	return target
}

func hasProperty(schema *highbase.Schema, name string) bool {
	if schema == nil {
		return false
	}
	if schema.Properties != nil && schema.Properties.GetOrZero(name) != nil {
		return true
	}
	for _, item := range schema.AllOf {
		if hasProperty(item.Schema(), name) {
			return true
		}
	}
	return false
}

func schemaInheritsFromRef(proxy *highbase.SchemaProxy, target string, seen map[string]struct{}) bool {
	if proxy == nil {
		return false
	}
	refName := refToSchemaName(proxy.GetReference())
	if refName == target {
		return true
	}
	key := refName
	if key == "" {
		key = fmt.Sprintf("%p", proxy)
	}
	if _, ok := seen[key]; ok {
		return false
	}
	seen[key] = struct{}{}

	schema := proxy.Schema()
	if schema == nil {
		return false
	}
	for _, item := range schema.AllOf {
		if schemaInheritsFromRef(item, target, seen) {
			return true
		}
	}
	return false
}

func refToSchemaName(ref string) string {
	const prefix = "#/components/schemas/"
	if strings.HasPrefix(ref, prefix) {
		return strings.TrimPrefix(ref, prefix)
	}
	return ""
}

func uniqueSorted(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	slices.Sort(out)
	return out
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	return append([]string(nil), values...)
}

func buildSummary(
	mf manifest,
	policy filterPolicy,
	doc v3.Document,
	spec normalizedSpec,
	keptOps, droppedOps int,
	reachableSchemas map[string]struct{},
	rendered []byte,
) generationSummary {
	totalSchemas := 0
	if doc.Components != nil && doc.Components.Schemas != nil {
		totalSchemas = doc.Components.Schemas.Len()
	}
	summary := generationSummary{
		PublishedVersion:   mf.PublishedVersion,
		SourceURL:          mf.SourceURL,
		RetrievalDate:      mf.RetrievalDate,
		KeptOperations:     keptOps,
		DroppedOperations:  droppedOps,
		KeptSchemas:        len(spec.Schemas),
		DroppedSchemas:     totalSchemas - len(spec.Schemas),
		ActiveDenylist:     summarizeDenylist(policy),
		FinalJSONSizeBytes: len(rendered),
	}

	parseTime, alloc := measureResolvedSpec(rendered)
	summary.ResolvedParseTimeMS = parseTime.Milliseconds()
	summary.StartupAllocBytes = alloc

	if len(rendered) > resolvedSpecSoftLimit {
		summary.Warnings = append(summary.Warnings, fmt.Sprintf("resolved spec exceeds soft limit: %d bytes > %d bytes", len(rendered), resolvedSpecSoftLimit))
	}
	if len(reachableSchemas) == 0 {
		summary.Warnings = append(summary.Warnings, "no reachable schemas were retained")
	}

	return summary
}

func summarizeDenylist(policy filterPolicy) []generationDenyEntry {
	var out []generationDenyEntry
	for _, rule := range policy.Denylist.Namespaces {
		out = append(out, generationDenyEntry{Kind: "namespace", Value: rule.Name, Rationale: rule.Rationale})
	}
	for _, rule := range policy.Denylist.PathPrefixes {
		out = append(out, generationDenyEntry{Kind: "pathPrefix", Value: rule.Prefix, Rationale: rule.Rationale})
	}
	for _, rule := range policy.Denylist.OperationIDs {
		out = append(out, generationDenyEntry{Kind: "operationId", Value: rule.ID, Rationale: rule.Rationale})
	}
	return out
}

func measureResolvedSpec(rendered []byte) (time.Duration, uint64) {
	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)

	start := time.Now()
	var decoded map[string]any
	_ = json.Unmarshal(rendered, &decoded)
	parseTime := time.Since(start)

	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	if after.Alloc >= before.Alloc {
		return parseTime, after.Alloc - before.Alloc
	}
	return parseTime, 0
}

func emitSummary(stdout, stderr io.Writer, summary generationSummary) error {
	machine, err := json.Marshal(summary)
	if err != nil {
		return err
	}
	if stdout != nil {
		if _, err := fmt.Fprintf(stdout, "%s\n", machine); err != nil {
			return err
		}
	}
	if stderr != nil {
		var buf bytes.Buffer
		fmt.Fprintf(&buf, "Generated Intersight spec %s\n", summary.PublishedVersion)
		fmt.Fprintf(&buf, "  operations: kept=%d dropped=%d\n", summary.KeptOperations, summary.DroppedOperations)
		fmt.Fprintf(&buf, "  schemas: kept=%d dropped=%d\n", summary.KeptSchemas, summary.DroppedSchemas)
		fmt.Fprintf(&buf, "  denylist entries: %d\n", len(summary.ActiveDenylist))
		fmt.Fprintf(&buf, "  resolved JSON size: %d bytes\n", summary.FinalJSONSizeBytes)
		fmt.Fprintf(&buf, "  pooled parse time: %dms\n", summary.ResolvedParseTimeMS)
		fmt.Fprintf(&buf, "  startup alloc delta: %d bytes\n", summary.StartupAllocBytes)
		for _, warning := range summary.Warnings {
			fmt.Fprintf(&buf, "  warning: %s\n", warning)
		}
		if _, err := io.Copy(stderr, &buf); err != nil {
			return err
		}
	}
	return nil
}
