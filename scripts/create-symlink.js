#!/usr/bin/env node
const fs = require('fs');
const path = require('path');

const binDir = path.join(__dirname, '..', 'bin');
const exeName = process.platform === 'win32' ? 'granola.exe' : 'granola';
const exePath = path.join(binDir, exeName);

// Check if binary exists
if (!fs.existsSync(exePath)) {
  console.error(`Binary not found at ${exePath}`);
  console.error('Please run: npm run build');
  process.exit(1);
}

// Get the .bin directory (where npm puts executables)
const packageDir = path.dirname(__dirname);

// Try to find .bin directory
let npmBinDir;

// If installed globally
if (process.env.npm_config_prefix) {
  npmBinDir = path.join(process.env.npm_config_prefix, 'lib', 'node_modules', '.bin');
} else {
  // Local install - check node_modules/.bin
  const parentDir = path.dirname(packageDir);
  if (path.basename(parentDir) === 'node_modules') {
    npmBinDir = path.join(parentDir, '.bin');
  } else {
    npmBinDir = packageDir;
  }
}

// Create .bin directory if it doesn't exist
if (!fs.existsSync(npmBinDir)) {
  try {
    fs.mkdirSync(npmBinDir, { recursive: true });
  } catch (err) {
    console.error(`Failed to create .bin directory: ${err.message}`);
    process.exit(1);
  }
}

const linkPath = path.join(npmBinDir, exeName);

// Remove existing symlink if it exists
if (fs.existsSync(linkPath)) {
  try {
    fs.unlinkSync(linkPath);
  } catch (err) {
    console.error(`Failed to remove existing link: ${err.message}`);
    process.exit(1);
  }
}

// Create symlink
try {
  if (process.platform === 'win32') {
    // On Windows, create a batch file
    fs.writeFileSync(
      linkPath,
      `@echo off\n"%~dp0\\..\\bin\\granola.exe" %*\n`
    );
  } else {
    // On Unix, create a symlink
    fs.symlinkSync(exePath, linkPath);
  }
  console.log(`Created symlink: ${linkPath}`);
} catch (err) {
  console.error(`Failed to create symlink: ${err.message}`);
  process.exit(1);
}