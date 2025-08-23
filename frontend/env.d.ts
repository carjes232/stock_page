/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_BASE?: string
  readonly FRONTEND_PORT?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
