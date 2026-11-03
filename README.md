# Wordweave · 生词本

个人本机生词本。支持收录、编辑和搜索英文单词或短语，记录多行纯文本释义，从词条出发搜索多选已有成员建立关系组，并沿成员导航进行对照阅读。词条和关系组均有重复提示与已有内容入口。数据保存在 SQLite 中，重启后继续使用。

## Windows 使用

运行 `wordweave.exe`，在浏览器打开程序输出的地址（默认 <http://127.0.0.1:8080>）。关闭窗口或按 Ctrl+C 停止程序。正式运行不需要 Go、Node、数据库服务或互联网。

默认布局：

```text
wordweave.exe
data/
  wordweave.db
  backups/          # 已有数据库升级前自动生成
```

数据目录按可执行文件位置定位，与启动工作目录无关。停止程序后，将 exe 与整个 `data` 目录一起搬移即可保留生词本。请将程序放在可写目录。

```powershell
.\wordweave.exe -data-dir D:\Notebooks\English
$env:WORDWEAVE_DATA_DIR = 'D:\Notebooks\English'
.\wordweave.exe
.\wordweave.exe -listen 127.0.0.1:8090
```

`-data-dir` 优先于 `WORDWEAVE_DATA_DIR`，两者均未设置时使用程序旁的 `data`；显式相对路径相对于启动工作目录。默认仅监听本机回环地址。

## 关系组与对照阅读

在词条详情点击「建立关系组」，搜索并勾选其他已有词条。切换搜索保留已选成员，当前词条始终包含在组中；选择一种关系类型，可填写多行纯文本说明，然后保存。至少有两个不同成员才能保存，取消建组不改变已有数据。

固定类型依次为形音相似、近义、反义、词族、屈折变化、拼写变体、语义相关。详情按此顺序分区，同类型组最近创建在前，组内成员按英文 A–Z 忽略大小写排列。每组完整展示成员、当前释义及共享说明；点击成员继续阅读，点击组标题查看独立组详情。重叠组各自保留边界，同一批成员可按不同类型分别建组。

类型与成员集合完全相同即为重复，改变成员顺序或说明不会产生新组；冲突时提供已有组入口。修改词条拼写或释义后，各组展示同步更新。

## 数据库升级与恢复

启动时执行嵌入的编号 SQL 迁移。新库直接初始化；已有库需要升级时，先用 SQLite `VACUUM INTO` 生成一致性快照，保存到数据目录的 `backups/wordweave-v旧版本-to-v新版本-随机后缀.db`，包含 WAL 中已提交的数据。快照方式见 [SQLite 官方文档](https://www.sqlite.org/lang_vacuum.html#vacuuminto)。

所有待执行迁移和版本记录在同一事务中提交。备份、脚本、版本记录或提交失败时停止启动并报告错误，原数据保留，不自动重建；数据库版本比程序新时拒绝打开。同版本重启不生成备份。

需要恢复时先停止程序，将选定备份复制到一个新的空目录，命名为 `wordweave.db`，再通过 `-data-dir` 指向该目录启动。原目录先保留，以便核对；恢复到旧版本的快照会按当前程序版本重新执行升级。

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

`internal/app/app_test.go` 通过公开 HTTP 和隔离临时 SQLite 验证收录、编辑、搜索、排序及持久化；`groups_test.go` 在同一入口验证建组、成员去重、无序判重、非法创建回滚、排序、跨成员共享、重叠组边界、编辑传播与重启。`app.Open(dataDir)` 是可复用的应用入口，不使用 mock 数据库。

`internal/app/upgrade_test.go` 用旧版数据库和受控失败验证启动结果，包括 WAL 一致性备份及恢复、脚本／版本记录／提交失败回滚、多项待迁移整体回滚、备份失败、无迁移启动和拒绝降级；通过 HTTP 确认升级及失败重试后仍可读取原词条。

`tests/portable_test.go` 构建正式程序，将它单独放入临时目录，从其他工作目录启动，验证本地前端资源、重启及整体搬移、参数和环境变量指定的数据目录。测试不访问个人生词本。

`tests/upgrade_test.go` 使用正式程序进程验证旧库升级、从备份恢复、失败后停止启动及重试读取原数据；覆盖迁移失败、备份失败和较新数据库拒绝启动。

浏览器基本冒烟：空列表 → 收录带多行释义的短语 → 详情 → 返回列表 → 大小写重复添加 → 进入已有词条 → 仅拼写收词 → 刷新详情。

关系组冒烟：从词条详情建组 → 单成员不能保存 → 搜索多选并切换搜索 → 保存多行说明 → 查看完整成员及 A–Z 排序 → 点击另一成员 → 反向顺序重复建组被阻止并进入已有组 → 同成员不同类型建组 → 重叠组各自显示说明、同类型最近创建在前 → 沿成员继续导航。

## HTTP 接口

| 方法与路径 | 行为 |
| --- | --- |
| `POST /api/entries` | JSON `{ "spelling": "Apple", "definition": "苹果\n苹果树" }`；释义可省略 |
| `GET /api/entries` | 词条数组，最近添加在前；可用 `q` 搜索拼写或释义，用 `sort` 指定排序；空列表为 `[]` |
| `GET /api/entries/{id}` | 词条详情，不存在时为 404 |
| `PUT /api/entries/{id}` | 修改拼写和释义，保留添加时间 |
| `GET /api/group-types` | 按固定顺序返回七种关系类型 |
| `POST /api/entries/{id}/groups` | JSON `{ "type": "近义", "memberIds": [2, 3], "note": "共享说明" }`；路径中的词条自动加入，说明可省略，重复成员去重 |
| `GET /api/entries/{id}/groups` | 该词条所属的完整关系组数组，按类型及创建时间排序；无组为 `[]`，词条不存在为 404 |
| `GET /api/groups/{id}` | 关系组详情，不存在为 404 |

词条返回 `id`、`spelling`、`definition`、`createdAt`（UTC RFC 3339）。创建成功为 201；拼写为空为 400；重复为 409，响应包含 `error` 和 `existingEntry`，不会覆盖原词条。

关系组返回 `id`、`type`、`note`、`createdAt` 和 `members`（完整词条数组）。创建成功为 201，`Location` 指向组详情接口；无效类型、不足两个不同成员或引用不存在的成员为 400。重复为 409，响应包含 `error` 和完整的 `existingGroup`；非法或重复创建不会留下部分组数据。

## 当前范围

词条收录见 [issue #2](https://github.com/Myraxion/wordweave/issues/2)，搜索见 [issue #4](https://github.com/Myraxion/wordweave/issues/4)。本次关系组创建与对照阅读对应 [issue #5](https://github.com/Myraxion/wordweave/issues/5)，并补齐新增迁移前的 [issue #10](https://github.com/Myraxion/wordweave/issues/10) 升级保护。当前数据库版本为 2，词条与关系组分别由 `0001_entries.sql`、`0002_groups.sql` 建立。组编辑、建组过程中新收词、关系说明搜索、按类型浏览及删除由后续票据实现。
