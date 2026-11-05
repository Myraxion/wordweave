import { test } from 'node:test'
import assert from 'node:assert/strict'
import { spawn } from 'node:child_process'
import { chromium } from 'playwright'

test('发音播放支持切换并在遮挡拼写时隐藏', async () => {
  const server = spawn(process.execPath, ['node_modules/vite/bin/vite.js', '--host', '127.0.0.1', '--port', '5191', '--strictPort'], { stdio: 'ignore' })
  let browser
  try {
    for (let i = 0; i < 80; i++) {
      try { if ((await fetch('http://127.0.0.1:5191')).ok) break } catch {}
      await new Promise(resolve => setTimeout(resolve, 100))
    }
    browser = await chromium.launch({ executablePath: process.env.CHROME_PATH || undefined, headless: true })
    const page = await browser.newPage()
    await page.addInitScript(() => {
      window.speechEvents = []
      window.SpeechSynthesisUtterance = class { constructor(text) { this.text = text } }
      Object.defineProperty(window, 'speechSynthesis', { value: {
        getVoices: () => [{ lang: 'en-US' }],
        cancel: () => window.speechEvents.push('cancel'),
        speak: utterance => window.speechEvents.push(utterance.text),
      } })
    })
    await page.route('**/api/**', route => {
      const path = new URL(route.request().url()).pathname
      const entries = ['quiet', 'quilt'].map((spelling, index) => ({ id: index + 1, spelling, definition: '', createdAt: '2026-11-04T12:00:00Z', updatedAt: null }))
      return route.fulfill({ json: path === '/api/stats' ? { entries: 2, groups: 0 } : { entries, groups: [] } })
    })
    await page.goto('http://127.0.0.1:5191/')
    await page.getByRole('button', { name: '播放 quiet 的发音' }).click()
    await page.getByRole('button', { name: '播放 quilt 的发音' }).click()
    assert.deepEqual((await page.evaluate(() => window.speechEvents)).slice(-4), ['cancel', 'quiet', 'cancel', 'quilt'])
    await page.getByRole('button', { name: '遮挡拼写' }).click()
    assert.equal(await page.locator('.pronunciation-button').count(), 0)
  } finally {
    await browser?.close()
    server.kill()
  }
})
