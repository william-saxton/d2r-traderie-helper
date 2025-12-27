# Traderie D2R Assistant

A desktop application for automating item listings on Traderie.com for Diablo 2 Resurrected.

## Features

- **Direct Memory Reading**: 100% accuracy by reading item data directly from game memory.
- **Hotkey Support**: Capture items instantly while in-game (default: F9).
- **API Integration**: Posts listings directly to Traderie via their API.
- **Companion Extension**: Includes a browser extension bridge to handle authentication and auto-relisting.

## Project Structure

- `d2r-traderie-wails/`: The main Wails application (Go + Svelte).
- `archive-browser-extension/`: Companion browser extension for API bridging and relisting.

## Getting Started

See the [Wails App README](./d2r-traderie-wails/README.md) for full setup and usage instructions.

## Disclaimer

This tool reads game memory which may violate Diablo 2 Resurrected's Terms of Service. Use at your own risk. The authors are not responsible for any consequences including account bans.

## License

MIT License
