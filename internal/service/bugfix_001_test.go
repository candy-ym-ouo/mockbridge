package service

import (
	"context"
	"testing"
	"time"

	"mockbridge/internal/model"
)

func TestBug01_CanceledMockRequestStopsDelay(t *testing.T) {
	mocks, _, closeDB := testRepositories(t)
	defer closeDB()
	ctx := context.Background()
	contract, err := mocks.CreateContract(ctx, model.ContractInput{Key: "cancel-delay", Name: "Cancel Delay", Enabled: true, Priority: 100})
	if err != nil {
		t.Fatal(err)
	}
	scenarios, err := NewScenarioService(ctx, mocks)
	if err != nil {
		t.Fatal(err)
	}
	_, err = scenarios.Update(ctx, contract.Key, contract.DefaultScenarioID, model.ScenarioInput{
		Name:       "default",
		MatchRules: model.MatchRules{Method: "GET", Path: "/api/mock/cancel-delay"},
		Response:   model.ResponseDef{Status: 200, Body: `{"ok":true}`},
		Delay:      model.DelayDef{FixedMS: 250},
	})
	if err != nil {
		t.Fatal(err)
	}
	requestCtx, cancel := context.WithCancel(ctx)
	cancel()
	started := time.Now()
	NewMockService(scenarios).Process(requestCtx, MockRequest{Method: "GET", Path: "/api/mock/cancel-delay"})
	if elapsed := time.Since(started); elapsed >= 150*time.Millisecond {
		t.Fatalf("canceled request remained in delay for %s", elapsed)
	}
}
