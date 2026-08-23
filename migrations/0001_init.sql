CREATE TABLE IF NOT EXISTS contracts (
 id INTEGER PRIMARY KEY AUTOINCREMENT, key TEXT NOT NULL UNIQUE, name TEXT NOT NULL,
 description TEXT NOT NULL DEFAULT '', enabled INTEGER NOT NULL DEFAULT 1,
 priority INTEGER NOT NULL DEFAULT 100, version INTEGER NOT NULL DEFAULT 1,
 created_at TEXT NOT NULL, updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_contracts_priority ON contracts(priority DESC, id ASC);
CREATE TABLE IF NOT EXISTS scenarios (
 id INTEGER PRIMARY KEY AUTOINCREMENT, contract_id INTEGER NOT NULL REFERENCES contracts(id) ON DELETE CASCADE,
 name TEXT NOT NULL, is_default INTEGER NOT NULL DEFAULT 0, is_active INTEGER NOT NULL DEFAULT 0,
 match_rules_json TEXT NOT NULL DEFAULT '{}', response_status INTEGER NOT NULL DEFAULT 200,
 response_headers_json TEXT NOT NULL DEFAULT '[]', response_body TEXT NOT NULL DEFAULT '',
 delay_fixed_ms INTEGER NOT NULL DEFAULT 0, delay_min_ms INTEGER NOT NULL DEFAULT 0, delay_max_ms INTEGER NOT NULL DEFAULT 0,
 fault_enabled INTEGER NOT NULL DEFAULT 0, fault_status INTEGER NOT NULL DEFAULT 500,
 fault_body TEXT NOT NULL DEFAULT '{"code":50000,"message":"injected fault"}', fault_rate REAL NOT NULL DEFAULT 0,
 fault_on_calls TEXT NOT NULL DEFAULT '[]', switch_after_calls INTEGER NOT NULL DEFAULT 0,
 switch_to_scenario_id INTEGER, switch_cron TEXT NOT NULL DEFAULT '', hit_count INTEGER NOT NULL DEFAULT 0,
 created_at TEXT NOT NULL, updated_at TEXT NOT NULL, UNIQUE(contract_id,name)
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_scenarios_contract_active ON scenarios(contract_id) WHERE is_active=1;
CREATE INDEX IF NOT EXISTS idx_scenarios_contract ON scenarios(contract_id);
CREATE TABLE IF NOT EXISTS call_records (
 id INTEGER PRIMARY KEY AUTOINCREMENT, request_id TEXT NOT NULL, method TEXT NOT NULL, path TEXT NOT NULL,
 query_string TEXT NOT NULL DEFAULT '', request_headers_json TEXT NOT NULL DEFAULT '[]', request_body TEXT NOT NULL DEFAULT '',
 client_ip TEXT NOT NULL DEFAULT '', contract_key TEXT NOT NULL DEFAULT '', scenario_name TEXT NOT NULL DEFAULT '',
 matched INTEGER NOT NULL DEFAULT 0, match_detail TEXT NOT NULL DEFAULT '', response_status INTEGER NOT NULL DEFAULT 0,
 response_headers_json TEXT NOT NULL DEFAULT '[]', response_body TEXT NOT NULL DEFAULT '', injected_delay_ms INTEGER NOT NULL DEFAULT 0,
 injected_fault INTEGER NOT NULL DEFAULT 0, total_ms INTEGER NOT NULL DEFAULT 0, created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_records_created ON call_records(created_at);
CREATE INDEX IF NOT EXISTS idx_records_contract ON call_records(contract_key);
CREATE INDEX IF NOT EXISTS idx_records_status ON call_records(response_status);
CREATE TABLE IF NOT EXISTS switch_logs (
 id INTEGER PRIMARY KEY AUTOINCREMENT, contract_key TEXT NOT NULL, from_scenario TEXT NOT NULL,
 to_scenario TEXT NOT NULL, trigger_type TEXT NOT NULL, created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_switch_logs_contract ON switch_logs(contract_key);
CREATE TABLE IF NOT EXISTS call_stats (
 id INTEGER PRIMARY KEY AUTOINCREMENT, contract_key TEXT NOT NULL, bucket_hour TEXT NOT NULL,
 total_calls INTEGER NOT NULL DEFAULT 0, matched_calls INTEGER NOT NULL DEFAULT 0,
 fault_calls INTEGER NOT NULL DEFAULT 0, error_calls INTEGER NOT NULL DEFAULT 0, sum_ms INTEGER NOT NULL DEFAULT 0,
 UNIQUE(contract_key,bucket_hour)
);
