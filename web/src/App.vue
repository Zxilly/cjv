<script setup lang="ts">
import CodeBlock from './components/CodeBlock.vue'
import Collapsible from './components/Collapsible.vue'
import { usePlatform } from './composables/usePlatform'

const { state, info, methods, mirrorMethods } = usePlatform()
</script>

<template>
  <div class="max-w-2xl mx-auto px-4 md:px-6 py-12 md:py-24 overflow-x-hidden w-full">
    <!-- Header -->
    <h1 class="text-6xl sm:text-7xl md:text-8xl" style="font-family: 'Patua One', serif; font-weight: 400">
      <span class="cj-gradient" style="padding-bottom: 0.15em; display: inline-block">cjv</span>
    </h1>
    <p class="mt-6 text-lg md:text-2xl text-gray-500 dark:text-gray-400">
      <a href="https://cangjie-lang.cn/" class="text-cj dark:text-cj-light hover:underline" style="font-family: 'Zhi Mang Xing', cursive; font-size: 1.4em; vertical-align: -0.1em; margin-right: 0.15em">仓颉</a>编程语言 SDK 工具链管理器
    </p>

    <!-- Install Card -->
    <div class="mt-10 rounded-lg border border-gray-200 dark:border-gray-800 divide-y divide-gray-200 dark:divide-gray-800">
      <!-- Primary: detected desktop OS -->
      <div v-if="state === 'ready'" class="p-6 text-center">
        <p class="text-base text-gray-500 dark:text-gray-400 mb-4">{{ info.hint }}</p>
        <CodeBlock :command="info.command!" primary />
        <p class="mt-4 text-sm text-gray-400 dark:text-gray-500">检测到你的平台：{{ info.label }}</p>
      </div>

      <!-- Unsupported mobile -->
      <div v-else-if="state === 'unsupported'" class="p-6 text-center">
        <p class="text-base text-gray-500 dark:text-gray-400">
          cjv 暂不支持 <strong class="text-gray-700 dark:text-gray-300">{{ info.label }}</strong> 平台。
        </p>
        <p class="mt-2 text-sm text-gray-400 dark:text-gray-500">cjv 目前支持 Linux、macOS 和 Windows 桌面系统。</p>
        <p class="mt-3 text-sm text-gray-400 dark:text-gray-500">
          请在桌面设备上访问此页面，或查看 <a href="https://github.com/Zxilly/cjv" class="text-cj dark:text-cj-light hover:underline">GitHub</a> 了解更多。
        </p>
      </div>

      <!-- Unknown -->
      <div v-else-if="state === 'unknown'" class="p-6 text-center">
        <p class="text-base text-gray-500 dark:text-gray-400">无法识别你的平台，以下是所有支持的安装方式。</p>
      </div>

      <!-- Other install methods -->
      <Collapsible title="其他安装方式" :initial="state === 'unknown'">
        <div class="mt-3 space-y-3 text-sm">
          <div v-for="m in methods" :key="m.label">
            <p class="text-sm font-medium text-gray-500 dark:text-gray-400 mb-1.5">{{ m.label }}</p>
            <CodeBlock :command="m.command" />
          </div>
          <p class="text-sm text-gray-400">
            或从 <a href="https://github.com/Zxilly/cjv/releases" class="text-cj dark:text-cj-light hover:underline">GitHub Releases</a> 手动下载。
          </p>
        </div>
      </Collapsible>

      <!-- Mirror -->
      <Collapsible title="使用镜像源安装">
        <div class="mt-3 space-y-3 text-sm">
          <div v-for="m in mirrorMethods" :key="m.label">
            <p class="text-sm font-medium text-gray-500 dark:text-gray-400 mb-1.5">{{ m.label }}</p>
            <CodeBlock :command="m.command" />
          </div>
        </div>
      </Collapsible>
    </div>

    <!-- Footer -->
    <div class="mt-12 text-center text-base text-gray-400 dark:text-gray-600 space-x-3">
      <a href="https://github.com/Zxilly/cjv" class="hover:text-cj dark:hover:text-cj-light">GitHub</a>
      <span>&middot;</span>
      <a href="https://github.com/Zxilly/cjv/releases" class="hover:text-cj dark:hover:text-cj-light">Releases</a>
      <span>&middot;</span>
      <a href="https://cangjie-lang.cn/" class="hover:text-cj dark:hover:text-cj-light">仓颉官网</a>
    </div>
  </div>
</template>
