import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'

// wdth.css, not index.css — it carries the width axis (font-stretch 62–125%)
// as well as weight. Headings are set expanded so they read as horizontal
// bands, which is the whole conceit.
import '@fontsource-variable/archivo/wdth.css'
import '@fontsource-variable/instrument-sans'
import '@fontsource-variable/jetbrains-mono'
import './index.css'

import App from './App'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
