#!/usr/bin/env node
import { writeFileSync, mkdirSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = dirname(fileURLToPath(import.meta.url))
const rabbitRoot = join(__dirname, '..')

const m = await import('file:///tmp/rov_vendor.mjs')
function strip(c) {
  const o = {}
  for (const [k, v] of Object.entries(c)) {
    o[k] = { safeFlags: v.safeFlags, respectsDoubleDash: v.respectsDoubleDash }
  }
  return o
}
const merged = {
  ...strip(m.GIT_READ_ONLY_COMMANDS),
  ...strip(m.GH_READ_ONLY_COMMANDS),
  ...strip(m.DOCKER_READ_ONLY_COMMANDS),
  ...strip(m.RIPGREP_READ_ONLY_COMMANDS),
  ...strip(m.PYRIGHT_READ_ONLY_COMMANDS),
}
const outDir = join(rabbitRoot, 'internal', 'readonlycmd')
mkdirSync(outDir, { recursive: true })
const f = join(outDir, 'allowlist_shared.json')
writeFileSync(f, JSON.stringify(merged))
console.error('wrote', f, 'commands', Object.keys(merged).length)
