// File: web/static/js/chat/chat-ui.js
// FIXED: Preserves existing chats, only updates streaming message

import { Utils } from '../utils.js';

export class ChatUI {
  constructor(elements) {
    this.elements = elements;
    this.lastAssistantMessage = null; // Track last assistant message for streaming
  }

  toggleLoading(isLoading) {
    this.elements.chatInput.disabled = isLoading;
    this.elements.submitButton.disabled = isLoading;
  }

  // Create and display a message - FIXED to track assistant messages
  displayMessage(content, role, returnElement = false) {
    const li = document.createElement("li");
    li.className = `msg ${role}`;

    const avatar = document.createElement("span");
    avatar.className = "avatar";
    avatar.textContent = role === "user" ? "You" : "AI";

    const container = document.createElement(role === "assistant" ? "div" : "p");

    if (role === "assistant") {
      container.classList.add("md");
      // Mark this as the last assistant message for streaming updates
      this.lastAssistantMessage = container;
      
      try {
        window.MarkdownRenderer?.render(container, content || "");
      } catch (e) {
        container.textContent = content || "";
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

  // NEW: Update only the last assistant message (for streaming)
  updateLastAssistantMessage(content, isMarkdown = true) {
    if (!this.lastAssistantMessage) return;

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

  // NEW: Get the last assistant message container for streaming
  getLastAssistantMessageContainer() {
    return this.lastAssistantMessage;
  }

  clearMessages() {
    this.elements.chatMessages.innerHTML = "";
    this.lastAssistantMessage = null; // Reset tracking
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
    this.clearMessages(); // This will reset lastAssistantMessage too

    if (Array.isArray(messages) && messages.length > 0) {
      messages.forEach(msg => {
        const role = msg.Role ?? msg.role ?? "assistant";
        const content = msg.Content ?? msg.content ?? "";
        this.displayMessage(content, role); // This will set lastAssistantMessage correctly
      });
    }
  }
}
