// File: web/static/js/chat/chat-stream.js
// REWRITTEN: Single persistent Markdown stream per assistant bubble, safe fallbacks, and clear lifecycle.

import { Utils } from '../utils.js';

/**
 * ChatStreamRenderer
 * - Creates one Markdown render stream per assistant message bubble.
 * - Appends incoming SSE chunks progressively and flushes on completion.
 * - Falls back to plain text if MarkdownRenderer is not available.
 */
export class ChatStreamRenderer {
  /**
   * @param {import('./chat-ui.js').ChatUI} chatUI
   * @param {{ debounceMs?: number }} options
   */
  constructor(chatUI, options = {}) {
    this.chatUI = chatUI;
    this.options = { debounceMs: options.debounceMs ?? 100 };
    this.container = null;
    this.stream = null;
    this.buffer = '';
    this.fallbackTimer = null;
    this.isDestroyed = false;
    this._initialized = false;
  }

  // Lazily initialize the stream on first chunk so we always target the latest assistant bubble
  _ensureInitialized() {
    if (this._initialized) return;
    this._initialized = true;

    // Resolve the current assistant message bubble
    this.container = this.chatUI?.getLastAssistantMessageContainer?.() || null;
    if (!this.container) return;

    // If a MarkdownRenderer with streaming is present, use it; otherwise, fallback to text
    const MR = window?.MarkdownRenderer;
    if (MR && typeof MR.createStream === 'function') {
      this.stream = MR.createStream(this.container, { debounceMs: this.options.debounceMs });
    } else {
      // Fallback mode: will set textContent with minimal batching
      this.stream = null;
    }
  }

  /**
   * Append a streamed chunk
   * @param {string} chunk
   */
  appendChunk(chunk) {
    if (this.isDestroyed || chunk == null) return;
    if (!this.container) this._ensureInitialized();
    if (!this.container) return; // nothing to render into yet

    const text = String(chunk);
    if (this.stream) {
      // Preferred: progressive Markdown rendering managed by MarkdownRenderer
      try {
        this.stream.append(text);
      } catch {
        // If streaming fails, gracefully fall back to buffered text
        this._fallbackAppend(text);
      }
    } else {
      // Fallback: accumulate text and render as plain text with minimal batching
      this._fallbackAppend(text);
    }
  }

  _fallbackAppend(text) {
    this.buffer += text;
    if (this.fallbackTimer) clearTimeout(this.fallbackTimer);
    // Batch updates slightly to avoid excessive DOM thrash
    this.fallbackTimer = setTimeout(() => {
      if (this.container) this.container.textContent = this.buffer;
      Utils.scrollToBottom?.(this.chatUI?.elements?.chatMessages);
    }, this.options.debounceMs);
  }

  /**
   * Finalize the stream: flush pending renders and render the complete buffer if needed
   */
  finalize() {
    if (this.isDestroyed) return;
    // Clear any fallback timer to avoid racing updates
    if (this.fallbackTimer) {
      clearTimeout(this.fallbackTimer);
      this.fallbackTimer = null;
    }

    try {
      if (this.stream && typeof this.stream.flush === 'function') {
        // Flush the Markdown render stream to ensure complete final HTML
        this.stream.flush();
      } else if (this.container) {
        // If we had to fall back to text but MarkdownRenderer exists now, render once as Markdown
        const MR = window?.MarkdownRenderer;
        if (MR && typeof MR.render === 'function') {
          MR.render(this.container, this.buffer);
        } else {
          this.container.textContent = this.buffer;
        }
      }
    } finally {
      Utils.scrollToBottom?.(this.chatUI?.elements?.chatMessages);
    }
  }

  /**
   * Destroy and cleanup references
   */
  destroy() {
    this.isDestroyed = true;
    if (this.fallbackTimer) {
      clearTimeout(this.fallbackTimer);
      this.fallbackTimer = null;
    }
    if (this.stream && typeof this.stream.destroy === 'function') {
      try { this.stream.destroy(); } catch {}
    }
    this.stream = null;
    this.container = null;
    this.buffer = '';
  }
}
