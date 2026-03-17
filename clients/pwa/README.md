# INDIS — Progressive Web App

> Fallback for unsupported devices — React + TypeScript (RTL-first)

## Technology

- **React** + TypeScript
- **RTL-first** design (Persian primary)
- Service Worker for offline credential presentation
- IndexedDB for encrypted credential storage

## Setup

```bash
cd clients/pwa
npm install
npm run dev
```

## Requirements

- Full credential presentation and ZK proof generation without network
- Up to 72 hours offline capability with cached revocation lists
- Responsive layout — mobile and desktop
