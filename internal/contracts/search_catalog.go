package contracts

import (
	"cmp"
	"fmt"
	"reflect"
	"slices"
	"strings"
)

type SearchCatalog struct {
	Metadata      ArtifactSourceMetadata    `json:"metadata"`
	Resources     map[string]SearchResource `json:"resources"`
	ResourceNames []string                  `json:"resourceNames"`
	Paths         map[string][]string       `json:"paths,omitempty"`
	Metrics       *SearchMetricsCatalog     `json:"metrics,omitempty"`
}

type SearchResource struct {
	Schema       string                 `json:"schema,omitempty"`
	CreateFields map[string]SearchField `json:"createFields,omitempty"`
	Path         string                 `json:"path,omitempty"`
	Rules        []SemanticRule         `json:"rules,omitempty"`
	Operations   []string               `json:"operations"`
}

type SearchField struct {
	Type     string   `json:"type,omitempty"`
	Format   string   `json:"format,omitempty"`
	Items    string   `json:"items,omitempty"`
	Ref      string   `json:"ref,omitempty"`
	Nullable bool     `json:"nullable,omitempty"`
	Enum     bool     `json:"enum,omitempty"`
	Required bool     `json:"required,omitempty"`
	OneOf    []string `json:"oneOf,omitempty"`
	Example  any      `json:"example,omitempty"`
}

func BuildSearchCatalog(spec NormalizedSpec, catalog SDKCatalog, rules RuleCatalog, metrics SearchMetricsCatalog) (SearchCatalog, error) {
	if spec.Metadata != catalog.Metadata || spec.Metadata != rules.Metadata {
		return SearchCatalog{}, fmt.Errorf("search catalog generation failed: spec, sdk catalog, and rule metadata must share identical source metadata")
	}

	var normalizedMetrics *SearchMetricsCatalog
	if !searchMetricsCatalogIsEmpty(metrics) {
		metrics = NormalizeSearchMetricsCatalog(metrics)
		if err := ValidateSearchMetricsCatalog(metrics); err != nil {
			return SearchCatalog{}, err
		}
		normalizedMetrics = &metrics
	}

	out := SearchCatalog{
		Metadata:      spec.Metadata,
		Resources:     map[string]SearchResource{},
		ResourceNames: []string{},
		Paths:         map[string][]string{},
		Metrics:       normalizedMetrics,
	}

	for _, sdkMethod := range sortedKeys(catalog.Methods) {
		method := catalog.Methods[sdkMethod]
		resourceKey, leaf, ok := splitSearchSDKMethod(method.SDKMethod)
		if !ok {
			return SearchCatalog{}, fmt.Errorf("search catalog generation failed: cannot derive resource key from sdk method %q", method.SDKMethod)
		}

		resource, exists := out.Resources[resourceKey]
		if !exists {
			resource = SearchResource{
				Operations: []string{},
			}
		}

		resource.Schema = selectSearchResourceSchema(spec, resource.Schema, method)
		methodRules := rules.Methods[sdkMethod]
		if leaf == "create" {
			resource.CreateFields = selectSearchCreateFields(spec, method, methodRules.Rules, resource.CreateFields)
		}
		resource.Rules = append(resource.Rules, unmergedSearchRules(methodRules.Rules)...)
		resource.Operations = append(resource.Operations, leaf)
		resource.Path = updateSearchResourcePath(resource.Path, method.Descriptor.PathTemplate)
		out.Resources[resourceKey] = resource
		indexSearchPath(out.Paths, method.Descriptor.PathTemplate, resourceKey)
	}

	for _, resourceKey := range sortedKeys(out.Resources) {
		resource := out.Resources[resourceKey]
		resource = normalizeSearchResource(resource)
		out.Resources[resourceKey] = resource
		out.ResourceNames = append(out.ResourceNames, resourceKey)
	}
	addSearchSchemaAliases(&out)

	return normalizeSearchCatalog(out), nil
}

