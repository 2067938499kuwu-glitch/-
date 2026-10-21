# 便携 FFmpeg

可把对应操作系统的 FFmpeg 可执行文件放在此目录：Windows 使用 `ffmpeg.exe`，macOS / Linux 使用 `ffmpeg`，后者需要可执行权限。

程序优先使用显式配置的 FFmpeg；默认自动模式先逐个检查 PATH 中的版本，再检查项目 `bin/`、程序目录和自动下载版本。每个候选必须具备实际播放能力，不兼容的自动候选会被跳过。也可通过 `-ffmpeg` 或自动生成的 `data/juku.json` 中的 `download.ffmpeg` 字段指定路径；显式指定的路径有误时会提示更正，不会擅自覆盖。

首次打开管理界面时，如果没有可用 FFmpeg，程序会在后台自动准备，不阻塞剧库加载；命令行下载时也会自动准备。下载、播放、合并及 HEIC 封面转换共用同一个安装任务，不会重复下载。状态和进度显示在下载模块及设置弹窗内，准备完成后自动隐藏下载模块中的提示，并重试当前失败的封面。

自动下载支持 macOS Intel / Apple Silicon、Windows x64、Linux x64。程序从 `eugeneware/ffmpeg-static` 的固定发布 `b6.1.1` 获取对应构建，而不是执行第三方安装脚本；使用现有代理设置，校验内置的安装包及可执行文件 SHA-256，检查 `libx264`、AAC 编码器并完成本地合成视频转码后，才将文件放入 `bin/<系统>-<架构>/verified-b6.1.1/`。原版许可和构建说明一并保存在该目录，无需管理员权限或额外解压工具，也不依赖 `ffprobe`。

自动准备约需下载 18–29 MiB，并占用约 44–80 MiB 可执行文件空间。网络或权限错误会明确显示，可修改路径或代理后点击“重新检查 FFmpeg”。程序应放在用户可写目录；不支持自动下载的平台仍可手动提供 FFmpeg。自动准备不会覆盖用户已有的 FFmpeg；旧版本不兼容时在独立子目录准备经过验证的版本。

发布来源：https://github.com/eugeneware/ffmpeg-static/releases/tag/b6.1.1 。FFmpeg 主页：https://ffmpeg.org/ 。不同平台构建的实际 FFmpeg 版本及许可证以配套说明为准；自动下载版本不会随远程 latest 标签静默变化。若另行捆绑分发 FFmpeg，需遵守 FFmpeg 及所用构建的许可证要求，保留许可和必要源码信息。

Windows 的 PATH 选择与 `preset` 报错修复、GitHub 实现对照及验证边界见 [FFmpeg 选择说明](../docs/ffmpeg-resolution.md)。设置页和 `data/logs/app.log` 会显示实际选中的路径；更改系统 PATH 后需新开终端并重启程序。
