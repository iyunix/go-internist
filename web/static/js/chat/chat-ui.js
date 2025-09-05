// File: web/static/js/chat/chat-ui.js
// FIXED: Complete skeleton loader functionality

import { Utils } from '../utils.js';

export class ChatUI {
  constructor(elements) {
    this.elements = elements;
    this.lastAssistantMessage = null;
    this.currentSkeletonLoader = null; // Track active skeleton
  }

  // Create skeleton loader HTML
  createSkeletonLoader() {
    return `
      <div class="skeleton-loader">
        <div class="skeleton-status">
          <div class="skeleton-status-icon searching"></div>
          <span class="skeleton-status-text">Searching knowledge base...</span>
        </div>
        <div class="skeleton-lines">
          <div class="skeleton-line medium"></div>
          <div class="skeleton-line long"></div>
          <div class="skeleton-line short"></div>
        </div>
      </div>
    `;
  }

  // Update skeleton status
  updateSkeletonStatus(status, message) {
    if (!this.currentSkeletonLoader) return;

    const statusIcon = this.currentSkeletonLoader.querySelector('.skeleton-status-icon');
    const statusText = this.currentSkeletonLoader.querySelector('.skeleton-status-text');

    if (statusIcon && statusText) {
      // Remove all status classes
      statusIcon.className = 'skeleton-status-icon';
      
      // Add new status class and update text
      switch (status) {
        case 'searching':
          statusIcon.classList.add('searching');
          statusText.textContent = message || 'Searching knowledge base...';
          break;
        case 'processing':
          statusIcon.classList.add('processing');
          statusText.textContent = message || 'Processing search results...';
          break;
        case 'thinking':
          statusIcon.classList.add('thinking');
          statusText.textContent = message || 'AI is thinking...';
          break;
        case 'responding':
          statusIcon.classList.add('responding');
          statusText.textContent = message || 'Generating response...';
          break;
        case 'completed':
          statusIcon.classList.add('completed');
          statusText.textContent = message || 'Response ready';
          break;
        case 'error':
          statusIcon.classList.add('error');
          statusText.textContent = message || 'Error occurred';
          break;
      }
    }
  }

  // Remove skeleton and prepare for content
  replaceSkeletonWithContent() {
    if (this.lastAssistantMessage && this.currentSkeletonLoader) {
      this.lastAssistantMessage.innerHTML = '';
      this.currentSkeletonLoader = null;
    }
  }

  toggleLoading(isLoading) {
    this.elements.chatInput.disabled = isLoading;
    this.elements.submitButton.disabled = isLoading;
  }

  // FIXED: Create and display message with skeleton support
  displayMessage(content, role, returnElement = false) {
    const li = document.createElement("li");
    li.className = `msg ${role}`;

    const avatar = document.createElement("span");
    avatar.className = "avatar";
    avatar.textContent = role === "user" ? "You" : "AI";

    const container = document.createElement(role === "assistant" ? "div" : "p");

    if (role === "assistant") {
      container.classList.add("md");
      this.lastAssistantMessage = container;
      
      // Show skeleton if no content provided
      if (!content) {
        container.innerHTML = this.createSkeletonLoader();
        this.currentSkeletonLoader = container.querySelector('.skeleton-loader');
      } else {
        try {
          window.MarkdownRenderer?.render(container, content);
        } catch (e) {
          container.textContent = content;
        }
      }
    } else {
      container.textContent = content || "";
    }

    li.appendChild(avatar);
    li.appendChild(container);
    this.elements.chatMessages.appendChild(li);
    Utils.scrollToBottom(this.elements.chatMessages, false);

    if (returnElement) return li;
  }

  // Update only the last assistant message (for streaming)
  updateLastAssistantMessage(content, isMarkdown = true) {
    if (!this.lastAssistantMessage) return;

    // Remove skeleton if it exists
    if (this.currentSkeletonLoader) {
      this.replaceSkeletonWithContent();
    }

    try {
      if (isMarkdown) {
        window.MarkdownRenderer?.render(this.lastAssistantMessage, content);
      } else {
        this.lastAssistantMessage.textContent = content;
      }
      Utils.scrollToBottom(this.elements.chatMessages, false);
    } catch (e) {
      this.lastAssistantMessage.textContent = content;
    }
  }

  // Get the last assistant message container for streaming
  getLastAssistantMessageContainer() {
    return this.lastAssistantMessage;
  }

  clearMessages() {
    this.elements.chatMessages.innerHTML = "";
    this.lastAssistantMessage = null;
    this.currentSkeletonLoader = null;
  }

  clearInput() {
    this.elements.chatInput.value = "";
  }

  setActiveChat(chatId) {
    document.querySelectorAll(".history-item.active").forEach(el => 
      el.classList.remove("active")
    );
    
    if (chatId) {
      const el = document.querySelector(`.history-item[data-chat-id='${chatId}']`);
      if (el) el.classList.add("active");
    }
  }

  renderHistory(chats) {
    this.elements.historyList.innerHTML = "";

    if (Array.isArray(chats)) {
      chats.forEach(chat => {
        const id = String(chat.ID ?? chat.id ?? "");
        const title = String(chat.Title ?? chat.title ?? "Untitled");
        if (!id) return;

        const li = document.createElement("li");
        li.className = "history-item";
        li.setAttribute("data-chat-id", id);
        li.innerHTML = `
          <a href="/chat?id=${id}">${title}</a>
          <button class="delete-chat-btn" title="Delete chat" aria-label="Delete chat">&times;</button>
        `;
        this.elements.historyList.appendChild(li);
      });
    }
  }

  renderMessages(messages) {
    this.clearMessages();

    if (Array.isArray(messages) && messages.length > 0) {
      messages.forEach(msg => {
        const role = msg.Role ?? msg.role ?? "assistant";
        const content = msg.Content ?? msg.content ?? "";
        this.displayMessage(content, role);
      });
    }
  }
}
