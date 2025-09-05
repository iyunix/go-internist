// File: web/static/js/chat/chat-stream.js
// FIXED: Progressive safe markdown rendering that preserves existing chats

import { Utils } from '../utils.js';

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

    // Check for unfinished code blocks
    const codeBlockMatches = (this.buffer.match(/```/g) || []).length;
    if (codeBlockMatches % 2 !== 0) return true;

    // Check for unfinished inline code
    const inlineCodeMatches = (buffer.match(/`/g) || []).length;
    if (inlineCodeMatches % 2 !== 0) return true;

    // Check for unfinished table rows
    const lines = buffer.split('\n');
    const lastNonEmptyLine = lines.filter(line => line.trim()).pop() || '';
    
    if (lastNonEmptyLine.includes('|') && !lastNonEmptyLine.endsWith('|')) {
      return true;
    }

    // Check for unfinished list items
    if (lastNonEmptyLine.match(/^\s*[-*+]\s*$/) || 
        lastNonEmptyLine.match(/^\s*\d+\.\s*$/)) {
      return true;
    }

    // Check for unfinished headers
    if (lastNonEmptyLine.match(/^\s*#{1,6}\s*$/)) {
      return true;
    }

    return false;
  }

  // Find safe content to render (complete sentences/paragraphs)
  findSafeContent() {
    const buffer = this.buffer;
    
    // Find last complete sentence
    const sentencePattern = /[.!?]\s+/g;
    let lastMatch;
    let match;
    
    while ((match = sentencePattern.exec(buffer)) !== null) {
      lastMatch = match;
    }

    if (lastMatch) {
      return buffer.substring(0, lastMatch.index + lastMatch.length);
    }

    // Fallback: find last complete paragraph
    const paragraphEnd = buffer.lastIndexOf('\n\n');
    if (paragraphEnd > 0) {
      return buffer.substring(0, paragraphEnd + 2);
    }

    return '';
  }

  // Progressive markdown rendering
  renderContent() {
    if (!this.buffer) return;

    try {
      if (this.hasIncompleteMarkdown()) {
        // Render safe portion as markdown, rest as monospace text
        const safeContent = this.findSafeContent();
        
        if (safeContent && safeContent !== this.lastSafeContent) {
          const remainingText = this.buffer.substring(safeContent.length);
          const combinedContent = safeContent + (remainingText ? `<span style="font-family:monospace;opacity:0.8">${this.escapeHtml(remainingText)}</span>` : '');
          
          // Create temp div to render markdown safely
          const tempDiv = document.createElement('div');
          window.MarkdownRenderer.render(tempDiv, safeContent);
          
          if (remainingText) {
            const textSpan = document.createElement('span');
            textSpan.style.fontFamily = 'monospace';
            textSpan.style.opacity = '0.8';
            textSpan.textContent = remainingText;
            tempDiv.appendChild(textSpan);
          }
          
          const container = this.chatUI.getLastAssistantMessageContainer();
          if (container) {
            container.innerHTML = tempDiv.innerHTML;
          }
          
          this.lastSafeContent = safeContent;
        } else if (!safeContent) {
          // No safe markdown found, show all as monospace
          this.chatUI.updateLastAssistantMessage(this.buffer, false);
        }
      } else {
        // Safe to render all as markdown
        this.chatUI.updateLastAssistantMessage(this.buffer, true);
        this.lastSafeContent = this.buffer;
      }

      this.lastRenderTime = Date.now();
    } catch (err) {
      console.warn("Streaming render error:", err);
      this.chatUI.updateLastAssistantMessage(this.buffer, false);
    }
  }

  // HTML escape utility
  escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
  }

  // Schedule render with intelligent timing
  scheduleRender(force = false) {
    clearTimeout(this.renderTimer);

    if (force) {
      this.renderContent();
      return;
    }

    const timeSinceLastRender = Date.now() - this.lastRenderTime;
    
    if (timeSinceLastRender > 300) {
      this.renderContent();
    } else {
      this.renderTimer = setTimeout(() => this.renderContent(), 150);
    }
  }

  // Add chunk and render progressively
  appendChunk(chunk) {
    if (!chunk) return;
    
    this.buffer += chunk;

    // Render on meaningful boundaries
    if (/[.!?]\s+/.test(chunk) || /\n\n/.test(chunk) || /\n#{1,6}\s/.test(chunk)) {
      this.scheduleRender();
    } else if (/\n/.test(chunk)) {
      this.scheduleRender();
    }
  }

  // Final render - always complete markdown
  finalize() {
    clearTimeout(this.renderTimer);
    this.chatUI.updateLastAssistantMessage(this.buffer, true);
  }

  // Cleanup
  destroy() {
    clearTimeout(this.renderTimer);
    this.buffer = "";
    this.lastSafeContent = "";
  }
}
