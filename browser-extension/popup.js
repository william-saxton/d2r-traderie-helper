// Popup script for the extension
// Simplified version focusing on relisting and API calls

// Initialize popup
document.addEventListener('DOMContentLoaded', async () => {
  await loadSettings();
  await loadStats();
  setupEventListeners();
});

// Setup event listeners
function setupEventListeners() {
  document.getElementById('saveRelistBtn').addEventListener('click', saveRelistSettings);
  document.getElementById('relistEnabled').addEventListener('change', handleRelistToggle);
}

// Save relist settings
async function saveRelistSettings() {
  const enabled = document.getElementById('relistEnabled').checked;
  const interval = parseInt(document.getElementById('relistInterval').value);
  const onlyExpired = document.getElementById('relistOnlyExpired').checked;
  
  const settings = {
    relistEnabled: enabled,
    relistInterval: interval,
    relistOnlyExpired: onlyExpired
  };
  
  await chrome.storage.local.set({ relistSettings: settings });
  
  // Send message to background script to update alarm
  chrome.runtime.sendMessage({
    action: 'updateRelistAlarm',
    settings: settings
  });
  
  const statusDiv = document.getElementById('relistStatus');
  showStatus(statusDiv, '✓ Settings saved!', 'success');
  
  setTimeout(() => {
    statusDiv.classList.remove('show');
  }, 2000);
}

// Handle relist toggle
async function handleRelistToggle(event) {
  const enabled = event.target.checked;
  const interval = parseInt(document.getElementById('relistInterval').value);
  
  if (enabled) {
    chrome.runtime.sendMessage({
      action: 'updateRelistAlarm',
      settings: {
        relistEnabled: true,
        relistInterval: interval,
        relistOnlyExpired: document.getElementById('relistOnlyExpired').checked
      }
    });
  } else {
    chrome.runtime.sendMessage({
      action: 'stopRelistAlarm'
    });
  }
}

// Load settings from storage
async function loadSettings() {
  const result = await chrome.storage.local.get('relistSettings');
  
  if (result.relistSettings) {
    document.getElementById('relistEnabled').checked = result.relistSettings.relistEnabled || false;
    document.getElementById('relistInterval').value = result.relistSettings.relistInterval || 30;
    document.getElementById('relistOnlyExpired').checked = result.relistSettings.relistOnlyExpired !== false;
  }
}

// Load stats
async function loadStats() {
  const result = await chrome.storage.local.get(['autoRelists']);
  document.getElementById('autoRelists').textContent = result.autoRelists || 0;
}

// Show status message
function showStatus(element, message, type) {
  element.textContent = message;
  element.className = `status show ${type}`;
}

