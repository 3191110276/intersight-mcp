package contracts

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
)

type ArtifactSourceMetadata struct {
	PublishedVersion string `json:"published_version"`
	SourceURL        string `json:"source_url"`
	SHA256           string `json:"sha256"`
	RetrievalDate    string `json:"retrieval_date"`
}

type NormalizedSpec struct {
	Metadata ArtifactSourceMetadata                    `json:"metadata"`
	Paths    map[string]map[string]NormalizedOperation `json:"paths"`
	Schemas  map[string]NormalizedSchema               `json:"schemas"`
	Tags     []NormalizedTag                           `json:"tags"`
}

type NormalizedTag struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type NormalizedOperation struct {
	Summary     string                        `json:"summary,omitempty"`
	OperationID string                        `json:"operationId,omitempty"`
	Tags        []string                      `json:"tags,omitempty"`
	Parameters  []NormalizedParameter         `json:"parameters,omitempty"`
	RequestBody *NormalizedRequestBody        `json:"requestBody,omitempty"`
	Responses   map[string]NormalizedResponse `json:"responses,omitempty"`
}

type NormalizedParameter struct {
	Name     string            `json:"name"`
	In       string            `json:"in"`
	Required bool              `json:"required"`
	Schema   *NormalizedSchema `json:"schema,omitempty"`
}

type NormalizedRequestBody struct {
	Required bool                              `json:"required"`
	Content  map[string]NormalizedMediaContent `json:"content"`
}

type NormalizedResponse struct {
	Description string                            `json:"description"`
	Content     map[string]NormalizedMediaContent `json:"content,omitempty"`
}

type NormalizedMediaContent struct {
	Schema *NormalizedSchema `json:"schema,omitempty"`
}

type NormalizedSchema struct {
	Type                   string                       `json:"type,omitempty"`
	Format                 string                       `json:"format,omitempty"`
	Properties             map[string]*NormalizedSchema `json:"properties,omitempty"`
	Required               []string                     `json:"required,omitempty"`
	Items                  *NormalizedSchema            `json:"items,omitempty"`
	Nullable               bool                         `json:"nullable,omitempty"`
	ReadOnly               bool                         `json:"readOnly,omitempty"`
	AdditionalProperties   any                          `json:"additionalProperties,omitempty"`
	OneOf                  []*NormalizedSchema          `json:"oneOf,omitempty"`
	AnyOf                  []*NormalizedSchema          `json:"anyOf,omitempty"`
	Circular               string                       `json:"$circular,omitempty"`
	ExpandTarget           string                       `json:"$expandTarget,omitempty"`
	Relationship           bool                         `json:"x-relationship,omitempty"`
	RelationshipTarget     string                       `json:"x-relationshipTarget,omitempty"`
	RelationshipWriteForms []string                     `json:"x-writeForms,omitempty"`
	Enum                   []any                        `json:"enum,omitempty"`
}

type SDKCatalog struct {
	Metadata ArtifactSourceMetadata `json:"metadata"`
	Methods  map[string]SDKMethod   `json:"methods"`
}

type SDKMethod struct {
	SDKMethod           string              `json:"sdkMethod"`
	Summary             string              `json:"summary,omitempty"`
	Tags                []string            `json:"tags,omitempty"`
	Descriptor          OperationDescriptor `json:"descriptor"`
	PathParameters      []string            `json:"pathParameters,omitempty"`
	QueryParameters     []string            `json:"queryParameters,omitempty"`
	HeaderParameters    []string            `json:"headerParameters,omitempty"`
	RequestBodyRequired bool                `json:"requestBodyRequired,omitempty"`
	RequestBodyFields   []string            `json:"requestBodyFields,omitempty"`
	RelatedSchemas      []string            `json:"relatedSchemas,omitempty"`
	Resource            string              `json:"resource,omitempty"`
}

