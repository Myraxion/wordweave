import { test } from 'node:test'
import assert from 'node:assert/strict'
import { spawn, execFileSync } from 'node:child_process'
import { mkdtemp, rm } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { chromium } from 'playwright'

test('group drafts protect changes and independently saved entries survive abandonment', async () => {
  const dir = await mkdtemp(join(tmpdir(), 'wordweave-draft-'))
  const server = spawn('go', ['run', './cmd/wordweave', '-listen', '127.0.0.1:0', '-data-dir', dir], { cwd: '..', stdio: ['ignore', 'pipe', 'ignore'] })
  let browser
  try {
    const base = await new Promise((resolve, reject) => {
      const timer = setTimeout(() => reject(new Error('Go service did not start')), 30000)
      server.stdout.on('data', chunk => {
        const match = chunk.toString().match(/http:\/\/127\.0\.0\.1:\d+/)
        if (match) { clearTimeout(timer); resolve(match[0]) }
      })
      server.once('error', reject)
      server.once('exit', () => reject(new Error('Go service exited before starting')))
    })
    browser = await chromium.launch({ executablePath: process.env.CHROME_PATH || undefined })
    const page = await browser.newPage()
    page.setDefaultTimeout(5000)
    let dialogs = 0
    let accept = false
    page.on('dialog', dialog => { dialogs++; void (accept ? dialog.accept() : dialog.dismiss()) })
    await page.goto(base + '/entries/new')
    await page.getByLabel('词条拼写').fill('apple')
    await page.getByRole('button', { name: '收录词条', exact: true }).click()
    await page.getByText('已收录 apple', { exact: false }).waitFor()
    await page.getByRole('button', { name: '取消建组' }).click()
    assert.equal(dialogs, 1)
    assert.match(await page.locator('.selected-members').innerText(), /apple/)
    accept = true
    await page.getByRole('button', { name: '取消建组' }).click()
    await page.waitForURL(base + '/')
    assert.equal(dialogs, 2)
    await page.reload()
    await page.getByText('apple', { exact: true }).first().waitFor()
    assert.equal((await (await fetch(base + '/api/stats')).json()).groups, 0)
    await page.goto(base + '/entries/new')
    await page.getByLabel('词条拼写').fill('pear')
    await page.getByRole('button', { name: '收录词条', exact: true }).click()
    await page.getByText('已收录 pear', { exact: false }).waitFor()
    await page.getByRole('checkbox', { name: /apple/ }).check()
    await page.locator('#group-type').selectOption('近义')
    await page.getByRole('button', { name: '保存关系组' }).click()
    await page.waitForURL('**/groups/*')
    await page.reload()
    await page.getByLabel('搜索并多选词条').waitFor()
    assert.match(await page.locator('.selected-members').innerText(), /apple/)
    assert.match(await page.locator('.selected-members').innerText(), /pear/)
    accept = false
    await page.getByLabel('搜索并多选词条').fill('apple')
    await page.getByRole('button', { name: '取消编辑' }).click()
    await page.waitForURL('**/relations')
    assert.equal(dialogs, 2, 'search alone is not a draft')
    await page.goBack()
    await page.locator('#group-note').fill('changed')
    await page.getByRole('button', { name: '取消编辑' }).click()
    assert.equal(dialogs, 3)
    assert.equal(await page.locator('#group-note').inputValue(), 'changed')
    await page.getByRole('link', { name: /返回关系浏览/ }).click()
    assert.equal(dialogs, 4)
    await page.getByLabel('搜索词条与关系说明').fill('apple')
    await page.getByRole('button', { name: /搜索/ }).click()
    assert.equal(dialogs, 5)
    assert.equal(await page.locator('#group-note').inputValue(), 'changed')
    await page.getByRole('button', { name: '＋ 新增缺少的词条' }).click()
    await page.getByLabel('词条拼写').fill('draft entry')
    await page.getByRole('button', { name: '取消新增' }).click()
    assert.equal(dialogs, 6)
    assert.equal(await page.getByLabel('词条拼写').inputValue(), 'draft entry')
    accept = true
    await page.getByRole('button', { name: '取消新增' }).click()
    assert.equal(dialogs, 7)
    assert.equal(await page.locator('#group-note').inputValue(), 'changed')
    accept = false
    await page.locator('#group-note').fill('')
    await page.locator('#group-type').selectOption('反义')
    await page.getByRole('button', { name: '取消编辑' }).click()
    assert.equal(dialogs, 8)
    await page.locator('#group-type').selectOption('近义')
    await page.getByRole('button', { name: '取消选择 apple' }).click()
    await page.getByRole('button', { name: '取消编辑' }).click()
    assert.equal(dialogs, 9)
    await page.getByRole('checkbox', { name: /apple/ }).check()
    await page.getByRole('button', { name: '取消编辑' }).click()
    await page.waitForURL('**/relations')
    assert.equal(dialogs, 9, 'restored values are clean')
  } finally {
    await browser?.close()
    if (process.platform === 'win32') { try { execFileSync('taskkill', ['/pid', String(server.pid), '/T', '/F'], { stdio: 'ignore' }) } catch {} }
    else server.kill()
    await rm(dir, { recursive: true, force: true, maxRetries: 10, retryDelay: 200 })
  }
})



