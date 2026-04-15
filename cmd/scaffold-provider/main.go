package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var providerPattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

func main() {
	var (
		provider    string
		title       string
		withFilter  bool
		withMetrics bool
	)

	flag.StringVar(&provider, "provider", "", "provider key, for example acme")
	flag.StringVar(&title, "title", "", "human-readable provider name")
	flag.BoolVar(&withFilter, "with-filter", false, "create a provider filter.yaml scaffold")
	flag.BoolVar(&withMetrics, "with-metrics", false, "create a provider metrics scaffold")
	flag.Parse()

	if err := run(runConfig{
		root:        ".",
		provider:    provider,
		title:       title,
		withFilter:  withFilter,
		withMetrics: withMetrics,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "scaffold-provider: %v\n", err)
		os.Exit(1)
	}
}

type runConfig struct {
	root        string
	provider    string
	title       string
	withFilter  bool
	withMetrics bool
}

func run(cfg runConfig) error {
	provider := strings.TrimSpace(cfg.provider)
	if !providerPattern.MatchString(provider) {
		return errors.New(`provider must match ^[a-z][a-z0-9-]*$`)
	}

	title := strings.TrimSpace(cfg.title)
	if title == "" {
		title = defaultTitle(provider)
	}

	files := scaffoldFiles(provider, title, cfg.withFilter, cfg.withMetrics)
	for path, contents := range files {
		absPath := filepath.Join(cfg.root, path)
		if _, err := os.Stat(absPath); err == nil {
			return fmt.Errorf("refusing to overwrite existing file %q", path)
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("stat %s: %w", path, err)
		}
		if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
			return fmt.Errorf("create directory for %s: %w", path, err)
		}
		if err := os.WriteFile(absPath, []byte(contents), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
	}

	return nil
}

func scaffoldFiles(provider, title string, withFilter, withMetrics bool) map[string]string {
	files := map[string]string{
		filepath.ToSlash(filepath.Join("cmd", provider+"-mcp", "main.go")):                               cmdMainTemplate(provider),
		filepath.ToSlash(filepath.Join("implementations", provider, "implementation.go")):                implementationTemplate(provider, title, withFilter, withMetrics),
		filepath.ToSlash(filepath.Join("implementations", provider, "config.go")):                        configTemplate(provider, title),
		filepath.ToSlash(filepath.Join("implementations", provider, "generated", "embed.go")):            embedTemplate(provider),
		filepath.ToSlash(filepath.Join("implementations", provider, "generated", "spec_resolved.json")):  "{}\n",
		filepath.ToSlash(filepath.Join("implementations", provider, "generated", "sdk_catalog.json")):    "{}\n",
		filepath.ToSlash(filepath.Join("implementations", provider, "generated", "rules.json")):          "{}\n",
		filepath.ToSlash(filepath.Join("implementations", provider, "generated", "search_catalog.json")): "{}\n",
		filepath.ToSlash(filepath.Join("third_party", provider, "openapi", "manifest.json")):             manifestTemplate(),
		filepath.ToSlash(filepath.Join("third_party", provider, "openapi", "raw", ".gitkeep")):           "",
	}
	if withFilter {
		files[filepath.ToSlash(filepath.Join("implementations", provider, "filter.yaml"))] = filterTemplate()
	}
	if withMetrics {
		files[filepath.ToSlash(filepath.Join("third_party", provider, "metrics", "search_metrics.json"))] = "{}\n"
	}
	return files
}

func cmdMainTemplate(provider string) string {
	return fmt.Sprintf(`package main

import (
	"fmt"
	"os"

	"github.com/mimaurer/intersight-mcp/implementations"
	_ "github.com/mimaurer/intersight-mcp/implementations/%[1]s"
	"github.com/mimaurer/intersight-mcp/internal/bootstrap"
)

var version = "dev"

var app = bootstrap.App{
	Target:  implementations.MustLookupTarget("%[1]s"),
	Version: version,
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "serve":
		if err := serve(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "serve: %%v\n", err)
			os.Exit(1)
		}
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	app.Usage(os.Stderr)
}

func serve(args []string) error {
	return app.Serve(args)
}
`, provider)
}