func BuildSDKCatalog(spec NormalizedSpec) (SDKCatalog, error) {
	catalog := SDKCatalog{
		Metadata: spec.Metadata,
		Methods:  make(map[string]SDKMethod),
	}

	paths := sortedKeys(spec.Paths)
	for _, path := range paths {
		methods := spec.Paths[path]
		for _, method := range sortedKeys(methods) {
			op := methods[method]
			if op.OperationID == "" {
				return SDKCatalog{}, fmt.Errorf("sdk catalog generation failed: %s %s is missing operationId", strings.ToUpper(method), path)
			}

			sdkMethodID, err := buildSDKMethodID(path, method, op.OperationID)
			if err != nil {
				return SDKCatalog{}, err
			}
			if existing, ok := catalog.Methods[sdkMethodID]; ok {
				alternateExistingID, existingErr := buildSDKMethodIDWithTerminalPathParam(existing.Descriptor.PathTemplate, existing.Descriptor.Method, existing.Descriptor.OperationID)
				alternateID, alternateErr := buildSDKMethodIDWithTerminalPathParam(path, method, op.OperationID)
				switch {
				case existingErr == nil && alternateExistingID != sdkMethodID:
					if _, alternateExists := catalog.Methods[alternateExistingID]; !alternateExists {
						delete(catalog.Methods, sdkMethodID)
						existing.SDKMethod = alternateExistingID
						catalog.Methods[alternateExistingID] = existing
					} else {
						return SDKCatalog{}, fmt.Errorf(
							"sdk catalog generation failed: duplicate sdk method %q for operations %q and %q",
							sdkMethodID,
							existing.Descriptor.OperationID,
							op.OperationID,
						)
					}
				case alternateErr == nil && alternateID != sdkMethodID:
					if _, alternateExists := catalog.Methods[alternateID]; !alternateExists {
						sdkMethodID = alternateID
					} else {
						return SDKCatalog{}, fmt.Errorf(
							"sdk catalog generation failed: duplicate sdk method %q for operations %q and %q",
							sdkMethodID,
							existing.Descriptor.OperationID,
							op.OperationID,
						)
					}
				default:
					return SDKCatalog{}, fmt.Errorf(
						"sdk catalog generation failed: duplicate sdk method %q for operations %q and %q",
						sdkMethodID,
						existing.Descriptor.OperationID,
						op.OperationID,
					)
				}
			}

			descriptor := NewHTTPOperationDescriptor(method, path)
			descriptor.OperationID = op.OperationID

			entry := SDKMethod{
				SDKMethod:      sdkMethodID,
				Summary:        op.Summary,
				Tags:           append([]string(nil), op.Tags...),
				Descriptor:     descriptor,
				RelatedSchemas: collectOperationSchemaNames(op),
			}
			entry.PathParameters, entry.QueryParameters, entry.HeaderParameters = groupParameterNames(op.Parameters)
			if body := op.RequestBody; body != nil {
				entry.RequestBodyRequired = body.Required
				if media, ok := body.Content["application/json"]; ok && media.Schema != nil {
					bodySchema := media.Schema
					if bodySchema.Circular != "" && strings.HasPrefix(bodySchema.Circular, "inline.") {
						bodySchema = dereferenceSchema(spec, bodySchema)
					}
					entry.RequestBodyFields = sortedKeys(bodySchema.Properties)
				}
			}
			resource, err := deriveCanonicalResource(spec, entry, op)
			if err != nil {
				return SDKCatalog{}, err
			}
			entry.Resource = resource
			catalog.Methods[sdkMethodID] = entry
		}
	}

	return catalog, nil
}

func ValidateSDKCatalogAgainstSpec(spec NormalizedSpec, catalog SDKCatalog) error {
	if spec.Metadata != catalog.Metadata {
		return fmt.Errorf("embedded artifact validation failed: spec and sdk catalog metadata do not match")
	}

	if err := validateCatalogReferences(spec, catalog); err != nil {
		return err
	}

	expected, err := BuildSDKCatalog(spec)
	if err != nil {
		return err
	}
	expected = normalizeSDKCatalog(expected)
	catalog = normalizeSDKCatalog(catalog)
	if reflect.DeepEqual(expected, catalog) {
		return nil
	}

	for name := range expected.Methods {
		if _, ok := catalog.Methods[name]; !ok {
			return fmt.Errorf("embedded artifact validation failed: sdk catalog missing method %q", name)
		}
	}
	for name := range catalog.Methods {
		if _, ok := expected.Methods[name]; !ok {
			return fmt.Errorf("embedded artifact validation failed: sdk catalog contains unknown method %q", name)
		}
		if !reflect.DeepEqual(expected.Methods[name], catalog.Methods[name]) {
			return fmt.Errorf("embedded artifact validation failed: sdk catalog entry %q does not match embedded spec", name)
		}
	}
	return fmt.Errorf("embedded artifact validation failed: sdk catalog does not match embedded spec")
}

