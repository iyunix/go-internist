// File: web/static/js/markdown_renderer.js
// MarkdownRenderer: Parse Markdown to HTML and sanitize it before inserting into the DOM.
// Requires global `marked` (Markdown parser) and `DOMPurify` (HTML sanitizer). [Docs]
// - marked.parse(md) -> HTML string [41]
// - DOMPurify.sanitize(html) -> safe HTML string [42]
//
// [41] marked: https://marked.js.org
// [42] DOMPurify: https://github.com/cure53/DOMPurify

(function (global) {
  'use strict';

  // Verify dependencies are present at load time
  function assertDeps() {
    if (typeof global.marked === 'undefined' || typeof global.marked.parse !== 'function') {
      throw new Error('MarkdownRenderer: missing dependency "marked" with marked.parse'); 
    }
    if (typeof global.DOMPurify === 'undefined' || typeof global.DOMPurify.sanitize !== 'function') {
      throw new Error('MarkdownRenderer: missing dependency "DOMPurify" with DOMPurify.sanitize'); 
    }
  }

  // Default configuration for marked
  const defaultMarkedOptions = {
    gfm: true,
    breaks: true
  };

  // Initialize marked once with defaults; callers can override via MarkdownRenderer.configure
  function initMarked(opts) {
    if (typeof global.marked?.setOptions === 'function') {
      global.marked.setOptions({ ...defaultMarkedOptions, ...(opts || {}) });
    }
  }

  // Core render function: markdown string -> sanitized HTML into target element
  function renderMarkdownTo(targetEl, markdown) {
    assertDeps();
    const md = typeof markdown === 'string' ? markdown : '';
    const html = global.marked.parse(md);
    const safe = global.DOMPurify.sanitize(html);
    targetEl.innerHTML = safe;
  }

  // Streaming helper: accumulates chunks and re-renders efficiently
  function createStreamRenderer(targetEl, options) {
    assertDeps();
    let buffer = '';
    let rafId = null;

    // Optional throttled render using requestAnimationFrame
    function scheduleRender() {
      if (rafId !== null) return;
      rafId = global.requestAnimationFrame(function () {
        rafId = null;
        renderMarkdownTo(targetEl, buffer);
        // Keep scrolled to bottom if container is in a scrollable area
        try {
          const scroller = targetEl.closest('.messages') || targetEl.parentElement || document.documentElement;
          scroller.scrollTop = scroller.scrollHeight;
        } catch (_) { /* no-op */ }
      });
    }

    return {
      // Append a streamed chunk of Markdown text
      append(chunk) {
        if (!chunk) return;
        buffer += String(chunk);
        scheduleRender();
      },
      // Replace the entire content with a new Markdown string
      set(markdown) {
        buffer = String(markdown || '');
        scheduleRender();
      },
      // Clear all content
      clear() {
        buffer = '';
        scheduleRender();
      },
      // Get current buffered Markdown (not HTML)
      get() {
        return buffer;
      },
      // Force an immediate render (synchronously)
      flush() {
        if (rafId !== null) {
          global.cancelAnimationFrame(rafId);
          rafId = null;
        }
        renderMarkdownTo(targetEl, buffer);
      },
      // Destroy references
      destroy() {
        if (rafId !== null) {
          global.cancelAnimationFrame(rafId);
          rafId = null;
        }
        buffer = '';
      }
    };
  }

  // Public API
  const MarkdownRenderer = {
    // Configure marked options globally (e.g., { gfm: true, breaks: true })
    configure(markedOptions) {
      initMarked(markedOptions);
    },
    // One-shot render: convert Markdown to sanitized HTML in the given element
    render(targetEl, markdown) {
      renderMarkdownTo(targetEl, markdown);
    },
    // Create a streaming renderer bound to an element
    createStream(targetEl, options) {
      return createStreamRenderer(targetEl, options);
    }
  };

  // Initialize defaults
  initMarked();

  // Attach to window
  global.MarkdownRenderer = MarkdownRenderer;

})(window);
