package worker

import (
	"context"
	"fmt"
	"testing"
	"time"

	"mockbridge/internal/database"
	"mockbridge/internal/model"
	"mockbridge/internal/repository"
	"mockbridge/internal/service"
)

func TestBug08_SwitcherPropagatesActivationFailure(t *testing.T) {
	db, err := database.Open(fmt.Sprintf("file:bug08_%d?mode=memory&cache=shared", time.Now().UnixNano()))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err = database.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	mocks := repository.NewMockRepository(db)
	contract, err := mocks.CreateContract(ctx, model.ContractInput{Key: "bug08", Name: "Bug 08", Enabled: true, Priority: 100})
	if err != nil {
		t.Fatal(err)
	}
	scenarios, err := service.NewScenarioService(ctx, mocks)
	if err != nil {
		t.Fatal(err)
	}
	second, err := scenarios.Create(ctx, contract.Key, model.ScenarioInput{
		Name:       "cron",
		MatchRules: model.MatchRules{Method: "GET", Path: "/api/mock/bug08"},
		Response:   model.ResponseDef{Status: 200, Body: `{}`},
		Switch:     model.SwitchDef{Cron: "* * * * *", SwitchToScenario: "default"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = scenarios.Activate(ctx, contract.Key, second.ID, "manual"); err != nil {
		t.Fatal(err)
	}
	if _, err = db.ExecContext(ctx, "UPDATE scenarios SET switch_to_scenario_id=?, switch_cron=? WHERE id=?", 999999, "* * * * *", second.ID); err != nil {
		t.Fatal(err)
	}
	if err = NewSwitcherJob(scenarios).Run(ctx); err == nil {
		t.Fatal("switcher reported success after activation failure")
	}
}
