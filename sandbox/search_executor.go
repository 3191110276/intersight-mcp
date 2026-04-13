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
	exec.search, err = loadSearchRuntime(specJSON, catalogJSON, rulesJSON, publicSearchJSON)
	if err != nil {
		return nil, err
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
		search:           bundle.search,
	}, nil
}

type searchExecutor struct {
	cfg               Config
	specJSON          []byte
	catalogJSON       []byte
	rulesJSON         []byte
	publicSearchJSON  []byte
	search            *searchRuntime
	beforeLoadGlobals func(context.Context) error
}

func (e *searchExecutor) loadGlobals(ctx context.Context, rt *qjs.Runtime) error {
	if err := ctx.Err(); err != nil {
		return normalizeJSError(ctx, err)
	}
	if e.search != nil {
		if err := e.search.install(rt.Context()); err != nil {
			return err
		}
	} else {
		search := rt.Context().ParseJSON(string(e.publicSearchJSON))
		rt.Context().Global().SetPropertyStr("catalog", search)
		spec := rt.Context().ParseJSON(string(e.specJSON))
		sdk := rt.Context().ParseJSON(string(e.catalogJSON))
		rules := rt.Context().ParseJSON(string(e.rulesJSON))
		rt.Context().Global().SetPropertyStr("spec", spec)
		rt.Context().Global().SetPropertyStr("sdk", sdk)
		rt.Context().Global().SetPropertyStr("rules", rules)
	}
	if err := ctx.Err(); err != nil {
		return normalizeJSError(ctx, err)
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

func (e *searchExecutor) Execute(ctx context.Context, code string, mode Mode) (result Result, err error) {
	if mode != ModeSearch {
		return Result{}, contracts.ValidationError{Message: "search executor only supports search mode"}
	}

	execCtx, cancel := context.WithTimeout(ctx, e.cfg.SearchTimeout)
	defer cancel()
	defer func() {
		if recovered := recover(); recovered != nil {
			err = normalizePanic(execCtx, recovered)
		}
	}()
	logs := newLogBuffer(e.cfg.MaxOutputBytes)
	if e.beforeLoadGlobals != nil {
		if err := e.beforeLoadGlobals(execCtx); err != nil {
			return Result{}, normalizeJSError(execCtx, err)
		}
	}
	rtCtx, rtCancel := context.WithCancel(context.Background())
	defer rtCancel()

	rt, err := qjs.New(qjs.Option{
		Context:            rtCtx,
		CloseOnContextDone: true,
		MemoryLimit:        e.cfg.WASMMemoryBytes,
		Stdout:             logs,
		Stderr:             logs,
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
	stopCancel := context.AfterFunc(execCtx, rtCancel)
	defer stopCancel()
	if err := e.loadGlobals(execCtx, rt); err != nil {
		return Result{}, err
	}

	result, err = executeWithRuntime(execCtx, rt, code, mode, e.cfg.MaxCodeSize, e.cfg.MaxOutputBytes)
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
