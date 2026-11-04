import { test } from 'node:test'
import assert from 'node:assert/strict'
import { spawn } from 'node:child_process'
import { chromium } from 'playwright'

const entries = [
  { id: 1, spelling: 'alpha', definition: 'one', createdAt: '2026-11-01T12:00:00Z', updatedAt: '2026-11-04T12:00:00Z' },
  { id: 2, spelling: 'beta', definition: 'two', createdAt: '2026-11-02T12:00:00Z', updatedAt: null },
]
const groups = [
  { id: 1, type: '近义', note: 'hit', createdAt: '2026-11-01T12:00:00Z', updatedAt: '2026-11-02T12:00:00Z', members: entries },
  { id: 2, type: '反义', note: 'hit', createdAt: '2026-11-02T12:00:00Z', updatedAt: '2026-11-04T12:00:00Z', members: entries },
]

test('browse pages group results by selected local date and retain other layouts', async () => {
  const server = spawn(process.execPath, ['node_modules/vite/bin/vite.js', '--host', '127.0.0.1', '--port', '5179', '--strictPort'], { stdio: 'ignore' })
  let browser
  try {
    for (let i = 0; i < 80; i++) {
      try { if ((await fetch('http://127.0.0.1:5179')).ok) break } catch {}
      await new Promise(resolve => setTimeout(resolve, 100))
    }
    browser = await chromium.launch({ executablePath: process.env.CHROME_PATH || undefined, headless: true })
    const page = await browser.newPage({ timezoneId: 'Asia/Shanghai' })
    await page.route('**/api/**', async route => {
      const url = new URL(route.request().url())
      let body
      if (url.pathname === '/api/stats') body = { entries: 2, groups: 2 }
      else if (url.pathname === '/api/group-types') body = ['近义', '反义']
      else if (url.pathname === '/api/groups') body = [...groups].reverse()
      else if (url.pathname === '/api/search') {
        const sorted = url.searchParams.get('sort')?.startsWith('created') ? [...entries].reverse() : entries
        body = { entries: sorted, groups: url.searchParams.get('q') ? groups : [] }
      } else throw new Error(`Unexpected API: ${url}`)
      await route.fulfill({ json: body })
    })
    await page.goto('http://127.0.0.1:5179/')
    await page.getByText('2026年11月4日').waitFor()
    assert.deepEqual(await page.locator('.timeline-date').allTextContents(), ['2026年11月4日', '2026年11月2日'])
    assert.equal(await page.locator('.compact-entry-row time').count(), 0)
    await page.selectOption('#sort-select', 'created_desc')
    await page.getByText('2026年11月1日').waitFor()
    assert.deepEqual(await page.locator('.timeline-date').allTextContents(), ['2026年11月2日', '2026年11月1日'])
    await page.selectOption('#sort-select', 'alpha_asc')
    await page.locator('.compact-entry-row time').first().waitFor()
    assert.equal(await page.locator('.timeline-date').count(), 0)
    await page.goto('http://127.0.0.1:5179/?q=hit')
    await page.getByLabel('说明命中的关系组').getByText('2026年11月4日').waitFor()
    assert.deepEqual(await page.getByLabel('说明命中的关系组').locator('.timeline-date').allTextContents(), ['2026年11月4日', '2026年11月2日'])
    assert.equal(await page.locator('.group-modified-date').count(), 0)
    await page.getByRole('button', { name: '瀑布流' }).click()
    await page.locator('.group-modified-date').first().waitFor()
    assert.equal(await page.locator('.timeline-date').count(), 0)
    await page.setViewportSize({ width: 390, height: 844 })
    await page.goto('http://127.0.0.1:5179/relations')
    await page.getByText('2026年11月4日').waitFor()
    const date = await page.locator('.timeline-date').first().boundingBox()
    const card = await page.locator('.relation-card').first().boundingBox()
    assert.ok(date.y < card.y, 'mobile date is above the card')
  } finally {
    await browser?.close()
    server.kill()
  }
})
