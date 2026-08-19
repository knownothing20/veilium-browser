import { describe, expect, it } from 'vitest'
import { en, zhCN } from '.'

function shape(value: unknown): unknown {
  if (typeof value === 'function') return 'function'
  if (typeof value !== 'object' || value === null) return typeof value
  return Object.fromEntries(Object.entries(value).map(([key, item]) => [key, shape(item)]))
}

describe('localization dictionaries', () => {
  it('keeps the English fallback structurally aligned with zh-CN', () => {
    expect(shape(en)).toEqual(shape(zhCN))
  })
})
