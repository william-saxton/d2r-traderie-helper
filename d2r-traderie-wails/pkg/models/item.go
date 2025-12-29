package models

// Item represents a D2R item with all its properties
type Item struct {
	Name         string            `json:"name"`
	Type         string            `json:"type"`
	Quality      string            `json:"quality"`      // Normal, Magic, Rare, Unique, Set, Crafted, Rune, Gem
	CraftedType  string            `json:"crafted_type,omitempty"` // Caster, Blood, Hitpower, Safety (for crafted items only)
	Properties   []Property        `json:"properties"`
	Requirements *Requirements     `json:"requirements,omitempty"`
	Sockets      int               `json:"sockets"`
	Defense      int               `json:"defense,omitempty"`      // For armor
	Damage       *DamageRange      `json:"damage,omitempty"`       // For weapons
	ItemLevel    int               `json:"item_level,omitempty"`
	IsIdentified bool              `json:"is_identified"`
	IsEthereal   bool              `json:"is_ethereal"`
}

// Property represents a single item property/stat
type Property struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"` // Can be int, string, or range
}

// Requirements for equipping the item
type Requirements struct {
	Level      int `json:"level,omitempty"`
	Strength   int `json:"strength,omitempty"`
	Dexterity  int `json:"dexterity,omitempty"`
	Intelligence int `json:"intelligence,omitempty"`
}

// DamageRange for weapons
type DamageRange struct {
	Min int `json:"min"`
	Max int `json:"max"`
	Type string `json:"type"` // Physical, Fire, Cold, Lightning, Poison, Magic
}

// TraderieItem represents the format expected by Traderie API (listings/create)
type TraderieItem struct {
	AcceptListingPrice  bool                   `json:"acceptListingPrice"`
	Captcha             string                 `json:"captcha"`
	CaptchaManaged      bool                   `json:"captchaManaged"`
	CurrencyGroupPrices []CurrencyGroupPrice    `json:"currencyGroupPrices"`
	EndTime             string                 `json:"endTime"`
	Free                bool                   `json:"free"`
	Item                string                 `json:"item"`     // Traderie Item ID
	ItemMode            *string                `json:"itemMode"` // null
	ItemType            string                 `json:"itemType"` // e.g., "sets", "uniques"
	MakeOffer           bool                   `json:"makeOffer"`
	NeedMaterials       bool                   `json:"needMaterials"`
	OfferBells          bool                   `json:"offerBells"`
	OfferNmt            bool                   `json:"offerNmt"`
	OfferWishlist       bool                   `json:"offerWishlist"`
	OfferWishlistId     string                 `json:"offerWishlistId"`
	Selling             bool                   `json:"selling"`
	StandingListing     bool                   `json:"standingListing"`
	StockListing        bool                   `json:"stockListing"`
	TouchTrading        bool                   `json:"touchTrading"`
	Wishlist            string                 `json:"wishlist"`
	Amount              string                 `json:"amount"` // e.g., "1"
	Properties          []TraderieListingProp  `json:"properties"`
	Offers              []string               `json:"offers,omitempty"` // For backward compatibility or specific use cases
	Items               []TraderieListingItem  `json:"items,omitempty"`
}

// TraderieListingItem represents the item being listed in the "items" array
type TraderieListingItem struct {
	Quantity   int           `json:"quantity"`
	Diy        bool          `json:"diy"`
	CanCatalog bool          `json:"canCatalog"`
	Properties []interface{} `json:"properties"` // Using interface{} to allow full metadata from Traderie database
	Variant    *string       `json:"variant"`
	Value      string        `json:"value"` // Item ID
	Label      string        `json:"label"` // Item Name
	Variants   *string       `json:"variants"`
	ImgURL     string        `json:"img_url"`
	Index      int           `json:"index"`
	Group      int           `json:"group"`
	OfferProps []string      `json:"offerProps"`
}

// CurrencyGroupPrice represents a group of items offered for the listing
type CurrencyGroupPrice struct {
	Items []PriceItem `json:"items"`
}

// PriceItem represents an item within a currency group
type PriceItem struct {
	Quantity int    `json:"quantity"`
	Item     string `json:"item"`     // Item ID
	ItemType string `json:"itemType"` // Item Type (e.g., "runes", "uniques")
}

// TraderieListingProp represents a property in the listings/create request
type TraderieListingProp struct {
	ID        int         `json:"id"`       // property_id
	Property  string      `json:"property"` // property name
	Option    interface{} `json:"option"`   // value (string, number, or bool)
	Type      string      `json:"type"`     // "string", "number", "bool"
	Preferred bool        `json:"preferred"`
}

// Price information for Traderie listing
type Price struct {
	Amount   float64 `json:"amount,omitempty"`
	Currency string  `json:"currency,omitempty"` // "rune", "fg", etc.
}

