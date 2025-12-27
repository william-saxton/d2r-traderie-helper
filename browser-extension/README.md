# Traderie Companion Browser Extension

This extension serves as a bridge for the main Go application, allowing it to make API calls to Traderie and providing auto-relisting capabilities.

## Features

- **API Bridge**: Allows the Go application to communicate with Traderie's API through the browser.
- **Auto-Relist**: Automatically refreshes your Traderie listings at configurable intervals.
- **Wails Integration**: Receives commands from the Wails application to post or refresh listings.

## Installation

1. Load as unpacked extension in Chrome/Edge:
   - Go to `chrome://extensions/`
   - Enable "Developer mode"
   - Click "Load unpacked"
   - Select this directory

2. Ensure the main Go application is running to use the API bridge features.

## Why use this extension?

This extension overcomes browser limitations by allowing the Go application to leverage the browser's session and cookies for authenticated API calls to Traderie.




