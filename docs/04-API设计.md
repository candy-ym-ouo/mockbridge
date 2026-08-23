# 04 API 设计

## 1. 通用约定

- **Base URL**：`http://{host}:{port}`
- 管理 API 前缀：`/admin/api`（JSON）；模拟 API 前缀：`/api/mock/`
- 统一信封（BR-8）：
```json
{ "code": 0, "message": "ok", "data": { ... } }
```
| code | 含义 |
| --- | --- |
| 0 | 成功 |
| 40000 | 参数错误 |
| 40400 | 资源不存在 / mock 未命中 |
| 40900 | 冲突（key 重复 / 乐观锁版本不一致） |
| 50000 | 服务内部错误 |
| 50300 | 服务暂不可用 |

- 时间格式：ISO8601 UTC（`2025-08-23T14:30:00Z`）。
- 分页：`page`（从 1 起）、`page_size`（默认 20，最大 100），返回 `{list, total, page, page_size}`。

---

## 2. 模拟 API（Simulation API）

### 2.1 模拟入口（任意方法）
```
任意方法  /api/mock/**（含任意子路径与查询参数）
```
- 说明：MockHandler 接收所有到达该前缀的请求，走"契约快照 → 匹配 → 渲染 → 故障/延迟 → 写出"流程（见 05 文档）；
- 未命中任何契约：`404 {"code":40400,"message":"mock not found"}`；
- 命中：状态码/头/体完全由命中场景决定（可能是 200、500、断连等）。

### 2.2 健康检查（供负载均衡/探活）
```
GET /api/mock/_ping → 200 {"code":0,"message":"pong"}
```

---

## 3. 管理 API（Admin API）

### 3.1 契约管理

#### 3.1.1 创建契约
```
POST /admin/api/contracts
```
请求体：
```json
{
  "key": "user/get-profile",
  "name": "获取用户资料",
  "description": "联调用契约",
  "priority": 100,
  "enabled": true
}
```
响应 `data`：
```json
{
  "id": 1, "key": "user/get-profile", "name": "获取用户资料",
  "description": "", "priority": 100, "enabled": true,
  "version": 1, "default_scenario_id": 1,
  "created_at": "2025-08-23T14:30:00Z", "updated_at": "2025-08-23T14:30:00Z"
}
```
- 自动创建 `default` 默认场景（BR-2）；`key` 非法/重复返回 40000/40900。

#### 3.1.2 契约列表
```
GET /admin/api/contracts?page=1&page_size=20&keyword=user&enabled=true
```
`keyword` 模糊匹配 key/name。响应 `data`：
```json
{ "list": [ {契约对象, 含 active_scenario_name} ], "total": 12, "page": 1, "page_size": 20 }
```

#### 3.1.3 契约详情
```
GET /admin/api/contracts/{key}
```
返回契约 + 其全部场景（含 `is_active` 标记）。

#### 3.1.4 更新契约
```
PUT /admin/api/contracts/{key}
```
请求体：`{"name":"...", "description":"...", "priority":200, "enabled":true, "version":1}`
- `version` 必须与库中一致，否则 40900（乐观锁，FR-1.4）。

#### 3.1.5 删除契约
```
DELETE /admin/api/contracts/{key}
```
- 级联删除场景（BR-3）；调用记录与切换记录保留。

### 3.2 场景管理

#### 3.2.1 新增场景
```
POST /admin/api/contracts/{key}/scenarios
```
请求体：
```json
{
  "name": "timeout",
  "is_default": false,
  "match_rules": {
    "method": "GET",
    "path": "/api/v1/user/{id}",
    "headers": [{"name": "X-Token", "value": "test-*"}],
    "query": [{"name": "type", "value": "vip"}],
    "body": [{"jsonpath": "$.userId", "value": "1001"}]
  },
  "response": {
    "status": 200,
    "headers": [{"name": "Content-Type", "value": "application/json"}],
    "body": "{\"id\":{{path.id}},\"name\":\"{{request.body.userName}}\",\"token\":\"{{uuid()}}\"}"
  },
  "delay": { "fixed_ms": 0, "min_ms": 100, "max_ms": 300 },
  "fault": {
    "enabled": true, "status": 503, "rate": 0.5,
    "body": "{\"code\":50300,\"message\":\"service unavailable\"}",
    "on_calls": [1, 5, 9]
  },
  "switch": { "after_calls": 10, "switch_to_scenario": "error", "cron": "" }
}
```
响应 `data`：场景对象（含 `id`、`is_active`）。

