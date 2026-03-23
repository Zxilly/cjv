const BASE = 'https://cjv.zxilly.dev';

const COPY_SVG = '<svg xmlns="http://www.w3.org/2000/svg" class="w-full h-full" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1"/></svg>';
const CHECK_SVG = '<svg xmlns="http://www.w3.org/2000/svg" class="w-full h-full text-cj dark:text-cj-light" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path d="M5 13l4 4L19 7"/></svg>';

function detectPlatform() {
  const ua = navigator.userAgent.toLowerCase();
  const platform = (navigator.platform || '').toLowerCase();

  if (/iphone|ipad|ipod/.test(ua)) return { os: 'ios', arch: 'arm64' };
  if (/android/.test(ua)) return { os: 'android', arch: 'arm64' };
  if (/harmonyos|hmos/.test(ua)) return { os: 'harmonyos', arch: 'arm64' };

  let os = 'unknown';
  if (ua.includes('win') || platform.includes('win')) os = 'windows';
  else if (ua.includes('mac') || platform.includes('mac')) os = 'darwin';
  else if (ua.includes('linux') || platform.includes('linux')) os = 'linux';

  if (os === 'unknown') {
    const av = (navigator.appVersion || '').toLowerCase();
    const oc = (navigator.oscpu || '').toLowerCase();
    if (av.includes('win') || oc.includes('win')) os = 'windows';
    else if (av.includes('mac') || oc.includes('mac')) os = 'darwin';
    else if (av.includes('linux') || oc.includes('linux')) os = 'linux';
  }

  let arch = 'amd64';
  if (navigator.userAgentData?.architecture?.toLowerCase() === 'arm') arch = 'arm64';
  else if (/arm64|aarch64/.test(ua) || platform.includes('arm')) arch = 'arm64';

  return { os, arch };
}

const UNSUPPORTED = { ios: 'iOS', android: 'Android', harmonyos: 'HarmonyOS' };

function getInstallInfo(os, arch) {
  if (UNSUPPORTED[os]) return { label: UNSUPPORTED[os] };
  if (os === 'windows') return { label: 'Windows x86_64', hint: '在 PowerShell 中运行：', command: `irm ${BASE}/install.ps1 | iex` };
  if (os === 'darwin' || os === 'linux') {
    const ol = os === 'darwin' ? 'macOS' : 'Linux';
    return { label: `${ol} ${arch === 'arm64' ? 'ARM64' : 'x86_64'}`, hint: '在终端中运行：', command: `curl -sSf ${BASE}/install.sh | sh` };
  }
  return { label: '未知平台' };
}

// Vue app
const { createApp, ref } = Vue;

const CodeBlock = {
  props: { command: String, primary: { type: Boolean, default: false } },
  setup(props) {
    const copied = ref(false);
    const { computed } = Vue;
    const icon = computed(() => copied.value ? CHECK_SVG : COPY_SVG);
    function copy(e) {
      const code = e.target.closest('.install-box').querySelector('code');
      if (!code) return;
      navigator.clipboard.writeText(code.textContent).then(() => {
        copied.value = true;
        setTimeout(() => { copied.value = false; }, 1500);
      }, () => {
        const r = document.createRange(); r.selectNodeContents(code);
        const s = window.getSelection(); s.removeAllRanges(); s.addRange(r);
      });
    }
    return { copied, copy, icon };
  },
  template: `
    <div class="install-box relative bg-gray-50 dark:bg-gray-900 border border-gray-200 dark:border-gray-700"
         :class="primary ? 'rounded-lg' : 'rounded'">
      <div class="overflow-x-auto" :class="primary ? 'px-5 py-4 pr-12' : 'px-3 py-2 pr-9'">
        <code class="font-mono text-gray-900 dark:text-gray-100 whitespace-nowrap"
              :class="primary ? 'text-sm md:text-base' : 'text-sm'">{{ command }}</code>
      </div>
      <button @click="copy"
              class="absolute top-1/2 -translate-y-1/2 rounded bg-gray-50/80 dark:bg-gray-900/80 backdrop-blur-sm hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors cursor-pointer text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
              :class="primary ? 'right-3 p-1.5' : 'right-2 p-1'"
              title="\u590d\u5236">
        <span v-html="icon" class="block" :class="primary ? 'w-4 h-4' : 'w-3.5 h-3.5'"></span>
      </button>
    </div>`
};

const Collapsible = {
  props: { title: String, initial: { type: Boolean, default: false } },
  setup(props) {
    const open = ref(props.initial);
    return { open };
  },
  template: `
    <div class="px-6 py-4">
      <button @click="open = !open"
              class="cursor-pointer text-base text-cj dark:text-cj-light hover:underline select-none flex items-center gap-1 w-full text-left">
        <span class="transition-transform duration-300" :class="open ? 'rotate-90' : ''">&#9654;</span>
        {{ title }}
      </button>
      <Transition name="slide">
        <div v-if="open"><div><slot></slot></div></div>
      </Transition>
    </div>`
};

createApp({
  components: { CodeBlock, Collapsible },
  setup() {
    const { os, arch } = detectPlatform();
    const info = getInstallInfo(os, arch);
    const state = UNSUPPORTED[os] ? 'unsupported' : (os === 'unknown' ? 'unknown' : 'ready');

    const methods = [
      { label: 'Linux / macOS', command: `curl -sSf ${BASE}/install.sh | sh` },
      { label: 'Windows (PowerShell)', command: `irm ${BASE}/install.ps1 | iex` },
      { label: '从源码编译', command: 'go install github.com/Zxilly/cjv/cmd/cjv@latest' },
    ];
    const mirrorMethods = [
      { label: 'Linux / macOS', command: `curl -sSf ${BASE}/install.sh | sh -s -- --mirror` },
      { label: 'Windows (PowerShell)', command: `$env:CJV_MIRROR = "1"; irm ${BASE}/install.ps1 | iex` },
    ];

    return { state, info, methods, mirrorMethods };
  }
}).mount('#app');
