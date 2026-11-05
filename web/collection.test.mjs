import { test } from 'node:test'
import assert from 'node:assert/strict'
import { spawn } from 'node:child_process'
import { chromium } from 'playwright'

test('collection and group tasks adapt without losing input', async () => {
  const server = spawn(process.execPath, ['node_modules/vite/bin/vite.js', '--host', '127.0.0.1', '--port', '5181', '--strictPort'], { stdio: 'ignore' })
  let browser
  try {
    for (let i = 0; i < 80; i++) { try { if ((await fetch('http://127.0.0.1:5181')).ok) break } catch {} await new Promise(resolve => setTimeout(resolve, 100)) }
    browser = await chromium.launch({ executablePath: process.env.CHROME_PATH || undefined })
    const page = await browser.newPage({ viewport: { width: 1440, height: 900 } })
    const entries = [{ id: 1, spelling: 'long phrase '.repeat(20), definition: '长释义'.repeat(100) }]
    await page.route('**/api/**', async route => {
      const path = new URL(route.request().url()).pathname
      if (path === '/api/group-types') return route.fulfill({ json: ['近义'] })
      if (path === '/api/stats') return route.fulfill({ json: { entries: entries.length, groups: 0 } })
      if (path === '/api/search') return route.fulfill({ json: { entries, groups: [] } })
      if (path === '/api/entries/2/groups' && route.request().method() === 'POST') return route.fulfill({ json: { id: 7, type: '近义', note: 'draft note', members: entries } })
      if (path === '/api/groups/7') return route.fulfill({ json: { id: 7, type: '近义', note: 'draft note', members: entries } })
      if (path === '/api/entries' && route.request().method() === 'GET') return route.fulfill({ json: entries })
      if (path === '/api/entries' && route.request().method() === 'POST') { const entry = { id: 2, ...route.request().postDataJSON() }; entries.push(entry); return route.fulfill({ json: entry }) }
      throw new Error(`Unexpected API: ${path}`)
    })
    await page.goto('http://127.0.0.1:5181/entries/new')
    const collector = page.getByRole('heading', { name: '收录词条', exact: true })
    const group = page.getByRole('heading', { name: '建立关系组' })
    await collector.waitFor()
    let a = await collector.boundingBox(), b = await group.boundingBox()
    assert.ok(a.x < b.x && Math.abs(a.y - b.y) < 100, 'desktop tasks are side by side')
    const candidate = page.locator('.member-options li').first()
    await candidate.waitFor()
    assert.equal(await candidate.locator('strong').innerText(), entries[0].spelling)
    const size = await candidate.locator('small').evaluate(el => ({ height: el.clientHeight, line: parseFloat(getComputedStyle(el).lineHeight), scroll: el.scrollHeight }))
    assert.ok(size.height <= size.line * 2 + 1 && size.scroll > size.height)
    assert.match(await page.locator('.group-actions').innerText(), /至少选择两个/)
    await page.getByLabel('词条拼写').fill('new entry')
    assert.match(await page.locator('.group-actions').innerText(), /未保存的词条/)
    await page.locator('#group-note').fill('draft note')
    await page.setViewportSize({ width: 320, height: 900 })
    a = await collector.boundingBox(); b = await group.boundingBox()
    assert.ok(a.y < b.y, 'narrow layout follows task order')
    assert.equal(await page.locator('#group-note').inputValue(), 'draft note')
    assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth))
    await page.getByRole('button', { name: '收录词条', exact: true }).click()
    await page.getByText('已选择 new entry', { exact: false }).waitFor()
    assert.match(await page.locator('.selected-members').innerText(), /new entry/)
    assert.match(await page.locator('.group-actions').innerText(), /至少/)
    await candidate.locator('input').check()
    assert.match(await page.locator('.group-actions').innerText(), /请选择关系类型/)
    await page.locator('#group-type').selectOption('近义')
    assert.equal(await page.getByRole('button', { name: '保存关系组' }).isEnabled(), true)
    for (const width of [390, 651, 768, 1440]) {
      await page.setViewportSize({ width, height: 900 })
      assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), `no overflow at ${width}`)
      assert.equal(await page.locator('#group-note').inputValue(), 'draft note')
    }
    await page.setViewportSize({ width: 390, height: 900 })
    await page.getByRole('button', { name: '保存关系组' }).click()
    await page.waitForURL('**/groups/7')
  } finally { await browser?.close(); server.kill() }
})


