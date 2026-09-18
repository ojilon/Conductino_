/**
 * PDF canvas leaf (issues 18-21).
 *
 * Renders workspace PDFs page-by-page on canvas with a pdf.js text layer,
 * so native text selection (and the whole selection → AI stack) works on
 * real page images instead of extracted paragraphs. Extracted blocks still
 * exist for AI context, search, and tools — this view is display only.
 *
 * - Bytes come from App.ReadRawFile (Resolve-jailed, 30 MiB cap).
 * - Worker is pdf.worker.min.mjs (frontend/public → dist, served by Wails);
 *   loaded via fetch→blob URL so it runs under the desktop custom scheme.
 * - Virtualization (issue 20): only near-visible pages rasterize; far pages
 *   drop back to sized placeholders (IntersectionObserver + rootMargin).
 * - Page containers carry data-block-id of the matching extraction
 *   page-break block, so chat anchors to a real block (issue 21).
 */

import { useEffect, useMemo, useRef, useState } from "react";
import * as pdfjs from "pdfjs-dist";
import { useApp } from "../../state/appState";
import { openRawFile } from "../../services/backend";
import { SelectionToolbar } from "./DocumentView";
import { Icon } from "../../components/icons";
import type { Document } from "../../types/domain";
import "./PdfView.css";

const RENDER_SCALE = 1.5;

let workerReady: Promise<void> | null = null;

/** Load the pdf.js worker via fetch→blob (works under Wails custom scheme). */
function ensureWorker(): Promise<void> {
  if (!workerReady) {
    workerReady = (async () => {
      const url = new URL("pdf.worker.min.mjs", window.location.href).href;
      try {
        const res = await fetch(url);
        if (!res.ok) throw new Error(`worker ${res.status}`);
        const blob = await res.blob();
        pdfjs.GlobalWorkerOptions.workerSrc = URL.createObjectURL(blob);
      } catch {
        // Fall back to the direct URL (plain browser dev serves it fine).
        pdfjs.GlobalWorkerOptions.workerSrc = url;
      }
    })();
  }
  return workerReady;
}

type PageState = "idle" | "rendering" | "ready" | "error";

function PdfPage({
  pdf,
  pageNumber,
  blockId,
  register,
}: {
  pdf: pdfjs.PDFDocumentProxy;
  pageNumber: number;
  blockId: string;
  register: (n: number, el: HTMLDivElement | null) => void;
}) {
  const hostRef = useRef<HTMLDivElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const textRef = useRef<HTMLDivElement>(null);
  const [dims, setDims] = useState<{ w: number; h: number } | null>(null);
  const [state, setState] = useState<PageState>("idle");
  const renderToken = useRef(0);

  // Page size first (cheap, no raster) so placeholders hold layout.
  useEffect(() => {
    let live = true;
    pdf.getPage(pageNumber).then((page) => {
      if (!live) return;
      const v = page.getViewport({ scale: RENDER_SCALE });
      setDims({ w: v.width, h: v.height });
    }).catch(() => live && setState("error"));
    return () => {
      live = false;
    };
  }, [pdf, pageNumber]);

  // Rasterize when near-visible; release when far away (issue 20).
  useEffect(() => {
    const host = hostRef.current;
    if (!host) return;
    const io = new IntersectionObserver(
      (entries) => {
        const vis = entries[0]?.isIntersecting ?? false;
        const my = ++renderToken.current;
        if (!vis) {
          // Drop the raster back to a placeholder to bound canvas memory.
          setState((s) => (s === "ready" ? "idle" : s));
          return;
        }
        setState("rendering");
        pdf.getPage(pageNumber).then(async (page) => {
          if (renderToken.current !== my) return; // superseded while loading
          const viewport = page.getViewport({ scale: RENDER_SCALE });
          const canvas = canvasRef.current;
          const textDiv = textRef.current;
          if (!canvas || !textDiv) return;
          canvas.width = Math.floor(viewport.width);
          canvas.height = Math.floor(viewport.height);
          textDiv.replaceChildren();
          try {
            await page.render({ canvas, viewport }).promise;
            if (renderToken.current !== my) return;
            const layer = new pdfjs.TextLayer({
              textContentSource: page.streamTextContent(),
              container: textDiv,
              viewport,
            });
            await layer.render();
            if (renderToken.current === my) setState("ready");
          } catch {
            if (renderToken.current === my) setState("error");
          }
        }).catch(() => {
          if (renderToken.current === my) setState("error");
        });
      },
      { rootMargin: "800px 0px" },
    );
    io.observe(host);
    return () => io.disconnect();
  }, [pdf, pageNumber]);

  useEffect(() => {
    register(pageNumber, hostRef.current);
    return () => register(pageNumber, null);
  }, [pageNumber, register]);

  return (
    <div
      ref={hostRef}
      data-block-id={blockId}
      className="pdf-page"
      style={dims ? { width: dims.w, height: dims.h } : { aspectRatio: "1 / 1.294", width: "100%" }}
    >
      <canvas ref={canvasRef} className="pdf-canvas" />
      <div ref={textRef} className="textLayer" />
      {state !== "ready" && (
        <div className="pdf-page-fallback">
          {state === "error" ? (
            <span className="flex items-center gap-1.5 text-[11px] text-mute">
              <Icon name="alert" size={12} /> page failed to render
            </span>
          ) : (
            <span className="text-[11px] text-mute">page {pageNumber}…</span>
          )}
        </div>
      )}
    </div>
  );
}

