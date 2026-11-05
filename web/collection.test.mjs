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

