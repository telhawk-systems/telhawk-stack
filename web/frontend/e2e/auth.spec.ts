import { test, expect } from '@playwright/test';

test.describe('Authentication', () => {
  test.beforeEach(async ({ page }) => {
    // Start from login page
    await page.goto('/login');
  });

  test('should display login page', async ({ page }) => {
    await expect(page.getByRole('heading', { name: 'TelHawk' })).toBeVisible();
    await expect(page.locator('input[type="text"]')).toBeVisible();
    await expect(page.locator('input[type="password"]')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Login' })).toBeVisible();
  });

  test('should show error with invalid credentials', async ({ page }) => {
    await page.locator('input[type="text"]').fill('wronguser');
    await page.locator('input[type="password"]').fill('wrongpassword');
    await page.getByRole('button', { name: 'Login' }).click();

    // Wait for error message
    await expect(page.getByText('Invalid username or password')).toBeVisible({ timeout: 10000 });
  });

  test('should login successfully with valid credentials', async ({ page }) => {
    await page.locator('input[type="text"]').fill('admin');
    await page.locator('input[type="password"]').fill('admin123');
    await page.getByRole('button', { name: 'Login' }).click();

    // Should redirect to dashboard (home page)
    await expect(page).toHaveURL('/', { timeout: 10000 });
  });

  test('should redirect unauthenticated users to login', async ({ page }) => {
    // Try to access protected route directly
    await page.goto('/events');

    // Should be redirected to login
    await expect(page).toHaveURL('/login');
  });
});
