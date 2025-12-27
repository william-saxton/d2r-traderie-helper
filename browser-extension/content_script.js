// Content script for Traderie pages - API-based approach
(function() {
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
