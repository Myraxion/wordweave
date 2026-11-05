## Agent skills

### Issue tracker

GitHub issues (`Myraxion/wordweave`). See `docs/agents/issue-tracker.md`.

### Triage labels

Canonical five-role vocabulary. See `docs/agents/triage-labels.md`.

### Domain docs

Single-context (`GLOSSARY.md` + `docs/adr/` at repo root). See `docs/agents/domain.md`.

## 内置浏览器预览

Windows 上沙箱内服务的入站回环可能被 AppContainer 阻止。使用内置浏览器检查页面时，先运行 `npm --prefix web run build`，再在宿主普通进程中启动 `go run ./cmd/wordweave -listen 127.0.0.1:5187 -data-dir .scratch/browser-preview`（需要时申请沙箱外执行），访问 `http://127.0.0.1:5187`。预览结束后停止服务；测试数据使用独立目录。
