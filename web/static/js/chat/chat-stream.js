// File: web/static/js/chat/chat-stream.js
// REFACTORED: Minor improvements for best practices.

import { Utils } from '../utils.js';

// NEW: Constants for readability
const RENDER_THROTTLE_MS = 150;
const RENDER_DEBOUNCE_MS = 300;

export class ChatStreamRenderer {
  constructor(chatUI) {
    this.chatUI = chatUI;
    this.buffer = "";
    this.renderTimer = null;
    this.lastRenderTime = 0;
    this.lastSafeContent = "";
  }

  // Check for incomplete markdown that shouldn't be rendered yet
  hasIncompleteMarkdown() {
    const buffer = this.buffer.trim();
    if (!buffer) return false;
    
    // Unfinished code blocks
    if ((this.buffer.match(/```/g) || []).length % 2 !== 0) return true;
    
    const lines = buffer.split('\n');
    const lastNonEmptyLine = lines.filter(line => line.trim()).pop() || '';
    
    // Unfinished table rows
    if (lastNonEmptyLine.includes('|') && !lastNonEmptyLine.trim().endsWith('|')) return true;
    
    // Unfinished list items or headers
    if (lastNonEmptyLine.match(/^\s*([-*+]|\d+\.|#{1,6})\s*$/)) return true;
    
    return false;
  }

  // Find safe content to render (complete sentences/paragraphs)
  findSafeContent() {
    // We only need to find safe content if the markdown is incomplete.
    // In other cases, the whole buffer is safe.
    const lastParagraphEnd = this.buffer.lastIndexOf('\n\n');
    if (lastParagraphEnd > 0) {
      return this.buffer.substring(0, lastParagraphEnd + 2);
    }
    return '';
  }

  // Progressive markdown rendering
  renderContent() {
    if (!this.buffer) return;

    try {
      const container = this.chatUI.getLastAssistantMessageContainer();
      if (!container) return;

      if (this.hasIncompleteMarkdown()) {
        const safeContent = this.findSafeContent();
        const remainingText = this.buffer.substring(safeContent.length);

        // Render the safe part as markdown, and the rest as a plain text fragment
        const tempDiv = document.createElement('div');
        window.MarkdownRenderer.render(tempDiv, safeContent);
        
        if (remainingText) {
          const textSpan = document.createElement('span');
          // UPDATED: Use a CSS class instead of inline style
          textSpan.className = 'streaming-text-fragment'; 
          textSpan.textContent = remainingText;
          tempDiv.appendChild(textSpan);
        }
        
        container.innerHTML = tempDiv.innerHTML;
        this.lastSafeContent = safeContent;
      } else {
        // Safe to render the entire buffer as markdown
        window.MarkdownRenderer.render(container, this.buffer);
        this.lastSafeContent = this.buffer;
      }

      this.lastRenderTime = Date.now();
      Utils.scrollToBottom(this.chatUI.elements.chatMessages);
    } catch (err) {
      console.warn("Streaming render error:", err);
      // Fallback to plain text on error
      const container = this.chatUI.getLastAssistantMessageContainer();
      if(container) container.textContent = this.buffer;
    }
  }

  // Schedule render with intelligent timing
  scheduleRender(force = false) {
    clearTimeout(this.renderTimer);

    if (force) {
      this.renderContent();
      return;
    }

    const timeSinceLastRender = Date.now() - this.lastRenderTime;
    
    if (timeSinceLastRender > RENDER_DEBOUNCE_MS) {
      this.renderContent();
    } else {
      this.renderTimer = setTimeout(() => this.renderContent(), RENDER_THROTTLE_MS);
    }
  }

  // Add chunk and render progressively
  appendChunk(chunk) {
    if (chunk === null || chunk === undefined) return;
    this.buffer += chunk;
    this.scheduleRender();
  }

  // Final render - always complete markdown
  finalize() {
    clearTimeout(this.renderTimer);
    const container = this.chatUI.getLastAssistantMessageContainer();
    if(container) window.MarkdownRenderer.render(container, this.buffer);
    Utils.scrollToBottom(this.chatUI.elements.chatMessages);
  }

  // Cleanup
  destroy() {
    clearTimeout(this.renderTimer);
    this.buffer = "";
    this.lastSafeContent = "";
  }
}