export const INSTALL_TABS = ['command', 'download', 'source'] as const

export type InstallTab = (typeof INSTALL_TABS)[number]

export function getInstallTabDirection(from: InstallTab, to: InstallTab) {
  const fromIndex = INSTALL_TABS.indexOf(from)
  const toIndex = INSTALL_TABS.indexOf(to)

  return Math.sign(toIndex - fromIndex)
}
