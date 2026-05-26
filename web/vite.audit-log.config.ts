import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import federation from '@originjs/vite-plugin-federation';

const dropCss = {
  name: 'drop-css',
  generateBundle(_: unknown, bundle: Record<string, { fileName: string }>) {
    for (const key of Object.keys(bundle)) {
      if (key.endsWith('.css')) delete bundle[key];
    }
  },
};

export default defineConfig({
  plugins: [
    react(),
    federation({
      name: 'mfe_audit_log',
      filename: 'remoteEntry.js',
      exposes: {
        './manifest': './src/manifest.ts',
        './AuthContext': './src/context/AuthContext.ts',
        './AuditLogPage': './src/AuditLogPage.tsx',
      },
      shared: {
        react: { singleton: true, requiredVersion: '^19.0.0', eager: true },
        'react-dom': { singleton: true, requiredVersion: '^19.0.0', eager: true },
        'react-router-dom': { singleton: true, requiredVersion: '^7.0.0', eager: true },
      },
    }),
    dropCss,
  ],
  build: {
    target: 'esnext',
    outDir: 'dist-mfe',
    assetsDir: 'assets/audit-log',
    cssCodeSplit: false,
  },
  server: {
    port: 3007,
    strictPort: true,
    proxy: {
      '/api/audit-log': {
        target: 'http://localhost:8000',
        changeOrigin: true,
        rewrite: (path: string) => path.replace(/^\/api\/audit-log/, ''),
      },
    },
  },
  preview: {
    port: 3007,
    strictPort: true,
  },
});
