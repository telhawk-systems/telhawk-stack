import { test, expect } from '@playwright/test';

test.describe('Search', () => {
  // Login before each test
  test.beforeEach(async ({ page }) => {
    await page.goto('/login');
    await page.locator('input[type="text"]').fill('admin');
    await page.locator('input[type="password"]').fill('admin123');
    await page.getByRole('button', { name: 'Login' }).click();

    // Wait for redirect to dashboard
    await expect(page).toHaveURL('/', { timeout: 10000 });
  });

  test('should navigate to events page', async ({ page }) => {
    // Navigate to events page via menu or direct URL
    await page.goto('/events');

    await expect(page).toHaveURL('/events');
    // Look for search-related elements
    await expect(page.getByText('Time Range')).toBeVisible({ timeout: 10000 });
  });

  test('should execute a search query', async ({ page }) => {
    await page.goto('/events');

    // Wait for the search console to load
    await expect(page.getByText('Time Range')).toBeVisible({ timeout: 10000 });

    // Select a time range (24h should be default or click it)
    await page.getByRole('button', { name: '24H' }).click();

    // Click the Search button
    await page.getByRole('button', { name: 'Search' }).click();

    // Wait for loading to complete (button text changes)
    await expect(page.getByRole('button', { name: 'Searching...' })).toBeVisible({ timeout: 5000 }).catch(() => {});
    await expect(page.getByRole('button', { name: 'Search' })).toBeVisible({ timeout: 30000 });
  });

  test('should change time range', async ({ page }) => {
    await page.goto('/events');

    await expect(page.getByText('Time Range')).toBeVisible({ timeout: 10000 });

    // Click different time ranges
    await page.getByRole('button', { name: '15M' }).click();
    await page.getByRole('button', { name: '1H' }).click();
    await page.getByRole('button', { name: '7D' }).click();

    // Verify the 7D button has the active styling
    const sevenDayButton = page.getByRole('button', { name: '7D' });
    await expect(sevenDayButton).toHaveClass(/bg-blue-600/);
  });

  test('should show/hide JSON query', async ({ page }) => {
    await page.goto('/events');

    await expect(page.getByText('Time Range')).toBeVisible({ timeout: 10000 });

    // Toggle JSON query visibility
    const toggleButton = page.getByRole('button', { name: /Show JSON Query/i });
    if (await toggleButton.isVisible()) {
      await toggleButton.click();
      await expect(page.getByText('Generated JSON')).toBeVisible();

      // Hide it again
      await page.getByRole('button', { name: /Hide JSON Query/i }).click();
      await expect(page.getByText('Generated JSON')).not.toBeVisible();
    }
  });
});