export default function PdfView({ doc }: { doc: Document }) {
  const { state, dispatch } = useApp();
  const [pdf, setPdf] = useState<pdfjs.PDFDocumentProxy | null>(null);
  const [numPages, setNumPages] = useState(0);
  const [error, setError] = useState<string | null>(null);
  const pagesRef = useRef(new Map<number, HTMLDivElement | null>());

  const relPath = doc.metadata.path;
  // Page-break block ids from extraction, in order — page N anchors to the
  // Nth break (issue 21). Fallback: first block, so chat always has an id.
  const pageBreakIds = useMemo(
    () => doc.blocks.filter((b) => b.type === "page").map((b) => b.id),
    [doc.blocks],
  );
  const blockIdForPage = (n: number) =>
    pageBreakIds[n - 1] ?? doc.blocks[0]?.id ?? doc.id;

  useEffect(() => {
    if (!relPath) {
      setError("No file path for this PDF — reopen it from the library.");
      return;
    }
    let live = true;
    let task: pdfjs.PDFDocumentLoadingTask | null = null;
    (async () => {
      try {
        await ensureWorker();
        const data = await openRawFile(relPath);
        if (!live) return;
        if (!data) {
          setError("Reading PDFs needs the desktop app — run `wails dev`.");
          return;
        }
        task = pdfjs.getDocument({ data });
        const proxy = await task.promise;
        if (!live) {
          await task.destroy();
          return;
        }
        setPdf(proxy);
        setNumPages(proxy.numPages);
      } catch (e) {
        if (live) setError(e instanceof Error ? e.message : "Could not render PDF.");
      }
    })();
    return () => {
      live = false;
      task?.destroy().catch(() => undefined);
    };
  }, [relPath]);

  const onMouseUp = () => {
    const sel = window.getSelection();
    const text = sel?.toString().trim() ?? "";
    if (!text || text.length < 4) {
      if (state.selection) dispatch({ type: "selection.set", selection: null });
      return;
    }
    const node = sel?.anchorNode;
    const el = node instanceof Element ? node : node?.parentElement;
    const pageEl = el?.closest(".pdf-page");
    if (!pageEl) {
      dispatch({ type: "selection.set", selection: null });
      return;
    }
    const domRange = sel && sel.rangeCount > 0 ? sel.getRangeAt(0) : null;
    const rect = domRange?.getBoundingClientRect();
    if (!rect) return;
    dispatch({
      type: "selection.set",
      selection: {
        documentId: doc.id,
        blockId: pageEl.getAttribute("data-block-id") as string,
        text,
        x: rect.left + rect.width / 2,
        y: rect.top,
      },
    });
  };

  if (error) {
    return (
      <div className="mx-auto max-w-[700px] px-8 py-6">
        <div className="rounded-lg border border-line-soft bg-cream-50 px-4 py-6 text-center">
          <p className="text-[13px] font-medium text-ink-900">Couldn’t render this PDF</p>
          <p className="mt-1 text-[12px] text-mute">{error}</p>
          <p className="mt-2 text-[11.5px] text-mute">Extracted text (if any) is still searchable via chat and tools.</p>
        </div>
      </div>
    );
  }

  if (!pdf) {
    return (
      <div className="mx-auto max-w-[700px] px-8 py-6">
        <p className="py-10 text-center text-[12.5px] text-mute">Loading PDF pages…</p>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-[760px] px-8 py-6" data-doc-area onMouseUp={onMouseUp}>
      <div className="mb-4 flex items-center justify-between border-b border-line pb-2.5 text-[12px] text-mute">
        <span className="truncate">{doc.metadata.title}</span>
        <span className="shrink-0 pl-3">{numPages} pages · canvas + selectable text</span>
      </div>
      <div className="space-y-5">
        {Array.from({ length: numPages }, (_, i) => (
          <PdfPage
            key={`${doc.id}-p${i + 1}`}
            pdf={pdf}
            pageNumber={i + 1}
            blockId={blockIdForPage(i + 1)}
            register={(n, el) => {
              if (el) pagesRef.current.set(n, el);
              else pagesRef.current.delete(n);
            }}
          />
        ))}
      </div>
      <SelectionToolbar doc={doc} />
    </div>
  );
}
