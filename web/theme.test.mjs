import { test } from 'node:test'
import assert from 'node:assert/strict'
import { spawn } from 'node:child_process'
import { chromium } from 'playwright'

const base = 'http://127.0.0.1:5188'
const entries = [
  { id: 1, spelling: 'affect', definition: '影响', createdAt: '2026-11-04T12:00:00Z', updatedAt: null },
  { id: 2, spelling: 'effect', definition: '结果', createdAt: '2026-11-04T12:00:00Z', updatedAt: null },
]
const groups = [{ id: 1, type: '形音相似', note: '注意用法区别', createdAt: entries[0].createdAt, updatedAt: null, members: entries }]

test('appearance supports system and manual preferences throughout the app', async t => {
  const server = spawn(process.execPath, ['node_modules/vite/bin/vite.js', '--host', '127.0.0.1', '--port', '5188', '--strictPort'], { stdio: 'ignore' })
  let browser
  try {
    for (let i = 0; i < 80; i++) {
      try { if ((await fetch(base)).ok) break } catch {}
      await new Promise(resolve => setTimeout(resolve, 100))
    }
    browser = await chromium.launch({ executablePath: process.env.CHROME_PATH || undefined, headless: true })
    const page = await browser.newPage({ colorScheme: 'dark' })
    await page.route('**/api/**', async route => {
      const path = new URL(route.request().url()).pathname
      let body
      if (path === '/api/stats') body = { entries: 2, groups: 1 }
      else if (path === '/api/group-types') body = ['形音相似']
      else if (path === '/api/groups' || path === '/api/entries/1/groups') body = groups
      else if (path === '/api/groups/1') body = groups[0]
      else if (path === '/api/entries/1') body = entries[0]
      else if (path === '/api/entries') body = entries
      else if (path === '/api/search') body = { entries, groups: new URL(route.request().url()).searchParams.get('q') ? groups : [] }
      else throw new Error(`Unexpected API: ${path}`)
      await route.fulfill({ json: body })
    })
    const scheme = () => page.evaluate(() => getComputedStyle(document.documentElement).colorScheme)
    const button = () => page.getByRole('button', { name: /^外观：当前/ })
    await page.goto(base)
    await button().waitFor({ timeout: 3000 })
    assert.equal(await scheme(), 'dark', 'first visit follows a dark system')
    assert.equal(await button().getAttribute('aria-label'), '外观：当前跟随系统；点击切换为浅色')
    await page.emulateMedia({ colorScheme: 'light' })
    await page.waitForFunction(() => getComputedStyle(document.documentElement).colorScheme === 'light')
    await button().focus()
    assert.equal(await page.getByRole('tooltip').innerText(), '当前：跟随系统；点击切换为浅色')
    await page.keyboard.press('Enter')
    assert.equal(await button().getAttribute('aria-label'), '外观：当前浅色；点击切换为深色')
    await page.emulateMedia({ colorScheme: 'dark' })
    assert.equal(await scheme(), 'light', 'manual light ignores a dark system')
    await page.reload()
    assert.equal(await scheme(), 'light', 'manual light survives a reload')
    await button().focus()
    await page.keyboard.press('Space')
    assert.equal(await scheme(), 'dark')
    assert.equal(await button().getAttribute('aria-label'), '外观：当前深色；点击切换为跟随系统')
    await page.emulateMedia({ colorScheme: 'light' })
    await page.reload()
    assert.equal(await scheme(), 'dark', 'manual dark survives a reload and system changes')
    await button().click()
    assert.equal(await scheme(), 'light', 'third click returns to the current system appearance')
    await page.emulateMedia({ colorScheme: 'dark' })
    await page.waitForFunction(() => getComputedStyle(document.documentElement).colorScheme === 'dark')
    await page.reload()
    assert.equal(await button().getAttribute('aria-label'), '外观：当前跟随系统；点击切换为浅色')

    await t.test('both palettes keep text and controls readable on desktop and narrow screens', async () => {
      for (const colorScheme of ['light', 'dark']) {
        await page.emulateMedia({ colorScheme })
        for (const width of [320, 768, 1440]) {
          await page.setViewportSize({ width, height: 900 })
          for (const path of ['/', '/relations', '/?q=affect', '/entries/1', '/entries/new', '/groups/1']) {
            await page.goto(base + path)
            await page.locator('main .loading').waitFor({ state: 'hidden' })
            await button().waitFor()
            assert.deepEqual(await page.evaluate(readability), [], `${colorScheme} ${width}px ${path}`)
            const bounds = await button().boundingBox()
            assert.ok(bounds && bounds.x >= 0 && bounds.x + bounds.width <= width, 'appearance button stays on screen')
            assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), 'page has no horizontal overflow')
          }
          await page.goto(base)
          await page.getByRole('button', { name: /遮挡释义/ }).click()
          assert.deepEqual(await page.evaluate(readability), [], `${colorScheme} masked definition at ${width}px`)
        }
      }
    })

    await t.test('saved appearance is applied before the application module loads', async () => {
      await button().click()
      await button().click()
      await page.emulateMedia({ colorScheme: 'light' })
      await page.route('**/src/main.ts', route => route.abort())
      await page.reload()
      assert.equal(await scheme(), 'dark', 'saved dark appearance reaches the browser before app startup')
      await page.unroute('**/src/main.ts')
      await page.reload()
      await button().click()
      await page.emulateMedia({ colorScheme: 'dark' })
      await page.route('**/src/main.ts', route => route.abort())
      await page.reload()
      assert.equal(await scheme(), 'dark', 'system dark appearance also reaches the browser before app startup')
      await page.unroute('**/src/main.ts')
    })

    await t.test('empty and error states remain readable in both appearances', async () => {
      for (const colorScheme of ['light', 'dark']) {
        await page.emulateMedia({ colorScheme })
        await page.route('**/api/search?**', route => route.fulfill({ json: { entries: [], groups: [] } }))
        await page.goto(base)
        await page.getByRole('heading', { name: '从第一个新词开始' }).waitFor()
        assert.deepEqual(await page.evaluate(readability), [], `${colorScheme} empty state`)
        await page.unroute('**/api/search?**')
        await page.route('**/api/search?**', route => route.fulfill({ status: 500, json: { error: '暂时无法读取词条' } }))
        await page.reload()
        await page.getByRole('alert').waitFor()
        assert.deepEqual(await page.evaluate(readability), [], `${colorScheme} error state`)
        await page.unroute('**/api/search?**')
      }
    })

    await t.test('appearance hint can be hovered and dismissed with Escape', async () => {
      await page.goto(base)
      await button().hover()
      const tip = page.getByRole('tooltip')
      assert.equal(await tip.isVisible(), true)
      const triggerBounds = await button().boundingBox()
      await page.mouse.move(triggerBounds.x + triggerBounds.width / 2, triggerBounds.y + triggerBounds.height + 4)
      assert.equal(await tip.isVisible(), true, 'hint remains visible across the gap below its trigger')
      await tip.hover()
      assert.equal(await tip.isVisible(), true)
      await button().focus()
      await page.keyboard.press('Escape')
      assert.equal(await tip.isVisible(), false, 'Escape dismisses the hint without moving focus')
      await button().blur()
      await button().focus()
      assert.equal(await tip.isVisible(), true, 'focusing the button again restores the hint')
    })
  } finally {
    await browser?.close()
    server.kill()
  }
})

