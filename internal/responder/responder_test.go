package responder

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"mockbridge/internal/model"
)

func TestRenderTemplateAndHeaders(t *testing.T) {
	ctx := Context{Method: "GET", Path: "/x", Query: map[string]string{"name": "Ada"}, Headers: map[string]string{"X-Token": "abc"}, Body: map[string]any{"id": float64(7)}, PathParams: map[string]string{"id": "42"}, State: map[string]any{"counter": 3}}
	def := model.ResponseDef{Status: 201, Headers: []model.KeyValueRule{{Name: "X-Echo", Value: "{{request.query.name}}"}}, Body: `{"id":"{{path.id}}","body":"{{request.body.id}}","count":{{state.counter}},"uuid":"{{uuid()}}","n":{{randInt(1,3)}},"s":"{{randStr(5)}}","literal":"\{{x}}"}`}
	out, err := Render(def, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != 201 || out.Headers.Get("X-Echo") != "Ada" {
		t.Fatalf("%+v", out)
	}
	text := string(out.Body)
	if !strings.Contains(text, `"id":"42"`) || !strings.Contains(text, `"literal":"{{x}}"`) {
		t.Fatal(text)
	}
	if !regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f-]{27}`).MatchString(text) {
		t.Fatal("uuid missing")
	}
}
func TestFallbacks(t *testing.T) {
	for _, def := range []model.ResponseDef{{Status: 99, Body: "x"}, {Status: 200, Body: "{{unknown}}"}, {Status: 200, Body: strings.Repeat("x", MaxBodySize+1)}} {
		out, err := Render(def, Context{})
		if err == nil || !out.Fallback || out.Status != 500 {
			t.Fatalf("expected fallback: %+v %v", out, err)
		}
		if !json.Valid(out.Body) {
			t.Fatalf("fallback body is not JSON: %q", out.Body)
		}
	}
	if _, err := RenderTemplate("{{randInt(4,1)}}", Context{}); err == nil {
		t.Fatal("bad range")
	}
	if _, err := RenderTemplate("{{oops", Context{}); err == nil {
		t.Fatal("unclosed")
	}
}
