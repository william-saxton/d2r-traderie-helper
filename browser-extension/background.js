// Background service worker
console.log('Traderie D2R Assistant: Background script loaded');

// Initialize extension
chrome.runtime.onInstalled.addListener((details) => {
  console.log('Extension installed:', details.reason);
  
  if (details.reason === 'install') {
    // Set default settings
    chrome.storage.local.set({
      relistSettings: {
        relistEnabled: false,
        relistInterval: 30,
        relistOnlyExpired: true
      },
      autoRelists: 0
    });
  }
});

// Listen for messages from popup and content scripts
chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
  if (request.action === 'updateRelistAlarm') {
    updateRelistAlarm(request.settings);
    sendResponse({ success: true });
  } else if (request.action === 'stopRelistAlarm') {
    stopRelistAlarm();
    sendResponse({ success: true });
  }
  return true;
});

// Handle alarms
chrome.alarms.onAlarm.addListener((alarm) => {
  if (alarm.name === 'autoRelist') {
    performAutoRelist();
  }
});

// Update relist alarm
async function updateRelistAlarm(settings) {
  console.log('Updating relist alarm:', settings);
  
  // Clear existing alarm
  await chrome.alarms.clear('autoRelist');
  
  if (settings.relistEnabled && settings.relistInterval > 0) {
    // Create new alarm
    chrome.alarms.create('autoRelist', {
      periodInMinutes: settings.relistInterval
    });
    
    console.log(`Auto-relist alarm set for every ${settings.relistInterval} minutes`);
  }
}

// Stop relist alarm
async function stopRelistAlarm() {
  await chrome.alarms.clear('autoRelist');
  console.log('Auto-relist alarm stopped');
}

// Perform auto-relist
async function performAutoRelist() {
  console.log('Auto-relist alarm triggered');
  
  // Get settings
  const result = await chrome.storage.local.get('relistSettings');
  const settings = result.relistSettings || {};
  
  // Find all Traderie tabs
  const tabs = await chrome.tabs.query({ url: 'https://traderie.com/*' });
  
  if (tabs.length === 0) {
    console.log('No Traderie tabs open, skipping auto-relist');
    return;
  }
  
  // Try to find listings page
  let listingsTab = tabs.find(tab => 
    tab.url.includes('/listings') || tab.url.includes('/profile')
  );
  
  if (!listingsTab) {
    // Use first Traderie tab
    listingsTab = tabs[0];
    
    // Navigate to listings page
    try {
      await chrome.tabs.update(listingsTab.id, { 
        url: 'https://traderie.com/diablo2resurrected/listings'
      });
      
      // Wait for page to load
      await new Promise(resolve => setTimeout(resolve, 2000));
    } catch (error) {
      console.error('Error navigating to listings page:', error);
      return;
    }
  }
  
  // Send message to content script to perform relist
  try {
    await chrome.tabs.sendMessage(listingsTab.id, {
      action: 'relistItems',
      settings: settings
    });
  } catch (error) {
    console.error('Error sending relist message:', error);
  }
}

// Handle extension icon click
chrome.action.onClicked.addListener((tab) => {
  // Open popup (default behavior)
  console.log('Extension icon clicked');
});

// --- Wails Bridge Logic ---
const BRIDGE_URL = 'http://127.0.0.1:8081';
let isPolling = false;

