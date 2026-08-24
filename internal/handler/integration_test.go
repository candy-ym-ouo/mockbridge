package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"mockbridge/internal/database"
	"mockbridge/internal/handler"
	"mockbridge/internal/middleware"
	"mockbridge/internal/model"
	"mockbridge/internal/repository"
	"mockbridge/internal/service"
	"mockbridge/internal/worker"
)

type testApp struct {
	server  *httptest.Server
	records *service.RecordService
	dbClose func()
}

func setup(t *testing.T) *testApp {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err = database.Migrate(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	mr := repository.NewMockRepository(db)
	rr := repository.NewRecordRepository(db)
	sc, err := service.NewScenarioService(context.Background(), mr)
	if err != nil {
		t.Fatal(err)
	}
	records := service.NewRecordService(rr, 32, slog.Default())
	contracts := service.NewContractService(mr, sc)
	scheduler := worker.NewScheduler(slog.Default())
	mocks := service.NewMockService(sc)
	mux := http.NewServeMux()
	mux.Handle("/api/mock/", handler.NewMockHandler(mocks, records))
	mux.Handle("/admin/api/", handler.NewAdminHandler(contracts, sc, records, scheduler))
	srv := httptest.NewServer(middleware.Chain(mux, slog.Default()))
	return &testApp{srv, records, func() { records.Close(); db.Close() }}
}
func (a *testApp) close() { a.server.Close(); a.dbClose() }

type envelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func call(t *testing.T, a *testApp, method, path string, body any) (int, envelope) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, a.server.URL+path, reader)
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var env envelope
	if err = json.NewDecoder(res.Body).Decode(&env); err != nil {
		t.Fatal(err)
	}
	return res.StatusCode, env
}
func decodeData[T any](t *testing.T, e envelope) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(e.Data, &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestFullAdminAndMockFlow(t *testing.T) {
	a := setup(t)
	defer a.close()
	status, e := call(t, a, "POST", "/admin/api/contracts", map[string]any{"key": "user/profile", "name": "Profile", "description": "test", "priority": 200, "enabled": true})
	if status != 201 || e.Code != 0 {
		t.Fatalf("create %d %+v", status, e)
	}
	c := decodeData[model.Contract](t, e)
	if c.DefaultScenarioID == 0 {
		t.Fatal("default scenario missing")
	}
	_, e = call(t, a, "GET", "/admin/api/contracts?page=1&page_size=10&enabled=true", nil)
	if decodeData[model.Page[model.Contract]](t, e).Total != 1 {
		t.Fatal("list")
	}
	_, e = call(t, a, "PUT", "/admin/api/contracts/user/profile", map[string]any{"name": "Profile v2", "description": "updated", "priority": 150, "enabled": true, "version": c.Version})
	if e.Code != 0 {
		t.Fatal(e.Message)
	}
	updated := decodeData[model.Contract](t, e)
	_, e = call(t, a, "PUT", "/admin/api/contracts/user/profile", map[string]any{"name": "stale", "priority": 1, "enabled": true, "version": c.Version})
	if e.Code != 40900 {
		t.Fatalf("expected conflict %+v", e)
	}
	defaultInput := scenario("default", "/api/mock/users/{id}", `{"id":"{{path.id}}","q":"{{request.query.q}}","count":{{state.counter}}}`, 0, false)
	_, e = call(t, a, "PUT", fmt.Sprintf("/admin/api/contracts/user/profile/scenarios/%d", updated.DefaultScenarioID), defaultInput)
	if e.Code != 0 {
		t.Fatalf("update scenario %s", e.Message)
	}
	_, e = call(t, a, "POST", "/admin/api/contracts/user/profile/scenarios", scenario("error", "/api/mock/users/{id}", `{"ok":false}`, 1, true))
	if e.Code != 0 {
		t.Fatalf("create scenario %s", e.Message)
	}
	errorScenario := decodeData[model.Scenario](t, e)
	res, err := http.Get(a.server.URL + "/api/mock/users/42?q=yes")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 || !bytes.Contains(raw, []byte(`"id":"42"`)) {
		t.Fatalf("mock %d %s", res.StatusCode, raw)
	}
	_, e = call(t, a, "POST", fmt.Sprintf("/admin/api/contracts/user/profile/scenarios/%d/activate", errorScenario.ID), nil)
	if e.Code != 0 {
		t.Fatal(e.Message)
	}
	res, err = http.Get(a.server.URL + "/api/mock/users/42")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ = io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 503 || !bytes.Contains(raw, []byte("fault")) {
		t.Fatalf("fault %d %s", res.StatusCode, raw)
	}
	res, err = http.Get(a.server.URL + "/api/mock/nope")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != 404 {
		t.Fatalf("nonmatching path should return 404, got %d", res.StatusCode)
	}
	waitRecords(t, a, 3)
	_, e = call(t, a, "GET", "/admin/api/records?page_size=20&contract_key=user/profile", nil)
	page := decodeData[model.Page[model.CallRecord]](t, e)
	if page.Total < 2 {
		t.Fatalf("records=%d", page.Total)
	}
	_, e = call(t, a, "GET", fmt.Sprintf("/admin/api/records/%d", page.List[0].ID), nil)
	if e.Code != 0 {
		t.Fatal("record detail")
	}
	_, e = call(t, a, "GET", "/admin/api/stats/overview", nil)
	if decodeData[model.Overview](t, e).TodayCalls < 3 {
		t.Fatal("overview")
	}
	_, e = call(t, a, "GET", "/admin/api/stats/trend?contract_key=user/profile&hours=24", nil)
	if e.Code != 0 {
		t.Fatal("trend")
	}
	_, e = call(t, a, "GET", "/admin/api/health", nil)
	if e.Code != 0 {
		t.Fatal("health")
	}
	_, e = call(t, a, "DELETE", fmt.Sprintf("/admin/api/contracts/user/profile/scenarios/%d", errorScenario.ID), nil)
	if e.Code != 0 {
		t.Fatal(e.Message)
	}
	_, e = call(t, a, "DELETE", "/admin/api/contracts/user/profile", nil)
	if e.Code != 0 {
		t.Fatal(e.Message)
	}
	_, e = call(t, a, "GET", "/admin/api/contracts/user/profile", nil)
	if e.Code != 40400 {
		t.Fatalf("expected deleted %+v", e)
	}
}
func TestValidationAndNotFound(t *testing.T) {
	a := setup(t)
	defer a.close()
	for _, tc := range []struct {
		method, path string
		body         any
		code         int
	}{{"POST", "/admin/api/contracts", map[string]any{"key": "bad key", "name": "x"}, 40000}, {"GET", "/admin/api/contracts?page=bad", nil, 40000}, {"GET", "/admin/api/contracts?page_size=bad", nil, 40000}, {"GET", "/admin/api/records?status=bad", nil, 40000}, {"GET", "/admin/api/stats/trend?contract_key=x&hours=bad", nil, 40000}, {"GET", "/admin/api/records/not-a-number", nil, 40000}, {"GET", "/admin/api/stats/trend", nil, 40000}, {"GET", "/admin/api/missing", nil, 40400}, {"GET", "/admin/api/records/999999", nil, 40400}, {"POST", "/admin/api/records/clean?before=bad", nil, 40000}} {
		_, e := call(t, a, tc.method, tc.path, tc.body)
		if e.Code != tc.code {
			t.Errorf("%s code=%d msg=%s", tc.path, e.Code, e.Message)
		}
	}
	_, e := call(t, a, "POST", "/admin/api/contracts", map[string]any{"key": "a/b", "name": "x", "priority": 100, "enabled": true})
	c := decodeData[model.Contract](t, e)
	_, e = call(t, a, "DELETE", fmt.Sprintf("/admin/api/contracts/a/b/scenarios/%d", c.DefaultScenarioID), nil)
	if e.Code != 40900 {
		t.Fatalf("delete default %+v", e)
	}
	_, e = call(t, a, "GET", "/admin/api/records?matched=bad", nil)
	if e.Code != 40000 {
		t.Fatal("matched validation")
	}
}
func scenario(name, path, body string, rate float64, after bool) map[string]any {
	sw := map[string]any{"after_calls": 0, "cron": ""}
	if after {
		sw["after_calls"] = 1
		sw["switch_to_scenario"] = "default"
	}
	return map[string]any{"name": name, "match_rules": map[string]any{"method": "GET", "path": path, "headers": []any{}, "query": []any{}, "body": []any{}}, "response": map[string]any{"status": 200, "headers": []map[string]string{{"name": "Content-Type", "value": "application/json"}}, "body": body}, "delay": map[string]any{"fixed_ms": 0, "min_ms": 0, "max_ms": 0}, "fault": map[string]any{"enabled": rate > 0, "status": 503, "rate": rate, "body": `{"message":"fault"}`, "on_calls": []int{}}, "switch": sw}
}
func waitRecords(t *testing.T, a *testApp, n int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		_, e := call(t, a, "GET", "/admin/api/records?page_size=100", nil)
		if decodeData[model.Page[model.CallRecord]](t, e).Total >= int64(n) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("records did not reach %d", n)
}

func TestDefaultAndUnmatchedResponsesAreValidJSON(t *testing.T) {
	t.Run("unmatched", func(t *testing.T) {
		a := setup(t)
		defer a.close()
		res, err := http.Get(a.server.URL + "/api/mock/missing")
		if err != nil {
			t.Fatal(err)
		}
		raw, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if res.StatusCode != http.StatusNotFound || !json.Valid(raw) {
			t.Fatalf("unmatched status=%d body=%q", res.StatusCode, raw)
		}
	})
	t.Run("default scenario and raw query", func(t *testing.T) {
		a := setup(t)
		defer a.close()
		_, e := call(t, a, http.MethodPost, "/admin/api/contracts", map[string]any{"key": "default/json", "name": "Default", "priority": 100, "enabled": true})
		if e.Code != 0 {
			t.Fatal(e.Message)
		}
		rawQuery := "b=two+words&a=1&a=x%26y"
		res, err := http.Get(a.server.URL + "/api/mock/default?" + rawQuery)
		if err != nil {
			t.Fatal(err)
		}
		raw, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if res.StatusCode != http.StatusOK || !json.Valid(raw) {
			t.Fatalf("default status=%d body=%q", res.StatusCode, raw)
		}
		waitRecords(t, a, 1)
		_, e = call(t, a, http.MethodGet, "/admin/api/records?contract_key=default%2Fjson&page_size=10", nil)
		page := decodeData[model.Page[model.CallRecord]](t, e)
		if len(page.List) != 1 || page.List[0].QueryString != rawQuery {
			t.Fatalf("records=%+v", page.List)
		}
	})
}

func TestAdminRejectsTrailingJSON(t *testing.T) {
	a := setup(t)
	defer a.close()
	body := `{"key":"one","name":"One","enabled":true}{"key":"two","name":"Two","enabled":true}`
	req, err := http.NewRequest(http.MethodPost, a.server.URL+"/admin/api/contracts", bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var e envelope
	if err = json.NewDecoder(res.Body).Decode(&e); err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusBadRequest || e.Code != 40000 {
		t.Fatalf("status=%d envelope=%+v", res.StatusCode, e)
	}
}

func TestDisabledContractRequestsAreNotRecorded(t *testing.T) {
	a := setup(t)
	defer a.close()
	_, e := call(t, a, http.MethodPost, "/admin/api/contracts", map[string]any{"key": "disabled", "name": "Disabled", "priority": 100, "enabled": false})
	if e.Code != 0 {
		t.Fatal(e.Message)
	}
	res, err := http.Get(a.server.URL + "/api/mock/disabled")
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d", res.StatusCode)
	}
	time.Sleep(300 * time.Millisecond)
	_, e = call(t, a, http.MethodGet, "/admin/api/records?page_size=10", nil)
	if page := decodeData[model.Page[model.CallRecord]](t, e); page.Total != 0 {
		t.Fatalf("disabled request was recorded: %+v", page.List)
	}
}
