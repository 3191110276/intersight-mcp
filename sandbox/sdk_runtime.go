package sandbox

import (
	"context"
	"encoding/json"
	"slices"
	"strings"

	"github.com/fastschema/qjs"
	"github.com/mimaurer/intersight-mcp/internal/contracts"
)

const telemetryQuerySDKMethod = "telemetry.query"

type sdkRuntime struct {
	catalog      contracts.SDKCatalog
	rules        contracts.RuleCatalog
	spec         contracts.NormalizedSpec
	specIndex    *dryRunSpecIndex
	namespaceMap map[string][]string
}

func loadSDKRuntime(specJSON, catalogJSON, rulesJSON []byte) (*sdkRuntime, error) {
	if len(specJSON) == 0 || len(catalogJSON) == 0 || len(rulesJSON) == 0 {
		return nil, nil
	}

	specIndex, err := loadDryRunSpecIndex(specJSON)
	if err != nil {
		return nil, err
	}

	var spec contracts.NormalizedSpec
	if err := json.Unmarshal(specJSON, &spec); err != nil {
		return nil, contracts.ValidationError{Message: "decode embedded spec", Err: err}
	}

	var catalog contracts.SDKCatalog
	if err := json.Unmarshal(catalogJSON, &catalog); err != nil {
		return nil, contracts.ValidationError{Message: "decode embedded sdk catalog", Err: err}
	}
	var rules contracts.RuleCatalog
	if err := json.Unmarshal(rulesJSON, &rules); err != nil {
		return nil, contracts.ValidationError{Message: "decode embedded rules", Err: err}
	}

	return &sdkRuntime{
		catalog:      catalog,
		rules:        rules,
		spec:         spec,
		specIndex:    specIndex,
		namespaceMap: buildSDKNamespaceMap(catalog),
	}, nil
}

func (r *sdkRuntime) install(ctx *qjs.Context, execCtx context.Context, bridge *apiBridge) error {
	if r == nil {
		return nil
	}

	ctx.SetAsyncFunc("__intersight_sdk_call_async__", func(this *qjs.This) {
		sdkMethod, args, err := decodeSDKCallArgs(this)
		if err != nil {
			rejectPromise(this, map[string]any{
				"kind":    "validation",
				"message": err.Error(),
			})
			return
		}

		response, err := bridge.callSDK(execCtx, sdkMethod, args)
		if err != nil {
			rejectPromise(this, err)
			return
		}
		if response["ok"] == false {
			rejectPromise(this, response["error"])
			return
		}
		resolvePromise(this, response["value"])
	})

	ctx.SetFunc("__intersight_sdk_has_method__", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return this.Context().NewBool(false), nil
		}
		_, ok := r.catalog.Methods[args[0].String()]
		return this.Context().NewBool(ok), nil
	})

	ctx.SetFunc("__intersight_sdk_list_children__", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		prefix := ""
		if len(args) > 0 {
			prefix = args[0].String()
		}
		children := r.namespaceMap[prefix]
		return qjs.ToJsValue(this.Context(), children)
	})

	sdkWrapper, err := ctx.Eval("sdk_helper.js", qjs.Code(`(() => {
  function dispatch(sdkMethod, args) {
    const normalizedArgs = args === undefined ? {} : args;
    return __intersight_sdk_call_async__(sdkMethod, normalizedArgs);
  }

  function createNamespace(prefix) {
    const children = __intersight_sdk_list_children__(prefix) || [];
    const target = Object.create(null);
    for (const child of children) {
      target[child] = true;
    }
    return new Proxy(target, {
      get(target, prop, receiver) {
        if (typeof prop !== 'string') {
          return Reflect.get(target, prop, receiver);
        }
        if (Object.prototype.hasOwnProperty.call(target, prop) && target[prop] !== true) {
          return Reflect.get(target, prop, receiver);
        }
        const next = prefix ? prefix + '.' + prop : prop;
        if (__intersight_sdk_has_method__(next)) {
          return function(args) {
            return dispatch(next, args);
          };
        }
        if (Object.prototype.hasOwnProperty.call(target, prop)) {
          return createNamespace(next);
        }
        return undefined;
      }
    });
  }

  const root = createNamespace('');
  if (!root.telemetry) {
    root.telemetry = {};
  }
  root.telemetry.query = function(args) {
    return dispatch('telemetry.query', args);
  };
  return root;
})()`))
	if err != nil {
		return contracts.InternalError{Message: "create sdk wrapper", Err: err}
	}
	ctx.Global().SetPropertyStr("sdk", sdkWrapper)
	return nil
}

func buildSDKNamespaceMap(catalog contracts.SDKCatalog) map[string][]string {
	namespaces := map[string]map[string]struct{}{
		"": {},
	}

	for sdkMethod := range catalog.Methods {
		parts := strings.Split(strings.TrimSpace(sdkMethod), ".")
		if len(parts) == 0 {
			continue
		}
		prefix := ""
		for i, part := range parts {
			if part == "" {
				continue
			}
			namespaces[prefix][part] = struct{}{}
			if i == len(parts)-1 {
				break
			}
			next := part
			if prefix != "" {
				next = prefix + "." + part
			}
			if _, ok := namespaces[next]; !ok {
				namespaces[next] = map[string]struct{}{}
			}
			prefix = next
		}
	}

	out := make(map[string][]string, len(namespaces))
	for prefix, children := range namespaces {
		items := make([]string, 0, len(children))
		for child := range children {
			items = append(items, child)
		}
		slices.Sort(items)
		out[prefix] = items
	}
	return out
}

func decodeSDKCallArgs(this *qjs.This) (string, map[string]any, error) {
	args := this.Args()
	if len(args) < 1 {
		return "", nil, contracts.ValidationError{Message: "sdk method dispatch is missing the operation id"}
	}

	sdkMethod := args[0].String()
	if len(args) < 2 || args[1].IsUndefined() || args[1].IsNull() {
		return sdkMethod, map[string]any{}, nil
	}

	decoded, err := qjs.ToGoValue[map[string]any](args[1])
	if err != nil {
		return "", nil, contracts.ValidationError{Message: "sdk method arguments must be an object", Err: err}
	}
	return sdkMethod, decoded, nil
}

func isCustomSDKMethod(sdkMethod string) bool {
	return strings.TrimSpace(sdkMethod) == telemetryQuerySDKMethod
}