func ValidateSearchCatalogAgainstArtifacts(spec NormalizedSpec, catalog SDKCatalog, rules RuleCatalog, search SearchCatalog) error {
	if spec.Metadata != catalog.Metadata || spec.Metadata != rules.Metadata || spec.Metadata != search.Metadata {
		return fmt.Errorf("embedded artifact validation failed: spec, sdk catalog, rule metadata, and search catalog must share identical source metadata")
	}

	search = normalizeSearchCatalog(search)
	if search.Metrics != nil {
		if err := ValidateSearchMetricsCatalog(*search.Metrics); err != nil {
			return err
		}
	}

	expected, err := BuildSearchCatalog(spec, catalog, rules, SearchMetricsCatalog{})
	if err != nil {
		return err
	}
	expected = normalizeSearchCatalog(expected)
	expected.Metrics = nil
	search.Metrics = nil

	if reflect.DeepEqual(expected, search) {
		return nil
	}

	for name := range expected.Resources {
		if _, ok := search.Resources[name]; !ok {
			return fmt.Errorf("embedded artifact validation failed: search catalog missing resource %q", name)
		}
	}
	for name := range search.Resources {
		if _, ok := expected.Resources[name]; !ok {
			return fmt.Errorf("embedded artifact validation failed: search catalog contains unknown resource %q", name)
		}
		if !reflect.DeepEqual(expected.Resources[name], search.Resources[name]) {
			return fmt.Errorf("embedded artifact validation failed: search catalog entry %q does not match generated search catalog", name)
		}
	}
	return fmt.Errorf("embedded artifact validation failed: search catalog does not match generated search catalog")
}

func normalizeSearchCatalog(catalog SearchCatalog) SearchCatalog {
	if catalog.Resources == nil {
		catalog.Resources = map[string]SearchResource{}
	}
	if catalog.ResourceNames == nil {
		catalog.ResourceNames = []string{}
	}
	if catalog.Paths == nil {
		catalog.Paths = map[string][]string{}
	}
	if catalog.Metrics != nil {
		normalized := NormalizeSearchMetricsCatalog(*catalog.Metrics)
		if searchMetricsCatalogIsEmpty(normalized) {
			catalog.Metrics = nil
		} else {
			catalog.Metrics = &normalized
		}
	}
	for key, resource := range catalog.Resources {
		resource = normalizeSearchResource(resource)
		catalog.Resources[key] = resource
	}
	catalog.ResourceNames = uniqueSortedStrings(catalog.ResourceNames)
	for key, resources := range catalog.Paths {
		catalog.Paths[key] = uniqueSortedStrings(resources)
	}
	return catalog
}

func addSearchSchemaAliases(catalog *SearchCatalog) {
	if catalog == nil || len(catalog.Resources) == 0 {
		return
	}
	for _, key := range sortedKeys(catalog.Resources) {
		resource := catalog.Resources[key]
		alias := searchResourceSchemaAlias(resource.Schema)
		if alias == "" || alias == key {
			continue
		}
		if existing, ok := catalog.Resources[alias]; ok {
			if reflect.DeepEqual(normalizeSearchResource(existing), normalizeSearchResource(resource)) {
				catalog.ResourceNames = append(catalog.ResourceNames, alias)
			}
			continue
		}
		catalog.Resources[alias] = resource
		catalog.ResourceNames = append(catalog.ResourceNames, alias)
	}
}

func normalizeSearchResource(resource SearchResource) SearchResource {
	resource.Schema = strings.TrimSpace(resource.Schema)
	resource.CreateFields = normalizeSearchFields(resource.CreateFields)
	resource.Path = strings.TrimSpace(resource.Path)
	resource.Rules = dedupeSemanticRules(normalizeSemanticRules(resource.Rules))
	resource.Operations = uniqueSortedStrings(resource.Operations)
	return resource
}

func normalizeSearchFields(fields map[string]SearchField) map[string]SearchField {
	if fields == nil {
		return nil
	}
	for key, field := range fields {
		field.Type = strings.TrimSpace(field.Type)
		field.Format = strings.TrimSpace(field.Format)
		field.Items = strings.TrimSpace(field.Items)
		field.Ref = strings.TrimSpace(field.Ref)
		field.OneOf = uniqueSortedStrings(field.OneOf)
		fields[key] = field
	}
	return fields
}

func searchMetricsCatalogIsEmpty(catalog SearchMetricsCatalog) bool {
	return len(catalog.Groups) == 0 && len(catalog.ByName) == 0 && len(catalog.Examples) == 0
}

func selectSearchCreateFields(spec NormalizedSpec, method SDKMethod, rules []SemanticRule, existing map[string]SearchField) map[string]SearchField {
	bodySchema, ok := searchRequestBodySchema(spec, method)
	var fields map[string]SearchField
	if ok && bodySchema != nil {
		fields = summarizeSearchCreateFields(bodySchema.Properties)
		for _, name := range bodySchema.Required {
			field, exists := fields[name]
			if !exists {
				continue
			}
			field.Required = true
			fields[name] = field
		}
	} else {
		fields = fallbackSearchCreateFields(spec, method)
	}
	if len(fields) == 0 {
		return existing
	}
	mergeRuleAnnotations(fields, rules)
	if len(fields) == 0 {
		return existing
	}
	return fields
}

