package sandbox

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/mimaurer/intersight-mcp/internal/contracts"
)

func (b *apiBridge) preflightMutation(execCtx context.Context, sdkMethod string, operation contracts.OperationDescriptor) error {
	if !isServerProfileWriteMethod(sdkMethod) {
		return nil
	}
	body, ok := operation.Body.(map[string]any)
	if !ok {
		return nil
	}
	requested := requestedPolicyBucketEntries(body)
	if len(requested) == 0 {
		return nil
	}

	targetPlatform, err := b.serverProfileTargetPlatform(execCtx, operation, body)
	if err != nil {
		return err
	}
	if strings.TrimSpace(targetPlatform) == "" {
		return nil
	}

	for i, entry := range requested {
		if strings.TrimSpace(entry.ObjectType) == "" || strings.TrimSpace(entry.Moid) == "" {
			continue
		}
		policy, err := b.fetchRelatedObject(execCtx, entry.ObjectType, entry.Moid)
		if err != nil {
			return err
		}
		policyTarget := strings.TrimSpace(stringField(policy, "TargetPlatform"))
		if policyTarget == "" || strings.EqualFold(policyTarget, targetPlatform) {
			continue
		}
		return newSemanticValidationError(
			sdkMethod,
			dryRunValidationError{
				Path:     fmt.Sprintf("PolicyBucket[%d].ObjectType", i),
				Type:     "compatibility",
				Source:   validationSourceRules,
				Rule:     "policyBucket.targetPlatform",
				Message:  fmt.Sprintf("Policy %q target platform %q is incompatible with server profile target platform %q.", entry.ObjectType, policyTarget, targetPlatform),
				Expected: targetPlatform,
				Actual:   policyTarget,
			},
		)
	}

	return nil
}

func (b *apiBridge) verifyMutationResult(execCtx context.Context, sdkMethod string, operation contracts.OperationDescriptor, result any) error {
	if !isServerProfileWriteMethod(sdkMethod) {
		return nil
	}
	body, ok := operation.Body.(map[string]any)
	if !ok {
		return nil
	}
	requested := requestedPolicyBucketEntries(body)
	if len(requested) == 0 {
		return nil
	}

	profileMoid := serverProfileMoid(operation, result)
	if profileMoid == "" {
		return nil
	}
	persisted, err := b.fetchServerProfile(execCtx, profileMoid)
	if err != nil {
		return err
	}
	persistedBucket := requestedPolicyBucketEntries(persisted)

	requestedKeys := policyBucketKeys(requested)
	persistedKeys := policyBucketKeys(persistedBucket)
	if slices.Equal(requestedKeys, persistedKeys) {
		return nil
	}

	missing := difference(requestedKeys, persistedKeys)
	extra := difference(persistedKeys, requestedKeys)
	message := "Persisted server profile policy bucket differs from the requested policy bucket."
	if len(missing) > 0 {
		message += " Missing after persistence: " + strings.Join(missing, ", ") + "."
	}
	if len(extra) > 0 {
		message += " Unexpected persisted entries: " + strings.Join(extra, ", ") + "."
	}

	return newSemanticValidationError(
		sdkMethod,
		dryRunValidationError{
			Path:     "PolicyBucket",
			Type:     "persistence_mismatch",
			Source:   validationSourceRules,
			Rule:     "policyBucket.persistedMatch",
			Message:  message,
			Expected: requestedKeys,
			Actual:   persistedKeys,
		},
	)
}

func (b *apiBridge) serverProfileTargetPlatform(execCtx context.Context, operation contracts.OperationDescriptor, body map[string]any) (string, error) {
	if target := strings.TrimSpace(stringField(body, "TargetPlatform")); target != "" {
		return target, nil
	}
	_, moid := splitCollectionPath(operation.Path)
	if moid == "" {
		return "", nil
	}
	profile, err := b.fetchServerProfile(execCtx, moid)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(stringField(profile, "TargetPlatform")), nil
}

func (b *apiBridge) fetchServerProfile(execCtx context.Context, moid string) (map[string]any, error) {
	return b.fetchObjectByPath(execCtx, "/api/v1/server/Profiles/"+strings.TrimSpace(moid))
}

func (b *apiBridge) fetchRelatedObject(execCtx context.Context, objectType, moid string) (map[string]any, error) {
	basePath, ok := b.relationshipPath(objectType)
	if !ok {
		return nil, nil
	}
	return b.fetchObjectByPath(execCtx, strings.TrimRight(basePath, "/")+"/"+strings.TrimSpace(moid))
}

func (b *apiBridge) fetchObjectByPath(execCtx context.Context, requestPath string) (map[string]any, error) {
	value, err := b.executeOperation(execCtx, contracts.NewHTTPOperationDescriptor(http.MethodGet, requestPath))
	if err != nil {
		return nil, err
	}
	obj, _ := value.(map[string]any)
	return obj, nil
}

func serverProfileMoid(operation contracts.OperationDescriptor, result any) string {
	if obj, ok := result.(map[string]any); ok {
		if moid := strings.TrimSpace(stringField(obj, "Moid")); moid != "" {
			return moid
		}
	}
	if body, ok := operation.Body.(map[string]any); ok {
		if moid := strings.TrimSpace(stringField(body, "Moid")); moid != "" {
			return moid
		}
	}
	_, moid := splitCollectionPath(operation.Path)
	return moid
}

type policyBucketEntry struct {
	Moid       string
	ObjectType string
}

func requestedPolicyBucketEntries(body map[string]any) []policyBucketEntry {
	raw, _ := body["PolicyBucket"].([]any)
	if len(raw) == 0 {
		return nil
	}
	out := make([]policyBucketEntry, 0, len(raw))
	for _, entry := range raw {
		item, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, policyBucketEntry{
			Moid:       strings.TrimSpace(stringField(item, "Moid")),
			ObjectType: strings.TrimSpace(stringField(item, "ObjectType")),
		})
	}
	return out
}

func policyBucketKeys(entries []policyBucketEntry) []string {
	if len(entries) == 0 {
		return nil
	}
	keys := make([]string, 0, len(entries))
	for _, entry := range entries {
		keys = append(keys, entry.ObjectType+":"+entry.Moid)
	}
	slices.Sort(keys)
	return slices.Compact(keys)
}

func difference(left, right []string) []string {
	if len(left) == 0 {
		return nil
	}
	rightSet := make(map[string]struct{}, len(right))
	for _, item := range right {
		rightSet[item] = struct{}{}
	}
	out := make([]string, 0, len(left))
	for _, item := range left {
		if _, ok := rightSet[item]; !ok {
			out = append(out, item)
		}
	}
	return out
}

func isServerProfileWriteMethod(sdkMethod string) bool {
	switch strings.TrimSpace(sdkMethod) {
	case "server.profile.create", "server.profile.post", "server.profile.update", "server.profile.patch",
		"server.profiles.create", "server.profiles.post", "server.profiles.update", "server.profiles.patch":
		return true
	default:
		return false
	}
}

func newSemanticValidationError(sdkMethod string, issue dryRunValidationError) error {
	layers := defaultValidationLayers(true)
	layers[2].Passed = false
	return contracts.ValidationError{
		Message: fmt.Sprintf("sdk method %q request body failed local validation", sdkMethod),
		Details: map[string]any{
			"sdkMethod": sdkMethod,
			"issues":    []dryRunValidationError{issue},
			"layers":    layers,
		},
	}
}