func implementationTemplate(provider, title string, withFilter, withMetrics bool) string {
	return fmt.Sprintf(`package %[1]s

import (
	"github.com/mimaurer/intersight-mcp/implementations"
	"github.com/mimaurer/intersight-mcp/implementations/%[2]s/generated"
	"github.com/mimaurer/intersight-mcp/internal/providerext"
)

type target struct{}

func init() {
	implementations.RegisterTarget(target{})
}

func (target) Name() string {
	return "%[2]s"
}

func (target) Artifacts() implementations.Artifacts {
	return implementations.Artifacts{
		ResolvedSpec:  generated.ResolvedSpecBytes(),
		SDKCatalog:    generated.SDKCatalogBytes(),
		Rules:         generated.RulesBytes(),
		SearchCatalog: generated.SearchCatalogBytes(),
	}
}

func (target) GenerationConfig() implementations.GenerationConfig {
	return implementations.StandardGenerationConfig("%[2]s", implementations.StandardGenerationConfigOptions{
		IncludeFilter:  %[4]t,
		IncludeMetrics: %[5]t,
	})
}

func (target) SandboxExtensions() providerext.Extensions {
	return providerext.Extensions{}
}

func (target) RuntimeMetadata() implementations.RuntimeMetadata {
	return implementations.RuntimeMetadata{
		ProviderName:    "%[3]s",
		ServerName:      "%[2]s-mcp",
		ConfigPrefix:    "%[6]s",
		DefaultEndpoint: "",
		AuthErrorHint:   "Check %[6]s_* provider configuration.",
		ToolDescriptions: implementations.ToolDescriptions{
			SearchTitle:       "%[3]s Spec Search",
			SearchDescription: "Search the %[3]s discovery catalog for resources and operations.",
			QueryTitle:        "%[3]s Query",
			QueryDescription:  "Run read-shaped SDK methods or offline validation for write-shaped methods.",
			MutateTitle:       "%[3]s Mutate",
			MutateDescription: "Run persistent write-shaped SDK methods against the %[3]s API.",
		},
	}
}

func (target) LoadConnectionConfig(args []string, environ []string) (implementations.ConnectionConfig, error) {
	return LoadConnectionConfig(args, environ)
}
`, packageName(provider), provider, title, withFilter, withMetrics, envPrefix(provider))
}

func configTemplate(provider, title string) string {
	return fmt.Sprintf(`package %[1]s

import (
	"context"
	"net/http"
	"time"

	"github.com/mimaurer/intersight-mcp/implementations"
	"github.com/mimaurer/intersight-mcp/internal/contracts"
)

type ConnectionConfig struct {
	ProxyURLRaw string
}

func (c ConnectionConfig) ProxyURL() string {
	return c.ProxyURLRaw
}

func (c ConnectionConfig) NewAPICaller(_ context.Context, _ time.Duration, _ *http.Client) implementations.APICaller {
	return unavailableAPICaller{}
}

func LoadConnectionConfig(_ []string, _ []string) (ConnectionConfig, error) {
	return ConnectionConfig{}, nil
}

type unavailableAPICaller struct{}

func (unavailableAPICaller) Do(_ context.Context, _ contracts.OperationDescriptor) (any, error) {
	return nil, contracts.InternalError{Message: "%[2]s API caller is not implemented"}
}
`, packageName(provider), title)
}

func embedTemplate(provider string) string {
	return fmt.Sprintf(`package generated

import _ "embed"

//go:generate sh -c "mkdir -p ../../../.cache/go-build ../../../.tmp && GOCACHE=$(cd ../../.. && pwd)/.cache/go-build GOTMPDIR=$(cd ../../.. && pwd)/.tmp go -C ../../.. run ./cmd/generate --provider %[1]s"

//go:embed spec_resolved.json
var specResolvedJSON []byte

//go:embed sdk_catalog.json
var sdkCatalogJSON []byte

//go:embed rules.json
var rulesJSON []byte

//go:embed search_catalog.json
var searchCatalogJSON []byte

func ResolvedSpecBytes() []byte {
	return append([]byte(nil), specResolvedJSON...)
}

func SDKCatalogBytes() []byte {
	return append([]byte(nil), sdkCatalogJSON...)
}

func RulesBytes() []byte {
	return append([]byte(nil), rulesJSON...)
}

func SearchCatalogBytes() []byte {
	return append([]byte(nil), searchCatalogJSON...)
}
`, provider)
}

func manifestTemplate() string {
	return "{\n  \"published_version\": \"\",\n  \"source_url\": \"\",\n  \"sha256\": \"\",\n  \"retrieval_date\": \"\"\n}\n"
}

func filterTemplate() string {
	return "denylist:\n  namespaces: []\n  pathPrefixes: []\n  operationIds: []\n"
}

func packageName(provider string) string {
	return strings.ReplaceAll(provider, "-", "")
}

func envPrefix(provider string) string {
	return strings.ToUpper(strings.ReplaceAll(provider, "-", "_"))
}

func defaultTitle(provider string) string {
	parts := strings.Split(provider, "-")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}
