export namespace clip {
	
	export class ClipItem {
	    id: number;
	    kind: number;
	    text: string;
	    preview: string;
	    // Go type: time
	    createdAt: any;
	    pinned: boolean;
	    format: string;
	
	    static createFrom(source: any = {}) {
	        return new ClipItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.kind = source["kind"];
	        this.text = source["text"];
	        this.preview = source["preview"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.pinned = source["pinned"];
	        this.format = source["format"];
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

export namespace snippet {
	
	export class Snippet {
	    id: number;
	    name: string;
	    content: string;
	
	    static createFrom(source: any = {}) {
	        return new Snippet(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.content = source["content"];
	    }
	}

}