async function pollWailsCommands() {
  if (isPolling) return;
  isPolling = true;

  try {
    const response = await fetch(`${BRIDGE_URL}/commands`);
    if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`);
    
    const commands = await response.json();
    if (commands && commands.length > 0) {
      console.log(`[Traderie Assistant] Received ${commands.length} commands from Wails`);
      for (const cmd of commands) {
        console.log(`[Traderie Assistant] Processing ${cmd.action} (ID: ${cmd.id})`);
        handleWailsCommand(cmd);
      }
    }
  } catch (error) {
    // Silently fail if Wails is not running, but log once in a while
    if (!error.message.includes('Failed to fetch')) {
      console.error('[Traderie Assistant] Error polling Wails commands:', error);
    }
  } finally {
    isPolling = false;
  }
}

async function handleWailsCommand(cmd) {
  console.log('Handling command:', cmd);
  let result = { id: cmd.id, success: false };

  try {
    switch (cmd.action) {
      case 'post_listing':
        result = await executePostListing(cmd.id, cmd.payload);
        break;
      case 'get_listings':
        result = await executeGetListings(cmd.id, cmd.payload);
        break;
      case 'test_connection':
        result = await executeTestConnection(cmd.id, cmd.payload);
        break;
      case 'refresh_listings':
        result = await executeRefreshListings(cmd.id, cmd.payload);
        break;
      case 'open_tab':
        result = await executeOpenTab(cmd.id, cmd.payload);
        break;
      default:
        result.error = `Unknown action: ${cmd.action}`;
    }
  } catch (error) {
    result.error = error.message;
  }

  // Send result back to Wails
  try {
    await fetch(`${BRIDGE_URL}/results`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(result)
    });
  } catch (error) {
    console.error('Error sending result back to Wails:', error);
  }
}

async function executePostListing(id, payload) {
  const { item, auth, baseURL } = payload;
  
  try {
    const formData = new FormData();
    formData.append('body', JSON.stringify(item));

    const response = await fetch(`${baseURL}/api/diablo2resurrected/listings/create`, {
      method: 'POST',
      headers: {
        'Accept': '*/*',
        'Authorization': auth
      },
      body: formData,
      credentials: 'include'
    });

    if (!response.ok) {
      const errorText = await response.text();
      return { id, success: false, error: `HTTP ${response.status}: ${errorText}` };
    }

    const data = await response.json();
    return { id, success: true, data };
  } catch (error) {
    return { id, success: false, error: error.message };
  }
}

async function executeGetListings(id, payload) {
  const { baseURL } = payload;
  
  try {
    const response = await fetch(`${baseURL}/api/diablo2resurrected/listings/user`, {
      method: 'GET',
      headers: { 'Accept': 'application/json' },
      credentials: 'include'
    });

    if (!response.ok) {
      return { id, success: false, error: `HTTP ${response.status}` };
    }

    const data = await response.json();
    // Traderie API returns listings in a specific format, we need to extract the array
    // Based on cloudflare_client.go, it expects an array of models.TraderieItem
    return { id, success: true, data };
  } catch (error) {
    return { id, success: false, error: error.message };
  }
}

async function executeTestConnection(id, payload) {
  const { baseURL } = payload;
  
  try {
    const response = await fetch(`${baseURL}/diablo2resurrected`, {
      method: 'GET',
      credentials: 'include'
    });

    if (!response.ok) {
      return { id, success: false, error: `HTTP ${response.status}` };
    }

    const text = await response.text();
    if (text.toLowerCase().includes('cloudflare') && text.toLowerCase().includes('challenge')) {
      return { id, success: false, error: 'Cloudflare challenge detected' };
    }

    return { id, success: true };
  } catch (error) {
    return { id, success: false, error: error.message };
  }
}

async function executeRefreshListings(id, payload) {
  const { baseURL } = payload;
  
  try {
    const response = await fetch(`${baseURL}/api/diablo2resurrected/listings/refresh`, {
      method: 'PUT',
      headers: {
        'Accept': 'application/json',
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ all: true, selling: true }),
      credentials: 'include'
    });

    if (!response.ok) {
      const errorText = await response.text();
      return { id, success: false, error: `HTTP ${response.status}: ${errorText}` };
    }

    const data = await response.json();
    return { id, success: true, data };
  } catch (error) {
    return { id, success: false, error: error.message };
  }
}

async function executeOpenTab(id, payload) {
  const { url } = payload;
  try {
    await chrome.tabs.create({ url });
    return { id, success: true };
  } catch (error) {
    return { id, success: false, error: error.message };
  }
}

// Start polling
pollWailsCommands();
setInterval(pollWailsCommands, 1000);

// Keep-alive mechanism for Manifest V3 Service Worker
chrome.runtime.onConnect.addListener((port) => {
  if (port.name === 'keep-alive') {
    port.onDisconnect.addListener(() => {
      // Reconnect or handle as needed
    });
  }
});

// Periodic alarm to wake up the service worker
chrome.alarms.create('keepAlive', { periodInMinutes: 0.5 });
chrome.alarms.onAlarm.addListener((alarm) => {
  if (alarm.name === 'keepAlive') {
    console.log('[Traderie Assistant] Service worker keep-alive alarm triggered');
    pollWailsCommands(); // Ensure we poll when woken up
  }
});

// Additional triggers to wake up the service worker
chrome.tabs.onUpdated.addListener((tabId, changeInfo, tab) => {
  if (changeInfo.status === 'complete' && tab.url && tab.url.includes('traderie.com')) {
    pollWailsCommands();
  }
});

chrome.tabs.onActivated.addListener(activeInfo => {
  pollWailsCommands();
});