func validateCatalogReferences(spec NormalizedSpec, catalog SDKCatalog) error {
	for name, method := range catalog.Methods {
		if method.Descriptor.OperationID == "" {
			return fmt.Errorf("embedded artifact validation failed: sdk catalog method %q is missing operationId", name)
		}

		specOp, bodySchema, ok := findSpecOperationForDescriptor(spec, method.Descriptor)
		if !ok {
			return fmt.Errorf("embedded artifact validation failed: sdk catalog method %q points at unknown operation %q", name, method.Descriptor.OperationID)
		}

		if err := validateCatalogFieldReferences(spec, name, bodySchema, method.RequestBodyFields); err != nil {
			return err
		}
		if err := validateCatalogSchemaReferences(spec, name, method.RelatedSchemas); err != nil {
			return err
		}
		if method.Resource != "" {
			if _, ok := spec.Schemas[method.Resource]; !ok {
				return fmt.Errorf("embedded artifact validation failed: sdk catalog method %q points at unknown resource schema %q", name, method.Resource)
			}
		}

		pathParams, queryParams, headerParams := groupParameterNames(specOp.Parameters)
		if !slices.Equal(pathParams, method.PathParameters) {
			return fmt.Errorf("embedded artifact validation failed: sdk catalog method %q path parameters do not match embedded spec", name)
		}
		if !slices.Equal(queryParams, method.QueryParameters) {
			return fmt.Errorf("embedded artifact validation failed: sdk catalog method %q query parameters do not match embedded spec", name)
		}
		if !slices.Equal(headerParams, method.HeaderParameters) {
			return fmt.Errorf("embedded artifact validation failed: sdk catalog method %q header parameters do not match embedded spec", name)
		}
	}
	return nil
}

func findSpecOperationForDescriptor(spec NormalizedSpec, descriptor OperationDescriptor) (NormalizedOperation, *NormalizedSchema, bool) {
	methods, ok := spec.Paths[descriptor.PathTemplate]
	if !ok {
		return NormalizedOperation{}, nil, false
	}
	op, ok := methods[strings.ToLower(descriptor.Method)]
	if !ok || op.OperationID == "" || op.OperationID != descriptor.OperationID {
		return NormalizedOperation{}, nil, false
	}

	var bodySchema *NormalizedSchema
	if op.RequestBody != nil {
		if media, ok := op.RequestBody.Content["application/json"]; ok {
			bodySchema = media.Schema
		}
	}
	return op, bodySchema, true
}

func validateCatalogFieldReferences(spec NormalizedSpec, name string, bodySchema *NormalizedSchema, fields []string) error {
	if len(fields) == 0 {
		return nil
	}
	if bodySchema == nil {
		return fmt.Errorf("embedded artifact validation failed: sdk catalog method %q declares request body fields for an operation without an application/json request body", name)
	}
	if bodySchema.Circular != "" && strings.HasPrefix(bodySchema.Circular, "inline.") {
		bodySchema = dereferenceSchema(spec, bodySchema)
	}
	for _, field := range fields {
		if _, ok := bodySchema.Properties[field]; !ok {
			return fmt.Errorf("embedded artifact validation failed: sdk catalog method %q points at unknown request body field %q", name, field)
		}
	}
	return nil
}

func validateCatalogSchemaReferences(spec NormalizedSpec, name string, schemaNames []string) error {
	for _, schemaName := range schemaNames {
		if _, ok := spec.Schemas[schemaName]; !ok {
			return fmt.Errorf("embedded artifact validation failed: sdk catalog method %q points at unknown schema %q", name, schemaName)
		}
	}
	return nil
}

func normalizeSDKCatalog(catalog SDKCatalog) SDKCatalog {
	if catalog.Methods == nil {
		catalog.Methods = map[string]SDKMethod{}
	}
	for key, method := range catalog.Methods {
		method.Descriptor = normalizeOperationDescriptor(method.Descriptor)
		catalog.Methods[key] = method
	}
	return catalog
}

