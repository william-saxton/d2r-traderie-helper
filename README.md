# Traderie D2R Assistant

A desktop application for automating item listings on Traderie.com for Diablo 2 Resurrected.

## Features

- **Direct Memory Reading**: 100% accuracy by reading item data directly from game memory.
- **Hotkey Support**: Capture items instantly while in-game (default: F9).
- **API Integration**: Posts listings directly to Traderie via their API.
- **Companion Extension**: Includes a browser extension bridge to handle authentication and auto-relisting.

## Project Structure

- `d2r-traderie-wails/`: The main Wails application (Go + Svelte).
- `browser-extension/`: Companion browser extension for API bridging and relisting.

## Installation

### 1. Install the Browser Extension
Since this extension is not yet on the Web Store, you must load it manually:

1.  **Download the extension**: Obtain the `traderie-d2r-assistant-extension.zip` from the latest [GitHub Release](https://github.com/your-repo/releases) and extract it to a folder.
2.  **Open Extensions page**: In your Chromium browser (Chrome, Edge, Brave, etc.), go to `chrome://extensions` (or `edge://extensions`).
3.  **Enable Developer Mode**: Toggle the switch in the top-right corner.
4.  **Load Unpacked**: Click the **"Load unpacked"** button and select the folder where you extracted the extension.
5.  **Pin it**: For ease of use, pin the extension icon to your toolbar.

### 2. Install the Desktop App
1.  Download the `d2r-traderie-wails.exe` (or the NSIS installer) from the latest [GitHub Release](https://github.com/your-repo/releases).
2.  Run the application. You may need to bypass the Windows SmartScreen warning as the app is not signed.

## Usage

1.  **Prepare the Browser**:
    *   Open your browser and ensure the **Traderie D2R Assistant** extension is active.
    *   Log in to your account at [Traderie.com](https://traderie.com/diablo2resurrected).
2.  **Launch the App**: Open the `d2r-traderie-wails` application.
3.  **Capture an Item**:
    *   Launch **Diablo 2 Resurrected** and hover your mouse over an item in your inventory or stash.
    *   Press **F9** (the default hotkey).
    *   The app will pop up with the item details automatically read from memory.
4.  **Post or Search**:
    *   Verify the item properties in the app.
    *   Click **"Post Listing"** to send the item directly to your Traderie listings.
    *   Alternatively, use **"Search on Traderie"** to quickly find similar items.

## Disclaimer

This tool reads game memory which may violate Diablo 2 Resurrected's Terms of Service. Use at your own risk. The authors are not responsible for any consequences including account bans.

## License

MIT License
