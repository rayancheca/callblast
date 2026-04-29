import { test, expect, Page } from '@playwright/test'
import path from 'path'
import { fileURLToPath } from 'url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const REPO_PATH = path.resolve(__dirname, '../../')
// Use commits that have real Go code changes (demo mode commit)
const BASE = 'HEAD~4'
const HEAD = 'HEAD~3'

async function waitForGraph(page: Page) {
  await page.waitForSelector('svg.graph-svg', { timeout: 45_000 })
  await page.waitForSelector('.node-circle', { timeout: 45_000 })
}

async function runAnalysis(page: Page) {
  await page.fill('#repoPath', REPO_PATH)
  await page.fill('#baseBranch', BASE)
  await page.fill('#headBranch', HEAD)
  await page.click('.form-submit')
}

test.describe('CallBlast UI', () => {
  test('landing page loads with correct title', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('h1')).toContainText('Analyze Blast Radius')
    await expect(page.locator('.form-card')).toBeVisible()
  })

  test('form has all required inputs', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('#repoPath')).toBeVisible()
    await expect(page.locator('#baseBranch')).toBeVisible()
    await expect(page.locator('#headBranch')).toBeVisible()
    await expect(page.locator('.demo-btn')).toBeVisible()
    await expect(page.locator('#prURL')).toBeVisible()
  })

  test('submit is disabled when head branch is empty', async ({ page }) => {
    await page.goto('/')
    await page.fill('#headBranch', '')
    await expect(page.locator('.form-submit')).toBeDisabled()
    await page.fill('#headBranch', 'feat/test')
    await expect(page.locator('.form-submit')).toBeEnabled()
  })

  test('demo button fills all three fields', async ({ page }) => {
    await page.goto('/')
    await page.click('.demo-btn')
    await expect(page.locator('#repoPath')).not.toHaveValue('', { timeout: 8_000 })
    await expect(page.locator('#baseBranch')).toHaveValue('HEAD~1')
    await expect(page.locator('#headBranch')).toHaveValue('HEAD')
  })

  test('GitHub PR import shows error for invalid URL', async ({ page }) => {
    await page.goto('/')
    await page.fill('#prURL', 'https://not-github.com/foo')
    await page.click('.pr-import-btn')
    await expect(page.locator('.pr-error')).toBeVisible({ timeout: 10_000 })
  })

  test('health endpoint is reachable', async ({ request }) => {
    const res = await request.get('/api/health')
    expect(res.ok()).toBeTruthy()
    const body = await res.json()
    expect(body.status).toBe('ok')
  })

  test('demo endpoint returns expected fields', async ({ request }) => {
    const res = await request.get('/api/demo')
    expect(res.ok()).toBeTruthy()
    const body = await res.json()
    expect(body.repoPath).toBeTruthy()
    expect(body.baseBranch).toBe('HEAD~1')
    expect(body.headBranch).toBe('HEAD')
  })

  test('full analysis run — form submission to graph render', async ({ page }) => {
    await page.goto('/')

    await page.fill('#repoPath', REPO_PATH)
    await page.fill('#baseBranch', BASE)
    await page.fill('#headBranch', HEAD)
    await page.screenshot({ path: '../docs/screenshots/01-form-filled.png' })

    await page.click('.form-submit')
    // Progress bar may flash by too fast to assert reliably, but take a screenshot
    // immediately after submit to capture either the running or completed state.
    await page.screenshot({ path: '../docs/screenshots/02-analysis-running.png' })

    await waitForGraph(page)
    await page.screenshot({ path: '../docs/screenshots/03-graph-rendered.png' })

    // Header should show non-zero stats after completion
    await expect(page.locator('.header-stat-value').first()).not.toHaveText('0', { timeout: 10_000 })
  })

  test('node detail panel opens on click', async ({ page }) => {
    await page.goto('/')
    await runAnalysis(page)
    await waitForGraph(page)

    const firstNode = page.locator('.node-circle').first()
    await firstNode.click()
    await expect(page.locator('.detail-panel')).toBeVisible({ timeout: 10_000 })
    await page.screenshot({ path: '../docs/screenshots/04-node-detail.png' })
  })

  test('impact list shows affected files', async ({ page }) => {
    await page.goto('/')
    await runAnalysis(page)
    await waitForGraph(page)

    await expect(page.locator('.impact-list')).toBeVisible({ timeout: 10_000 })
    await page.screenshot({ path: '../docs/screenshots/05-impact-list.png' })
  })

  test('reset returns to form', async ({ page }) => {
    await page.goto('/')
    await runAnalysis(page)
    await waitForGraph(page)

    await page.click('.header-reset')
    await expect(page.locator('.form-card')).toBeVisible({ timeout: 5_000 })
    await page.screenshot({ path: '../docs/screenshots/06-reset-to-form.png' })
  })
})
