import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'

// Same font loading as main.tsx — wdth.css for the width axis, which the
// headings depend on. Duplicated rather than shared because each entry has to
// stand alone; a shared bootstrap module would put every page's imports into
// every bundle.
import '@fontsource-variable/archivo/wdth.css'
import '@fontsource-variable/instrument-sans'
import '@fontsource-variable/jetbrains-mono'
import './index.css'

import InstallPage from './InstallPage'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <InstallPage />
  </StrictMode>,
)
