#!/usr/bin/env node

const { spawnSync } = require('node:child_process')
const fs = require('node:fs')
const path = require('node:path')

function platformAssetName() {
  const p = process.platform
  const a = process.arch

  if (p === 'darwin' && a === 'arm64') return 'granola-darwin-arm64'
  if (p === 'darwin' && (a === 'x64' || a === 'amd64')) return 'granola-darwin-amd64'
  if (p === 'linux' && a === 'arm64') return 'granola-linux-arm64'
  if (p === 'linux' && (a === 'x64' || a === 'amd64')) return 'granola-linux-amd64'
  if (p === 'win32' && (a === 'x64' || a === 'amd64')) return 'granola-windows-amd64.exe'

  return null
}

const asset = platformAssetName()
if (!asset) {
  console.error(`Unsupported platform: ${process.platform}/${process.arch}`)
  process.exit(1)
}

const binaryPath = path.join(__dirname, '..', 'dist', asset)
if (!fs.existsSync(binaryPath)) {
  console.error(`Granola binary not found for ${process.platform}/${process.arch}`)
  console.error(`Expected: ${binaryPath}`)
  process.exit(1)
}

const result = spawnSync(binaryPath, process.argv.slice(2), {
  stdio: 'inherit',
  windowsHide: false,
})

if (result.error) {
  console.error(result.error.message)
  process.exit(1)
}

process.exit(result.status ?? 0)
