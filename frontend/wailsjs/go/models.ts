export namespace models {
	
	export class AIRequest {
	    operation: string;
	    query?: string;
	    documentId?: string;
	    sourceId?: string;
	    changeId?: string;
	    selection?: string;
	
	    static createFrom(source: any = {}) {
	        return new AIRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.operation = source["operation"];
	        this.query = source["query"];
	        this.documentId = source["documentId"];
	        this.sourceId = source["sourceId"];
	        this.changeId = source["changeId"];
	        this.selection = source["selection"];
	    }
	}
	export class FileTreeNode {
	    id: string;
	    label: string;
	    kind: string;
	    ext?: string;
	    children?: FileTreeNode[];
	    documentId?: string;
	
	    static createFrom(source: any = {}) {
	        return new FileTreeNode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.kind = source["kind"];
	        this.ext = source["ext"];
	        this.children = this.convertValues(source["children"], FileTreeNode);
	        this.documentId = source["documentId"];
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
	export class Source {
	    id: string;
	    kind: string;
	    title: string;
	    origin: string;
	    url?: string;
	    typeLabel: string;
	    abstract: string;
	    rank?: number;
	    relevance?: string;
	    saved: boolean;
	    readerDocId?: string;
	
	    static createFrom(source: any = {}) {
	        return new Source(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.kind = source["kind"];
	        this.title = source["title"];
	        this.origin = source["origin"];
	        this.url = source["url"];
	        this.typeLabel = source["typeLabel"];
	        this.abstract = source["abstract"];
	        this.rank = source["rank"];
	        this.relevance = source["relevance"];
	        this.saved = source["saved"];
	        this.readerDocId = source["readerDocId"];
	    }
	}

}

