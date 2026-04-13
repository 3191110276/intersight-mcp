package sandbox

import (
	"encoding/json"
	"slices"

	"github.com/fastschema/qjs"
	"github.com/mimaurer/intersight-mcp/internal/contracts"
)

type searchRuntime struct {
	catalog           contracts.SearchCatalog
	spec              contracts.NormalizedSpec
	sdk               contracts.SDKCatalog
	rules             contracts.RuleCatalog
	resourceKeys      []string
	catalogPaths      []string
	metricGroupKeys   []string
	metricByNameKeys  []string
	metricExampleKeys []string
	specPaths         []string
	schemaKeys        []string
	sdkMethodKeys     []string
	ruleMethodKeys    []string
	baseCatalog       map[string]any
	baseSpec          map[string]any
	baseSDK           map[string]any
	baseRules         map[string]any
}

func loadSearchRuntime(specJSON, catalogJSON, rulesJSON, searchJSON []byte) (*searchRuntime, error) {
	if len(specJSON) == 0 || len(catalogJSON) == 0 || len(rulesJSON) == 0 || len(searchJSON) == 0 {
		return nil, nil
	}

	var spec contracts.NormalizedSpec
	if err := json.Unmarshal(specJSON, &spec); err != nil {
		return nil, contracts.ValidationError{Message: "decode embedded spec", Err: err}
	}

	var sdk contracts.SDKCatalog
	if err := json.Unmarshal(catalogJSON, &sdk); err != nil {
		return nil, contracts.ValidationError{Message: "decode embedded sdk catalog", Err: err}
	}

	var rules contracts.RuleCatalog
	if err := json.Unmarshal(rulesJSON, &rules); err != nil {
		return nil, contracts.ValidationError{Message: "decode embedded rules", Err: err}
	}

	var catalog contracts.SearchCatalog
	if err := json.Unmarshal(searchJSON, &catalog); err != nil {
		return nil, contracts.ValidationError{Message: "decode embedded search catalog", Err: err}
	}

	return newSearchRuntime(spec, sdk, rules, catalog), nil
}

func newSearchRuntime(spec contracts.NormalizedSpec, sdk contracts.SDKCatalog, rules contracts.RuleCatalog, catalog contracts.SearchCatalog) *searchRuntime {
	resourceKeys := append([]string(nil), catalog.ResourceNames...)
	catalogPaths := sortedMapKeys(catalog.Paths)
	metricGroupKeys := sortedMapKeys(catalog.Metrics.Groups)
	metricByNameKeys := sortedMapKeys(catalog.Metrics.ByName)
	metricExampleKeys := sortedMapKeys(catalog.Metrics.Examples)
	specPaths := sortedNestedKeys(spec.Paths)
	schemaKeys := sortedMapKeys(spec.Schemas)
	sdkMethodKeys := sortedMapKeys(sdk.Methods)
	ruleMethodKeys := sortedMapKeys(rules.Methods)

	return &searchRuntime{
		catalog:           catalog,
		spec:              spec,
		sdk:               sdk,
		rules:             rules,
		resourceKeys:      resourceKeys,
		catalogPaths:      catalogPaths,
		metricGroupKeys:   metricGroupKeys,
		metricByNameKeys:  metricByNameKeys,
		metricExampleKeys: metricExampleKeys,
		specPaths:         specPaths,
		schemaKeys:        schemaKeys,
		sdkMethodKeys:     sdkMethodKeys,
		ruleMethodKeys:    ruleMethodKeys,
		baseCatalog: map[string]any{
			"metadata":      catalog.Metadata,
			"resourceNames": resourceKeys,
		},
		baseSpec: map[string]any{
			"metadata": spec.Metadata,
			"tags":     append([]contracts.NormalizedTag(nil), spec.Tags...),
		},
		baseSDK: map[string]any{
			"metadata": sdk.Metadata,
		},
		baseRules: map[string]any{
			"metadata": rules.Metadata,
		},
	}
}

func sortedMapKeys[T any](items map[string]T) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func sortedNestedKeys[T any](items map[string]T) []string {
	return sortedMapKeys(items)
}

