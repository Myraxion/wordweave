# 开发、验证与发布

## 构建与开发

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

前端页面测试首次运行前执行 `npx --prefix web playwright install chromium --only-shell`，之后通过 `npm --prefix web test` 验证浏览行为。

`internal/app/app_test.go` 通过公开 HTTP 和隔离临时 SQLite 验证收录、编辑、搜索、排序及持久化；`groups_test.go` 在同一入口验证建组、成员去重、无序判重、非法创建回滚、排序、跨成员共享、重叠组边界、编辑传播与重启。`app.Open(dataDir)` 是可复用的应用入口，不使用 mock 数据库。

`internal/app/upgrade_test.go` 用旧版数据库和受控失败验证启动结果，包括 WAL 一致性备份及恢复、脚本／版本记录／提交失败回滚、多项待迁移整体回滚、备份失败、无迁移启动和拒绝降级；通过 HTTP 确认升级及失败重试后仍可读取原词条。

`tests/upgrade_test.go` 使用正式程序进程验证旧库升级、从备份恢复、失败后停止启动及重试读取原数据；覆盖迁移失败、备份失败和较新数据库拒绝启动。

浏览器基本冒烟：空列表 → 收录带多行释义的短语 → 详情 → 返回列表 → 大小写重复添加 → 进入已有词条 → 仅拼写收词 → 刷新详情。

关系组冒烟：从词条详情建组 → 单成员不能保存 → 搜索多选并切换搜索 → 保存多行说明 → 查看完整成员及 A–Z 排序 → 点击另一成员 → 反向顺序重复建组被阻止并进入已有组 → 同成员不同类型建组 → 重叠组各自显示说明、同类型最近创建在前 → 沿成员继续导航。

关系组编辑冒烟：从组详情编辑类型、说明和成员 → 从各成员详情核对同一份更新 → 尝试保存为已有组并查看冲突入口 → 删除整组时取消确认并检查组仍存在 → 编辑组移除成员至不足两个，检查删除提示并先取消、再确认 → 检查组已删除而词条仍存在。

## 发布


将 `v` 开头的 tag 推送到 GitHub 会运行 Release 工作流，构建 Windows x64 压缩包和 SHA-256 校验文件，自动生成 GitHub Release 说明，并发布同名 tag 的 GHCR 镜像。例如：

```sh
git tag v1.0.0
git push origin v1.0.0
```

包含 `-` 的 tag 会标记为预发布且不会更新镜像 `latest`；其他 tag 会更新 `latest`。工作流使用内置 `GITHUB_TOKEN`，需要仓库允许 Actions 写入 Release 和 Packages。镜像发布失败时可在 Actions 中重新运行工作流。
