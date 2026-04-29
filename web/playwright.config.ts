import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
  testDir: './tests',
  timeout: 60_000,
  expect: { timeout: 15_000 },
  fullyParallel: false,
  retries: 1,
  reporter: [['list'], ['html', { open: 'never' }]],

  use: {
    baseURL: 'http://localhost:7334',
    screenshot: 'only-on-failure',
    trace: 'retain-on-failure',
  },

  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],

  // Start the Go binary before tests; serve from pre-built web/dist
  webServer: {
    command: './callblast --port 7334 --static web/dist',
    url: 'http://localhost:7334/api/health',
    reuseExistingServer: false,
    timeout: 10_000,
    cwd: '../',
  },
})
