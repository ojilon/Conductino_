# 03-21 Canvas ownership sketch (PdfView you will rewrite)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: own zoom + text layer. Where: `PdfView.tsx` + `PdfView.css`, pdf.js canvas leaf, selectable text divs.
Sketch: `renderPage(canvas, viewport, pdfBytes)` TS fn; text layer offsets map to blockIds; selection -> TextSelection{range}.
Exercise: implement zoom as pure viewport math (no React state for pixels).
Done when: zoom + selection survive re-render.
