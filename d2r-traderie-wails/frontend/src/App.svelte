<script>
  import { onMount } from 'svelte';
  import { 
    GetAllItems, 
    GetTradingOptions, 
    PostItem, 
    SaveTradingOptions, 
    SetupCookies, 
    TestConnection, 
    HasSavedCookies, 
    GetCookieSetupInstructions,
    SetAuthToken,
    GetAuthToken,
    GetPropertyMapping,
    SavePropertyMappings,
    GenerateSearchURL,
    RefreshListings,
    OpenURLInExtension
  } from '../wailsjs/go/main/App';
  import { EventsOn } from '../wailsjs/runtime/runtime';

  let currentItem = null;
  let traderieProperties = [];
  let allItems = [];
  let propertyMappings = [];
  
  // Auth/Settings management
  let showSettings = false;
  let authToken = '';
  let cookieInput = '';
  let cookieStatus = '';
  let cookieInstructions = '';
  let hasSavedCookies = false;
  let showAdvanced = false;
  
  // Trading options
  let platform = 'PC';
  let mode = 'softcore';
  let ladder = 'Non-Ladder';
  let region = 'Americas';
  let ethereal = false;
  let upgraded = false;
  let unidentified = false;
  
  // New features state
  let searchRange = 20;
  let excludedProperties = new Set();
  let autoRefreshEnabled = false;
  let autoRefreshInterval = 60;
  let isRefreshing = false;
  
    // Pricing options
    let askForOffers = true;
  let priceOffers = []; // Array of offers, each with items array and additional flag
  
  let searchQuery = '';
  let filteredItems = [];
  let showDropdown = false;
  let selectedOfferIndex = -1;
  let selectedItemIndex = -1;
  let isPosting = false;
  let initialized = false;
  let backendVersion = 'unknown';
  
  // Reactive statements to save options on change
  $: if (initialized && (platform || mode || ladder || region || autoRefreshEnabled || autoRefreshInterval || searchRange)) {
    saveOptions();
  }

  async function saveOptions() {
    if (!initialized) return;
    // Only save if we have values
    if (platform && mode && ladder && region) {
      console.log('Saving options:', { platform, mode, ladder, region, autoRefreshEnabled, autoRefreshInterval, searchRange });
      await SaveTradingOptions({ 
        platform, 
        mode, 
        ladder: ladder === 'Ladder', 
        region,
        autoRefreshEnabled,
        autoRefreshInterval,
        searchRange
      });
    }
  }

  async function handleMappingChange() {
    if (propertyMappings.length > 0) {
      await SavePropertyMappings(propertyMappings);
    }
  }

  onMount(async () => {
    // Wait for backend to be ready
    EventsOn('backend-ready', async (data) => {
      console.log('Backend is ready, loading initial data...', data);
      if (data && data.version) {
        backendVersion = data.version;
      }
      
      try {
        // Load all items for autocomplete
        const items = await GetAllItems();
        allItems = items || [];
        console.log(`Loaded ${allItems.length} items`);
        
        // Load trading options
        const opts = await GetTradingOptions();
        console.log('Loaded options from backend:', opts);
        if (opts) {
          platform = opts.platform || 'PC';
          mode = opts.mode || 'softcore';
          ladder = opts.ladder ? 'Ladder' : 'Non-Ladder';
          region = opts.region || 'Americas';
          autoRefreshEnabled = opts.autoRefreshEnabled || false;
          autoRefreshInterval = opts.autoRefreshInterval || 60;
          searchRange = opts.searchRange || 20;
        }
        
        // Check if cookies are saved
        hasSavedCookies = await HasSavedCookies();
        
        // Load auth token
        const token = await GetAuthToken();
        authToken = token || '';
        
        // Load cookie instructions
        const instructions = await GetCookieSetupInstructions();
        cookieInstructions = instructions || '';

        // Mark as initialized so reactive saves can start
        setTimeout(() => {
          initialized = true;
          console.log('Initialization complete, ready to save changes');
        }, 500);
      } catch (err) {
        console.error('Error during initialization:', err);
      }
    });
    
    // Listen for item scans
    EventsOn('item-scanned', (data) => {
      console.log('Item scanned:', data);
      currentItem = data.item;
      traderieProperties = data.traderieProperties || [];
      
      if (traderieProperties.length === 0) {
        console.warn('Warning: No Traderie properties found for this item type.');
      }
      if (data.mappings && data.mappings.length > 0) {
        propertyMappings = data.mappings;
      } else {
        // Fallback
        propertyMappings = currentItem.properties.map(prop => ({
          d2rProp: `${prop.name}: ${prop.value}`,
          traderieProp: prop.name
        }));
      }
      
      // Set item-specific options
      ethereal = currentItem.is_ethereal || false;
      unidentified = !currentItem.is_identified || false;
    });
    
    EventsOn('item-scan-error', (error) => {
      alert(`Error scanning item: ${error}`);
    });
  });
  
  function addPropertyMapping() {
    propertyMappings = [...propertyMappings, { d2rProp: '', traderieProp: '' }];
  }
  
  function removePropertyMapping(index) {
    propertyMappings = propertyMappings.filter((_, i) => i !== index);
  }
  
  function addPriceOffer() {
    priceOffers = [...priceOffers, { items: [{ quantity: 1, itemName: '' }], additional: false }];
  }
  
  function removePriceOffer(index) {
    priceOffers = priceOffers.filter((_, i) => i !== index);
  }
  
  function addItemToOffer(offerIndex) {
    priceOffers[offerIndex].items = [...priceOffers[offerIndex].items, { quantity: 1, itemName: '' }];
    priceOffers = priceOffers; // Trigger reactivity
  }
  
  function removeItemFromOffer(offerIndex, itemIndex) {
    if (priceOffers[offerIndex].items.length > 1) {
      priceOffers[offerIndex].items = priceOffers[offerIndex].items.filter((_, i) => i !== itemIndex);
      priceOffers = priceOffers; // Trigger reactivity
    }
  }
  
  function filterItems(query, offerIndex, itemIndex) {
    console.log('filterItems called:', query, offerIndex, itemIndex, 'allItems count:', allItems.length);
    
    if (query.length < 2) {
      filteredItems = [];
      showDropdown = false;
      return;
    }
    
    const lower = query.toLowerCase();
    filteredItems = allItems
      .filter(item => item.toLowerCase().includes(lower))
      .slice(0, 10); // Max 10 results
    
    console.log('Filtered results:', filteredItems.length, filteredItems.slice(0, 3));
    showDropdown = true;
    selectedOfferIndex = offerIndex;
    selectedItemIndex = itemIndex;
  }
  
  function selectItem(item, offerIndex, itemIndex) {
    priceOffers[offerIndex].items[itemIndex].itemName = item;
    showDropdown = false;
    filteredItems = [];
  }
  
  async function postItem() {
    if (!currentItem) {
      alert('No item to post!');
      return;
    }
    
    if (isPosting) return;
    
    isPosting = true;
    const tradingOpts = { 
      platform, 
      mode, 
      ladder: ladder === 'Ladder', // Convert to boolean for backend
      region, 
      ethereal, 
      upgraded, 
      unidentified,
      mappings: propertyMappings // Include mappings to be learned
    };
    const pricingOpts = { askForOffers, offers: priceOffers };
    
    try {
      await PostItem(currentItem, platform, tradingOpts, pricingOpts);
      await SaveTradingOptions({ 
        platform, 
        mode, 
        ladder: ladder === 'Ladder', 
        region 
      });
      alert('Item posted successfully!');
      currentItem = null;
    } catch (err) {
      const errMsg = String(err);
      console.error('Post failed:', errMsg);
      
      // Check if it's a Cloudflare block or cookie-related error
      if (
        errMsg.toLowerCase().includes('cloudflare') || 
        errMsg.toLowerCase().includes('cookie') || 
        errMsg.toLowerCase().includes('just a moment') ||
        errMsg.includes('403')
      ) {
        alert(`Failed to post: Cloudflare Blocked your request.\n\n⚠️ The "Auth Token" is usually blocked by Traderie. You MUST use the "Cloudflare Bypass (Advanced)" section in Settings to post items.`);
        showSettings = true; // Open settings automatically
        showAdvanced = true; // Show the advanced section
      } else {
        alert(`Failed to post item: ${err}`);
      }
    } finally {
      isPosting = false;
    }
  }
  
  async function saveAuthToken() {
    if (!authToken.trim()) {
      cookieStatus = '❌ Please enter an auth token';
      return;
    }
    
    cookieStatus = '⏳ Saving token...';
    try {
      await SetAuthToken(authToken);
      cookieStatus = '✅ Token saved successfully!';
      setTimeout(() => {
        showSettings = false;
      }, 1500);
    } catch (err) {
      cookieStatus = `❌ Error: ${err}`;
    }
  }

  async function saveCookies() {
    if (!cookieInput.trim()) {
      cookieStatus = '❌ Please paste your cookies first';
      return;
    }
    
    cookieStatus = '⏳ Saving cookies and testing connection...';
    
    try {
      await SetupCookies(cookieInput);
      cookieStatus = '✅ Cookies saved and tested successfully!';
      hasSavedCookies = true;
      cookieInput = ''; // Clear input for security
      
      setTimeout(() => {
        showSettings = false; // Auto-close after success
      }, 2000);
    } catch (err) {
      cookieStatus = `❌ Error: ${err}`;
    }
  }
  
  async function testConnection() {
    cookieStatus = '⏳ Testing connection...';
    
    try {
      await TestConnection();
      cookieStatus = '✅ Connection test successful!';
    } catch (err) {
      cookieStatus = `❌ Connection test failed: ${err}`;
    }
  }
  
  function toggleSettings() {
    showSettings = !showSettings;
    if (showSettings) {
      cookieStatus = ''; // Clear status when opening
    }
  }
  
  function toggleExcludedProperty(propName) {
    if (excludedProperties.has(propName)) {
      excludedProperties.delete(propName);
    } else {
      excludedProperties.add(propName);
    }
    excludedProperties = excludedProperties; // Trigger reactivity
  }

  async function openSearchOnTraderie() {
    if (!currentItem) return;
    try {
      const opts = {
        platform,
        mode,
        ladder: ladder === 'Ladder',
        region
      };
      const url = await GenerateSearchURL(currentItem, searchRange, propertyMappings, Array.from(excludedProperties), opts);
      await OpenURLInExtension(url);
    } catch (err) {
      alert(`Error opening search on extension: ${err}`);
    }
  }

  async function refreshNow() {
    if (isRefreshing) return;
    isRefreshing = true;
    try {
      await RefreshListings();
      alert('Listings refreshed successfully!');
    } catch (err) {
      alert(`Error refreshing listings: ${err}`);
    } finally {
      isRefreshing = false;
    }
  }

  function cancel() {
    currentItem = null;
    propertyMappings = [];
    priceOffers = [];
  }
