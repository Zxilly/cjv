interface ImportMetaEnv {
  readonly VITE_EXPECTED_BINARY_GOARCH?: string
  readonly VITE_EXPECTED_BINARY_GOOS?: string
  readonly VITE_EXPECTED_BROWSER?: string
  readonly VITE_EXPECTED_LABEL?: string
  readonly VITE_EXPECTED_PLATFORM_ARCH?: string
  readonly VITE_EXPECTED_PLATFORM_OS?: string
  readonly VITE_EXPECTED_PLAYWRIGHT_BROWSER?: string
  readonly VITE_EXPECTED_RUNNER?: string
  readonly VITE_EXPECTED_RUNNER_ARCH?: string
  readonly VITE_EXPECTED_STATE?: 'ready' | 'unsupported' | 'unknown'
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
