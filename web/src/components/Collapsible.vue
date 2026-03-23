<script setup lang="ts">
import { ref } from 'vue'

const props = defineProps<{
  title: string
  initial?: boolean
}>()

const open = ref(props.initial ?? false)
</script>

<template>
  <div class="px-6 py-4">
    <button
      class="cursor-pointer text-base text-cj dark:text-cj-light hover:underline select-none flex items-center gap-1 w-full text-left"
      @click="open = !open"
    >
      <span class="transition-transform duration-300" :class="open ? 'rotate-90' : ''">&#9654;</span>
      {{ title }}
    </button>
    <Transition name="slide">
      <div v-if="open">
        <div>
          <slot />
        </div>
      </div>
    </Transition>
  </div>
</template>
