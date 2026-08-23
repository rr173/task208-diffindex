# task208-diffindex — 蛋白质晶体衍射峰索引校验服务

判断一批晶体衍射峰是否被正确索引：从探测器峰位反推晶格候选、分配 Miller 索引、
计算残差与系统性缺峰证据，并发布一份不可变的索引版本快照。

## 业务闭环

1. 创建衍射批次，配置实验几何（波长、探测器距离、束心）。
2. 导入衍射峰（探测器坐标 + 强度，按采集序号幂等去重）。
3. 触发索引：搜索晶格候选并评分。
4. 复核：锁定参考峰、排除遮挡/杂散峰、查看冲突峰。
5. 确认晶格候选，系统给每个峰分配 Miller 索引并标记冲突峰。
6. 发布索引版本（绑定确认晶格与排除清单），批次封存。

## 标准命令

```bash
CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go vet   ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go test  ./...
go run ./cmd/diffindex --smoke-test
go run ./cmd/diffindex --addr :8080 --db ./diffindex.db
```

## API 入口（前缀 /api）

| 能力 | 方法 + 路径 |
| --- | --- |
| 创建批次 | `POST /api/batches` |
| 列出批次 | `GET /api/batches` |
| 批次详情 | `GET /api/batches/{id}` |
| 配置几何 | `PUT /api/batches/{id}/geometry` |
| 查询几何 | `GET /api/batches/{id}/geometry` |
| 导入峰 | `POST /api/batches/{id}/peaks` |
| 列出峰 | `GET /api/batches/{id}/peaks` |
| 触发索引 | `POST /api/batches/{id}/index` |
| 晶格候选 | `GET /api/batches/{id}/lattices` |
| 候选详情 | `GET /api/lattices/{id}` |
| 确认晶格 | `POST /api/batches/{id}/confirm` |
| 残差报告 | `GET /api/batches/{id}/residuals` |
| 缺峰诊断 | `GET /api/batches/{id}/missing` |
| 锁定参考峰 | `POST /api/peaks/{id}/lock` |
| 取消锁定 | `POST /api/peaks/{id}/unlock` |
| 排除峰 | `POST /api/peaks/{id}/exclude` |
| 恢复峰 | `POST /api/peaks/{id}/restore` |
| 冲突峰 | `GET /api/batches/{id}/conflicts` |
| 排除峰列表 | `GET /api/batches/{id}/excluded` |
| 发布版本 | `POST /api/batches/{id}/versions` |
| 版本列表 | `GET /api/batches/{id}/versions` |
| 版本详情 | `GET /api/versions/{id}` |
| 统计 | `GET /api/stats` |
| 健康检查 | `GET /api/health` |

## 持久化与重启恢复

数据全部持久化到 SQLite（纯 Go 驱动，CGO 无关）。批次、几何、峰、晶格候选、索引版本
均落表；重启后重新打开同一数据库路径即可恢复全部状态。峰以（批次 + 采集序号）为幂等键，
发布版本为不可变快照（绑定晶格 + 排除清单），封存批次拒绝后续修改。

## 模块结构

```
cmd/diffindex       入口（--addr / --db / --smoke-test）
internal/model      实体、状态机与错误
internal/store      SQLite 持久化
internal/geometry   探测器坐标 → 散射角/倒易矢量
internal/lattice    晶胞 ↔ 倒易格子、Miller 索引运算
internal/indexing   晶格搜索与 Miller 分配
internal/scoring    残差统计与缺峰诊断
internal/review     参考峰锁定、峰排除、冲突
internal/versioning 索引版本生命周期
internal/service    编排层
internal/httpapi    HTTP 层
```
