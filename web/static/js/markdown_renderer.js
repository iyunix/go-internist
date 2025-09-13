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

  // ---- Footnote Support ----
  const footnotes = {};

  // Inline footnote references: [^id]
  const footnoteRefExtension = {
    name: "footnoteRef",
    level: "inline",
    start(src) {
      return src.match(/\[\^/)?.index;
    },
    tokenizer(src) {
      const rule = /^\[\^(.+?)\]/;
      const match = rule.exec(src);
      if (match) {
        return {
          type: "footnoteRef",
          raw: match[0],
          id: match[1]
        };
      }
    },
    renderer(token) {
      const id = token.id.toLowerCase();
      return `<sup id="fnref:${id}"><a href="#fn:${id}" class="footnote-ref">[${id}]</a></sup>`;
    }
  };

  // Footnote definitions: [^id]: text
  const footnoteDefExtension = {
    name: "footnoteDef",
    level: "block",
    start(src) {
      return src.match(/^\[\^.+?\]:/)?.index;
    },
    tokenizer(src) {
      const rule = /^\[\^(.+?)\]: (.+)$/m;
      const match = rule.exec(src);
      if (match) {
        footnotes[match[1].toLowerCase()] = match[2];
        return {
          type: "footnoteDef",
          raw: match[0],
          id: match[1],
          text: match[2]
        };
      }
    },
    renderer() {
      // Don’t render definition inline; will render at bottom later
      return "";
    }
  };

  function renderFootnoteSection() {
    const keys = Object.keys(footnotes);
    if (!keys.length) return "";
    let out = '<section class="footnotes"><hr><ol>';
    keys.forEach(id => {
      const text = footnotes[id];
      out += `<li id="fn:${id}">${text} <a href="#fnref:${id}" class="footnote-backref">↩</a></li>`;
    });
    out += "</ol></section>";
    return out;
  }

  // Initialize marked once with defaults; callers can override via MarkdownRenderer.configure
  function initMarked(opts) {
    if (typeof global.marked?.setOptions === 'function') {
      global.marked.setOptions({ ...defaultMarkedOptions, ...(opts || {}) });
    }
    if (typeof global.marked?.use === 'function') {
      global.marked.use({ extensions: [footnoteRefExtension, footnoteDefExtension] });
    }
  }

  // Core render function: markdown string -> sanitized HTML into target element
  function renderMarkdownTo(targetEl, markdown) {
    assertDeps();
    const md = typeof markdown === 'string' ? markdown : '';

    // Reset footnote storage
    for (const k in footnotes) delete footnotes[k];

    const htmlMain = global.marked.parse(md);
    const htmlFootnotes = renderFootnoteSection();
    const fullHtml = htmlMain + htmlFootnotes;

    const safe = global.DOMPurify.sanitize(fullHtml);
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
        const scroller = targetEl.closest('.messages') || targetEl.parentElement;
        if (scroller) {
          scroller.scrollTop = scroller.scrollHeight;
        }
      } catch (err) {
        console.warn('MarkdownRenderer render error:', err);
        targetEl.textContent = buffer;
      }
    }

    function scheduleRender(immediate = false) {
      if (isDestroyed) return;
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
    configure(markedOptions) {
      initMarked(markedOptions);
    },
    render(targetEl, markdown) {
      renderMarkdownTo(targetEl, markdown);
    },
    createStream(targetEl, options) {
      return createStreamRenderer(targetEl, options);
    }
  };

  // Initialize defaults
  initMarked();

  // Attach to window
  global.MarkdownRenderer = MarkdownRenderer;

})(window);