func normalizeOperationDescriptor(descriptor OperationDescriptor) OperationDescriptor {
	if descriptor.PathParams == nil {
		descriptor.PathParams = map[string]string{}
	}
	if descriptor.QueryParams == nil {
		descriptor.QueryParams = map[string][]string{}
	}
	if descriptor.Headers == nil {
		descriptor.Headers = map[string][]string{}
	}
	return descriptor
}

func buildSDKMethodID(path, method, operationID string) (string, error) {
	return buildSDKMethodIDWithVerbStrategy(path, method, operationID, false)
}

func buildSDKMethodIDWithTerminalPathParam(path, method, operationID string) (string, error) {
	return buildSDKMethodIDWithVerbStrategy(path, method, operationID, true)
}

func buildSDKMethodIDWithVerbStrategy(path, method, operationID string, useTerminalPathParam bool) (string, error) {
	segments := sdkMethodPathSegments(path)
	if len(segments) == 0 {
		return "", fmt.Errorf("sdk catalog generation failed: cannot derive sdk method for %s %s", strings.ToUpper(method), path)
	}
	namespace := lowerCamelIdentifier(segments[0])
	if namespace == "" {
		return "", fmt.Errorf("sdk catalog generation failed: cannot derive namespace for %s %s", strings.ToUpper(method), path)
	}

	var resourceParts []string
	var action string
	hasPathParam := false
	lastSegmentIsPathParam := false
	for i := 1; i < len(segments); i++ {
		segment := segments[i]
		if isPathParameter(segment) {
			hasPathParam = true
			if i == len(segments)-1 {
				lastSegmentIsPathParam = true
			}
			continue
		}
		if strings.EqualFold(segment, "Actions") {
			if i+1 >= len(segments) || isPathParameter(segments[i+1]) {
				return "", fmt.Errorf("sdk catalog generation failed: malformed action path for %s %s", strings.ToUpper(method), path)
			}
			action = segments[i+1]
			break
		}
		resourceParts = append(resourceParts, segment)
	}

	resource := lowerCamelIdentifier(joinPathParts(resourceParts))
	if resource == "" {
		resource = namespace
	}
	if resource == "" {
		resource = lowerCamelIdentifier(operationID)
	}
	pathParamForVerb := hasPathParam
	if useTerminalPathParam {
		pathParamForVerb = lastSegmentIsPathParam
	}
	verb := deriveSDKVerb(method, pathParamForVerb, action)
	if verb == "" {
		return "", fmt.Errorf("sdk catalog generation failed: cannot derive sdk method verb for %s %s", strings.ToUpper(method), path)
	}
	return namespace + "." + resource + "." + verb, nil
}

func sdkMethodPathSegments(path string) []string {
	rawSegments := strings.Split(strings.Trim(strings.TrimSpace(path), "/"), "/")
	if len(rawSegments) == 0 {
		return nil
	}

	segments := make([]string, 0, len(rawSegments))
	for _, segment := range rawSegments {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			continue
		}
		segments = append(segments, segment)
	}
	if len(segments) == 0 {
		return nil
	}

	if len(segments) >= 2 && strings.EqualFold(segments[0], "api") && isVersionSegment(segments[1]) {
		segments = segments[2:]
	} else if isVersionSegment(segments[0]) {
		segments = segments[1:]
	}

	if len(segments) == 0 {
		return nil
	}
	return segments
}

func isVersionSegment(segment string) bool {
	segment = strings.TrimSpace(segment)
	if len(segment) < 2 {
		return false
	}
	if segment[0] != 'v' && segment[0] != 'V' {
		return false
	}
	for i := 1; i < len(segment); i++ {
		if segment[i] < '0' || segment[i] > '9' {
			return false
		}
	}
	return true
}

func deriveSDKVerb(method string, hasPathParam bool, action string) string {
	if action != "" {
		return lowerCamelIdentifier(action)
	}
	switch strings.ToLower(strings.TrimSpace(method)) {
	case "get":
		if hasPathParam {
			return "get"
		}
		return "list"
	case "post":
		if !hasPathParam {
			return "create"
		}
		return "post"
	case "patch":
		return "update"
	case "put":
		return "replace"
	case "delete":
		return "delete"
	case "head":
		return "head"
	case "options":
		return "options"
	case "trace":
		return "trace"
	default:
		return strings.ToLower(strings.TrimSpace(method))
	}
}

