# HTTP 自动化调用

服务默认在 `http://127.0.0.1:8080` 提供 API。完整路径、字段和响应见 [OpenAPI 规范](openapi.json)。响应使用 UTF-8 JSON；请求 JSON 必须是对象，未知字段、`null` 或额外内容均会被拒绝。拼写去除首尾空格并忽略大小写判重。

`PUT` 完整替换，`PATCH` 仅修改提供的字段；空字符串清空释义或说明。`memberIds` 一旦提供即替换完整成员集合，不足两个成员返回 400，删除组须使用 DELETE。收录和建组独立保存。词条或组重复创建返回 409 和 `existingEntry` 或 `existingGroup`，不会覆盖现有内容。

词条与组详情返回 `ETag`；更新或删除时可将原值传入 `If-Match`。内容发生变化时返回 412，调用者应重新读取并决定如何处理，不要盲目重试。错误响应包含 `code` 和 `error`；`code` 包括 `invalid_request`、`not_found`、`method_not_allowed`、`duplicate_entry`、`duplicate_group`、`version_conflict` 和 `internal_error`。

删除词条返回 `deleted`、`updatedGroupIds`、`deletedGroupIds`；当删除导致重复关系组时返回 409，所有数据保持不变。空列表为 `[]`，时间是 UTC RFC 3339，旧记录的 `updatedAt` 可能为 `null`。

`GET /api/groups` 可选用 `limit`（1 至 500）和 `offset`（非负整数，默认为 0）分页，例如 `/api/groups?sort=updated_desc&limit=50&offset=50`。不传 `limit` 时保持返回全部结果的原有行为。分页使用当前数据的排序，读取期间数据变化可能改变后续页。POST、PUT 和 PATCH 的请求正文上限为 16 MiB，超过时返回 HTTP 413 和 `request_too_large` 错误码。

下面的 PowerShell 7 示例连续收录两个词条并建组；遇到重复时复用已有对象。运行前先启动程序。

```powershell
$base = 'http://127.0.0.1:8080/api'
function Add-Entry($spelling, $definition) {
  $body = @{ spelling = $spelling; definition = $definition } | ConvertTo-Json
  $r = Invoke-WebRequest "$base/entries" -Method POST -ContentType 'application/json; charset=utf-8' -Body ([System.Text.Encoding]::UTF8.GetBytes($body)) -SkipHttpErrorCheck
  $data = $r.Content | ConvertFrom-Json
  if ($r.StatusCode -eq 201) { return $data }
  if ($r.StatusCode -eq 409 -and $data.existingEntry) { return $data.existingEntry }
  throw "HTTP $($r.StatusCode): $($data.error)"
}
$a = Add-Entry 'apple' '苹果'
$b = Add-Entry 'pear' '梨'
$body = @{ type = '语义相关'; memberIds = @($a.id, $b.id); note = '水果' } | ConvertTo-Json
$r = Invoke-WebRequest "$base/groups" -Method POST -ContentType 'application/json; charset=utf-8' -Body ([System.Text.Encoding]::UTF8.GetBytes($body)) -SkipHttpErrorCheck
$data = $r.Content | ConvertFrom-Json
if ($r.StatusCode -eq 201) { $data }
elseif ($r.StatusCode -eq 409 -and $data.existingGroup) { $data.existingGroup }
else { throw "HTTP $($r.StatusCode): $($data.error)" }
```

curl 示例可查询词条，然后使用版本条件修改释义（以下为 POSIX shell）：

```sh
base=http://127.0.0.1:8080/api
curl -fsSG "$base/entries/exact" --data-urlencode 'spelling=apple'
curl -fsS -D headers.txt "$base/entries/1" -o entry.json
etag=$(awk 'tolower($0) ~ /^etag:/ {sub(/^[^:]*: */, ""); sub(/\r$/, ""); print}' headers.txt)
curl -fsS -X PATCH "$base/entries/1" -H 'Content-Type: application/json' -H "If-Match: $etag" --data '{"definition":"苹果"}'
```

在上述 PowerShell 会话中，可运行以下示例演示省略拼写并清空释义、输出结构化错误，以及查看删除对关系组的影响。最后一步会实际删除刚才选定的 `apple` 词条。

```powershell
$detail = Invoke-WebRequest "$base/entries/$($a.id)"
$etag = $detail.Headers.ETag
$clear = Invoke-WebRequest "$base/entries/$($a.id)" -Method PATCH -Headers @{ 'If-Match' = $etag } -ContentType 'application/json' -Body '{"definition":""}' -SkipHttpErrorCheck
if ($clear.StatusCode -ne 200) {
  $problem = $clear.Content | ConvertFrom-Json
  Write-Output "错误码：$($problem.code)；说明：$($problem.error)"
  throw '清空释义失败'
}
$updated = $clear.Content | ConvertFrom-Json
Write-Output "拼写保持为 $($updated.spelling)，释义已清空。"
$deletion = Invoke-WebRequest "$base/entries/$($a.id)" -Method DELETE -Headers @{ 'If-Match' = $clear.Headers.ETag } -SkipHttpErrorCheck
$result = $deletion.Content | ConvertFrom-Json
if ($deletion.StatusCode -eq 200) {
  Write-Output "修改的组：$($result.updatedGroupIds -join ', ')；删除的组：$($result.deletedGroupIds -join ', ')"
} else {
  Write-Output "错误码：$($result.code)；说明：$($result.error)"
}
```
