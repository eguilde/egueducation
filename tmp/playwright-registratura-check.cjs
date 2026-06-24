const { chromium } = require('playwright-core');
const fs = require('fs/promises');
const path = require('path');

const OUTPUT_DIR = path.join('E:', 'dev', 'egueducation', 'outputs');
const APP_URL = 'http://localhost:4200/auth/start?returnUrl=%2Fdocumente';
const OTP_CODE = '123456';
const USERNAME = 'thomasgalambos';
const CHROME_PATH = 'C:/Program Files/Google/Chrome/Application/chrome.exe';

async function wait(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function saveJson(name, value) {
  await fs.mkdir(OUTPUT_DIR, { recursive: true });
  await fs.writeFile(path.join(OUTPUT_DIR, name), JSON.stringify(value, null, 2), 'utf8');
}

async function saveText(name, value) {
  await fs.mkdir(OUTPUT_DIR, { recursive: true });
  await fs.writeFile(path.join(OUTPUT_DIR, name), value, 'utf8');
}

async function run() {
  const browser = await chromium.launch({
    headless: true,
    executablePath: CHROME_PATH,
  });

  const page = await browser.newPage({
    viewport: { width: 1440, height: 960 },
    colorScheme: 'dark',
  });

  try {
    await page.goto(APP_URL, { waitUntil: 'domcontentloaded' });
    await page.waitForLoadState('networkidle');
    await page.screenshot({ path: path.join(OUTPUT_DIR, 'auth-step-1-initial.png'), fullPage: true });
    await saveText('auth-step-1-initial.html', await page.content());
    console.log('initial-url', page.url());

    if (!page.url().includes('/documente')) {
      const smsButton = page.getByRole('button', { name: 'SMS Cod OTP' });
      if (await smsButton.count()) {
        await smsButton.click();
        await page.waitForLoadState('domcontentloaded');
      }

      await page.screenshot({ path: path.join(OUTPUT_DIR, 'auth-step-1b-after-sms-click.png'), fullPage: true });
      await saveText('auth-step-1b-after-sms-click.html', await page.content());

      const identifier = page.locator('#identifier');
      if (await identifier.count()) {
        await identifier.fill(USERNAME);
        await page.screenshot({ path: path.join(OUTPUT_DIR, 'auth-step-2-identifier.png'), fullPage: true });
        await saveText('auth-step-2-identifier.html', await page.content());
        await page.getByRole('button', { name: 'Trimite codul prin SMS' }).click();
        await page.waitForLoadState('domcontentloaded');
        await wait(1000);
        await page.screenshot({ path: path.join(OUTPUT_DIR, 'auth-step-3-after-submit.png'), fullPage: true });
        await saveText('auth-step-3-after-submit.html', await page.content());
      }

      const otpBoxes = page.locator('.otp-box');
      await otpBoxes.first().waitFor({ state: 'visible', timeout: 30000 });
      const bannerText = (await page.locator('.info-banner').textContent()) ?? '';
      const runtimeCode = bannerText.match(/(\d{6})/)?.[1] ?? OTP_CODE;
      await otpBoxes.first().click();
      await page.keyboard.type(runtimeCode);
      await page.screenshot({ path: path.join(OUTPUT_DIR, 'auth-step-4-otp-filled.png'), fullPage: true });
      await saveText('auth-step-4-otp-filled.html', await page.content());
      await page.getByRole('button', { name: 'Verifica codul' }).click();
      await page.waitForLoadState('domcontentloaded');
      const consentButton = page.getByRole('button', { name: 'Accepta si continua' });
      if (await consentButton.count()) {
        await consentButton.click();
      }
      await page.waitForURL(/\/auth\/callback|\/documente/, { timeout: 30000 });
      await page.waitForLoadState('networkidle');
    }

    console.log('post-nav-url', page.url());

    if (!page.url().includes('/documente')) {
      await page.goto(APP_URL, { waitUntil: 'domcontentloaded' });
      await page.waitForLoadState('networkidle');
    }

    await page.waitForSelector('.registry-header', { timeout: 30000 });
    await wait(1200);

    const metrics = await page.evaluate(() => {
      const header = document.querySelector('.registry-header');
      const searchButton = header?.querySelector('button');
      const centerGroup = header?.children?.[1];
      const registryLabel = header?.querySelector('.registry-selector');
      const registrySelect = registryLabel?.querySelector('.p-select');
      const exportButton = header?.querySelector('button[icon=\"pi pi-file-pdf\"], .pi-file-pdf')?.closest('button');
      const drawer = document.querySelector('.p-drawer');
      const tableRoot = document.querySelector('.workspace-table');
      const paginator = tableRoot?.querySelector('.p-paginator');
      const datatable = tableRoot?.querySelector('.p-datatable');
      const grid = tableRoot?.querySelector('table');

      const rect = (el) => {
        if (!el) {
          return null;
        }
        const r = el.getBoundingClientRect();
        return {
          x: Math.round(r.x),
          y: Math.round(r.y),
          width: Math.round(r.width),
          height: Math.round(r.height),
        };
      };

      const centerTexts = centerGroup
        ? Array.from(centerGroup.querySelectorAll('button')).map((button) => button.textContent?.trim()).filter(Boolean)
        : [];

      return {
        url: window.location.href,
        filterPanelVisible: !!drawer && getComputedStyle(drawer).visibility !== 'hidden' && drawer.getBoundingClientRect().width > 0,
        headerColumns: header ? getComputedStyle(header).gridTemplateColumns : null,
        searchButtonRect: rect(searchButton),
        centerGroupRect: rect(centerGroup),
        centerTexts,
        registryLabelRect: rect(registryLabel),
        registrySelectRect: rect(registrySelect),
        exportButtonRect: rect(exportButton),
        tableRootRect: rect(tableRoot),
        datatableRect: rect(datatable),
        gridRect: rect(grid),
        paginatorRect: rect(paginator),
        viewport: {
          width: window.innerWidth,
          height: window.innerHeight,
        },
      };
    });

    await page.screenshot({ path: path.join(OUTPUT_DIR, 'registratura-documente-dark.png'), fullPage: true });
    await saveJson('registratura-documente-metrics.json', metrics);

    console.log(JSON.stringify(metrics, null, 2));
  } finally {
    await browser.close();
  }
}

run().catch((error) => {
  console.error(error);
  process.exit(1);
});
