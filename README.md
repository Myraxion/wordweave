# Wordweave · 生词本

Wordweave 是一个在本机运行的英文生词本。收录单词和短语，记录释义，把相关词条组织成关系组，在浏览器中对照和复习。数据保存在自己的设备上，安装后日常使用无需联网。

## 可以做什么

- **收录与搜索**：保存英文单词或短语及多行释义，通过拼写、释义或关系说明搜索。
- **关联与对照**：将词条按近义、反义、词族等类型组成关系组，添加说明，并沿成员进入其他词条。编辑词条后，组中的内容随之更新。
- **浏览与复习**：按时间或字母排序，切换网格和瀑布流布局；遮挡拼写或释义，点击后逐项揭示。
- **个性化显示**：切换浅色、深色或跟随系统外观；浏览布局和显示模式会保存在当前浏览器中。

## 开始使用

### Windows

1. 从 [GitHub Releases](https://github.com/Myraxion/wordweave/releases) 下载 Windows x64 压缩包，解压到可写目录。
2. 运行 `wordweave.exe`，在浏览器打开程序显示的地址，默认为 <http://127.0.0.1:8080>。
3. 保持程序窗口打开；关闭窗口或按 Ctrl+C 即可停止。

程序不需要单独安装 Go、Node 或数据库。首次启动会在程序旁创建 `data` 目录，用于保存词条和关系组。

### Docker Compose

镜像支持 Linux amd64 和 arm64。在包含 [compose.yaml](compose.yaml) 的目录中运行：

```sh
docker compose up -d
```

打开 <http://127.0.0.1:8080>。配置使用 `wordweave-data` 数据卷，并只将端口绑定到宿主机回环地址。首次发布后，GHCR 包需要设为公开才能匿名拉取。

```sh
docker compose logs -f      # 查看日志
docker compose pull         # 拉取更新
docker compose up -d        # 启动或应用更新
docker compose down         # 停止并移除容器，保留数据卷
```

不要使用 `docker compose down -v`，除非确定要删除数据卷及其中的生词本。如需固定版本，可在 `compose.yaml` 中将 `latest` 改成具体版本，例如 `v1.0.0`。

也可以直接运行镜像：

```sh
docker volume create wordweave-data
docker run -d --name wordweave --restart unless-stopped \
  -p 127.0.0.1:8080:8080 -v wordweave-data:/data \
  ghcr.io/myraxion/wordweave:latest
```

Wordweave 没有身份验证，请勿直接将端口暴露到不可信网络。

## 使用生词本

在「收录与建组」中添加词条，或选择已有词条建立关系组。每组至少包含两个不同的词条，可选择关系类型并填写说明；同一批词条可以按不同类型建立多个组。在词条详情中点击成员，可以继续查看相关词条。

使用顶部搜索框查找词条或关系说明。在全部词条和关系浏览页面，可以切换布局、排序和复习模式。关系组可编辑或删除；删除关系组不会删除其中的词条。

## 数据、备份与恢复

Windows 默认将数据保存在 `wordweave.exe` 旁的 `data` 目录中：

```text
wordweave.exe
data/
  wordweave.db
  backups/          # 数据库升级前自动生成的备份
```

停止程序后，复制整个 `data` 目录即可备份或迁移数据。Docker 用户应停止容器后备份整个 `wordweave-data` 卷，而不是只复制运行中的数据库文件。

数据库需要升级时，程序会先在 `backups` 目录创建备份；如果备份或升级失败，程序会停止启动。恢复时先停止程序，将选定的备份复制到一个新的空目录并命名为 `wordweave.db`，再指定该目录启动。保留原数据目录以便核对；旧版备份可能在启动时再次升级。

Windows 下可通过命令行指定数据目录或监听地址：

```powershell
.\wordweave.exe -data-dir D:\Notebooks\English
$env:WORDWEAVE_DATA_DIR = 'D:\Notebooks\English'
.\wordweave.exe
.\wordweave.exe -listen 127.0.0.1:8090
```

`-data-dir` 优先于环境变量 `WORDWEAVE_DATA_DIR`；都未设置时使用程序旁的 `data` 目录。显式指定的相对路径相对于启动时的工作目录。

## 更多文档

- [HTTP 自动化调用](docs/http-api.md)和 [OpenAPI 规范](docs/openapi.json)
- [开发、验证与发布](docs/development.md)