func summarizeSearchCreateFields(properties map[string]*NormalizedSchema) map[string]SearchField {
	if len(properties) == 0 {
		return nil
	}
	out := make(map[string]SearchField, len(properties))
	for name, schema := range properties {
		if !shouldIncludeSearchField(name, schema) {
			continue
		}
		out[name] = summarizeSearchField(schema)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func summarizeSearchField(schema *NormalizedSchema) SearchField {
	if schema == nil {
		return SearchField{}
	}
	field := SearchField{
		Type:     strings.TrimSpace(schema.Type),
		Format:   strings.TrimSpace(schema.Format),
		Nullable: schema.Nullable,
		Enum:     len(schema.Enum) > 0,
	}
	if ref := searchFieldRef(schema); ref != "" {
		field.Ref = ref
	}
	if item := searchFieldItem(schema.Items); item != "" {
		field.Items = item
	}
	if schema.Relationship || field.Ref != "" {
		field.Example = searchFieldExample(schema)
	}
	return field
}

func fallbackSearchCreateFields(spec NormalizedSpec, method SDKMethod) map[string]SearchField {
	if strings.TrimSpace(method.Resource) == "" {
		return nil
	}
	schema, ok := spec.Schemas[method.Resource]
	if !ok {
		return nil
	}
	if len(method.RequestBodyFields) == 0 {
		return summarizeSearchCreateFields(schema.Properties)
	}
	fields := make(map[string]SearchField, len(method.RequestBodyFields))
	for _, name := range method.RequestBodyFields {
		property, ok := schema.Properties[name]
		if !ok || !shouldIncludeSearchField(name, property) {
			continue
		}
		fields[name] = summarizeSearchField(property)
	}
	if len(fields) == 0 {
		return nil
	}
	return fields
}

func searchRequestBodySchema(spec NormalizedSpec, method SDKMethod) (*NormalizedSchema, bool) {
	_, bodySchema, ok := findSpecOperationForDescriptor(spec, method.Descriptor)
	if !ok || bodySchema == nil {
		return nil, false
	}
	return dereferenceSchema(spec, bodySchema), true
}

func mergeRuleAnnotations(fields map[string]SearchField, rules []SemanticRule) {
	for _, rule := range rules {
		if rule.When != nil {
			continue
		}
		if strings.TrimSpace(rule.Kind) == "one_of" && len(rule.RequireAny) > 1 {
			names := make([]string, 0, len(rule.RequireAny))
			for _, requirement := range rule.RequireAny {
				if _, ok := fields[requirement.Field]; ok {
					names = append(names, requirement.Field)
				}
			}
			names = uniqueSortedStrings(names)
			for _, name := range names {
				field := fields[name]
				field.OneOf = append(field.OneOf, names...)
				fields[name] = field
			}
		}
	}
}

func unmergedSearchRules(rules []SemanticRule) []SemanticRule {
	if len(rules) == 0 {
		return nil
	}
	out := make([]SemanticRule, 0, len(rules))
	for _, rule := range rules {
		if rule.When == nil {
			switch strings.TrimSpace(rule.Kind) {
			case "one_of":
				continue
			}
		}
		out = append(out, rule)
	}
	return out
}

func searchFieldRef(schema *NormalizedSchema) string {
	if schema == nil {
		return ""
	}
	for _, ref := range []string{schema.RelationshipTarget, schema.ExpandTarget, schema.Circular} {
		ref = strings.TrimSpace(ref)
		if ref != "" {
			return ref
		}
	}
	if nested, ok := schema.AdditionalProperties.(*NormalizedSchema); ok {
		return searchFieldRef(nested)
	}
	return ""
}

func searchFieldItem(schema *NormalizedSchema) string {
	if schema == nil {
		return ""
	}
	if ref := searchFieldRef(schema); ref != "" {
		return ref
	}
	if nestedType := strings.TrimSpace(schema.Type); nestedType != "" {
		return nestedType
	}
	return ""
}

func searchFieldExample(schema *NormalizedSchema) any {
	if schema == nil {
		return nil
	}
	target := strings.TrimSpace(schema.RelationshipTarget)
	if target == "" {
		target = searchFieldRef(schema)
	}
	if target == "" {
		return nil
	}
	if schema.Items != nil {
		if example, ok := searchFieldExample(schema.Items).(map[string]any); ok {
			return []map[string]any{example}
		}
		return nil
	}
	return map[string]any{
		"Moid": fmt.Sprintf("<%s-moid>", relationshipExampleName(target)),
	}
}

func relationshipExampleName(target string) string {
	target = strings.TrimSpace(target)
	if target == "" {
		return "resource"
	}
	if idx := strings.IndexByte(target, '.'); idx >= 0 {
		target = target[idx+1:]
	}
	target = strings.TrimSpace(target)
	if target == "" {
		return "resource"
	}
	return strings.ToLower(target[:1]) + target[1:]
}

func shouldIncludeSearchField(name string, schema *NormalizedSchema) bool {
	if schema == nil || schema.ReadOnly {
		return false
	}
	switch strings.TrimSpace(name) {
	case "ApplianceAccount", "ClassId", "Moid", "ObjectType", "Owners", "Parent", "VersionContext":
		return false
	}
	switch strings.TrimSpace(searchFieldRef(schema)) {
	case "iam.Account.Relationship", "mo.VersionContext":
		return false
	}
	return true
}

func dedupeSemanticRules(rules []SemanticRule) []SemanticRule {
	if len(rules) == 0 {
		return nil
	}
	deduped := make([]SemanticRule, 0, len(rules))
	seen := map[string]struct{}{}
	for _, rule := range rules {
		key := semanticRuleFingerprint(rule)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		deduped = append(deduped, rule)
	}
	return deduped
}

func semanticRuleFingerprint(rule SemanticRule) string {
	var b strings.Builder
	b.WriteString(strings.TrimSpace(rule.Kind))
	b.WriteByte('|')
	b.WriteString(strings.TrimSpace(rule.Description))
	b.WriteByte('|')
	if rule.When != nil {
		b.WriteString(strings.TrimSpace(rule.When.Field))
		b.WriteByte('=')
		b.WriteString(fmt.Sprintf("%v", rule.When.Equals))
		b.WriteByte('|')
		for _, v := range rule.When.In {
			b.WriteString(fmt.Sprintf("%v,", v))
		}
	}
	b.WriteByte('|')
	for _, req := range rule.Require {
		b.WriteString(strings.TrimSpace(req.Field))
		b.WriteByte('>')
		b.WriteString(strings.TrimSpace(req.Target))
		b.WriteByte('#')
		b.WriteString(fmt.Sprintf("%d", req.MinCount))
		b.WriteByte(',')
	}
	b.WriteByte('|')
	for _, req := range rule.RequireAny {
		b.WriteString(strings.TrimSpace(req.Field))
		b.WriteByte('>')
		b.WriteString(strings.TrimSpace(req.Target))
		b.WriteByte('#')
		b.WriteString(fmt.Sprintf("%d", req.MinCount))
		b.WriteByte(',')
	}
	b.WriteByte('|')
	for _, field := range rule.Forbid {
		b.WriteString(strings.TrimSpace(field))
		b.WriteByte(',')
	}
	b.WriteByte('|')
	for _, minimum := range rule.Minimum {
		b.WriteString(strings.TrimSpace(minimum.Field))
		b.WriteByte('>')
		b.WriteString(fmt.Sprintf("%g", minimum.Value))
		b.WriteByte(',')
	}
	return b.String()
}

func updateSearchResourcePath(existingPath, nextPath string) string {
	existingPath = strings.TrimSpace(existingPath)
	nextPath = strings.TrimSpace(nextPath)
	switch {
	case existingPath == "":
		return nextPath
	case nextPath == "", nextPath == existingPath:
		return existingPath
	}
	return deriveSearchResourcePath([]string{existingPath, nextPath})
}

func deriveSearchResourcePath(paths []string) string {
	paths = uniqueSortedStrings(paths)
	switch len(paths) {
	case 0:
		return ""
	case 1:
		return paths[0]
	}

	for _, base := range paths {
		paramName := ""
		matches := true
		for _, candidate := range paths {
			if candidate == base {
				continue
			}
			nextParam, ok := optionalItemPathParameter(base, candidate)
			if !ok {
				matches = false
				break
			}
			if paramName == "" {
				paramName = nextParam
				continue
			}
			if paramName != nextParam {
				matches = false
				break
			}
		}
		if matches && paramName != "" {
			return base + "/{" + paramName + "?}"
		}
	}

	return ""
}

func optionalItemPathParameter(basePath, candidatePath string) (string, bool) {
	basePath = strings.TrimSpace(basePath)
	candidatePath = strings.TrimSpace(candidatePath)
	if basePath == "" || candidatePath == "" || !strings.HasPrefix(candidatePath, basePath) {
		return "", false
	}
	suffix := strings.TrimPrefix(candidatePath, basePath)
	if !strings.HasPrefix(suffix, "/{") || !strings.HasSuffix(suffix, "}") {
		return "", false
	}
	if strings.Count(suffix, "/") != 1 {
		return "", false
	}
	paramName := strings.TrimSuffix(strings.TrimPrefix(suffix, "/{"), "}")
	if paramName == "" || strings.Contains(paramName, "/") {
		return "", false
	}
	return paramName, true
}

func indexSearchPath(index map[string][]string, rawPath, resourceKey string) {
	path := strings.TrimSpace(rawPath)
	if path == "" || resourceKey == "" {
		return
	}

	keys := []string{path}
	keys = append(keys, searchPathAliases(path)...)

	for _, key := range keys {
		index[key] = append(index[key], resourceKey)
		lower := strings.ToLower(key)
		if lower != key {
			index[lower] = append(index[lower], resourceKey)
		}
	}
}

func searchPathAliases(path string) []string {
	segments := strings.Split(strings.Trim(strings.TrimSpace(path), "/"), "/")
	if len(segments) == 0 {
		return nil
	}

	var start int
	switch {
	case len(segments) >= 2 && strings.EqualFold(segments[0], "api") && isVersionSegment(segments[1]):
		start = 2
	case isVersionSegment(segments[0]):
		start = 1
	default:
		return nil
	}
	if start >= len(segments) {
		return []string{"/"}
	}
	return []string{"/" + strings.Join(segments[start:], "/")}
}

func splitSearchSDKMethod(sdkMethod string) (resourceKey, leaf string, ok bool) {
	parts := strings.Split(strings.TrimSpace(sdkMethod), ".")
	if len(parts) != 3 {
		return "", "", false
	}
	if parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", false
	}
	return parts[0] + "." + parts[1], parts[2], true
}

func searchResourceSchemaAlias(schemaName string) string {
	schemaName = strings.TrimSpace(schemaName)
	parts := strings.Split(schemaName, ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return ""
	}
	return parts[0] + "." + lowerCamelIdentifier(parts[1])
}

func selectSearchResourceSchema(spec NormalizedSpec, current string, method SDKMethod) string {
	current = strings.TrimSpace(current)
	if current != "" {
		return current
	}
	if candidate := canonicalResourceFromSDKStem(spec, method.SDKMethod); candidate != "" {
		return candidate
	}
	if candidate := strings.TrimSpace(method.Resource); candidate != "" && !isSupportSchemaName(candidate) {
		if _, ok := spec.Schemas[candidate]; ok {
			return candidate
		}
	}
	for _, candidate := range preferredSearchRelatedSchemas(method.RelatedSchemas) {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" || isSupportSchemaName(candidate) {
			continue
		}
		if _, ok := spec.Schemas[candidate]; ok {
			return candidate
		}
	}
	return ""
}

func preferredSearchRelatedSchemas(candidates []string) []string {
	if len(candidates) == 0 {
		return nil
	}
	out := append([]string(nil), candidates...)
	slices.SortStableFunc(out, func(a, b string) int {
		return cmp.Compare(searchSchemaPriority(a), searchSchemaPriority(b))
	})
	return out
}

func searchSchemaPriority(name string) int {
	name = strings.TrimSpace(name)
	switch {
	case strings.Contains(name, ".response."):
		return 0
	case strings.Contains(name, ".request"):
		return 1
	case strings.Contains(name, ".parameter."):
		return 3
	default:
		return 2
	}
}

func canonicalResourceFromSDKStem(spec NormalizedSpec, sdkMethod string) string {
	resourceKey, _, ok := splitSearchSDKMethod(sdkMethod)
	if !ok {
		return ""
	}
	parts := strings.Split(resourceKey, ".")
	if len(parts) != 2 {
		return ""
	}
	candidate := parts[0] + "." + upperCamelIdentifier(parts[1])
	if _, ok := spec.Schemas[candidate]; !ok || isSupportSchemaName(candidate) {
		return ""
	}
	return candidate
}
