package sandbox

import (
	"encoding/json"
	"slices"

	"github.com/fastschema/qjs"
	"github.com/mimaurer/intersight-mcp/internal/contracts"
)

type searchRuntime struct {
	catalog          contracts.SearchCatalog
	schemas          map[string]contracts.NormalizedSchema
	resourceKeys     []string
	catalogPaths     []string
	metricGroupKeys  []string
	metricByNameKeys []string
	schemaKeys       []string
	baseCatalog      map[string]any
}

func loadSearchRuntime(specJSON, catalogJSON, rulesJSON, searchJSON []byte) (*searchRuntime, error) {
	if len(specJSON) == 0 || len(catalogJSON) == 0 || len(rulesJSON) == 0 || len(searchJSON) == 0 {
		return nil, nil
	}

	var spec contracts.NormalizedSpec
	if err := json.Unmarshal(specJSON, &spec); err != nil {
		return nil, contracts.ValidationError{Message: "decode embedded spec", Err: err}
	}

	var catalog contracts.SearchCatalog
	if err := json.Unmarshal(searchJSON, &catalog); err != nil {
		return nil, contracts.ValidationError{Message: "decode embedded search catalog", Err: err}
	}

	return newSearchRuntime(catalog, spec.Schemas), nil
}

func newSearchRuntime(catalog contracts.SearchCatalog, schemas map[string]contracts.NormalizedSchema) *searchRuntime {
	resourceKeys := append([]string(nil), catalog.ResourceNames...)
	catalogPaths := sortedMapKeys(catalog.Paths)
	metricGroupKeys := sortedMapKeys(catalog.Metrics.Groups)
	metricByNameKeys := sortedMapKeys(catalog.Metrics.ByName)
	schemaKeys := sortedMapKeys(schemas)

	return &searchRuntime{
		catalog:          catalog,
		schemas:          schemas,
		resourceKeys:     resourceKeys,
		catalogPaths:     catalogPaths,
		metricGroupKeys:  metricGroupKeys,
		metricByNameKeys: metricByNameKeys,
		schemaKeys:       schemaKeys,
		baseCatalog: map[string]any{
			"metadata":      catalog.Metadata,
			"resourceNames": resourceKeys,
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
	registerLookup("__intersight_search_metric_group_get__", func(key string) (any, bool) {
		value, ok := r.catalog.Metrics.Groups[key]
		return value, ok
	})
	registerLookup("__intersight_search_metric_by_name_get__", func(key string) (any, bool) {
		value, ok := r.catalog.Metrics.ByName[key]
		return value, ok
	})
	registerLookup("__intersight_search_schema_get__", func(key string) (any, bool) {
		value, ok := r.schemas[key]
		return value, ok
	})

	values := map[string]any{
		"__search_catalog_base":        r.baseCatalog,
		"__search_resource_keys":       r.resourceKeys,
		"__search_catalog_paths":       r.catalogPaths,
		"__search_metric_group_keys":   r.metricGroupKeys,
		"__search_metric_by_name_keys": r.metricByNameKeys,
		"__search_schema_keys":         r.schemaKeys,
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
	  const catalog = {
	    metadata: catalogBase.metadata,
	    resourceNames: catalogBase.resourceNames || [],
	    metrics: {
	      groups: createLookupProxy(__search_metric_group_keys, key => __intersight_search_metric_group_get__(key)),
      byName: createLookupProxy(__search_metric_by_name_keys, key => __intersight_search_metric_by_name_get__(key))
	    },
	    resources: createLookupProxy(__search_resource_keys, key => __intersight_search_resource_get__(key)),
	    paths: createLookupProxy(__search_catalog_paths, key => __intersight_search_catalog_path_get__(key)),
	    schema(name) {
	      if (typeof name !== 'string') {
	        return undefined;
	      }
	      return __intersight_search_schema_get__(name);
	    }
	  };

	  Object.freeze(catalog.resourceNames);
	  Object.freeze(catalog.metrics);
	  Object.freeze(catalog);

	  return { catalog };
	})()`))
	if err != nil {
		return contracts.InternalError{Message: "create search discovery wrapper", Err: err}
	}

	ctx.Global().SetPropertyStr("catalog", discoveryValue.GetPropertyStr("catalog"))
	return nil
}
