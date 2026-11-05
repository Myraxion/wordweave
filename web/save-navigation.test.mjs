import { test } from 'node:test'
import assert from 'node:assert/strict'
import { spawn } from 'node:child_process'
import { chromium } from 'playwright'

test('saving a group blocks switching tasks and preserves the draft on failure', async () => {
  const server = spawn(process.execPath, ['node_modules/vite/bin/vite.js', '--host', '127.0.0.1', '--port', '5183', '--strictPort'], { stdio: 'ignore' })
  let browser
  try {
    for (let i = 0; i < 80; i++) { try { if ((await fetch('http://127.0.0.1:5183')).ok) break } catch {} await new Promise(resolve => setTimeout(resolve, 100)) }
    browser = await chromium.launch({ executablePath: process.env.CHROME_PATH || undefined })
    const page = await browser.newPage()
    const entries = [{ id: 1, spelling: 'apple', definition: 'fruit', createdAt: '2025-01-01T00:00:00Z' }, { id: 2, spelling: 'pear', definition: 'fruit', createdAt: '2025-01-01T00:00:00Z' }]
    let release
    const pending = new Promise(resolve => { release = resolve })
    await page.route('**/api/**', async route => {
      const path = new URL(route.request().url()).pathname
      if (path === '/api/entries/1/groups' && route.request().method() === 'POST') { await pending; return route.fulfill({ status: 500, json: { error: '保存失败' } }) }
      if (path === '/api/entries/1/groups') return route.fulfill({ json: [] })
      if (path === '/api/entries/1') return route.fulfill({ json: entries[0] })
      if (path === '/api/group-types') return route.fulfill({ json: ['近义'] })
      if (path === '/api/stats') return route.fulfill({ json: { entries: 2, groups: 0 } })
      if (path === '/api/entries') return route.fulfill({ json: entries })
      throw new Error(`Unexpected API: ${path}`)
    })
    page.on('dialog', dialog => void dialog.accept())
    await page.goto('http://127.0.0.1:5183/entries/1')
    await page.getByRole('button', { name: '＋ 建立关系组' }).click()
    await page.getByRole('checkbox', { name: /pear/ }).check()
    await page.locator('#group-type').selectOption('近义')
    await page.locator('#group-note').fill('保留的说明')
    await page.getByRole('button', { name: '保存关系组' }).click()
    await page.getByRole('button', { name: /编辑词条/ }).click()
    await page.getByText('正在保存，请稍候').waitFor({ timeout: 2000 })
    assert.equal(await page.locator('#group-note').inputValue(), '保留的说明')
    release()
    await page.getByText('保存失败').waitFor()
    assert.equal(await page.locator('#group-note').inputValue(), '保留的说明')
  } finally { await browser?.close(); server.kill() }
})

test('entry and group saves block navigation, retain failures, and do not replay blocked jumps', async () => {
  const server = spawn(process.execPath, ['node_modules/vite/bin/vite.js', '--host', '127.0.0.1', '--port', '5184', '--strictPort'], { stdio: 'ignore' })
  let browser
  try {
    for (let i = 0; i < 80; i++) { try { if ((await fetch('http://127.0.0.1:5184')).ok) break } catch {} await new Promise(resolve => setTimeout(resolve, 100)) }
    browser = await chromium.launch({ executablePath: process.env.CHROME_PATH || undefined })
    const page = await browser.newPage()
    const entries = [{ id: 1, spelling: 'apple', definition: 'fruit', createdAt: '2025-01-01T00:00:00Z' }, { id: 2, spelling: 'pear', definition: 'fruit', createdAt: '2025-01-01T00:00:00Z' }]
    const group = { id: 7, type: '近义', note: 'original', members: entries }
    let release, started, fail = true
    let gate = new Promise(resolve => { release = resolve })
    let requestStarted = new Promise(resolve => { started = resolve })
    const nextRequest = () => { gate = new Promise(resolve => { release = resolve }); requestStarted = new Promise(resolve => { started = resolve }) }
    await page.route('**/api/**', async route => {
      const path = new URL(route.request().url()).pathname
      const method = route.request().method()
      if (method === 'PUT' && (path === '/api/entries/1' || path === '/api/groups/7')) {
        started(); await gate
        if (fail) return route.fulfill({ status: 500, json: { error: '保存失败' } })
        const data = route.request().postDataJSON()
        return route.fulfill({ json: path.includes('/groups/') ? { ...group, ...data } : { ...entries[0], ...data } })
      }
      if (path === '/api/entries/1/groups') return route.fulfill({ json: [group] })
      if (path === '/api/entries/1') return route.fulfill({ json: entries[0] })
      if (path === '/api/groups/7') return route.fulfill({ json: group })
      if (path === '/api/group-types') return route.fulfill({ json: ['近义'] })
      if (path === '/api/stats') return route.fulfill({ json: { entries: 2, groups: 1 } })
      if (path === '/api/entries') return route.fulfill({ json: entries })
      if (path === '/api/groups') return route.fulfill({ json: [group] })
      if (path === '/api/search') return route.fulfill({ json: { entries, groups: [] } })
      throw new Error(`Unexpected API: ${path}`)
    })
    let dialogs = 0
    page.on('dialog', dialog => { dialogs++; void dialog.dismiss() })
    const checkBlocked = async () => {
      const url = page.url()
      await page.getByRole('link', { name: /全部词条/, exact: false }).first().click()
      await page.getByText('正在保存，请稍候').waitFor()
      await page.getByLabel('搜索词条与关系说明').fill('pear')
      await page.getByRole('button', { name: /搜索/, exact: false }).first().click()
      assert.equal(page.url(), url)
      assert.equal(dialogs, 0)
    }
    await page.goto('http://127.0.0.1:5184/entries/1')
    await page.getByRole('button', { name: /编辑词条/ }).click()
    await page.locator('#definition').fill('changed entry')
    await page.getByRole('button', { name: /保存修改/ }).click()
    await requestStarted; await checkBlocked(); release()
    await page.getByText('保存失败').waitFor()
    assert.equal(await page.locator('#definition').inputValue(), 'changed entry')
    assert.match(page.url(), /\/entries\/1$/)
    fail = false; nextRequest()
    await page.getByRole('button', { name: /保存修改/ }).click()
    await requestStarted; await checkBlocked(); release()
    await page.getByText('changed entry', { exact: true }).first().waitFor()
    assert.match(page.url(), /\/entries\/1$/)
    await page.goto('http://127.0.0.1:5184/groups/7')
    fail = true; nextRequest()
    await page.locator('#group-note').fill('changed group')
    await page.getByRole('button', { name: '保存关系组' }).click()
    await requestStarted; await checkBlocked(); release()
    await page.getByText('保存失败').waitFor()
    assert.equal(await page.locator('#group-note').inputValue(), 'changed group')
    assert.match(page.url(), /\/groups\/7$/)
    fail = false; nextRequest()
    await page.getByRole('button', { name: '保存关系组' }).click()
    await requestStarted; await checkBlocked(); release()
    await page.waitForURL('**/relations')
    assert.equal(dialogs, 0)
  } finally { await browser?.close(); server.kill() }
})

