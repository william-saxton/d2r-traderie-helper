# Traderie D2R Assistant (Wails App)

This is the desktop application for the Traderie D2R Assistant. It uses Wails (Go + Svelte) to provide a user interface and core logic for reading Diablo 2 Resurrected item memory and communicating with the Traderie API.

## Features

- **Memory Reading**: Uses `d2go` to read item data directly from game memory.
- **API Bridge**: Communicates with the companion browser extension to make authenticated API calls to Traderie.
- **Hotkey Listener**: Listens for the F9 key to trigger item capture.
- **Svelte Frontend**: Modern UI for configuring settings and viewing item data.

## Prerequisites

- [Wails CLI](https://wails.io/docs/gettingstarted/installation)
- Go 1.21+
- Node.js & NPM
- Diablo 2 Resurrected

## Development

To run in live development mode:

```bash
wails dev
```

This will run a Vite development server for the frontend and recompile the Go backend on changes.

## Building

To build a production executable:

```bash
wails build
```

The output will be in the `build/bin/` directory.

## Configuration

The application settings are stored in `internal/config/settings.go` and can be adjusted through the UI.
