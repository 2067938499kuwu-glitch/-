# Emby 接入

果果剧库可导出 Emby 支持的 STRM 分集，并提供持续有效的 HLS 播放 API。接入对象是现有 Emby 服务器的“电视剧”媒体库；本项目不实现完整的 Emby 服务器协议。

## 界面操作

1. 使用 Emby 服务器能访问的地址打开果果剧库，例如 `http://192.168.1.10:8999`。两者分别运行在不同电脑或容器时，避免通过 `localhost` 导出。
2. 点击剧名打开详情，再点击“导出 Emby”。浏览器下载 ZIP，包含剧名目录、`tvshow.nfo`、`Season 01/S01E001.strm` 和对应的分集 NFO。
3. 解压到 Emby 的电视剧媒体库目录，执行扫描。以后有新集，重新导出到相同目录即可补充。

播放期间需要果果剧库和 FFmpeg 保持运行。分集先匹配本地已下载文件，否则按原分集 ID 重新解析在线地址，经 HLS 输出；不会把会过期的源站地址直接写进 STRM。导出包只含文字资料与分集链接。

## 导出 API

```http
POST /api/emby/export
Content-Type: application/json

{"dramaId":"hongguo:实际剧ID","baseUrl":"http://192.168.1.10:8999"}
```

返回 `application/zip`。`dramaId` 必须存在于当前剧库；`baseUrl` 是 Emby 能访问的果果剧库地址。单次最多 2000 集，必须有稳定分集 ID。此管理接口与现有 Web UI 一样面向可信局域网。

STRM 内的 `GET /api/emby/stream.m3u8` 带有绑定剧 ID 和分集 ID 的签名。客户端按清单请求 `GET /api/emby/segment.ts`，支持 HEAD 和字节 Range。HLS 分片沿用有界转码缓存，与网页播放器合计最多四个活跃会话，空闲两分钟自动回收。过期后重新打开 STRM 即可建立新会话。

签名密钥首次导出时保存为 `data/emby-key`，重启后链接继续有效；请将它与数据目录一起保留，两个项目各自生成。分享 STRM 等同于分享相应分集的播放权限。更改访问地址或删除密钥后，需要重新导出；删除密钥后重启程序使旧签名失效。

Emby 的观看进度和用户由 Emby 管理，本轮不回写果果剧库观看记录。在线资源的可用性仍取决于源站。Docker 环境中的 `baseUrl` 应填写 Emby 可访问的宿主映射地址或容器地址。

## 验证

两版已通过 ZIP 路径与 XML 转义、签名篡改拒绝、重启后链接有效、真实合成媒体的 HLS/Range/HEAD/会话清理，以及 FFprobe 读取完整四秒 HLS 时长的检查。浏览器验证导出失败重试及 ZIP 下载，无源站图片请求。

本次环境没有可操作的 Emby 服务器，尚未进行 Emby 实际扫库与客户端端到端验证。格式依据：[Emby 官方 STRM 说明](https://emby.media/support/articles/Strm-Files.html)。
