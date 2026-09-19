export namespace models {
	
	export class FocusedChange {
	    id?: string;
	    op?: string;
	    blockId?: string;
	    oldContent?: string;
	    newContent?: string;
	
	    static createFrom(source: any = {}) {
	        return new FocusedChange(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.op = source["op"];
	        this.blockId = source["blockId"];
	        this.oldContent = source["oldContent"];
	        this.newContent = source["newContent"];
	    }
	}
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
	    summaryContent?: string;
	    primarySummaryId?: string;
	    mentionIds?: string[];
	    summaryPath?: string;
	    focusedChange?: FocusedChange;
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
	        this.summaryContent = source["summaryContent"];
	        this.primarySummaryId = source["primarySummaryId"];
	        this.mentionIds = source["mentionIds"];
	        this.summaryPath = source["summaryPath"];
	        this.focusedChange = this.convertValues(source["focusedChange"], FocusedChange);
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
	export class ChangeRecord {
	    id: string;
	    documentId: string;
	    workspaceId?: string;
	    type: string;
	    blockId?: string;
	    oldContent?: string;
	    newContent?: string;
	    status: string;
	    sourceId?: string;
	    activityId?: string;
	    createdAt?: number;
	
	    static createFrom(source: any = {}) {
	        return new ChangeRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.documentId = source["documentId"];
	        this.workspaceId = source["workspaceId"];
	        this.type = source["type"];
	        this.blockId = source["blockId"];
	        this.oldContent = source["oldContent"];
	        this.newContent = source["newContent"];
	        this.status = source["status"];
	        this.sourceId = source["sourceId"];
	        this.activityId = source["activityId"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class ChatMessageRecord {
	    id: string;
	    threadId: string;
	    role: string;
	    content: string;
	    documentId?: string;
	    toolTrace?: string;
	    createdAt?: number;
	
	    static createFrom(source: any = {}) {
	        return new ChatMessageRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.threadId = source["threadId"];
	        this.role = source["role"];
	        this.content = source["content"];
	        this.documentId = source["documentId"];
	        this.toolTrace = source["toolTrace"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class ChatThreadRecord {
	    id: string;
	    workspaceId: string;
	    documentId?: string;
	    title?: string;
	    createdAt?: number;
	    updatedAt?: number;
	
	    static createFrom(source: any = {}) {
	        return new ChatThreadRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.workspaceId = source["workspaceId"];
	        this.documentId = source["documentId"];
	        this.title = source["title"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	
	export class DocumentRecord {
	    id: string;
	    workspaceId: string;
	    kind: string;
	    sourceId?: string;
	    title: string;
	    blocksJson?: string;
	    metaJson?: string;
	    updatedAt?: number;
	
	    static createFrom(source: any = {}) {
	        return new DocumentRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.workspaceId = source["workspaceId"];
	        this.kind = source["kind"];
	        this.sourceId = source["sourceId"];
	        this.title = source["title"];
	        this.blocksJson = source["blocksJson"];
	        this.metaJson = source["metaJson"];
	        this.updatedAt = source["updatedAt"];
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

}

