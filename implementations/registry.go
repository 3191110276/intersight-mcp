package implementations

import (
	"fmt"
	"slices"
	"strings"
	"sync"
)

var (
	targetRegistryMu sync.RWMutex
	targetRegistry   = map[string]Target{}
)

func RegisterTarget(target Target) {
	if target == nil {
		panic("implementations: cannot register nil target")
	}

	name := strings.TrimSpace(target.Name())
	if name == "" {
		panic("implementations: target name is required")
	}

	targetRegistryMu.Lock()
	defer targetRegistryMu.Unlock()

	if _, exists := targetRegistry[name]; exists {
		panic(fmt.Sprintf("implementations: target %q already registered", name))
	}
	targetRegistry[name] = target
}

func LookupTarget(name string) (Target, error) {
	targetRegistryMu.RLock()
	defer targetRegistryMu.RUnlock()

	target, ok := targetRegistry[strings.TrimSpace(name)]
	if !ok {
		return nil, fmt.Errorf("unknown provider %q", name)
	}
	return target, nil
}

func MustLookupTarget(name string) Target {
	target, err := LookupTarget(name)
	if err != nil {
		panic(err)
	}
	return target
}

func RegisteredTargetNames() []string {
	targetRegistryMu.RLock()
	defer targetRegistryMu.RUnlock()

	names := make([]string, 0, len(targetRegistry))
	for name := range targetRegistry {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}
