# Traderie Companion Browser Extension

This extension serves as a bridge for the main Go application, allowing it to make API calls to Traderie and providing auto-relisting capabilities.

## Features

- **API Bridge**: Allows the Go application to communicate with Traderie's API through the browser.
- **Auto-Relist**: Automatically refreshes your Traderie listings at configurable intervals.
- **Wails Integration**: Receives commands from the Wails application to post or refresh listings.
- **Authentication Extraction**: Automatically extracts JWT tokens and Cloudflare cookies from your browser session.

## Installation

1. Load as unpacked extension in Chrome/Edge:
   - Go to `chrome://extensions/`
   - Enable "Developer mode"
   - Click "Load unpacked"
   - Select this directory

2. **IMPORTANT**: Make sure you're logged in to Traderie in your browser
   - Visit https://traderie.com
   - Log in with your account
   - The extension will automatically extract your authentication token

3. Ensure the main Go application is running to use the API bridge features.

## How Authentication Works

The extension now automatically:
1. Extracts the JWT authentication token from Traderie's localStorage
2. Retrieves the `cf_clearance` Cloudflare cookie
3. Uses these credentials when making API calls on behalf of the Wails application

This means you no longer need to manually copy cookies or tokens - just stay logged in to Traderie in your browser!

## Troubleshooting

If you're getting "Unauthorized jwt" errors:

1. Make sure you're logged in to Traderie (https://traderie.com) in your browser
2. Reload the extension in chrome://extensions/
3. Try posting an item again from the Wails application
4. Check the browser console (F12) for any error messages

If the extension can't find your JWT token:
- Open Traderie in a new tab
- Open the browser console (F12)
- Look for messages from "[Traderie Assistant]" showing which localStorage keys are available
- This will help diagnose what key Traderie is using for authentication

## Why use this extension?

This extension overcomes browser limitations by allowing the Go application to leverage the browser's session and cookies for authenticated API calls to Traderie.




