package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(p, []byte("server:\n  address: ':9999'\ndatabase:\n  path: test.db\ntasks:\n  cleaner_interval: 2s\n  stats_interval: 3s\n  switcher_interval: 4s\n  record_retention_days: 2\nrecord_queue_size: 12\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MOCKBRIDGE_ADDRESS", ":7777")
	c, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if c.Server.Address != ":7777" || c.Tasks.CleanerInterval != 2*time.Second || c.RecordQueueSize != 12 {
		t.Fatalf("%+v", c)
	}
	bad := filepath.Join(t.TempDir(), "bad.yaml")
	_ = os.WriteFile(bad, []byte("tasks:\n  cleaner_interval: nope\n"), 0600)
	if _, err = Load(bad); err == nil {
		t.Fatal("expected invalid duration")
	}
}
