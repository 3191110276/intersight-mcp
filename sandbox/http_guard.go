package sandbox

import (
	"net/http"
	"strings"

	"github.com/mimaurer/intersight-mcp/internal/contracts"
)

func guardMethod(mode Mode, method string, dryRun bool) error {
	if mode == ModeValidate {
		return contracts.ValidationError{
			Message: "offline validation only supports write-shaped sdk calls; direct HTTP requests are unavailable during offline validation",
		}
	}
	if mode != ModeQuery {
		return nil
	}
	if dryRun {
		return contracts.ValidationError{
			Message: "query no longer supports api.call dry-run previews; use write-shaped sdk methods in query for non-persisting validation or mutate for persistent writes",
		}
	}
	normalized := strings.ToUpper(strings.TrimSpace(method))
	if normalized == http.MethodGet {
		return nil
	}
	return contracts.ValidationError{
		Message: "query only allows GET requests; use write-shaped sdk methods in query for offline validation or mutate for persistent writes",
	}
}