</script>

<main>
  <div class="header">
    <h1>D2R Traderie <span class="version-tag">{backendVersion}</span></h1>
    <div class="header-actions">
      <div class="cookie-status" class:has-cookies={hasSavedCookies} class:no-cookies={!hasSavedCookies}>
        {hasSavedCookies ? '✅ Cloudflare Bypass Active' : '⚠️ No Cookies'}
      </div>
      <button class="btn-settings" on:click={toggleSettings}>
        ⚙️ Settings
      </button>
    </div>
  </div>
  
  {#if showSettings}
    <div class="settings-panel">
      <div class="settings-header">
        <h2>Authentication Settings</h2>
        <button class="btn-close" on:click={toggleSettings}>✕</button>
      </div>
      
      <div class="auth-section">
        <label for="auth-token">Traderie Auth Token:</label>
        <div class="input-with-button">
          <input 
            id="auth-token"
            type="password"
            bind:value={authToken} 
            placeholder="Paste your auth token here..."
          />
          <button class="btn-save-small" on:click={saveAuthToken}>Save</button>
        </div>
        <p class="help">This token is used to authenticate your requests to Traderie.</p>
      </div>

      <div class="settings-section">
        <h3>Listing Auto-Refresh</h3>
        <div class="form-group">
          <label>Auto-Refresh:</label>
          <div class="checkbox-wrapper">
            <input type="checkbox" bind:checked={autoRefreshEnabled}>
            <span>{autoRefreshEnabled ? 'Enabled' : 'Disabled'}</span>
          </div>
        </div>
        <div class="form-group">
          <label>Interval (hours):</label>
          <select bind:value={autoRefreshInterval}>
            <option value={60}>1 hour</option>
            <option value={120}>2 hours</option>
            <option value={180}>3 hours</option>
            <option value={360}>6 hours</option>
            <option value={720}>12 hours</option>
            <option value={1440}>24 hours</option>
          </select>
        </div>
        <div class="settings-actions">
          <button class="btn-test" on:click={refreshNow} disabled={isRefreshing}>
            {isRefreshing ? '⏳ Refreshing...' : '🔄 Refresh All Listings Now'}
          </button>
        </div>
      </div>

      <div class="advanced-toggle">
        <button class="btn-link" on:click={() => showAdvanced = !showAdvanced}>
          {showAdvanced ? '▼ Hide Cloudflare Bypass' : '▶ Show Cloudflare Bypass (Advanced)'}
        </button>
      </div>
      
      {#if showAdvanced}
        <div class="advanced-section">
          <div class="instructions">
            <pre>{cookieInstructions}</pre>
          </div>
          
          <div class="cookie-input-section">
            <label for="cookie-input">Advanced: Paste all cookies here:</label>
            <textarea 
              id="cookie-input"
              bind:value={cookieInput} 
              placeholder="Alternative: Paste full document.cookie here..."
              rows="3"
            ></textarea>
          </div>
          
          <div class="settings-actions">
            <button class="btn-save" on:click={saveCookies}>💾 Save Cookies</button>
            <button class="btn-test" on:click={testConnection} disabled={!hasSavedCookies}>
              🔌 Test Connection
            </button>
          </div>
        </div>
      {/if}
      
      {#if cookieStatus}
        <div class="cookie-status-message" class:success={cookieStatus.includes('✅')} class:error={cookieStatus.includes('❌')}>
          {cookieStatus}
        </div>
      {/if}
    </div>
  {:else if currentItem}
    <div class="item-card">
      <div class="item-header-row">
        <h2>{currentItem.name}</h2>
        {#if currentItem.quality === 'Rare' || currentItem.quality === 'Magic' || currentItem.quality === 'Crafted'}
          <div class="quality-badge {currentItem.quality.toLowerCase()}">{currentItem.quality}</div>
        {/if}
      </div>
      <p class="quality">{currentItem.quality} {currentItem.type}</p>
      <p class="info">Sockets: {currentItem.sockets} | Ethereal: {currentItem.is_ethereal}</p>
      
      <section>
        <h3>Trading Options</h3>
        <div class="form-group">
          <label>Platform:</label>
          <select bind:value={platform}>
            <option>PC</option>
            <option>PlayStation</option>
            <option>Xbox</option>
            <option>Switch</option>
          </select>
        </div>
        
        <div class="form-group">
          <label>Mode:</label>
          <select bind:value={mode}>
            <option>softcore</option>
            <option>hardcore</option>
          </select>
        </div>
        
        <div class="form-group">
          <label>Ladder:</label>
          <select bind:value={ladder}>
            <option>Ladder</option>
            <option>Non-Ladder</option>
          </select>
        </div>
        
        <div class="form-group">
          <label>Region:</label>
          <select bind:value={region}>
            <option>Americas</option>
            <option>Europe</option>
            <option>Asia</option>
          </select>
        </div>
        
        <div class="checkboxes">
          <label><input type="checkbox" bind:checked={ethereal}> Ethereal</label>
          <label><input type="checkbox" bind:checked={upgraded}> Upgraded</label>
          <label><input type="checkbox" bind:checked={unidentified}> Unidentified</label>
        </div>

        {#if currentItem.quality === 'Rare' || currentItem.quality === 'Magic' || currentItem.quality === 'Crafted' || currentItem.name.toLowerCase() === currentItem.type.toLowerCase()}
          <div class="form-group rarity-group">
            <label>Listing Rarity:</label>
            <select bind:value={currentItem.quality} on:change={() => {
              // Update the mapping if it exists
              const mapping = propertyMappings.find(m => m.traderieProp === 'Rarity');
              if (mapping) {
                mapping.d2rProp = `Quality: ${currentItem.quality}`;
                propertyMappings = propertyMappings;
              }
            }}>
              <option value="Rare">Rare</option>
              <option value="Magic">Magic</option>
              <option value="Crafted">Crafted</option>
              <option value="Normal">Normal</option>
              <option value="Superior">Superior</option>
            </select>
          </div>
        {/if}
      </section>
      
      <section>
        <h3>Search Settings</h3>
        <div class="form-group">
          <label>Fuzzy Range (%):</label>
          <select bind:value={searchRange}>
            <option value={0}>Exact Match</option>
            <option value={5}>5% Range</option>
            <option value={10}>10% Range</option>
            <option value={15}>15% Range</option>
            <option value={20}>20% Range</option>
            <option value={25}>25% Range</option>
            <option value={30}>30% Range</option>
          </select>
        </div>
        <p class="help">Fuzzy range applies to numeric properties (e.g. Defense, Enhanced Defense).</p>
      </section>

      <section>
        <h3>Property Mappings & Exclusion</h3>
        <p class="help" style="font-size: 11px; margin-bottom: 10px;">Check the "X" to exclude a property from Traderie search.</p>
        {#each propertyMappings as mapping, i}
          <div class="property-row">
            <input 
              type="checkbox" 
              checked={!excludedProperties.has(mapping.d2rProp.split(':')[0])} 
              on:change={() => toggleExcludedProperty(mapping.d2rProp.split(':')[0])}
              title="Include in search"
            />
            <select bind:value={mapping.traderieProp} on:change={handleMappingChange}>
              <option value="">Select Traderie property...</option>
              {#each traderieProperties as prop}
                <option value={prop}>{prop}</option>
              {/each}
            </select>
            <span>←</span>
            <select bind:value={mapping.d2rProp} on:change={handleMappingChange}>
              <option value="">Select D2R property...</option>
              {#each currentItem.properties as prop}
                <option value={`${prop.name}: ${prop.value}`}>{prop.name}: {prop.value}</option>
              {/each}
            </select>
            <button class="btn-remove" on:click={() => { removePropertyMapping(i); handleMappingChange(); }}>✕</button>
          </div>
        {/each}
        <button class="btn-add" on:click={addPropertyMapping}>+ Add Property</button>
      </section>
      
      <section>
        <h3>Price / Items Wanted</h3>
        <p class="help" style="font-size: 12px; color: #888;">Loaded {allItems.length} items for search</p>
        <label>
          <input type="checkbox" bind:checked={askForOffers}>
          Ask for offers (no specific items)
        </label>
        
        {#if !askForOffers}
          {#each priceOffers as offer, i}
            <div class="offer-container">
              {#if i > 0}<div class="or-label">OR</div>{/if}
              
              {#each offer.items as item, j}
                <div class="offer-row">
                  {#if j > 0}<span class="and-label">+</span>{/if}
                  <input type="number" bind:value={item.quantity} class="qty-input" placeholder="Qty">
                  <span>×</span>
                  <div class="autocomplete-wrapper">
                    <input 
                      type="text" 
                      bind:value={item.itemName}
                      on:input={(e) => filterItems(e.target.value, i, j)}
                      placeholder="Type to search items..."
                      class="item-search"
                    />
                    {#if showDropdown && selectedOfferIndex === i && selectedItemIndex === j}
                      <div class="dropdown">
                        {#if filteredItems.length > 0}
                          {#each filteredItems as filteredItem}
                            <div class="dropdown-item" on:click={() => selectItem(filteredItem, i, j)}>
                              {filteredItem}
                            </div>
                          {/each}
                        {:else}
                          <div class="dropdown-item" style="color: #888;">
                            No matches found
                          </div>
                        {/if}
                      </div>
                    {/if}
                  </div>
                  {#if j > 0}
                    <button class="btn-remove-item" on:click={() => removeItemFromOffer(i, j)}>✕</button>
                  {:else}
                    <button class="btn-remove" on:click={() => removePriceOffer(i)}>✕ Remove Offer</button>
                  {/if}
                </div>
              {/each}
              
              <div class="offer-actions">
                <button class="btn-add-item" on:click={() => addItemToOffer(i)}>+ Add Item</button>
              </div>
              
              <div class="offer-checkbox">
                <label class="additional">
                  <input type="checkbox" bind:checked={offer.additional}>
                  + additional items
                </label>
              </div>
            </div>
          {/each}
          <button class="btn-add" on:click={addPriceOffer}>+ Add Offer Option (OR)</button>
        {/if}
      </section>
      
      <div class="actions">
        <button class="btn-cancel" on:click={cancel} disabled={isPosting}>✕ Cancel</button>
        <button class="btn-search" on:click={openSearchOnTraderie} disabled={isPosting}>
          🔍 Search on Traderie
        </button>
        <button class="btn-post" on:click={postItem} disabled={isPosting}>
          {isPosting ? '⏳ Posting...' : '✓ Post to Traderie'}
        </button>
      </div>
    </div>
  {:else}
    <div class="waiting">
      <p>⏳ Waiting for item scan...</p>
      <p class="help">Press <kbd>F9</kbd> while holding/hovering an item in D2R</p>
    </div>
  {/if}
</main>

<style>
  :global(body) {
    margin: 0;
    padding: 0;
    background: #121212;
    color: #e0e0e0;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
  }
  
  main {
    padding: 20px;
    max-width: 100%;
  }
  
  .header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 20px;
  }
  
  h1 {
    color: #fff;
    margin: 0;
    flex: 1;
    display: flex;
    align-items: baseline;
    gap: 10px;
  }

  .version-tag {
    font-size: 12px;
    background: #333;
    color: #888;
    padding: 2px 8px;
    border-radius: 10px;
    font-family: monospace;
  }
  
  .header-actions {
    display: flex;
    align-items: center;
    gap: 10px;
  }
  
  .cookie-status {
    padding: 6px 12px;
    border-radius: 4px;
    font-size: 13px;
    font-weight: bold;
  }
  
  .cookie-status.has-cookies {
    background: #1b5e20;
    color: #4caf50;
    border: 1px solid #4caf50;
  }
  
  .cookie-status.no-cookies {
    background: #4a2c00;
    color: #ff9800;
    border: 1px solid #ff9800;
  }
  
  .btn-settings {
    background: #424242;
    color: #fff;
    border: none;
    padding: 8px 16px;
    border-radius: 4px;
    cursor: pointer;
    font-size: 14px;
  }
  
  .btn-settings:hover {
    background: #525252;
  }
  
  .settings-panel {
    background: #1e1e1e;
    border-radius: 8px;
    padding: 20px;
    margin-bottom: 20px;
    border: 2px solid #ff9800;
  }
  
  .settings-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 15px;
  }
  
  .settings-header h2 {
    margin: 0;
    color: #ff9800;
    font-size: 18px;
  }
  
  .btn-close {
    background: #d32f2f;
    color: white;
    border: none;
    padding: 6px 12px;
    border-radius: 4px;
    cursor: pointer;
    font-size: 18px;
    line-height: 1;
  }
  
  .btn-close:hover {
    background: #b71c1c;
  }
  
  .auth-section {
    margin-bottom: 20px;
  }

  .auth-section label {
    display: block;
    margin-bottom: 8px;
    color: #ccc;
    font-weight: bold;
  }

  .input-with-button {
    display: flex;
    gap: 10px;
  }

  .input-with-button input {
    flex: 1;
  }

  .btn-save-small {
    background: #2196F3;
    color: white;
    border: none;
    padding: 8px 16px;
    border-radius: 4px;
    cursor: pointer;
    font-weight: bold;
  }

  .btn-save-small:hover {
    background: #1976D2;
  }

  .advanced-toggle {
    margin-bottom: 10px;
  }

  .btn-link {
    background: none;
    border: none;
    color: #2196F3;
    cursor: pointer;
    font-size: 13px;
    padding: 0;
    text-decoration: underline;
  }

  .advanced-section {
    background: #252525;
    padding: 15px;
    border-radius: 6px;
    margin-top: 10px;
    border: 1px dashed #444;
  }

  .instructions {
    background: #1e1e1e;
    border-radius: 4px;
    padding: 10px;
    margin-bottom: 15px;
    border: 1px solid #333;
  }
  
  .instructions pre {
    margin: 0;
    white-space: pre-wrap;
    word-wrap: break-word;
    font-size: 12px;
    color: #aaa;
    line-height: 1.5;
  }
  
  .cookie-input-section {
    margin-bottom: 15px;
  }
  
  .cookie-input-section label {
    display: block;
    margin-bottom: 8px;
    color: #ccc;
    font-weight: bold;
  }
  
  textarea {
    width: 100%;
    background: #333;
    border: 1px solid #555;
    color: #fff;
    padding: 10px;
    border-radius: 4px;
    font-size: 13px;
    font-family: monospace;
    resize: vertical;
  }
  
  .settings-actions {
    display: flex;
    gap: 10px;
    margin-bottom: 15px;
  }
  
  .btn-save, .btn-test {
    flex: 1;
    padding: 12px;
    border: none;
    border-radius: 6px;
    font-size: 15px;
    cursor: pointer;
    font-weight: bold;
  }
  
  .btn-save {
    background: #2196F3;
    color: white;
  }
  
  .btn-save:hover {
    background: #1976D2;
  }
  
  .settings-section {
    margin-bottom: 25px;
    padding-bottom: 15px;
    border-bottom: 1px solid #333;
  }

  .settings-section h3 {
    color: #ff9800;
    margin-bottom: 15px;
    font-size: 16px;
  }

  .checkbox-wrapper {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .btn-test {
    background: #4CAF50;
    color: white;
  }
  
  .btn-test:hover {
    background: #388E3C;
  }
  
  .btn-test:disabled {
    background: #424242;
    color: #888;
    cursor: not-allowed;
  }
  
  .cookie-status-message {
    padding: 12px;
    border-radius: 4px;
    text-align: center;
    font-weight: bold;
  }
  
  .cookie-status-message.success {
    background: #1b5e20;
    color: #4caf50;
    border: 1px solid #4caf50;
  }
  
  .cookie-status-message.error {
    background: #4a1414;
    color: #f44336;
    border: 1px solid #f44336;
  }
  
  .waiting {
    text-align: center;
    padding: 60px 20px;
  }
  
  .waiting p {
    font-size: 18px;
    margin: 10px 0;
  }
  
  .help {
    color: #888;
  }
  
  kbd {
    background: #333;
    padding: 4px 8px;
    border-radius: 4px;
    font-family: monospace;
    border: 1px solid #555;
  }
  
  .item-card {
    background: #1e1e1e;
    border-radius: 8px;
    padding: 20px;
    border: 1px solid #333;
  }
  
  h2 {
    margin: 0;
    color: #ffd700;
  }
  
  .item-header-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 5px;
  }

  .quality-badge {
    padding: 2px 8px;
    border-radius: 4px;
    font-size: 12px;
    font-weight: bold;
    text-transform: uppercase;
  }

  .quality-badge.rare {
    background: #ffff00;
    color: #000;
  }

  .quality-badge.magic {
    background: #0000ff;
    color: #fff;
  }

  .quality-badge.crafted {
    background: #ff8c00;
    color: #fff;
  }

  .quality {
    color: #888;
    font-style: italic;
    margin: 5px 0;
  }
  
  .info {
    color: #aaa;
    font-family: monospace;
    font-size: 14px;
  }
  
  section {
    margin: 20px 0;
    padding: 15px;
    background: #252525;
    border-radius: 6px;
  }
  
  h3 {
    margin-top: 0;
    color: #fff;
    font-size: 16px;
  }
  
  .form-group {
    display: grid;
    grid-template-columns: 120px 1fr;
    align-items: center;
    margin: 10px 0;
  }
  
  label {
    color: #ccc;
  }
  
  select, input[type="text"], input[type="number"] {
    background: #333;
    border: 1px solid #555;
    color: #fff;
    padding: 8px;
    border-radius: 4px;
    font-size: 14px;
  }
  
  select {
    width: 100%;
  }
  
  .checkboxes {
    display: flex;
    gap: 20px;
    margin-top: 10px;
  }
  
  .rarity-group {
    margin-top: 15px;
    padding-top: 10px;
    border-top: 1px solid #333;
  }
  
  .checkboxes label {
    display: flex;
    align-items: center;
    gap: 6px;
  }
  
  .property-row, .offer-row {
    display: flex;
    align-items: center;
    gap: 10px;
    margin: 10px 0;
  }
  
  .property-row select {
    flex: 1;
  }
  
  .offer-container {
    margin: 15px 0;
  }
  
  .offer-row {
    display: flex;
    align-items: center;
    gap: 10px;
  }
  
  .offer-checkbox {
    padding-left: 10px;
    margin-top: 8px;
  }
  
  .or-label {
    width: 100%;
    text-align: center;
    color: #888;
    font-weight: bold;
    margin: 10px 0;
    font-size: 14px;
  }
  
  .and-label {
    color: #888;
    font-weight: bold;
    margin: 0 5px;
  }
  
  .offer-actions {
    margin-top: 8px;
    padding-left: 10px;
  }
  
  .btn-add-item {
    background: #424242;
    color: #fff;
    border: none;
    padding: 6px 12px;
    border-radius: 4px;
    cursor: pointer;
    font-size: 13px;
  }
  
  .btn-add-item:hover {
    background: #525252;
  }
  
  .btn-remove-item {
    background: #d32f2f;
    color: white;
    border: none;
    padding: 6px 12px;
    border-radius: 4px;
    cursor: pointer;
  }
  
  .btn-remove-item:hover {
    background: #b71c1c;
  }
  
  .qty-input {
    width: 60px;
  }
  
  .autocomplete-wrapper {
    flex: 1;
    position: relative;
    min-width: 0; /* Prevents flex item overflow */
  }
  
  .item-search {
    width: 100%;
  }
  
  .dropdown {
    position: absolute;
    top: 100%;
    left: 0;
    right: 0;
    background: #2a2a2a;
    border: 2px solid #4CAF50;
    border-top: none;
    max-height: 200px;
    overflow-y: auto;
    z-index: 9999;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.5);
    margin-top: 2px;
  }
  
  .dropdown-item {
    padding: 10px;
    cursor: pointer;
    border-bottom: 1px solid #333;
  }
  
  .dropdown-item:hover {
    background: #3a3a3a;
  }
  
  .additional {
    white-space: nowrap;
  }
  
  .btn-remove {
    background: #d32f2f;
    color: white;
    border: none;
    padding: 6px 12px;
    border-radius: 4px;
    cursor: pointer;
  }
  
  .btn-remove:hover {
    background: #b71c1c;
  }
  
  .btn-add {
    background: #424242;
    color: #fff;
    border: none;
    padding: 8px 16px;
    border-radius: 4px;
    cursor: pointer;
    margin-top: 10px;
  }
  
  .btn-add:hover {
    background: #525252;
  }
  
  .actions {
    display: flex;
    gap: 10px;
    margin-top: 20px;
  }
  
  .btn-cancel, .btn-post {
    flex: 1;
    padding: 12px;
    border: none;
    border-radius: 6px;
    font-size: 16px;
    cursor: pointer;
    font-weight: bold;
  }
  
  .btn-cancel {
    background: #424242;
    color: #fff;
  }
  
  .btn-cancel:hover {
    background: #525252;
  }
  
  .btn-search {
    flex: 1;
    padding: 12px;
    border: none;
    border-radius: 6px;
    font-size: 16px;
    cursor: pointer;
    font-weight: bold;
    background: #ff9800;
    color: #000;
  }

  .btn-search:hover {
    background: #f57c00;
  }

  .btn-post {
    background: #2196F3;
    color: white;
  }
  
  .btn-post:disabled {
    background: #424242;
    color: #888;
    cursor: not-allowed;
  }
</style>