test('collecting an entry preserves an unrelated group draft', async () => {
  const server = spawn(process.execPath, ['node_modules/vite/bin/vite.js', '--host', '127.0.0.1', '--port', '5185', '--strictPort'], { stdio: 'ignore' })
  let browser
  try {
    for (let i = 0; i < 80; i++) { try { if ((await fetch('http://127.0.0.1:5185')).ok) break } catch {} await new Promise(resolve => setTimeout(resolve, 100)) }
    browser = await chromium.launch({ executablePath: process.env.CHROME_PATH || undefined })
    const page = await browser.newPage()
    let release, started
    const gate = new Promise(resolve => { release = resolve })
    const requestStarted = new Promise(resolve => { started = resolve })
    await page.route('**/api/**', async route => {
      const path = new URL(route.request().url()).pathname
      if (path === '/api/entries' && route.request().method() === 'POST') { started(); await gate; return route.fulfill({ json: { id: 1, spelling: 'apple', definition: '' } }) }
      if (path === '/api/entries') return route.fulfill({ json: [] })
      if (path === '/api/group-types') return route.fulfill({ json: ['近义'] })
      if (path === '/api/stats') return route.fulfill({ json: { entries: 0, groups: 0 } })
      throw new Error(`Unexpected API: ${path}`)
    })
    let dialogs = 0
    page.on('dialog', dialog => { dialogs++; void dialog.dismiss() })
    await page.goto('http://127.0.0.1:5185/entries/new')
    await page.locator('#group-note').fill('group draft')
    await page.getByLabel('词条拼写').fill('apple')
    await page.getByRole('button', { name: '收录词条', exact: true }).click()
    await requestStarted
    await page.getByRole('link', { name: /关系浏览/ }).click()
    await page.getByText('正在保存，请稍候').waitFor()
    release()
    await page.getByText('已选择 apple', { exact: false }).waitFor()
    assert.equal(await page.getByLabel('词条拼写').inputValue(), '')
    assert.equal(await page.locator('#group-note').inputValue(), 'group draft')
    assert.match(page.url(), /\/entries\/new$/)
    await page.getByRole('link', { name: /关系浏览/ }).click()
    assert.equal(dialogs, 1)
    assert.equal(await page.locator('#group-note').inputValue(), 'group draft')
  } finally { await browser?.close(); server.kill() }
})

test('stale entry edit sends its version and preserves the draft', async () => {
  const server = spawn(process.execPath, ['node_modules/vite/bin/vite.js', '--host', '127.0.0.1', '--port', '5185', '--strictPort'], { stdio: 'ignore' })
  let browser
  try {
    for (let i = 0; i < 80; i++) { try { if ((await fetch('http://127.0.0.1:5185')).ok) break } catch {} await new Promise(resolve => setTimeout(resolve, 100)) }
    browser = await chromium.launch({ executablePath: process.env.CHROME_PATH || undefined })
    const page = await browser.newPage()
    let writes = 0
    await page.route('**/api/**', async route => {
      const path = new URL(route.request().url()).pathname
      if (path === '/api/entries/1' && route.request().method() === 'PUT') {
        writes++
        assert.equal(route.request().headers()['if-match'], '"entry-version"')
        return route.fulfill({ status: 412, json: { code: 'version_conflict', error: '内容已变化，请重新读取后再操作。' } })
      }
      if (path === '/api/entries/1') return route.fulfill({ headers: { etag: '"entry-version"' }, json: { id: 1, spelling: 'apple', definition: 'old', createdAt: '2025-01-01T00:00:00Z', updatedAt: null } })
      if (path === '/api/entries/1/groups') return route.fulfill({ json: [] })
      if (path === '/api/stats') return route.fulfill({ json: { entries: 1, groups: 0 } })
      throw new Error(`Unexpected API: ${path}`)
    })
    await page.goto('http://127.0.0.1:5185/entries/1')
    await page.getByRole('button', { name: /编辑词条/ }).click()
    await page.locator('#definition').fill('draft')
    await page.getByRole('button', { name: /保存修改/ }).click()
    await page.getByText('内容已变化，请重新读取后再操作。').waitFor()
    assert.equal(await page.locator('#definition').inputValue(), 'draft')
    assert.equal(writes, 1)
  } finally { await browser?.close(); server.kill() }
})
