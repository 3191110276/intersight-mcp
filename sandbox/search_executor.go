package sandbox

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/fastschema/qjs"
	"github.com/mimaurer/intersight-mcp/internal/contracts"
)

// NewSearchExecutor builds a search executor backed by immutable JSON artifacts.
// It creates a fresh QuickJS runtime for each execution; runtimes are not pooled.
func NewSearchExecutor(cfg Config, specJSON, catalogJSON, rulesJSON, searchJSON []byte) (Executor, error) {
	cfg = normalizeConfig(cfg)
	if !json.Valid(specJSON) {
		return nil, contracts.ValidationError{Message: "embedded spec is not valid JSON"}
	}
	if !json.Valid(catalogJSON) {
		return nil, contracts.ValidationError{Message: "embedded sdk catalog is not valid JSON"}
	}
	if !json.Valid(rulesJSON) {
		return nil, contracts.ValidationError{Message: "embedded rules are not valid JSON"}
	}
	if !json.Valid(searchJSON) {
		return nil, contracts.ValidationError{Message: "embedded search catalog is not valid JSON"}
	}

	publicSearchJSON, err := redactSearchCatalogPublicFields(searchJSON)
	if err != nil {
		return nil, err
	}

	exec := &searchExecutor{
		cfg:              cfg,
		specJSON:         append([]byte(nil), specJSON...),
		catalogJSON:      append([]byte(nil), catalogJSON...),
		rulesJSON:        append([]byte(nil), rulesJSON...),
		publicSearchJSON: publicSearchJSON,
	}
	return exec, nil
}

// NewSearchExecutorFromBundle reuses the parsed immutable artifact bundle, but
// still creates a fresh QuickJS runtime for each search execution.
func NewSearchExecutorFromBundle(cfg Config, bundle *ArtifactBundle) (Executor, error) {
	if bundle == nil {
		return nil, contracts.ValidationError{Message: "artifact bundle is required"}
	}
	cfg = normalizeConfig(cfg)
	return &searchExecutor{
		cfg:              cfg,
		specJSON:         append([]byte(nil), bundle.specJSON...),
		catalogJSON:      append([]byte(nil), bundle.catalogJSON...),
		rulesJSON:        append([]byte(nil), bundle.rulesJSON...),
		publicSearchJSON: append([]byte(nil), bundle.publicSearchJSON...),
	}, nil
}

type searchExecutor struct {
	cfg              Config
	specJSON         []byte
	catalogJSON      []byte
	rulesJSON        []byte
	publicSearchJSON []byte
}

func (e *searchExecutor) loadGlobals(rt *qjs.Runtime) error {
	spec := rt.Context().ParseJSON(string(e.specJSON))
	sdk := rt.Context().ParseJSON(string(e.catalogJSON))
	rules := rt.Context().ParseJSON(string(e.rulesJSON))
	search := rt.Context().ParseJSON(string(e.publicSearchJSON))
	rt.Context().Global().SetPropertyStr("catalog", search)
	rt.Context().Global().SetPropertyStr("spec", spec)
	rt.Context().Global().SetPropertyStr("sdk", sdk)
	rt.Context().Global().SetPropertyStr("rules", rules)
	if _, err := rt.Context().Eval("freeze_spec.js", qjs.Code(`
const __freezeSeen = new WeakSet();
function __deepFreeze(value) {
  if (!value || typeof value !== 'object' || __freezeSeen.has(value)) {
    return value;
  }
  __freezeSeen.add(value);
  for (const key of Object.getOwnPropertyNames(value)) {
    __deepFreeze(value[key]);
  }
  return Object.freeze(value);
}
__deepFreeze(sdk);
__deepFreeze(rules);
__deepFreeze(catalog);
__deepFreeze(spec);
`)); err != nil {
		return normalizeJSError(context.Background(), err)
	}
	return nil
}

func redactSearchCatalogPublicFields(searchJSON []byte) ([]byte, error) {
	var catalog contracts.SearchCatalog
	if err := json.Unmarshal(searchJSON, &catalog); err != nil {
		return nil, contracts.ValidationError{
			Message: fmt.Sprintf("embedded search catalog is not valid JSON: %v", err),
		}
	}

	for resourceKey, resource := range catalog.Resources {
		if slices.Contains(resource.Operations, "update") {
			filtered := make([]string, 0, len(resource.Operations))
			for _, operation := range resource.Operations {
				if operation != "post" {
					filtered = append(filtered, operation)
				}
			}
			resource.Operations = filtered
		}
		catalog.Resources[resourceKey] = resource
	}

	redacted, err := json.Marshal(catalog)
	if err != nil {
		return nil, contracts.InternalError{Message: "serialize redacted search catalog", Err: err}
	}
	return redacted, nil
}

func (e *searchExecutor) Execute(ctx context.Context, code string, mode Mode) (Result, error) {
	if mode != ModeSearch {
		return Result{}, contracts.ValidationError{Message: "search executor only supports search mode"}
	}

	execCtx, cancel := context.WithTimeout(ctx, e.cfg.SearchTimeout)
	defer cancel()
	rtCtx, rtCancel := context.WithCancel(context.Background())
	defer rtCancel()

	rt, err := qjs.New(qjs.Option{
		Context:            rtCtx,
		CloseOnContextDone: true,
		MemoryLimit:        e.cfg.WASMMemoryBytes,
	})
	if err != nil {
		return Result{}, contracts.InternalError{Message: "create search QuickJS runtime", Err: err}
	}
	defer func() {
		defer func() {
			_ = recover()
		}()
		rt.Close()
	}()
	if err := e.loadGlobals(rt); err != nil {
		return Result{}, err
	}

	logs := &logBuffer{}
	stopCancel := context.AfterFunc(execCtx, rtCancel)
	defer stopCancel()
	result, err := executeWithRuntime(execCtx, rt, code, mode, e.cfg.MaxCodeSize, e.cfg.MaxOutputBytes)
	if err != nil {
		if len(result.Logs) == 0 {
			result.Logs = logs.Lines()
		}
		return result, err
	}
	if len(result.Logs) == 0 {
		result.Logs = logs.Lines()
	}
	return result, nil
}

func (e *searchExecutor) Close() error {
	return nil
}
