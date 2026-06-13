import { defineConfig } from '@lingui/cli'

// Source language is Simplified Chinese; English is a translation in src/locales/en.
export default defineConfig({
  sourceLocale: 'zh',
  locales: ['zh', 'en'],
  catalogs: [
    {
      path: '<rootDir>/src/locales/{locale}/messages',
      include: ['<rootDir>/src'],
    },
  ],
})
