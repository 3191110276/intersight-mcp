package sandbox

import (
	"net/http"
	"strings"

	"github.com/mimaurer/intersight-mcp/internal/contracts"
)

func guardMethod(mode Mode, method string, dryRun bool) error {
	if mode == ModeValidate {
		return contracts.ValidationError{
			Message: "validate only supports write-shaped sdk calls; api.call is unavailable in validate",
		}
	}
	if mode != ModeQuery {
		return nil
	}
	if dryRun {
		return contracts.ValidationError{
			Message: "query no longer supports api.call dry-run previews; use validate for non-persisting write checks",
		}
	}
	normalized := strings.ToUpper(strings.TrimSpace(method))
	if normalized == http.MethodGet {
		return nil
	}
	return contracts.ValidationError{
		Message: "query only allows GET requests; use validate for offline write validation or mutate for persistent writes",
	}
}
