package matcher

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
	"time"

	"mockbridge/internal/model"
)

func TestMatchDimensionsAndPriority(t *testing.T) {
	var body any
	_ = json.Unmarshal([]byte(`{"user":{"id":1001},"items":[{"name":"alpha"}]}`), &body)
	low := snapshot("low", 10, model.MatchRules{Method: "*", Path: "/api/mock/**"})
	high := snapshot("high", 100, model.MatchRules{Method: "POST", Path: "/api/mock/users/{id}", Headers: []model.KeyValueRule{{Name: "X-Token", Value: "test-*"}}, Query: []model.KeyValueRule{{Name: "type", Value: "vip"}}, Body: []model.JSONPathRule{{JSONPath: "$.user.id", Value: "1001"}, {JSONPath: "$.items[*].name", Value: "a*"}}})
	req := Request{Method: "POST", Path: "/api/mock/users/88", Header: http.Header{"X-Token": []string{"test-abc"}}, Query: url.Values{"type": []string{"vip"}}, Body: body}
	got, fail := Match(req, []model.ContractSnapshot{low, high})
	if got == nil || got.Contract.Key != "high" || got.PathParams["id"] != "88" {
		t.Fatalf("unexpected match %#v failures=%v", got, fail)
	}
	if got.Detail != "priority=100 matched: method,path,headers,query,body" {
		t.Fatalf("detail=%s", got.Detail)
	}
	req.Header.Del("X-Token")
	got, _ = Match(req, []model.ContractSnapshot{high})
	if got != nil {
		t.Fatal("missing header should not match")
	}
}
func TestPathAndJSONPathEdges(t *testing.T) {
	for _, tc := range []struct {
		path, pattern string
		ok            bool
	}{{"/files/a/b", "/files/**", true}, {"/a/x", "/a/*", true}, {"/a/x/y", "/a/*", false}, {"/", "/", true}, {"/u/7", "/u/{id}", true}} {
		_, ok := pathMatch(tc.path, tc.pattern)
		if ok != tc.ok {
			t.Errorf("%s %s=%v", tc.path, tc.pattern, ok)
		}
	}
	root := map[string]any{"a": []any{map[string]any{"id": float64(7)}, map[string]any{"id": float64(8)}}}
	v, ok := JSONPath(root, "$.a[1].id")
	if !ok || normalize(v[0]) != "8" {
		t.Fatalf("jsonpath %#v %v", v, ok)
	}
	if _, ok = JSONPath(root, "$.a[x]"); ok {
		t.Fatal("invalid index")
	}
	if wildcardMatch("abc", "a*d") {
		t.Fatal("wildcard mismatch")
	}
	if !wildcardMatch("abcd", "a*d") {
		t.Fatal("wildcard match")
	}
}
func snapshot(key string, priority int, rules model.MatchRules) model.ContractSnapshot {
	return model.ContractSnapshot{Key: key, Priority: priority, Enabled: true, CreatedAt: time.Now(), Scenario: model.ScenarioSnapshot{Name: "default", Rules: rules}}
}

func TestDisabledContractDoesNotBlockEnabledMatch(t *testing.T) {
	rules := model.MatchRules{Method: "GET", Path: "/api/mock/**"}
	disabled := snapshot("disabled", 200, rules)
	disabled.Enabled = false
	enabled := snapshot("enabled", 100, rules)
	got, failures := Match(Request{Method: "GET", Path: "/api/mock/x"}, []model.ContractSnapshot{disabled, enabled})
	if got == nil || got.Contract.Key != "enabled" {
		t.Fatalf("match=%+v failures=%+v", got, failures)
	}
	got, failures = Match(Request{Method: "GET", Path: "/api/mock/x"}, []model.ContractSnapshot{disabled})
	if got != nil || len(failures) != 1 || failures[0].Dimension != "disabled" {
		t.Fatalf("disabled match=%+v failures=%+v", got, failures)
	}
}
