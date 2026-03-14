#!/usr/bin/env node
const fs = require('fs');
const path = require('path');

const binDir = path.join(__dirname, '..', '..', 'bin');
const exeName = process.platform === 'win32' ? 'granola.exe' : 'granola';
const exePath = path.join(binDir, exeName);

if (!fs.existsSync(exePath)) {
  console.error(`Binary not found at ${exePath}`);
  console.error('Please run: npm run build');
  process.exit(1);
}

const packageRoot = path.dirname(path.dirname(__dirname));
const npmBinDir = path.join(packageRoot, '..', '.bin');

if (!fs.existsSync(npmBinDir)) {
  fs.mkdirSync(npmBinDir, { recursive: true });
}

const linkPath = path.join(npmBinDir, exeName);

if (fs.existsSync(linkPath)) {
  fs.unlinkSync(linkPath);
}

if (process.platform === 'win32') {
  fs.writeFileSync(linkPath, `@echo off\n"%~dp0\\..\\bin\\granola.exe" %*\n`);
} else {
  fs.symlinkSync(exePath, linkPath);
}

console.log(`Created symlink: ${linkPath}`);