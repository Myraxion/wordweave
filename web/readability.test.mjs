import { test } from 'node:test'
import assert from 'node:assert/strict'
import { spawn } from 'node:child_process'
import { chromium } from 'playwright'

const definition = '这是很长的释义，需要在列表中截断，同时在详情和关系组中完整显示。'.repeat(8)
const entries = [
  { id: 1, spelling: 'a very long phrase for comparison', definition, createdAt: '2026-11-04T12:00:00Z', updatedAt: null },
  { id: 2, spelling: 'short', definition: '简短释义', createdAt: '2026-11-04T12:00:00Z', updatedAt: null },
]
const groups = [{ id: 1, type: '近义', note: 'hit', createdAt: entries[0].createdAt, updatedAt: null, members: entries }]

test('list summaries and relation members remain readable across screen sizes', async () => {
  const server = spawn(process.execPath, ['node_modules/vite/bin/vite.js', '--host', '127.0.0.1', '--port', '5180', '--strictPort'], { stdio: 'ignore' })
  let browser
  try {
    for (let i = 0; i < 80; i++) {
      try { if ((await fetch('http://127.0.0.1:5180')).ok) break } catch {}
      await new Promise(resolve => setTimeout(resolve, 100))
    }
    browser = await chromium.launch({ executablePath: process.env.CHROME_PATH || undefined, headless: true })
    const page = await browser.newPage({ timezoneId: 'Asia/Shanghai' })
    await page.route('**/api/**', async route => {
      const path = new URL(route.request().url()).pathname
      let body
      if (path === '/api/stats') body = { entries: 2, groups: 1 }
      else if (path === '/api/group-types') body = ['近义']
      else if (path === '/api/groups' || path === '/api/entries/1/groups') body = groups
      else if (path === '/api/entries/1') body = entries[0]
      else if (path === '/api/search') body = { entries, groups }
      else throw new Error(`Unexpected API: ${path}`)
      await route.fulfill({ json: body })
    })
    await page.goto('http://127.0.0.1:5180/')
    const summary = page.locator('.compact-entry-row').first().locator('p')
    await summary.waitFor()
    const dimensions = await summary.evaluate(el => ({ height: el.clientHeight, line: parseFloat(getComputedStyle(el).lineHeight), scroll: el.scrollHeight }))
    assert.ok(dimensions.height <= dimensions.line * 2 + 1 && dimensions.scroll > dimensions.height, 'long list definition is limited to two lines')
    const shortSummary = page.locator('.compact-entry-row').nth(1).locator('p')
    assert.equal(await shortSummary.innerText(), '简短释义')
    assert.ok(await shortSummary.evaluate(el => el.scrollHeight <= el.clientHeight), 'short definition stays visible')
    await page.getByRole('button', { name: '遮挡释义' }).click()
    const mask = summary.locator('button')
    await mask.focus()
    assert.equal(await mask.getAttribute('aria-pressed'), 'false')
    assert.equal(await mask.getAttribute('aria-describedby'), null)
    await page.keyboard.press('Enter')
    assert.equal(await mask.getAttribute('aria-pressed'), 'true')
    const descriptionId = await mask.getAttribute('aria-describedby')
    assert.ok(descriptionId, 'revealed answer describes the control')
    const answer = page.locator(`[id="${descriptionId}"]`)
    assert.equal(await answer.innerText(), definition)
    await page.keyboard.press('Tab')
    assert.equal(await answer.evaluate(el => el === document.activeElement), true)
    await page.keyboard.press('ArrowDown')
    await page.waitForFunction(id => document.getElementById(id)?.scrollTop > 0, descriptionId)
    await page.keyboard.press('Shift+Tab')
    await page.keyboard.press('Space')
    assert.equal(await mask.getAttribute('aria-describedby'), null)
    assert.equal(await mask.evaluate(el => el === document.activeElement), true)
    const maskHeight = await mask.evaluate(el => el.getBoundingClientRect().height)
    await summary.locator('.reveal-text').hover({ force: true })
    await summary.locator('.study-content').click()
    assert.equal(await mask.evaluate(el => el.getBoundingClientRect().height), maskHeight)
    assert.ok(maskHeight > dimensions.height, 'answer region keeps its fixed three-line height')
    await page.getByRole('button', { name: '遮挡拼写' }).click()
    const spellingMask = page.locator('.compact-entry-row').first().locator('h3 button')
    await spellingMask.focus()
    assert.equal(await spellingMask.evaluate(el => el === document.activeElement), true)
    assert.equal(await spellingMask.getAttribute('aria-describedby'), null)
    await page.keyboard.press('Space')
    assert.equal(await spellingMask.getAttribute('aria-pressed'), 'true')
    const spellingId = await spellingMask.getAttribute('aria-describedby')
    assert.equal(await page.locator(`[id="${spellingId}"]`).innerText(), entries[0].spelling)
    assert.equal(await page.getByRole('link', { name: '查看词条详情' }).count(), 2)
    await page.getByRole('button', { name: '遮挡释义' }).click()
    await page.getByRole('button', { name: '遮挡拼写' }).click()
    assert.equal(await page.getByRole('button', { name: '显示拼写' }).first().getAttribute('aria-describedby'), null)
    await page.getByRole('button', { name: '完整模式' }).click()
    for (const width of [320, 390, 768, 1440]) {
      await page.setViewportSize({ width, height: 900 })
      for (const path of ['/relations', '/?q=hit', '/entries/1']) {
        await page.goto(`http://127.0.0.1:5180${path}`)
        const card = page.locator('.relation-card').first()
        await card.waitFor()
        const member = card.locator('.group-members li').first()
        const spelling = await member.locator('.member-heading').boundingBox()
        const meaning = await member.locator('.definition').boundingBox()
        if (width <= 390) assert.ok(meaning.y >= spelling.y + spelling.height - 1, `${path} stacks member content at ${width}px`)
        if (width === 1440) assert.ok(meaning.x >= spelling.x + spelling.width - 1, `${path} compares member content at ${width}px`)
        assert.equal(await member.locator('.definition').innerText(), definition)
        assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), `${path} fits ${width}px`)
        assert.equal(await card.locator('.group-members li').count(), 2)
        assert.ok(await card.locator('.group-members li').nth(1).evaluate(el => parseFloat(getComputedStyle(el).borderTopWidth) > 0), 'member separator remains visible')
        if (path === '/entries/1') {
          assert.equal(await card.getByText('当前').count(), 1)
          assert.equal(await page.locator('.detail-card .definition').innerText(), definition)
        }
      }
    }
    await page.goto('http://127.0.0.1:5180/?q=hit')
    assert.ok(await page.locator('.timeline-date').count() > 0, 'grid retains date groups')
    await page.getByRole('button', { name: '瀑布流' }).click()
    assert.equal(await page.locator('.timeline-date').count(), 0)
    assert.ok(await page.locator('.relation-card').count() > 0, 'masonry retains groups')
  } finally {
    await browser?.close()
    server.kill()
  }
})