func groupParameterNames(params []NormalizedParameter) (pathParams, queryParams, headerParams []string) {
	for _, param := range params {
		switch strings.ToLower(param.In) {
		case "path":
			pathParams = append(pathParams, param.Name)
		case "query":
			queryParams = append(queryParams, param.Name)
		case "header":
			headerParams = append(headerParams, param.Name)
		}
	}
	return uniqueSortedStrings(pathParams), uniqueSortedStrings(queryParams), uniqueSortedStrings(headerParams)
}

func collectOperationSchemaNames(op NormalizedOperation) []string {
	seen := map[string]struct{}{}
	for _, param := range op.Parameters {
		collectSchemaNames(param.Schema, seen)
	}
	if op.RequestBody != nil {
		for _, media := range op.RequestBody.Content {
			collectSchemaNames(media.Schema, seen)
		}
	}
	for _, response := range op.Responses {
		for _, media := range response.Content {
			collectSchemaNames(media.Schema, seen)
		}
	}
	return sortedSetKeys(seen)
}

func collectSchemaNames(schema *NormalizedSchema, out map[string]struct{}) {
	if schema == nil {
		return
	}
	if schema.Circular != "" {
		out[schema.Circular] = struct{}{}
	}
	if schema.ExpandTarget != "" {
		out[schema.ExpandTarget] = struct{}{}
	}
	if schema.RelationshipTarget != "" {
		out[schema.RelationshipTarget] = struct{}{}
	}
	for _, prop := range schema.Properties {
		collectSchemaNames(prop, out)
	}
	collectSchemaNames(schema.Items, out)
	if nested, ok := schema.AdditionalProperties.(*NormalizedSchema); ok {
		collectSchemaNames(nested, out)
	}
	for _, item := range schema.OneOf {
		collectSchemaNames(item, out)
	}
	for _, item := range schema.AnyOf {
		collectSchemaNames(item, out)
	}
}

func deriveCanonicalResource(spec NormalizedSpec, method SDKMethod, op NormalizedOperation) (string, error) {
	for _, candidates := range canonicalResourceCandidateGroups(spec, method, op) {
		if len(candidates) > 1 {
			return "", fmt.Errorf("sdk catalog generation failed: %q has ambiguous canonical resource candidates %q", method.SDKMethod, strings.Join(candidates, ", "))
		}
		if len(candidates) == 1 {
			return candidates[0], nil
		}
	}
	return "", nil
}

func canonicalResourceCandidateGroups(spec NormalizedSpec, method SDKMethod, op NormalizedOperation) [][]string {
	var groups [][]string
	if body := op.RequestBody; body != nil {
		if media, ok := body.Content["application/json"]; ok {
			if candidates := canonicalResourceCandidatesFromSchema(spec, media.Schema, nil); len(candidates) > 0 {
				groups = append(groups, candidates)
			}
		}
	}

	for _, status := range []string{"200", "201", "202", "203", "204", "default"} {
		response, ok := op.Responses[status]
		if !ok {
			continue
		}
		if media, ok := response.Content["application/json"]; ok {
			if candidates := canonicalResourceCandidatesFromSchema(spec, media.Schema, nil); len(candidates) > 0 {
				groups = append(groups, candidates)
			}
		}
	}

	if resource := canonicalResourceFromSummary(spec, method.Summary); resource != "" {
		groups = append(groups, []string{resource})
	}
	if resource := canonicalResourceFromSDKMethodName(spec, method.SDKMethod); resource != "" {
		groups = append(groups, []string{resource})
	}

	return groups
}