func (r *searchRuntime) install(ctx *qjs.Context) error {
	if r == nil {
		return nil
	}

	registerLookup := func(name string, getter func(string) (any, bool)) {
		ctx.SetFunc(name, func(this *qjs.This) (*qjs.Value, error) {
			args := this.Args()
			if len(args) < 1 {
				return this.Context().NewUndefined(), nil
			}
			value, ok := getter(args[0].String())
			if !ok {
				return this.Context().NewUndefined(), nil
			}
			return qjs.ToJsValue(this.Context(), value)
		})
	}

	registerLookup("__intersight_search_resource_get__", func(key string) (any, bool) {
		value, ok := r.catalog.Resources[key]
		return value, ok
	})
	registerLookup("__intersight_search_catalog_path_get__", func(key string) (any, bool) {
		value, ok := r.catalog.Paths[key]
		return value, ok
	})
	registerLookup("__intersight_search_spec_path_get__", func(key string) (any, bool) {
		value, ok := r.spec.Paths[key]
		return value, ok
	})
	registerLookup("__intersight_search_schema_get__", func(key string) (any, bool) {
		value, ok := r.spec.Schemas[key]
		return value, ok
	})
	registerLookup("__intersight_search_sdk_method_get__", func(key string) (any, bool) {
		value, ok := r.sdk.Methods[key]
		return value, ok
	})
	registerLookup("__intersight_search_rule_method_get__", func(key string) (any, bool) {
		value, ok := r.rules.Methods[key]
		return value, ok
	})
	registerLookup("__intersight_search_metric_group_get__", func(key string) (any, bool) {
		value, ok := r.catalog.Metrics.Groups[key]
		return value, ok
	})
	registerLookup("__intersight_search_metric_by_name_get__", func(key string) (any, bool) {
		value, ok := r.catalog.Metrics.ByName[key]
		return value, ok
	})
	registerLookup("__intersight_search_metric_example_get__", func(key string) (any, bool) {
		value, ok := r.catalog.Metrics.Examples[key]
		return value, ok
	})

	values := map[string]any{
		"__search_catalog_base":        r.baseCatalog,
		"__search_spec_base":           r.baseSpec,
		"__search_sdk_base":            r.baseSDK,
		"__search_rules_base":          r.baseRules,
		"__search_resource_keys":       r.resourceKeys,
		"__search_catalog_paths":       r.catalogPaths,
		"__search_metric_group_keys":   r.metricGroupKeys,
		"__search_metric_by_name_keys": r.metricByNameKeys,
		"__search_metric_example_keys": r.metricExampleKeys,
		"__search_spec_paths":          r.specPaths,
		"__search_schema_keys":         r.schemaKeys,
		"__search_sdk_method_keys":     r.sdkMethodKeys,
		"__search_rule_method_keys":    r.ruleMethodKeys,
	}
	for name, value := range values {
		jsValue, err := qjs.ToJsValue(ctx, value)
		if err != nil {
			return contracts.InternalError{Message: "create search runtime globals", Err: err}
		}
		ctx.Global().SetPropertyStr(name, jsValue)
	}

	discoveryValue, err := ctx.Eval("search_helper.js", qjs.Code(`(() => {
  function createLookupProxy(keys, getter) {
    const target = Object.create(null);
    for (const key of keys || []) {
      target[key] = true;
    }
    return new Proxy(target, {
      get(target, prop, receiver) {
        if (typeof prop === 'string' && Object.prototype.hasOwnProperty.call(target, prop)) {
          return getter(prop);
        }
        return Reflect.get(target, prop, receiver);
      },
      set() {
        return false;
      },
      defineProperty() {
        return false;
      },
      deleteProperty() {
        return false;
      }
    });
  }

  const catalogBase = __search_catalog_base || {};
  const specBase = __search_spec_base || {};
  const sdkBase = __search_sdk_base || {};
  const rulesBase = __search_rules_base || {};

  const catalog = {
    metadata: catalogBase.metadata,
    resourceNames: catalogBase.resourceNames || [],
    metrics: {
      groups: createLookupProxy(__search_metric_group_keys, key => __intersight_search_metric_group_get__(key)),
      byName: createLookupProxy(__search_metric_by_name_keys, key => __intersight_search_metric_by_name_get__(key)),
      examples: createLookupProxy(__search_metric_example_keys, key => __intersight_search_metric_example_get__(key))
    },
    resources: createLookupProxy(__search_resource_keys, key => __intersight_search_resource_get__(key)),
    paths: createLookupProxy(__search_catalog_paths, key => __intersight_search_catalog_path_get__(key))
  };
  const spec = {
    metadata: specBase.metadata,
    tags: specBase.tags || [],
    paths: createLookupProxy(__search_spec_paths, key => __intersight_search_spec_path_get__(key)),
    schemas: createLookupProxy(__search_schema_keys, key => __intersight_search_schema_get__(key))
  };
  const sdk = {
    metadata: sdkBase.metadata,
    methods: createLookupProxy(__search_sdk_method_keys, key => __intersight_search_sdk_method_get__(key))
  };
  const rules = {
    metadata: rulesBase.metadata,
    methods: createLookupProxy(__search_rule_method_keys, key => __intersight_search_rule_method_get__(key))
  };

  Object.freeze(catalog.resourceNames);
  Object.freeze(catalog.metrics);
  Object.freeze(spec.tags);
  Object.freeze(catalog);
  Object.freeze(spec);
  Object.freeze(sdk);
  Object.freeze(rules);

  return { catalog, spec, sdk, rules };
})()`))
	if err != nil {
		return contracts.InternalError{Message: "create search discovery wrapper", Err: err}
	}

	ctx.Global().SetPropertyStr("catalog", discoveryValue.GetPropertyStr("catalog"))
	ctx.Global().SetPropertyStr("spec", discoveryValue.GetPropertyStr("spec"))
	ctx.Global().SetPropertyStr("sdk", discoveryValue.GetPropertyStr("sdk"))
	ctx.Global().SetPropertyStr("rules", discoveryValue.GetPropertyStr("rules"))
	return nil
}
