import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import App from './App'
import './styles.css'
import './kernel.css'
import './runtime.css'
import './credential.css'
import './diagnostics.css'
import './evidence.css'
import './adapter.css'

createRoot(document.getElementById('root')!).render(<StrictMode><App /></StrictMode>)
