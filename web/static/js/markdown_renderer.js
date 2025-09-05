// File: web/static/js/markdown_renderer.js
// MarkdownRenderer: Parse Markdown to HTML and sanitize it before inserting into the DOM.
// Requires global `marked` (Markdown parser) and `DOMPurify` (HTML sanitizer).

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

  // Streaming helper: accumulate chunks and render progressively
  function createStreamRenderer(targetEl, options) {
    assertDeps();
    
    let buffer = '';
    let rafId = null;
    let timeoutId = null;
    const debounceMs = (options && options.debounceMs) || 100;
    let isDestroyed = false;

    function doRender() {
      if (isDestroyed) return;
      
      rafId = null;
      
      try {
        renderMarkdownTo(targetEl, buffer);
        
        // Auto-scroll to bottom
        const scroller = targetEl.closest('.messages') || targetEl.parentElement;
        if (scroller) {
          scroller.scrollTop = scroller.scrollHeight;
        }
      } catch (err) {
        console.warn('MarkdownRenderer render error:', err);
        // Fallback to plain text on render error
        targetEl.textContent = buffer;
      }
    }

    function scheduleRender(immediate = false) {
      if (isDestroyed) return;
      
      // Clear any existing scheduled renders
      if (rafId !== null) {
        cancelAnimationFrame(rafId);
        rafId = null;
      }
      if (timeoutId !== null) {
        clearTimeout(timeoutId);
        timeoutId = null;
      }

      if (immediate) {
        rafId = requestAnimationFrame(doRender);
      } else {
        // Debounced render for smoother streaming
        timeoutId = setTimeout(() => {
          timeoutId = null;
          rafId = requestAnimationFrame(doRender);
        }, debounceMs);
      }
    }

    return {
      append(chunk) {
        if (isDestroyed || !chunk) return;
        
        chunk = String(chunk);
        buffer += chunk;
        
        // Render immediately on newlines, otherwise debounce
        const hasNewline = chunk.includes('\n');
        scheduleRender(hasNewline);
      },
      
      set(markdown) {
        if (isDestroyed) return;
        
        buffer = String(markdown || '');
        scheduleRender(true);
      },
      
      clear() {
        if (isDestroyed) return;
        
        buffer = '';
        scheduleRender(true);
      },
      
      get() {
        return buffer;
      },
      
      flush() {
        if (isDestroyed) return;
        
        // Cancel any pending renders and do immediate final render
        if (rafId !== null) {
          cancelAnimationFrame(rafId);
          rafId = null;
        }
        if (timeoutId !== null) {
          clearTimeout(timeoutId);
          timeoutId = null;
        }
        
        doRender();
      },
      
      destroy() {
        isDestroyed = true;
        
        if (rafId !== null) {
          cancelAnimationFrame(rafId);
          rafId = null;
        }
        if (timeoutId !== null) {
          clearTimeout(timeoutId);
          timeoutId = null;
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