#### 3.2.2 更新场景
```
PUT /admin/api/contracts/{key}/scenarios/{scenarioId}
```
- 字段同创建；`is_default`、`is_active` 不可通过本接口修改（由专用接口控制）。

#### 3.2.3 删除场景
```
DELETE /admin/api/contracts/{key}/scenarios/{scenarioId}
```
- 禁止删除默认场景（40900）；删除当前生效场景时，自动把默认场景置为生效。

#### 3.2.4 手动切换场景
```
POST /admin/api/contracts/{key}/scenarios/{scenarioId}/activate
```
- 将该场景置为生效（事务内先清后设，BR-6），写一条 `manual` 切换记录；
- 响应 `data`：`{"contract_key":"...","from":"success","to":"timeout","trigger":"manual"}`。

### 3.3 调用记录

#### 3.3.1 分页查询
```
GET /admin/api/records?page=1&page_size=20&contract_key=user/get-profile&status=200&matched=true&start=2025-08-20T00:00:00Z&end=2025-08-23T23:59:59Z&keyword=1001
```
响应 `data`：
```json
{
  "list": [{
    "id": 100, "request_id": "req_8f3a", "method": "POST", "path": "/api/mock/user/get-profile",
    "query_string": "type=vip", "client_ip": "127.0.0.1",
    "contract_key": "user/get-profile", "scenario_name": "timeout", "matched": true,
    "match_detail": "priority=100 rules matched: method,path,body",
    "response_status": 503, "response_body": "{\"code\":50300,...}",
    "injected_delay_ms": 250, "injected_fault": true, "total_ms": 262,
    "created_at": "2025-08-23T14:31:02Z"
  }],
  "total": 356, "page": 1, "page_size": 20
}
```

#### 3.3.2 记录详情
```
GET /admin/api/records/{id}
```
返回单条记录完整字段（含请求/响应头与体）。

#### 3.3.3 清理记录（手动触发）
```
POST /admin/api/records/clean?before=2025-08-16T00:00:00Z
```
- 删除指定时间之前的记录，返回删除条数。后台任务自动执行同一逻辑。

### 3.4 统计与健康

#### 3.4.1 概览统计
```
GET /admin/api/stats/overview
```
响应 `data`：
```json
{
  "contract_count": 12,
  "enabled_contract_count": 10,
  "today_calls": 3560,
  "today_matched": 3421,
  "today_faults": 210,
  "today_error_rate": 0.059,
  "avg_ms": 38,
  "top_contracts": [ {"contract_key":"user/get-profile","calls":1200,"faults":60} ]
}
```
- 实时部分（今日）由统计表聚合 + 少量实时查询混合得出。

#### 3.4.2 契约调用趋势
```
GET /admin/api/stats/trend?contract_key=user/get-profile&hours=24
```
返回按小时聚合的 `[{bucket_hour, total_calls, matched_calls, fault_calls, error_calls, avg_ms}]`。

#### 3.4.3 服务健康
```
GET /admin/api/health
```
响应 `data`：
```json
{
  "status": "ok",
  "uptime_seconds": 3600,
  "db": "ok",
  "tasks": [
    {"name": "stats-aggregate", "last_run_at": "...", "last_error": ""},
    {"name": "record-cleaner", "last_run_at": "...", "last_error": ""},
    {"name": "scenario-switcher", "last_run_at": "...", "last_error": ""}
  ],
  "record_queue_depth": 12
}
```

### 3.5 错误示例
```json
// 404 未命中
{ "code": 40400, "message": "mock not found", "data": null }
// 409 版本冲突
{ "code": 40900, "message": "contract version conflict: expect 1, got 2", "data": null }
// 400 参数错误
{ "code": 40000, "message": "key format invalid", "data": null }
```

---

## 4. 前端静态资源

```
GET /admin/           → web/index.html（管理界面）
GET /admin/static/*   → web/ 下静态文件（app.js、style.css）
```

## 5. API 与需求映射

| API | 需求 |
| --- | --- |
| 模拟入口 | FR-2/FR-3/FR-5/FR-6 |
| 契约 CRUD | FR-1 |
| 场景 CRUD + activate | FR-4.1/FR-4.2 |
| 记录查询/清理 | FR-5.3/FR-5.4 |
| 统计/健康 | FR-7.5、FR-8 |
