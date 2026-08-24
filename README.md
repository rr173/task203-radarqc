# task203-radarqc 气象雷达双偏振质量标记服务

面向天气预报算法工程师的纯后端服务：为每个雷达门生成可解释的降水质量标签，
区分降水、杂波与异常传播，支持规则版本化与标签重算追溯。

## 业务闭环

1. 登记雷达站并发布校准参数（ZDR 偏差 / ZH 偏移 / RHOHV 偏差）；
2. 上传体扫（元数据 + 门数据：ZH / ZDR / RHOHV 三个双偏振变量）；
3. 创建并发布杂波分类规则版本；
4. 对体扫执行门级质量标记（分类 → 证据 → 摘要），输出可解释标签；
5. 发布新规则后对未封存体扫重算，旧标签经历史表可追溯；封存体扫结果冻结。

## 状态机

- 雷达站：`active → calibrating → disabled`，`disabled → calibrating → active`；
- 体扫：`receiving → pending → marked → sealed`（sealed 为终态，禁止追加/重算）；
- 雷达门：`raw → valid / clutter / anomaly / missing`；
- 规则版本：`draft → effective → retired`。

## 质量分类规则（默认参数）

| 规则 | 判定条件 |
| --- | --- |
| CLUTTER_HIGH_ZH | ZH > 65 dBZ |
| CLUTTER_LOW_RHOHV | RHOHV < 0.85 |
| CLUTTER_ZERO_ZDR_HIGH_ZH | \|ZDR\| < 0.3 dB 且 ZH > 35 dBZ |
| ANOMALOUS_ZDR | ZDR ∉ [-0.5, 4.5] dB |
| VALID | 通过全部阈值 |

## 标准命令

```bash
# 构建
CGO_ENABLED=0 go build ./...
# 静态检查
CGO_ENABLED=0 go vet ./...
# 单元测试
CGO_ENABLED=0 go test ./...
# 端到端冒烟（创建数据 → 标记 → 关闭重开验证重启恢复）
go run ./cmd/task203-radarqc --smoke-test
# 启动服务
go run ./cmd/task203-radarqc --addr :8080 --db ./radarqc.db
```

## API 入口（均以 /api 前缀）

- 站点：`POST /api/stations`、`GET /api/stations`、`GET /api/stations/{id}`、
  `POST /api/stations/{id}/status`、`POST /api/stations/{id}/calibrations`、
  `GET /api/stations/{id}/calibrations`、`POST /api/calibrations/{id}/publish`
- 体扫：`POST /api/scans`、`GET /api/scans`、`GET /api/scans/{id}`、
  `POST /api/scans/{id}/gates`、`GET /api/scans/{id}/gates`、
  `POST /api/scans/{id}/finalize`、`POST /api/scans/{id}/mark`、
  `POST /api/scans/{id}/recompute`、`POST /api/scans/{id}/seal`、
  `GET /api/scans/{id}/summary`、`GET /api/scans/{id}/evidence`、
  `GET /api/scans/{id}/marks`
- 规则：`POST /api/rules`、`GET /api/rules`、`GET /api/rules/{id}`、
  `POST /api/rules/{id}/publish`、`POST /api/rules/{id}/retire`、
  `GET /api/rules/compare?from=&to=`
- 统计与健康：`GET /api/stats`、`GET /api/health`

## 持久化

SQLite（modernc.org/sqlite，纯 Go 无 CGO）。核心表：stations、calibrations、
scans、gates、rule_versions、rule_hits、scan_summaries、quality_marks。
重复门按 `(scan_id, elevation_index, azimuth_bin, range_index)` 幂等；重启后
未完成体扫可继续追加门数据（断点续传）。
