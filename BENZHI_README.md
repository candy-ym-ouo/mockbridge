# MockBridge 评测与打包说明

MockBridge 是一个使用 Go 1.22、SQLite 和原生 HTML/CSS/JavaScript 实现的接口契约模拟服务。管理前端是静态资源，不需要 Node.js 或额外的前端构建链。

## 本地验证

```bash
go mod download
go build ./...
go test ./...
make verify
```

## 本地运行

```bash
go run ./cmd/server
```

默认地址为 `http://localhost:8080`：

- 管理前端：`http://localhost:8080/admin/`
- 健康检查：`http://localhost:8080/admin/api/health`
- Mock API：`http://localhost:8080/api/mock/**`

可通过环境变量覆盖监听地址和数据库路径：

```bash
MOCKBRIDGE_ADDRESS=:18080 MOCKBRIDGE_DB_PATH=/tmp/mockbridge.db go run ./cmd/server
```

## Docker 评测镜像

构建并进入保留完整 Go 工具链的评测镜像：

```bash
./build_benzhi_docker.sh mockbridge linux/amd64
docker run --rm -it mockbridge:latest
```

Apple Silicon 架构验证：

```bash
./build_benzhi_docker.sh mockbridge linux/arm64
```

进入容器后可离线执行：

```bash
go build ./...
go test ./...
make verify
```
