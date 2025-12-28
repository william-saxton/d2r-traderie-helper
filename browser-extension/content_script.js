// Content script for Traderie pages - API-based approach
(function () {
  'use strict';

  console.log('Traderie D2R Assistant: Content script loaded');

  // Auto-relist items (placeholder for future implementation)
  async function performAutoRelist(settings) {
    console.log('Auto-relist requested with settings:', settings);
    // showNotification('Auto-relist feature coming soon!', 'info');
  }

  // Show notification to user
  function showNotification(message, type = 'info') {
    console.log(`[${type.toUpperCase()}] ${message}`);

    // Create notification element
    const notification = document.createElement('div');
    notification.className = `traderie-bot-notification traderie-bot-${type}`;
    notification.textContent = message;
    notification.style.cssText = `
      position: fixed;
      top: 20px;
      right: 20px;
      padding: 15px 20px;
      background: ${type === 'success' ? '#4CAF50' : type === 'error' ? '#f44336' : '#2196F3'};
      color: white;
      border-radius: 8px;
      box-shadow: 0 4px 6px rgba(0,0,0,0.3);
      z-index: 100000;
      font-family: Arial, sans-serif;
      font-size: 14px;
      max-width: 400px;
      animation: slideIn 0.3s ease-out;
    `;

    document.body.appendChild(notification);

    // Remove after 5 seconds
    setTimeout(() => {
      notification.style.animation = 'slideOut 0.3s ease-out';
      setTimeout(() => notification.remove(), 300);
    }, 5000);
  }

  // Add notification animations
  const style = document.createElement('style');
  style.textContent = `
    @keyframes slideIn {
      from { transform: translateX(400px); opacity: 0; }
      to { transform: translateX(0); opacity: 1; }
    }
    @keyframes slideOut {
      from { transform: translateX(0); opacity: 1; }
      to { transform: translateX(400px); opacity: 0; }
    }
  `;
  document.head.appendChild(style);

  // Listen for messages from popup or background
  chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
    if (request.action === 'relistItems') {
      performAutoRelist(request.settings);
      sendResponse({ success: true });
    } else if (request.action === 'getAuth') {
      // Extract JWT from localStorage
      try {
        // Try different possible localStorage keys that Traderie might use
        const possibleKeys = [
          'jwt',
          'token', 
          'authToken',
          'auth_token',
          'accessToken',
          'access_token',
          'traderieToken',
          'user_token',
          'sessionToken'
        ];
        
        let jwt = null;
        
        // Try each possible key
        for (const key of possibleKeys) {
          const value = localStorage.getItem(key);
          if (value) {
            jwt = value;
            console.log(`[Traderie Assistant] Found JWT in localStorage key: ${key}`);
            break;
          }
        }
        
        // If still no JWT found, try to get it from any key that looks like a token
        if (!jwt) {
          console.log('[Traderie Assistant] Checking all localStorage keys for JWT...');
          for (let i = 0; i < localStorage.length; i++) {
            const key = localStorage.key(i);
            const value = localStorage.getItem(key);
            // Check if value looks like a JWT (has dots and is reasonably long)
            if (value && typeof value === 'string' && value.includes('.') && value.length > 50) {
              jwt = value;
              console.log(`[Traderie Assistant] Found JWT-like value in localStorage key: ${key}`);
              break;
            }
          }
        }
        
        if (!jwt) {
          console.log('[Traderie Assistant] No JWT found in localStorage. Available keys:', 
            Array.from({length: localStorage.length}, (_, i) => localStorage.key(i)));
        }
        
        sendResponse({ jwt: jwt });
      } catch (error) {
        console.error('[Traderie Assistant] Error extracting JWT from localStorage:', error);
        sendResponse({ jwt: null, error: error.message });
      }
    }
    return true;
  });

  // Keep-alive connection to background script
  function connectToBackground() {
    try {
      const port = chrome.runtime.connect({ name: 'keep-alive' });
      port.onDisconnect.addListener(() => {
        console.log('[Traderie Assistant] Keep-alive port disconnected, reconnecting...');
        setTimeout(connectToBackground, 1000);
      });
    } catch (e) {
      console.warn('[Traderie Assistant] Could not connect to background for keep-alive:', e);
    }
  }
  connectToBackground();

})();
