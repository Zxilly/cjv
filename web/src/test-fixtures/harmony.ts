// Original UA samples: keep browser versions distinct from the OS version.
export const HARMONY_SAMPLES = [
  {
    name: 'OpenHarmony 6.1 tablet',
    userAgent: 'Mozilla/5.0 (Tablet; OpenHarmony 6.1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36 ArkWeb/6.1.0.120 HuaweiBrowser/6.1.8.301',
    version: '6.1',
    device: 'tablet',
    message: '你正在使用鸿蒙平板。',
    englishMessage: 'You are using a HarmonyOS tablet.',
  },
  {
    name: 'OpenHarmony 7.0 convertible in tablet mode',
    userAgent: 'Mozilla/5.0 (Tablet; OpenHarmony 7.0; Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36 ArkWeb/7.0.0.105 HuaweiBrowser/6.1.8.301',
    version: '7.0',
    device: 'convertible-tablet',
    message: '你正在使用鸿蒙二合一设备的平板模式，Windows 安装包不适用于此系统。',
    englishMessage: 'You are using a HarmonyOS 2-in-1 in tablet mode. The Windows installer is not compatible with this system.',
  },
  {
    name: 'OpenHarmony 7.0 phone with ArkWeb .109',
    userAgent: 'Mozilla/5.0 (Phone; OpenHarmony 7.0; Android 10) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36 ArkWeb/7.0.0.109 Mobile HuaweiBrowser/6.1.8.301',
    version: '7.0',
    device: 'phone',
    message: '你正在使用鸿蒙手机。',
    englishMessage: 'You are using a HarmonyOS phone.',
  },
  {
    name: 'OpenHarmony 7.0 phone with ArkWeb .107',
    userAgent: 'Mozilla/5.0 (Phone; OpenHarmony 7.0; Android 10) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36 ArkWeb/7.0.0.107 Mobile HuaweiBrowser/6.1.8.301',
    version: '7.0',
    device: 'phone',
    message: '你正在使用鸿蒙手机。',
    englishMessage: 'You are using a HarmonyOS phone.',
  },
] as const
