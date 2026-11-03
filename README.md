# Wordweave · 生词本

个人本机生词本。支持收录英文单词或短语、记录多行纯文本释义、浏览、搜索词条列表和详情。重复拼写忽略大小写，提示进入已有词条。数据保存在 SQLite 中，重启后继续使用。

## Windows 使用

运行 `wordweave.exe`，在浏览器打开程序输出的地址（默认 <http://127.0.0.1:8080>）。关闭窗口或按 Ctrl+C 停止程序。正式运行不需要 Go、Node、数据库服务或互联网。

默认布局：

```text
wordweave.exe
data/
  wordweave.db
```

数据目录按可执行文件位置定位，与启动工作目录无关。停止程序后，将 exe 与整个 `data` 目录一起搬移即可保留生词本。请将程序放在可写目录。

```powershell
.\wordweave.exe -data-dir D:\Notebooks\English
$env:WORDWEAVE_DATA_DIR = 'D:\Notebooks\English'
.\wordweave.exe
.\wordweave.exe -listen 127.0.0.1:8090
```

`-data-dir` 优先于 `WORDWEAVE_DATA_DIR`，两者均未设置时使用程序旁的 `data`；显式相对路径相对于启动工作目录。默认仅监听本机回环地址。

## 开发与构建

需要 Go 1.26+、Node.js 22.12+（或 20.19+）和 npm。提交锁定的 npm 依赖和 Go 模块校验和。

```powershell
.\scripts\build.ps1
```

产物为 `bin/wordweave.exe`。脚本先构建 Vue，再将资源与编号 SQL 迁移一起嵌入 Go 程序。依赖下载和编译需要网络，交付后的程序离线运行。也可以手动执行：

```powershell
npm --prefix web ci
npm --prefix web run build
go build -trimpath -ldflags='-s -w' -o bin/wordweave.exe ./cmd/wordweave
```

开发时先完成一次前端构建，再在两个终端运行：

```powershell
go run ./cmd/wordweave -data-dir .scratch/dev-data
npm --prefix web run dev
```

Vite 页面代理 `/api` 到 `127.0.0.1:8080`。`go run` 使用临时可执行文件，开发时请显式指定数据目录。

## 验证

先构建前端（Go 编译需要 `web/dist`），再执行：

```powershell
npm --prefix web run build
go vet ./...
go test ./...
```

`internal/app/app_test.go` 通过公开 HTTP 和隔离临时 SQLite 验证收录、输入保留、判重、列表、详情及重启持久化；`app.Open(dataDir)` 是可复用的应用入口，不使用 mock 数据库。

`tests/portable_test.go` 构建正式程序，将它单独放入临时目录，从其他工作目录启动，验证本地前端资源、重启及整体搬移、参数和环境变量指定的数据目录。测试不访问个人生词本。

浏览器基本冒烟：空列表 → 收录带多行释义的短语 → 详情 → 返回列表 → 大小写重复添加 → 进入已有词条 → 仅拼写收词 → 刷新详情。

## HTTP 接口

| 方法与路径 | 行为 |
| --- | --- |
| `POST /api/entries` | JSON `{ "spelling": "Apple", "definition": "苹果\n苹果树" }`；释义可省略 |
| `GET /api/entries` | 词条数组，最近添加在前；可用 `q` 搜索拼写或释义，用 `sort` 指定排序；空列表为 `[]` |
| `GET /api/entries/{id}` | 词条详情，不存在时为 404 |

词条返回 `id`、`spelling`、`definition`、`createdAt`（UTC RFC 3339）。创建成功为 201；拼写为空为 400；重复为 409，响应包含 `error` 和 `existingEntry`，不会覆盖原词条。

## 当前范围

本次实现对应 [issue #2](https://github.com/Myraxion/wordweave/issues/2)。词条搜索见 [issue #4](https://github.com/Myraxion/wordweave/issues/4)；关系组和删除由后续票据实现。

启动时使用嵌入的 `0001_entries.sql` 初始化数据库，并在 `schema_migrations` 中记录版本；迁移与版本记录在同一事务中提交。当前支持新库初始化和同版本重启，数据库版本较新时拒绝启动。已有数据库升级前的一致性备份、失败升级验证等完整保护流程属于 T9（[issue #10](https://github.com/Myraxion/wordweave/issues/10)），需在发布下一版迁移前完成。
