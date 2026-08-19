#!/usr/bin/env node
// puppeteer_print.js <input.html> <output.pdf>
//
// Renders an HTML file to PDF through headless Chrome using puppeteer-core.
// This is the puppeteer side of the bench-external process comparison
// (same generated report fixture as the wkhtmltopdf and WeasyPrint runs).
// The puppeteer-core module lives in scripts/puppeteer/node_modules
// (npm ci --prefix scripts/puppeteer). The Chrome binary defaults to
// /usr/bin/google-chrome; override with PUPPETEER_EXECUTABLE_PATH.
//
// The script intentionally prints nothing on success so timing captures
// renderer work only.
'use strict';

const { resolve } = require('path');

const puppeteer = require(resolve(__dirname, 'puppeteer', 'node_modules', 'puppeteer-core'));

async function main() {
  const [htmlPath, pdfPath] = process.argv.slice(2);
  if (!htmlPath || !pdfPath) {
    console.error('usage: puppeteer_print.js <input.html> <output.pdf>');
    process.exit(2);
  }

  const executablePath = process.env.PUPPETEER_EXECUTABLE_PATH || '/usr/bin/google-chrome';

  const browser = await puppeteer.launch({
    executablePath,
    headless: true,
    args: [
      '--no-sandbox',
      '--disable-setuid-sandbox',
      '--disable-dev-shm-usage',
      '--disable-gpu',
      '--hide-scrollbars',
      '--force-device-scale-factor=1',
    ],
  });

  try {
    const page = await browser.newPage();
    await page.goto('file://' + resolve(htmlPath), { waitUntil: 'networkidle0' });
    await page.pdf({
      path: pdfPath,
      format: 'A4',
      printBackground: true,
      preferCSSPageSize: true,
    });
  } finally {
    await browser.close();
  }
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});