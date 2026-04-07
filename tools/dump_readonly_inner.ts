/**
 * Run from rabbit-code directory:
 *   npx tsx tools/dump_readonly_inner.ts
 */
import { writeFileSync, mkdirSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import {
  GIT_READ_ONLY_COMMANDS,
  GH_READ_ONLY_COMMANDS,
  DOCKER_READ_ONLY_COMMANDS,
  RIPGREP_READ_ONLY_COMMANDS,
  PYRIGHT_READ_ONLY_COMMANDS,
} from '../../claude-code-sourcemap/restored-src/src/utils/shell/readOnlyCommandValidation.ts'

function strip(c: Record<string, { safeFlags: Record<string, string>; respectsDoubleDash?: boolean }>) {
  const o: Record<string, { safeFlags: Record<string, string>; respectsDoubleDash?: boolean }> = {}
  for (const [k, v] of Object.entries(c)) {
    o[k] = { safeFlags: v.safeFlags, respectsDoubleDash: v.respectsDoubleDash }
  }
  return o
}

const merged = {
  ...strip(GIT_READ_ONLY_COMMANDS),
  ...strip(GH_READ_ONLY_COMMANDS),
  ...strip(DOCKER_READ_ONLY_COMMANDS),
  ...strip(RIPGREP_READ_ONLY_COMMANDS),
  ...strip(PYRIGHT_READ_ONLY_COMMANDS),
}

const __dirname = dirname(fileURLToPath(import.meta.url))
const outDir = join(__dirname, '..', 'internal', 'readonlycmd')
mkdirSync(outDir, { recursive: true })
const outFile = join(outDir, 'allowlist_shared.json')
writeFileSync(outFile, JSON.stringify(merged))
console.error('wrote', outFile, 'commands', Object.keys(merged).length)
