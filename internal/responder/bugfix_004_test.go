package responder

import (
	"testing"

	"mockbridge/internal/model"
)

func TestBug04_RenderAlwaysReturnsWritableHeaders(t *testing.T) {
	out, err := Render(model.ResponseDef{Status: 200, Body: "plain response"}, Context{})
	if err != nil {
		t.Fatal(err)
	}
	if out.Headers == nil {
		t.Fatal("render returned nil headers for a response without configured headers")
	}
}
