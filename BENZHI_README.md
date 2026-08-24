# BENZHI 评测说明：task203-radarqc

气象雷达双偏振质量标记服务（纯后端 Go 服务，无前端）。

## 技术栈

- Go 1.26.3（`GOTOOLCHAIN=local`，`CGO_ENABLED=0`）
- SQLite 3.46.1（modernc.org/sqlite v1.52.0 纯 Go 驱动，无 CGO）
- 无第三方 HTTP 框架，标准库 `net/http`（Go 1.22+ 路由模式）

## 标准构建与验证命令

```bash
CGO_ENABLED=0 go build ./...
CGO_ENABLED=0 go vet ./...
CGO_ENABLED=0 go test ./...
go run ./cmd/task203-radarqc --smoke-test
```

## 入口契约

`cmd/task203-radarqc/main.go` 支持：

- `--addr :8080`：HTTP 监听地址；
- `--db ./radarqc.db`：SQLite 数据库路径；
- `--smoke-test`：端到端冒烟。真实创建站点→校准→规则→体扫（36 门）→
  幂等验证→非法数据拒绝→质量标记→摘要→关闭重开验证持久化→严格规则
  重算（标签翻转）→封存拒绝重算，全程以 0 退出码结束；任一断言失败以非 0 退出。

## Docker 双架构

```bash
bash build_benzhi_docker.sh <镜像名> <平台>   # 例如 my-project linux/arm64
docker run --rm <镜像名>:latest --smoke-test  # 容器内冒烟验证
```

Dockerfile / benzhi.Dockerfile 单阶段构建，`ENTRYPOINT ["/app/radarqc"]`、
`CMD ["--smoke-test"]`。容器内显式设置 `GOPROXY=https://goproxy.cn,direct` 与
`GOSUMDB=sum.golang.google.cn`。

## API 概览

| 能力 | 入口 |
| --- | --- |
| 站点管理 | `POST/GET /api/stations`、`GET /api/stations/{id}`、`POST /api/stations/{id}/status` |
| 校准版本 | `POST /api/stations/{id}/calibrations`、`GET /api/stations/{id}/calibrations`、`POST /api/calibrations/{id}/publish` |
| 体扫接收 | `POST /api/scans`、`POST /api/scans/{id}/gates`、`POST /api/scans/{id}/finalize` |
| 质量标记 | `POST /api/scans/{id}/mark`、`POST /api/scans/{id}/recompute`、`POST /api/scans/{id}/seal` |
| 查询 | `GET /api/scans/{id}/gates`、`GET /api/scans/{id}/summary`、`GET /api/scans/{id}/evidence`、`GET /api/scans/{id}/marks` |
| 规则版本 | `POST /api/rules`、`GET /api/rules/{id}`、`POST /api/rules/{id}/publish`、`POST /api/rules/{id}/retire`、`GET /api/rules/compare` |
| 统计与健康 | `GET /api/stats`、`GET /api/health` |

## 业务核心

- 双偏振变量：ZH（反射率，dBZ）、ZDR（差分反射率，dB）、RHOHV（相关系数，无量纲）。
- 杂波分类：反射率超限 / 相关系数过低 / 近零 ZDR 高反射 / ZDR 越界 → 杂波或异常；
  全部通过 → 有效降水。
- 校准修正：ZDR 扣偏差、ZH 加偏移、RHOHV 加偏差并夹取 [0,1]。
- 规则版本化：草稿 → 生效 → 废止；发布新规则自动废止旧版；未封存体扫可重算，
  标签翻转可通过历史表与证据追溯。