func canonicalResourceCandidatesFromSchema(spec NormalizedSpec, schema *NormalizedSchema, visited map[string]struct{}) []string {
	if schema == nil {
		return nil
	}
	if visited == nil {
		visited = map[string]struct{}{}
	}

	var candidates []string
	seen := map[string]struct{}{}
	add := func(items ...string) {
		for _, item := range items {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			if _, ok := spec.Schemas[item]; !ok {
				continue
			}
			if isSupportSchemaName(item) {
				continue
			}
			if _, ok := seen[item]; ok {
				continue
			}
			seen[item] = struct{}{}
			candidates = append(candidates, item)
		}
	}

	if schema.Circular != "" {
		if _, ok := visited[schema.Circular]; ok {
			return nil
		}
		if strings.HasPrefix(schema.Circular, "inline.") {
			nextVisited := cloneStringSet(visited)
			nextVisited[schema.Circular] = struct{}{}
			if target, ok := spec.Schemas[schema.Circular]; ok {
				return canonicalResourceCandidatesFromSchema(spec, &target, nextVisited)
			}
			return nil
		}
		if !isEnvelopeSchemaName(schema.Circular) {
			return []string{schema.Circular}
		}
		nextVisited := cloneStringSet(visited)
		nextVisited[schema.Circular] = struct{}{}
		if target, ok := spec.Schemas[schema.Circular]; ok {
			add(canonicalResourceCandidatesFromSchema(spec, &target, nextVisited)...)
		}
		return candidates
	}

	add(canonicalResourceCandidatesFromPayloadProperty(spec, schema.Properties["Results"], visited)...)
	add(canonicalResourceCandidatesFromPayloadProperty(spec, schema.Properties["Result"], visited)...)
	if len(candidates) == 0 && len(schema.Properties) == 1 {
		for _, property := range schema.Properties {
			add(canonicalResourceCandidatesFromSchema(spec, property, visited)...)
		}
	}
	add(canonicalResourceCandidatesFromSchema(spec, schema.Items, visited)...)

	if len(schema.OneOf) == 1 {
		add(canonicalResourceCandidatesFromSchema(spec, schema.OneOf[0], visited)...)
	}
	if len(schema.AnyOf) == 1 {
		add(canonicalResourceCandidatesFromSchema(spec, schema.AnyOf[0], visited)...)
	}

	return candidates
}

func canonicalResourceCandidatesFromPayloadProperty(spec NormalizedSpec, schema *NormalizedSchema, visited map[string]struct{}) []string {
	if schema == nil {
		return nil
	}
	if schema.Items != nil {
		return canonicalResourceCandidatesFromSchema(spec, schema.Items, visited)
	}
	return canonicalResourceCandidatesFromSchema(spec, schema, visited)
}

func canonicalResourceFromSummary(spec NormalizedSpec, summary string) string {
	start := strings.IndexByte(summary, '\'')
	if start < 0 {
		return ""
	}
	rest := summary[start+1:]
	end := strings.IndexByte(rest, '\'')
	if end < 0 {
		return ""
	}
	candidate := rest[:end]
	if _, ok := spec.Schemas[candidate]; !ok || isSupportSchemaName(candidate) {
		return ""
	}
	return candidate
}

func canonicalResourceFromSDKMethodName(spec NormalizedSpec, sdkMethod string) string {
	parts := strings.Split(strings.TrimSpace(sdkMethod), ".")
	if len(parts) < 3 {
		return ""
	}
	namespace := parts[0]
	resourcePart := upperCamelIdentifier(parts[1])
	if resourcePart == "" {
		return ""
	}
	candidate := namespace + "." + resourcePart
	if _, ok := spec.Schemas[candidate]; !ok || isSupportSchemaName(candidate) {
		return ""
	}
	return candidate
}

func isEnvelopeSchemaName(name string) bool {
	return strings.HasSuffix(name, ".Response")
}

func isSupportSchemaName(name string) bool {
	return name == "Error" || isEnvelopeSchemaName(name)
}

func cloneStringSet(in map[string]struct{}) map[string]struct{} {
	if len(in) == 0 {
		return map[string]struct{}{}
	}
	out := make(map[string]struct{}, len(in))
	for key := range in {
		out[key] = struct{}{}
	}
	return out
}

func joinPathParts(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	var builder strings.Builder
	for _, part := range parts {
		builder.WriteString(part)
	}
	return builder.String()
}

func lowerCamelIdentifier(value string) string {
	if value == "" {
		return ""
	}
	runes := []rune(value)
	runes[0] = []rune(strings.ToLower(string(runes[0])))[0]
	return string(runes)
}

func upperCamelIdentifier(value string) string {
	if value == "" {
		return ""
	}
	runes := []rune(value)
	runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
	return string(runes)
}

func isPathParameter(segment string) bool {
	return strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}")
}

func sortedKeys[T any](in map[string]T) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	for key := range in {
		out = append(out, key)
	}
	slices.Sort(out)
	return out
}

func sortedSetKeys(in map[string]struct{}) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	for key := range in {
		out = append(out, key)
	}
	slices.Sort(out)
	return out
}

func uniqueSortedStrings(values []string) []string {
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
