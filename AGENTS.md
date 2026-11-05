## Agent skills

### Issue tracker

GitHub issues (`Myraxion/wordweave`). See `docs/agents/issue-tracker.md`.

### Triage labels

Canonical five-role vocabulary. See `docs/agents/triage-labels.md`.

### Domain docs

Single-context (`GLOSSARY.md` + `docs/adr/` at repo root). See `docs/agents/domain.md`.

## 内置浏览器预览

Windows 上沙箱内服务的入站回环可能被 AppContainer 阻止。使用内置浏览器检查页面时，先运行 `npm --prefix web run build`，再在宿主普通进程中启动 `go run ./cmd/wordweave -listen 127.0.0.1:5187 -data-dir .scratch/browser-preview`（需要时申请沙箱外执行），访问 `http://127.0.0.1:5187`。预览结束后停止服务；测试数据使用独立目录。

## Playwright 浏览器测试

使用 Playwright 自带的 Chromium，首次运行前按 README 安装。Windows 上在独立 PowerShell 进程中清除 `CHROME_PATH` 后运行测试，以保留用户环境变量：

```powershell
powershell -NoProfile -Command '$env:CHROME_PATH = ""; npm --prefix web test; exit $LASTEXITCODE'
```

若沙箱阻止 Chromium 启动或服务回环访问，在宿主环境运行同一命令。
