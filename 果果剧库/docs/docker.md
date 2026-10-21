# 果果剧库 Docker 部署

需要 Docker Engine/Desktop 与 Compose v2。镜像由用户在自己的机器上构建，本项目不附带预制镜像。

## 构建与启动

在项目根目录运行：

```bash
./scripts/docker-up.sh
```

脚本执行 `docker compose up -d --build`。Windows PowerShell 可直接运行该命令。默认打开 `http://localhost:8999`，局域网使用 Docker 宿主机地址。单独构建镜像：

```bash
./scripts/docker-build.sh
```

可用 `JUKU_IMAGE` 更改默认镜像名 `guoguo-juku:local`；构建脚本支持附加 `docker build` 参数，例如 `--platform linux/arm64`。默认构建当前 Docker 平台。Dockerfile 使用 Go 多阶段构建，运行层含 FFmpeg（H.264/AAC）、证书、时区和进程回收器，以 UID/GID 1000 运行。健康检查只读取本地页面，不访问站源。

## 数据与端口

Compose 的独立命名卷 `data` 保存配置、剧库、观看记录、收藏、任务和日志，`downloads` 保存视频；卷名带项目名 `guoguo-juku` 前缀。两版本不会共用数据。停止时运行 `docker compose down`，卷会保留；不要加 `-v`，除非确实要删除运行数据。

宿主端口可调整，例如：

```bash
JUKU_PORT=9099 ./scripts/docker-up.sh
```

`JUKU_BIND_ADDRESS=127.0.0.1` 可仅开放宿主本机。管理界面适合可信局域网，默认没有网页登录认证。

如需把文件直接写入 NAS/宿主目录，在 `compose.yaml` 将两条卷映射改为绝对路径，例如 `/srv/guoguo-juku/data:/data` 和 `/srv/guoguo-juku/downloads:/downloads`，提前创建目录并确保 UID/GID 1000 可写；也可在服务中设置 `user: "实际UID:实际GID"`。程序中的下载目录应保持 `/downloads` 或其子目录，不能填写宿主机路径。启动参数固定输出到 `/downloads`，重新启动后仍使用该挂载点。

Docker 构建上下文只允许 Go 入口、模块与 `internal/` 源码；现有数据、视频、配置、便携 FFmpeg、dist 程序和测试文件不会打入镜像。

## 代理与日志

容器的 `127.0.0.1` 指向容器自身。如需使用宿主代理，可运行：

```bash
JUKU_PROXY_URL=http://host.docker.internal:7897 ./scripts/docker-up.sh
```

宿主代理须允许容器网段访问；也可在页面“代理 / 设置”输入可访问的局域网代理地址。Docker 拉基础镜像的代理需在 Docker Engine/Desktop 单独配置，程序代理不会影响镜像拉取。

```bash
docker compose logs --tail=100 -f
docker compose exec app ls /data/logs
```

持久化诊断日志在 `/data/logs/`。标准输出日志由 Compose 限制为每份 10 MiB、最多三份。

## 更新与验证边界

更新源码后再次执行 `./scripts/docker-up.sh`，保留原卷即可。这里仅交付构建材料：本次已验证脚本参数、Compose 结构、Linux amd64/arm64 Go 编译；本机没有 Docker 引擎，因此未构建镜像或声称容器实跑通过。