test('group editing keeps member and relation tasks in order while resizing', async () => {
  const server = spawn(process.execPath, ['node_modules/vite/bin/vite.js', '--host', '127.0.0.1', '--port', '5182', '--strictPort'], { stdio: 'ignore' })
  let browser
  try {
    for (let i = 0; i < 80; i++) { try { if ((await fetch('http://127.0.0.1:5182')).ok) break } catch {} await new Promise(resolve => setTimeout(resolve, 100)) }
    browser = await chromium.launch({ executablePath: process.env.CHROME_PATH || undefined })
    const page = await browser.newPage({ viewport: { width: 1440, height: 900 } })
    const entries = [{ id: 1, spelling: 'apple', definition: 'fruit' }, { id: 2, spelling: 'pear', definition: 'fruit' }, { id: 3, spelling: 'long phrase', definition: 'long definition '.repeat(40) }]
    const group = { id: 7, type: '近义', note: 'original', members: entries.slice(0, 2) }
    let attempts = 0
    let deleted = 0
    let saved
    await page.route('**/api/**', async route => {
      const path = new URL(route.request().url()).pathname
      if (path === '/api/groups/7' && route.request().method() === 'DELETE') { deleted++; return route.fulfill({ json: { deleted: true } }) }
      if (path === '/api/groups/7' && route.request().method() === 'PUT') {
        saved = route.request().postDataJSON()
        if (++attempts === 1) return route.fulfill({ status: 409, json: { error: '关系组已存在', existingGroup: group } })
        return route.fulfill({ json: group })
      }
      if (path === '/api/group-types') return route.fulfill({ json: ['近义', '反义'] })
      if (path === '/api/stats') return route.fulfill({ json: { entries: 3, groups: 1 } })
      if (path === '/api/search') return route.fulfill({ json: { entries, groups: [] } })
      if (path === '/api/entries') return route.fulfill({ json: entries })
      if (path === '/api/groups') return route.fulfill({ json: [] })
      if (path === '/api/groups/7') return route.fulfill({ json: group })
      throw new Error(`Unexpected API: ${path}`)
    })
    await page.goto('http://127.0.0.1:5182/groups/7')
    const members = page.getByLabel('搜索并多选词条')
    const type = page.getByLabel('关系类型 必填')
    await members.waitFor()
    let a = await members.boundingBox(), b = await type.boundingBox()
    assert.ok(a.x < b.x, 'desktop member and relation tasks are side by side')
    await page.locator('#group-note').fill('changed')
    await page.getByText('long phrase', { exact: true }).click()
    for (const width of [320, 390, 651, 768, 1440]) {
      await page.setViewportSize({ width, height: 900 })
      a = await members.boundingBox(); b = await type.boundingBox()
      if (width === 320) assert.ok(a.y < b.y, 'mobile members precede relation fields')
      assert.equal(await page.locator('#group-note').inputValue(), 'changed')
      assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), `no overflow at ${width}`)
    }
    await page.getByRole('button', { name: '＋ 新增缺少的词条' }).focus()
    await page.keyboard.press('Tab')
    assert.equal(await page.evaluate(() => document.activeElement.id), 'group-type')
    await page.getByRole('button', { name: '＋ 新增缺少的词条' }).click()
    assert.match(await page.locator('.group-actions').innerText(), /完成或取消新增/)
    await page.getByRole('button', { name: '取消新增' }).click()
    await page.getByRole('button', { name: '保存关系组' }).click()
    await page.getByRole('link', { name: /编辑已有组/ }).waitFor()
    assert.equal(await page.locator('#group-note').inputValue(), 'changed')
    await page.getByRole('button', { name: '保存关系组' }).click()
    await page.waitForURL('**/relations')
    assert.deepEqual(saved, { type: '近义', note: 'changed', memberIds: [1, 2, 3] })
    await page.goto('http://127.0.0.1:5182/groups/7')
    await page.getByRole('button', { name: '取消编辑' }).click()
    await page.waitForURL('**/relations')
    await page.goto('http://127.0.0.1:5182/groups/7')
    page.once('dialog', dialog => dialog.dismiss())
    await page.getByRole('button', { name: '删除关系组' }).click()
    assert.equal(deleted, 0)
    await page.getByRole('button', { name: '取消选择 pear' }).click()
    page.once('dialog', dialog => dialog.accept())
    await page.getByRole('button', { name: '保存关系组' }).click()
    await page.waitForURL('**/relations')
    assert.equal(deleted, 1)
  } finally { await browser?.close(); server.kill() }
})



