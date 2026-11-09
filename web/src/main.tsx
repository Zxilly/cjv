import { StrictMode } from 'react'
import { createRoot, hydrateRoot } from 'react-dom/client'
import App from './App'
import './style.css'

const app = (
  <StrictMode>
    <App />
  </StrictMode>
)
const root = document.getElementById('app')!
if (document.documentElement.dataset.prerendered === 'true') hydrateRoot(root, app)
else createRoot(root).render(app)
