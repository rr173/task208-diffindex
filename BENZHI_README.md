# task208-diffindex 评测说明（BENZHI）

蛋白质晶体衍射峰索引校验服务。评测构建与运行方式如下。

## 构建（双架构 Docker）

```bash
bash build_benzhi_docker.sh <镜像名> <平台>
# 例：bash build_benzhi_docker.sh my-project linux/amd64
#     bash build_benzhi_docker.sh my-project linux/arm64
```

镜像默认 `ENTRYPOINT ["/app/diffindex"]`，`CMD ["--smoke-test"]`。

## 冒烟契约

```bash
docker run --rm <镜像名>:latest --smoke-test
```

退出码 0 表示端到端冒烟通过：创建批次 → 配置几何 → 导入峰 → 索引 → 确认晶格 →
发布版本 → 封存守卫 → 重启恢复验证。

启动长驻服务：

```bash
docker run --rm -p 8080:8080 <镜像名>:latest --addr :8080
```

## API 前缀

全部 HTTP 路由以 `/api` 开头，核心入口见根目录 `README.md`。

## 环境

- Go 1.26.3（`GOTOOLCHAIN=local`，`CGO_ENABLED=0`）
- SQLite 3.46.1（`modernc.org/sqlite`，纯 Go 驱动）
- 构建镜像 `golang:1.26.3-bookworm`，`GOPROXY=https://goproxy.cn,direct`
