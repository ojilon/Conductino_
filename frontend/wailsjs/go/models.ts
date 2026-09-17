export namespace models {
	
	export class ChatTurn {
	    role: string;
	    content: string;
	
	    static createFrom(source: any = {}) {
	        return new ChatTurn(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.role = source["role"];
	        this.content = source["content"];
	    }
	}
	export class AIRequest {
	    operation: string;
	    query?: string;
	    documentId?: string;
	    sourceId?: string;
	    changeId?: string;
	    selection?: string;
	    selectionText?: string;
	    blockId?: string;
	    workspaceId?: string;
	    customPrompt?: string;
	    includeDocumentContext?: boolean;
	    contextPack?: string;
	    messageHistory?: ChatTurn[];
	    mode?: string;
	    requestId?: string;
	
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
	        this.selectionText = source["selectionText"];
	        this.blockId = source["blockId"];
	        this.workspaceId = source["workspaceId"];
	        this.customPrompt = source["customPrompt"];
	        this.includeDocumentContext = source["includeDocumentContext"];
	        this.contextPack = source["contextPack"];
	        this.messageHistory = this.convertValues(source["messageHistory"], ChatTurn);
	        this.mode = source["mode"];
	        this.requestId = source["requestId"];
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
	
	export class FileTreeNode {
	    id: string;
	    label: string;
	    kind: string;
	    ext?: string;
	    path?: string;
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
	        this.path = source["path"];
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
	export class OpenedDocument {
	    title: string;
	    blocksJSON: string;
	    pageCount?: number;
	    kind?: string;
	    root?: string;
	    reason?: string;
	    detail?: string;
	
	    static createFrom(source: any = {}) {
	        return new OpenedDocument(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.blocksJSON = source["blocksJSON"];
	        this.pageCount = source["pageCount"];
	        this.kind = source["kind"];
	        this.root = source["root"];
	        this.reason = source["reason"];
	        this.detail = source["detail"];
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

