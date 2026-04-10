package testutil

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"
)

const FakeAccessToken = "test-token"

type FakeIntersight struct {
	t      *testing.T
	server *TCP4Server

	mu                sync.Mutex
	rackUnits         map[string]map[string]any
	createdPolicies   []map[string]any
	createdLanPolicies []map[string]any
	createdEthIfs     []map[string]any
	tokenRequestCount int
}

func NewFakeIntersight(t *testing.T) *FakeIntersight {
	t.Helper()

	f := &FakeIntersight{
		t: t,
		rackUnits: map[string]map[string]any{
			"rack-1": {
				"Moid":         "rack-1",
				"ObjectType":   "compute.RackUnit",
				"Name":         "rack-alpha",
				"Model":        "UCS C240 M6",
				"Serial":       "ABC12345",
				"ManagementIp": "10.0.0.10",
			},
			"rack-2": {
				"Moid":         "rack-2",
				"ObjectType":   "compute.RackUnit",
				"Name":         "rack-beta",
				"Model":        "UCS C220 M7",
				"Serial":       "XYZ98765",
				"ManagementIp": "10.0.0.11",
			},
		},
	}

	f.server = NewTCP4Server(t, http.HandlerFunc(f.serveHTTP))
	return f
}

func (f *FakeIntersight) URL() string {
	f.t.Helper()
	return f.server.URL
}

func (f *FakeIntersight) Client() *http.Client {
	f.t.Helper()
	return f.server.Client()
}

func (f *FakeIntersight) Close() {
	f.t.Helper()
	f.server.Close()
}

func (f *FakeIntersight) TokenRequestCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.tokenRequestCount
}

func (f *FakeIntersight) LastCreatedPolicy() map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.createdPolicies) == 0 {
		return nil
	}
	return cloneMap(f.createdPolicies[len(f.createdPolicies)-1])
}

func (f *FakeIntersight) LastCreatedLanConnectivityPolicy() map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.createdLanPolicies) == 0 {
		return nil
	}
	return cloneMap(f.createdLanPolicies[len(f.createdLanPolicies)-1])
}

func (f *FakeIntersight) CreatedEthIfs() []map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.createdEthIfs) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(f.createdEthIfs))
	for _, payload := range f.createdEthIfs {
		out = append(out, cloneMap(payload))
	}
	return out
}

func (f *FakeIntersight) serveHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/iam/token":
		f.mu.Lock()
		f.tokenRequestCount++
		f.mu.Unlock()
		writeJSON(f.t, w, http.StatusOK, map[string]any{
			"access_token": FakeAccessToken,
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
		return
	case r.URL.Path == "/api/v1/iam/UserPreferences":
		if !f.requireBearer(w, r) {
			return
		}
		writeJSON(f.t, w, http.StatusOK, map[string]any{"Results": []any{}})
		return
	case r.URL.Path == "/api/v1/compute/RackUnits" && r.Method == http.MethodGet:
		if !f.requireBearer(w, r) {
			return
		}
		results := []any{
			cloneMap(f.rackUnits["rack-1"]),
			cloneMap(f.rackUnits["rack-2"]),
		}
		writeJSON(f.t, w, http.StatusOK, map[string]any{
			"Results": results,
			"Count":   len(results),
		})
		return
	case strings.HasPrefix(r.URL.Path, "/api/v1/compute/RackUnits/") && r.Method == http.MethodGet:
		if !f.requireBearer(w, r) {
			return
		}
		moid := strings.TrimPrefix(r.URL.Path, "/api/v1/compute/RackUnits/")
		payload, ok := f.rackUnits[moid]
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(f.t, w, http.StatusOK, cloneMap(payload))
		return
	case r.URL.Path == "/api/v1/ntp/Policies" && r.Method == http.MethodPost:
		if !f.requireBearer(w, r) {
			return
		}
		payload, ok := decodeJSONBody(f.t, w, r)
		if !ok {
			return
		}
		payload["Moid"] = "policy-1"
		payload["ObjectType"] = "ntp.Policy"

		f.mu.Lock()
		f.createdPolicies = append(f.createdPolicies, cloneMap(payload))
		f.mu.Unlock()

		writeJSON(f.t, w, http.StatusOK, payload)
		return
	case r.URL.Path == "/api/v1/vnic/LanConnectivityPolicies" && r.Method == http.MethodPost:
		if !f.requireBearer(w, r) {
			return
		}
		payload, ok := decodeJSONBody(f.t, w, r)
		if !ok {
			return
		}
		payload["Moid"] = "lan-policy-1"
		payload["ObjectType"] = "vnic.LanConnectivityPolicy"

		f.mu.Lock()
		f.createdLanPolicies = append(f.createdLanPolicies, cloneMap(payload))
		f.mu.Unlock()

		writeJSON(f.t, w, http.StatusOK, payload)
		return
	case r.URL.Path == "/api/v1/vnic/EthIfs" && r.Method == http.MethodPost:
		if !f.requireBearer(w, r) {
			return
		}
		payload, ok := decodeJSONBody(f.t, w, r)
		if !ok {
			return
		}
		index := 0
		f.mu.Lock()
		index = len(f.createdEthIfs) + 1
		payload["Moid"] = "ethif-" + strings.TrimSpace(string(rune('0'+index)))
		payload["ObjectType"] = "vnic.EthIf"
		f.createdEthIfs = append(f.createdEthIfs, cloneMap(payload))
		f.mu.Unlock()

		writeJSON(f.t, w, http.StatusOK, payload)
		return
	default:
		http.NotFound(w, r)
	}
}

func (f *FakeIntersight) requireBearer(w http.ResponseWriter, r *http.Request) bool {
	if got := r.Header.Get("Authorization"); got != "Bearer "+FakeAccessToken {
		writeJSON(f.t, w, http.StatusUnauthorized, map[string]any{"message": "missing or invalid bearer token"})
		return false
	}
	return true
}

func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func decodeJSONBody(t *testing.T, w http.ResponseWriter, r *http.Request) (map[string]any, bool) {
	t.Helper()

	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(t, w, http.StatusBadRequest, map[string]any{"message": "invalid JSON body"})
		return nil, false
	}
	return cloneMap(payload), true
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, payload any) {
	t.Helper()

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(body); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
}
