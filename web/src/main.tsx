import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import App from './App'
import './style.css'

createRoot(document.getElementById('app')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)

const fallback = setTimeout(reveal, 3000)
function reveal() {
  clearTimeout(fallback)
  document.documentElement.classList.add('fonts-loaded')
}

Promise.all([
  document.fonts.load('1em "Noto Sans SC Variable"'),
  document.fonts.load('1em "Patua One"'),
]).then(reveal, reveal)
