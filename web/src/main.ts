import { createApp } from 'vue'
import App from './App.vue'
import './style.css'

createApp(App).mount('#app')

const show = () => document.documentElement.classList.add('fonts-loaded')

Promise.all([
  document.fonts.load('1em "Noto Sans SC Variable"'),
  document.fonts.load('1em "Patua One"'),
]).then(show, show)

setTimeout(show, 3000)
