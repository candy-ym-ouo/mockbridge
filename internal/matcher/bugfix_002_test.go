package matcher

import (
	"testing"

	"mockbridge/internal/model"
)

func TestBug02_MatchDoesNotReorderPublishedSnapshot(t *testing.T) {
	contracts := []model.ContractSnapshot{
		{Key: "low", Priority: 10, Enabled: true, Scenario: model.ScenarioSnapshot{Rules: model.MatchRules{Method: "GET", Path: "/api/mock/**"}}},
		{Key: "high", Priority: 20, Enabled: true, Scenario: model.ScenarioSnapshot{Rules: model.MatchRules{Method: "GET", Path: "/api/mock/**"}}},
	}
	if result, _ := Match(Request{Method: "GET", Path: "/api/mock/item"}, contracts); result == nil || result.Contract.Key != "high" {
		t.Fatalf("match result=%+v", result)
	}
	if contracts[0].Key != "low" || contracts[1].Key != "high" {
		t.Fatalf("published snapshot was reordered: %+v", contracts)
	}
}