function readability() {
  const rgb = value => (value.match(/[\d.]+/g) || []).map(Number)
  const luminance = color => color.slice(0, 3).map(v => {
    v /= 255
    return v <= .04045 ? v / 12.92 : ((v + .055) / 1.055) ** 2.4
  }).reduce((sum, v, i) => sum + v * [.2126, .7152, .0722][i], 0)
  const contrast = (a, b) => (Math.max(luminance(a), luminance(b)) + .05) / (Math.min(luminance(a), luminance(b)) + .05)
  function background(el) {
    for (let current = el; current; current = current.parentElement) {
      const color = rgb(getComputedStyle(current).backgroundColor)
      if (color.length === 3 || color[3] === 1) return color
    }
    return [255, 255, 255]
  }
  const failures = []
  for (const el of document.querySelectorAll('html, body *')) {
    const style = getComputedStyle(el)
    if (!el.getClientRects().length || style.visibility === 'hidden' || el.closest(':disabled, [aria-hidden="true"]')) continue
    if (getComputedStyle(document.documentElement).colorScheme === 'dark' && el.matches('html, .sidebar, .topbar, .form-card, .detail-card, .relation-card, .compact-entry-row, input, textarea, select, .reveal-text')) {
      if (luminance(background(el)) > .15) failures.push(`bright dark-mode surface ${el.className}`)
    }
    const text = [...el.childNodes].filter(node => node.nodeType === 3).map(node => node.textContent.trim()).join(' ')
    if (text && contrast(rgb(style.color), background(el)) < 4.5) failures.push(`text ${el.className}: ${text.slice(0, 30)}`)
    if (el.matches('.global-search, input, textarea, select, .display-modes, .reveal-text.concealed') && parseFloat(style.borderTopWidth) > 0) {
      if (contrast(rgb(style.borderTopColor), background(el)) < 3) failures.push(`control border ${el.className}`)
    }
  }
  return [...new Set(failures)]
}