test('candidate list has an entry point and keyboard selection mode', async () => {
  const server = spawn(process.execPath, ['node_modules/vite/bin/vite.js', '--host', '127.0.0.1', '--port', '5183', '--strictPort'], { stdio: 'ignore' })
  let browser
  try {
    for (let i = 0; i < 80; i++) { try { if ((await fetch('http://127.0.0.1:5183')).ok) break } catch {} await new Promise(resolve => setTimeout(resolve, 100)) }
    browser = await chromium.launch({ executablePath: process.env.CHROME_PATH || undefined })
    const page = await browser.newPage()
    const entries = [{ id: 1, spelling: 'apple', definition: 'fruit' }, { id: 2, spelling: 'pear', definition: 'fruit' }]
    await page.route('**/api/**', async route => {
      const path = new URL(route.request().url()).pathname
      if (path === '/api/group-types') return route.fulfill({ json: ['近义'] })
      if (path === '/api/stats') return route.fulfill({ json: { entries: 2, groups: 0 } })
      if (path === '/api/entries') return route.fulfill({ json: entries })
      throw new Error(`Unexpected API: ${path}`)
    })
    await page.goto('http://127.0.0.1:5183/entries/new')
    const search = page.getByLabel('搜索并多选词条')
    const list = page.getByRole('list', { name: '可选词条' })
    const apple = page.getByRole('checkbox', { name: /apple/ })
    const pear = page.getByRole('checkbox', { name: /pear/ })
    await apple.waitFor()
    assert.equal(await page.getByText('尚未选择成员').isVisible(), true)
    const searchTop = await search.evaluate(el => el.getBoundingClientRect().top + window.scrollY)
    await search.focus()
    await page.keyboard.press('Tab')
    assert.equal(await list.evaluate(el => el === document.activeElement), true)
    await page.keyboard.press('Enter')
    assert.equal(await apple.evaluate(el => el === document.activeElement), true)
    const focusedRow = apple.locator('..')
    assert.notEqual(await focusedRow.evaluate(el => getComputedStyle(el).backgroundColor), 'rgba(0, 0, 0, 0)')
    await page.keyboard.press('Space')
    assert.equal(await apple.isChecked(), true)
    assert.equal(await page.getByText('尚未选择成员').count(), 0)
    assert.ok(Math.abs(await search.evaluate(el => el.getBoundingClientRect().top + window.scrollY) - searchTop) < 1, 'selecting the first member keeps the form in place')
    await page.keyboard.press('Space')
    assert.equal(await page.getByText('尚未选择成员').isVisible(), true)
    assert.ok(Math.abs(await search.evaluate(el => el.getBoundingClientRect().top + window.scrollY) - searchTop) < 1, 'removing the last member keeps the form in place')
    await page.keyboard.press('Space')
    await page.keyboard.press('Tab')
    assert.equal(await pear.evaluate(el => el === document.activeElement), true)
    await page.keyboard.press('Enter')
    assert.equal(await pear.isChecked(), true)
    await page.keyboard.press('Escape')
    assert.equal(await list.evaluate(el => el === document.activeElement), true)
    await page.keyboard.press('Space')
    await page.keyboard.press('Tab')
    await page.keyboard.press('Tab')
    assert.equal(await page.getByRole('button', { name: '收录缺少的词条' }).evaluate(el => el === document.activeElement), true)
    assert.equal(await apple.isChecked(), true)
    assert.equal(await pear.isChecked(), true)
    await search.focus()
    await page.keyboard.press('Tab')
    await page.keyboard.press('Tab')
    assert.equal(await page.getByRole('button', { name: '收录缺少的词条' }).evaluate(el => el === document.activeElement), true)
  } finally { await browser?.close(); server.kill() }
})
