import { readdirSync, readFileSync, statSync } from 'node:fs'
import { extname, join } from 'node:path'
import { describe, expect, it } from 'vitest'

/**
 * Prohibe clases de paleta cruda de Tailwind (bg-blue-500, text-gray-400,
 * border-slate-200, etc) fuera de este archivo de tokens. Todo color debe
 * consumirse via los tokens definidos en tokens.css (brand-*, secondary-*,
 * success/warning/danger/info, surface-*, fg-*, border).
 */
const BANNED_PALETTE_CLASS =
  /\b(bg|text|border|ring|from|to|via)-(slate|gray|zinc|neutral|stone|red|orange|amber|yellow|lime|green|emerald|teal|cyan|sky|blue|indigo|violet|purple|fuchsia|pink|rose)-[0-9]{2,3}\b/g

function collectTsxFiles(dir: string, files: string[] = []): string[] {
  for (const entry of readdirSync(dir)) {
    const fullPath = join(dir, entry)
    if (statSync(fullPath).isDirectory()) {
      collectTsxFiles(fullPath, files)
      continue
    }
    if (extname(fullPath) === '.tsx' && !fullPath.endsWith('tokens.css')) {
      files.push(fullPath)
    }
  }
  return files
}

describe('tokens guard', () => {
  it('no usa clases de paleta cruda de Tailwind en src/**/*.tsx', () => {
    const srcDir = join(process.cwd(), 'src')
    const files = collectTsxFiles(srcDir)
    expect(files.length).toBeGreaterThan(0)

    const violations: string[] = []
    for (const file of files) {
      const content = readFileSync(file, 'utf8')
      const matches = content.match(BANNED_PALETTE_CLASS)
      if (matches) {
        violations.push(`${file}: ${matches.join(', ')}`)
      }
    }

    expect(violations).toEqual([])
  })
})
