import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { UgosViteBuilder } from '@ugreen-nas/builder-open'

const builder = new UgosViteBuilder({
  windowConfig: { width: 480, height: 640, minWidth: 360, minHeight: 400 },
  getIgnoreFolder: (c) => c,
})

export default defineConfig({
  plugins: [vue(), ...builder.pluginEntry()],
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
})
