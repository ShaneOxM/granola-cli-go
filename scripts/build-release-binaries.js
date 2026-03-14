#!/usr/bin/env node

const { execFileSync } = require('node:child_process')
const fs = require('node:fs')
const path = require('node:path')

const root = path.resolve(__dirname, '..')
const dist = path.join(root, 'dist')

const targets = [
  { goos: 'darwin', goarch: 'arm64', output: 'granola-darwin-arm64' },
  { goos: 'darwin', goarch: 'amd64', output: 'granola-darwin-amd64' },
  { goos: 'linux', goarch: 'arm64', output: 'granola-linux-arm64' },
  { goos: 'linux', goarch: 'amd64', output: 'granola-linux-amd64' },
  { goos: 'windows', goarch: 'amd64', output: 'granola-windows-amd64.exe' },
]

fs.rmSync(dist, { recursive: true, force: true })
fs.mkdirSync(dist, { recursive: true })

for (const target of targets) {
  const env = {
    ...process.env,
    CGO_ENABLED: '0',
    GOOS: target.goos,
    GOARCH: target.goarch,
  }
  const out = path.join(dist, target.output)
  console.log(`Building ${target.output}...`)
  execFileSync('go', ['build', '-trimpath', '-ldflags=-s -w', '-o', out, './cmd/granola'], {
    cwd: root,
    env,
    stdio: 'inherit',
  })
}

console.log('Release binaries built in dist/')
