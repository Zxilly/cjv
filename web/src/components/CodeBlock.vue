<script setup lang="ts">
import { ref, computed, onUnmounted } from 'vue'

const COPY_SVG = '<svg xmlns="http://www.w3.org/2000/svg" class="w-full h-full" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1"/></svg>'
const CHECK_SVG = '<svg xmlns="http://www.w3.org/2000/svg" class="w-full h-full text-cj dark:text-cj-light" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path d="M5 13l4 4L19 7"/></svg>'

const props = defineProps<{
  command: string
  primary?: boolean
}>()

const copied = ref(false)
const icon = computed(() => copied.value ? CHECK_SVG : COPY_SVG)

let resetTimer: ReturnType<typeof setTimeout> | undefined

onUnmounted(() => {
  if (resetTimer) clearTimeout(resetTimer)
})

function copy() {
  navigator.clipboard.writeText(props.command).then(() => {
    copied.value = true
    if (resetTimer) clearTimeout(resetTimer)
    resetTimer = setTimeout(() => { copied.value = false }, 1500)
  })
}
</script>

<template>
  <div
    class="install-box relative bg-gray-50 dark:bg-gray-900 border border-gray-200 dark:border-gray-700"
    :class="primary ? 'rounded-lg' : 'rounded'"
  >
    <div class="overflow-x-auto" :class="primary ? 'px-5 py-4 pr-12' : 'px-3 py-2 pr-9'">
      <code
        class="font-mono text-gray-900 dark:text-gray-100 whitespace-nowrap"
        :class="primary ? 'text-sm md:text-base' : 'text-sm'"
      >{{ command }}</code>
    </div>
    <button
      class="absolute top-1/2 -translate-y-1/2 rounded bg-gray-50/80 dark:bg-gray-900/80 backdrop-blur-sm hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors cursor-pointer text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
      :class="primary ? 'right-3 p-1.5' : 'right-2 p-1'"
      title="复制"
      @click="copy"
    >
      <span v-html="icon" class="block" :class="primary ? 'w-4 h-4' : 'w-3.5 h-3.5'" />
    </button>
  </div>
</template>
