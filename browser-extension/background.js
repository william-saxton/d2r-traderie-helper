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
let pollInterval = null;

async function pollWailsCommands() {
  if (isPolling) return;
  isPolling = true;

  try {
    const response = await fetch(`${BRIDGE_URL}/commands`, {
      method: 'GET',
      headers: { 'Accept': 'application/json' }
    });
    if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`);

    const commands = await response.json();
    if (commands && commands.length > 0) {
      console.log(`[Traderie Assistant] Received ${commands.length} commands from Wails`);
      for (const cmd of commands) {
        console.log(`[Traderie Assistant] Processing ${cmd.action} (ID: ${cmd.id})`);
        await handleWailsCommand(cmd);
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

// Ensure polling interval is always active
function ensurePollingActive() {
  if (pollInterval) {
    clearInterval(pollInterval);
  }
  pollInterval = setInterval(pollWailsCommands, 1000);
  console.log('[Traderie Assistant] Polling interval (re)started');
}

async function handleWailsCommand(cmd) {
  console.log('[Traderie Assistant] Handling command:', cmd);
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
    console.error('[Traderie Assistant] Error executing command:', error);
    result.error = error.message;
  }

  // Send result back to Wails
  console.log('[Traderie Assistant] Sending result back to Wails:', result);
  try {
    const response = await fetch(`${BRIDGE_URL}/results`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(result)
    });
    if (!response.ok) {
      console.error('[Traderie Assistant] Failed to send result, HTTP status:', response.status);
    } else {
      console.log('[Traderie Assistant] Result sent successfully');
    }
  } catch (error) {
    console.error('[Traderie Assistant] Error sending result back to Wails:', error);
  }
}

// Helper function to get JWT token from Traderie localStorage and cf_clearance cookie
async function getTraderieAuth(baseURL) {
  try {
    // Find a Traderie tab to extract the JWT from
    let tabs = await chrome.tabs.query({ url: 'https://traderie.com/*' });
    
    // Filter out tabs that might not be fully loaded
    tabs = tabs.filter(tab => tab.status === 'complete');
    
    // If no loaded Traderie tab is open, try to open one and wait for it to load
    if (tabs.length === 0) {
      console.log('[Traderie Assistant] No Traderie tab found, opening one...');
      const newTab = await chrome.tabs.create({ url: 'https://traderie.com/diablo2resurrected', active: false });
      
      // Wait for the tab to load completely
      await new Promise((resolve) => {
        const listener = (tabId, changeInfo) => {
          if (tabId === newTab.id && changeInfo.status === 'complete') {
            chrome.tabs.onUpdated.removeListener(listener);
            // Add a small delay to ensure content script is injected
            setTimeout(resolve, 500);
          }
        };
        chrome.tabs.onUpdated.addListener(listener);
        
        // Timeout after 10 seconds
        setTimeout(() => {
          chrome.tabs.onUpdated.removeListener(listener);
          resolve();
        }, 10000);
      });
      
      tabs = [newTab];
    }

    let jwt = null;
    let cf_clearance = null;

    // Try to get JWT from localStorage via content script with retry logic
    if (tabs.length > 0) {
      const maxRetries = 3;
      const retryDelay = 1000; // 1 second between retries
      
      for (let attempt = 1; attempt <= maxRetries; attempt++) {
        try {
          console.log(`[Traderie Assistant] Attempt ${attempt}/${maxRetries} to get JWT from tab ${tabs[0].id}`);
          
          // Try to send message to content script
          const result = await chrome.tabs.sendMessage(tabs[0].id, { action: 'getAuth' });
          if (result && result.jwt) {
            jwt = result.jwt;
            console.log('[Traderie Assistant] Successfully extracted JWT token');
            break;
          } else {
            console.log('[Traderie Assistant] Content script responded but no JWT found');
          }
        } catch (error) {
          console.error(`[Traderie Assistant] Attempt ${attempt} failed:`, error.message);
          
          // If this is not the last attempt, wait before retrying
          if (attempt < maxRetries) {
            console.log(`[Traderie Assistant] Waiting ${retryDelay}ms before retry...`);
            await new Promise(resolve => setTimeout(resolve, retryDelay));
            
            // Refresh the tab list to see if content script loaded
            const refreshedTabs = await chrome.tabs.query({ url: 'https://traderie.com/*' });
            if (refreshedTabs.length > 0) {
              tabs = refreshedTabs.filter(tab => tab.status === 'complete');
            }
          }
        }
      }
      
      // If still no JWT, try to inject content script manually
      if (!jwt && tabs.length > 0) {
        try {
          console.log('[Traderie Assistant] Trying to manually inject content script...');
          await chrome.scripting.executeScript({
            target: { tabId: tabs[0].id },
            files: ['content_script.js']
          });
          
          // Wait a bit for the script to initialize
          await new Promise(resolve => setTimeout(resolve, 500));
          
          // Try one more time
          const result = await chrome.tabs.sendMessage(tabs[0].id, { action: 'getAuth' });
          if (result && result.jwt) {
            jwt = result.jwt;
            console.log('[Traderie Assistant] Successfully extracted JWT after manual injection');
          }
        } catch (injectError) {
          console.error('[Traderie Assistant] Could not manually inject content script:', injectError);
        }
      }
    }

    // Get cf_clearance cookie
    try {
      const cookies = await chrome.cookies.getAll({ domain: '.traderie.com' });
      const cfCookie = cookies.find(c => c.name === 'cf_clearance');
      if (cfCookie) {
        cf_clearance = cfCookie.value;
        console.log('[Traderie Assistant] Found cf_clearance cookie');
      } else {
        console.log('[Traderie Assistant] No cf_clearance cookie found');
      }
    } catch (error) {
      console.error('[Traderie Assistant] Error getting cf_clearance cookie:', error);
    }

    return { jwt, cf_clearance };
  } catch (error) {
    console.error('[Traderie Assistant] Error in getTraderieAuth:', error);
    return { jwt: null, cf_clearance: null };
  }
}

async function executePostListing(id, payload) {
  const { item, baseURL } = payload;

  try {
    // Get JWT token and cf_clearance cookie from Traderie
    const authData = await getTraderieAuth(baseURL);
    if (!authData.jwt) {
      return { 
        id, 
        success: false, 
        error: 'No JWT token found. Please:\n1. Log in to https://traderie.com in your browser\n2. Reload this extension\n3. Try again' 
      };
    }

    const formData = new FormData();
    formData.append('body', JSON.stringify(item));

    const response = await fetch(`${baseURL}/api/diablo2resurrected/listings/create`, {
      method: 'POST',
      headers: {
        'Accept': '*/*',
        'Authorization': `Bearer ${authData.jwt}`
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
    // Get JWT token
    const authData = await getTraderieAuth(baseURL);
    if (!authData.jwt) {
      return { id, success: false, error: 'No JWT token found. Please log in to Traderie first.' };
    }

    const response = await fetch(`${baseURL}/api/diablo2resurrected/listings/user`, {
      method: 'GET',
      headers: { 
        'Accept': 'application/json',
        'Authorization': `Bearer ${authData.jwt}`
      },
      credentials: 'include'
    });

    if (!response.ok) {
      const errorText = await response.text();
      return { id, success: false, error: `HTTP ${response.status}: ${errorText}` };
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
    // Get JWT token
    const authData = await getTraderieAuth(baseURL);
    if (!authData.jwt) {
      return { id, success: false, error: 'No JWT token found. Please log in to Traderie first.' };
    }

    const response = await fetch(`${baseURL}/api/diablo2resurrected/listings/refresh`, {
      method: 'PUT',
      headers: {
        'Accept': 'application/json',
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${authData.jwt}`
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
  console.log('[Traderie Assistant] Opening tab with URL:', url);
  try {
    const tab = await chrome.tabs.create({ url });
    console.log('[Traderie Assistant] Tab opened successfully:', tab.id);
    return { id, success: true };
  } catch (error) {
    console.error('[Traderie Assistant] Error opening tab:', error);
    return { id, success: false, error: error.message };
  }
}

// Start polling
ensurePollingActive();
pollWailsCommands();

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
    ensurePollingActive(); // Restart polling interval
    pollWailsCommands(); // Ensure we poll when woken up
  }
});

// Additional triggers to wake up the service worker
chrome.tabs.onUpdated.addListener((tabId, changeInfo, tab) => {
  if (changeInfo.status === 'complete' && tab.url && tab.url.includes('traderie.com')) {
    ensurePollingActive(); // Ensure polling is active
    pollWailsCommands();
  }
});

chrome.tabs.onActivated.addListener(activeInfo => {
  ensurePollingActive(); // Ensure polling is active
  pollWailsCommands();
});





