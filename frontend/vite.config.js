import react from '@vitejs/plugin-react'

// Deploys to GitHub Pages under the repo name (https://<user>.github.io/gowkhtmltopdf/).
// Override with VITE_BASE_PATH for a custom repo / root deployment.
const basePath = process.env.VITE_BASE_PATH || '/gowkhtmltopdf/'

export default {
  plugins: [react()],
  base: basePath,
  build: {
    outDir: 'dist',
    assetsDir: 'assets',
    chunkSizeWarningLimit: 500,
  },
}
