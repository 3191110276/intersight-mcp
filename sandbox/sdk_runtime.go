package sandbox

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/fastschema/qjs"
	"github.com/mimaurer/intersight-mcp/internal/contracts"
)

const telemetryQuerySDKMethod = "telemetry.query"

type sdkRuntime struct {
	catalog     contracts.SDKCatalog
	rules       contracts.RuleCatalog
	spec        contracts.NormalizedSpec
	specIndex   *dryRunSpecIndex
	catalogJSON []byte
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
		catalog:     catalog,
		rules:       rules,
		spec:        spec,
		specIndex:   specIndex,
		catalogJSON: append([]byte(nil), catalogJSON...),
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

	catalogValue := ctx.ParseJSON(string(r.catalogJSON))
	ctx.Global().SetPropertyStr("__sdk_catalog", catalogValue)

	sdkWrapper, err := ctx.Eval("sdk_helper.js", qjs.Code(`(() => {
  const root = {};
  const methods = (__sdk_catalog && __sdk_catalog.methods) || {};
  for (const sdkMethod of Object.keys(methods)) {
    const parts = sdkMethod.split('.');
    let cursor = root;
    for (let i = 0; i < parts.length - 1; i++) {
      const key = parts[i];
      if (!cursor[key]) {
        cursor[key] = {};
      }
      cursor = cursor[key];
    }
    const leaf = parts[parts.length - 1];
    cursor[leaf] = function(args) {
      const normalizedArgs = args === undefined ? {} : args;
      return __intersight_sdk_call_async__(sdkMethod, normalizedArgs);
    };
  }
  if (!root.telemetry) {
    root.telemetry = {};
  }
	root.telemetry.query = function(args) {
		const normalizedArgs = args === undefined ? {} : args;
		return __intersight_sdk_call_async__('telemetry.query', normalizedArgs);
	};
  return root;
})()`))
	if err != nil {
		return contracts.InternalError{Message: "create sdk wrapper", Err: err}
	}
	ctx.Global().SetPropertyStr("sdk", sdkWrapper)
	return nil
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
