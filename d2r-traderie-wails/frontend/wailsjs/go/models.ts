export namespace models {
	
	export class DamageRange {
	    min: number;
	    max: number;
	    type: string;
	
	    static createFrom(source: any = {}) {
	        return new DamageRange(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.min = source["min"];
	        this.max = source["max"];
	        this.type = source["type"];
	    }
	}
	export class Requirements {
	    level?: number;
	    strength?: number;
	    dexterity?: number;
	    intelligence?: number;
	
	    static createFrom(source: any = {}) {
	        return new Requirements(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.level = source["level"];
	        this.strength = source["strength"];
	        this.dexterity = source["dexterity"];
	        this.intelligence = source["intelligence"];
	    }
	}
	export class Property {
	    name: string;
	    value: any;
	
	    static createFrom(source: any = {}) {
	        return new Property(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.value = source["value"];
	    }
	}
	export class Item {
	    name: string;
	    type: string;
	    quality: string;
	    properties: Property[];
	    requirements?: Requirements;
	    sockets: number;
	    defense?: number;
	    damage?: DamageRange;
	    item_level?: number;
	    is_identified: boolean;
	    is_ethereal: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Item(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.quality = source["quality"];
	        this.properties = this.convertValues(source["properties"], Property);
	        this.requirements = this.convertValues(source["requirements"], Requirements);
	        this.sockets = source["sockets"];
	        this.defense = source["defense"];
	        this.damage = this.convertValues(source["damage"], DamageRange);
	        this.item_level = source["item_level"];
	        this.is_identified = source["is_identified"];
	        this.is_ethereal = source["is_ethereal"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	

}

export namespace traderie {
	
	export class TraderieTag {
	    tag_id: number;
	    tag: string;
	    category: string;
	
	    static createFrom(source: any = {}) {
	        return new TraderieTag(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tag_id = source["tag_id"];
	        this.tag = source["tag"];
	        this.category = source["category"];
	    }
	}
	export class TraderieProperty {
	    property_id: number;
	    property: string;
	    options: string[];
	    type: string;
	    required: boolean;
	    min?: number;
	    max?: number;
	
	    static createFrom(source: any = {}) {
	        return new TraderieProperty(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.property_id = source["property_id"];
	        this.property = source["property"];
	        this.options = source["options"];
	        this.type = source["type"];
	        this.required = source["required"];
	        this.min = source["min"];
	        this.max = source["max"];
	    }
	}
	export class TraderieItem {
	    id: string;
	    name: string;
	    img: string;
	    type: string;
	    description: string;
	    properties: TraderieProperty[];
	    tags: TraderieTag[];
	
	    static createFrom(source: any = {}) {
	        return new TraderieItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.img = source["img"];
	        this.type = source["type"];
	        this.description = source["description"];
	        this.properties = this.convertValues(source["properties"], TraderieProperty);
	        this.tags = this.convertValues(source["tags"], TraderieTag);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	

}

