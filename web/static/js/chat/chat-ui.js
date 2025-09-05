// File: web/static/js/chat/chat-ui.js
// UPDATED: Aligned with new HTML structure and added methods for new UX features.

import { Utils } from '../utils.js';

export class ChatUI {
  constructor(elements) {
    this.elements = elements;
    this.lastAssistantMessageBubble = null; // Changed to target the bubble specifically
    this.currentSkeletonLoader = null;
    this.currentSources = [];
  }

  // --- Footnote & Source Methods (Unchanged) ---
  createFootnote(sources) {
    if (!sources || !sources.length === 0) return '';
    const sourceItems = sources.map(source => `<li>${this.escapeHtml(source)}</li>`).join('');
    return `<div class="message-footnote"><h6>Sources</h6><ul>${sourceItems}</ul></div>`;
  }
  escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
  }
  addFootnote(sources) {
    if (!this.lastAssistantMessageBubble || !sources || !sources.length === 0) return;
    const existing = this.lastAssistantMessageBubble.parentNode.querySelector('.message-footnote');
    if (existing) existing.remove();
    this.lastAssistantMessageBubble.parentNode.insertAdjacentHTML('beforeend', this.createFootnote(sources));
    Utils.scrollToBottom(this.elements.chatMessages);
  }
  setSources(sources) { this.currentSources = sources || []; }
  getSources() { return this.currentSources; }

  // --- Skeleton Loader Methods (Unchanged Logic, just target bubble) ---
  createSkeletonLoader() {
    return `<div class="skeleton-loader"><div class="skeleton-status"><div class="skeleton-status-icon searching"></div><span class="skeleton-status-text">Searching...</span></div><div class="skeleton-lines"><div class="skeleton-line"></div><div class="skeleton-line"></div></div></div>`;
  }
  updateSkeletonStatus(status, message) {
    if (!this.currentSkeletonLoader) return;
    const icon = this.currentSkeletonLoader.querySelector('.skeleton-status-icon');
    const text = this.currentSkeletonLoader.querySelector('.skeleton-status-text');
    if (icon) icon.className = `skeleton-status-icon ${status}`;
    if (text) text.textContent = message;
  }
  replaceSkeletonWithContent() {
    if (this.lastAssistantMessageBubble && this.currentSkeletonLoader) {
      this.lastAssistantMessageBubble.innerHTML = '';
      this.currentSkeletonLoader = null;
    }
  }

  // --- Core UI Methods ---
  
  toggleLoading(isLoading) {
    this.elements.chatInput.disabled = isLoading;
    this.elements.submitButton.disabled = isLoading;
  }
  
  // UPDATED: Rewritten to match new HTML structure from chat.html template
  displayMessage(content, role) {
    const li = document.createElement("li");
    li.className = `message-item ${role}`;

    const avatar = document.createElement("img");
    avatar.className = "avatar";
    // NOTE: You'll want to set the src for your avatars, e.g., based on role
    avatar.src = role === "user" ? "/static/img/user-avatar.png" : "/static/img/ai-avatar.png"; 
    avatar.alt = `${role} avatar`;

    const messageBubble = document.createElement("div");
    messageBubble.className = "message-bubble";

    if (role === "assistant") {
      this.lastAssistantMessageBubble = messageBubble;
      
      if (!content) { // If no content, show skeleton loader
        messageBubble.innerHTML = this.createSkeletonLoader();
        this.currentSkeletonLoader = messageBubble.querySelector('.skeleton-loader');
      } else {
        window.MarkdownRenderer?.render(messageBubble, content);
      }
    } else { // User message
      messageBubble.textContent = content || "";
    }

    li.appendChild(avatar);
    li.appendChild(messageBubble);
    this.elements.chatMessages.appendChild(li);
    Utils.scrollToBottom(this.elements.chatMessages);
  }
  
  getLastAssistantMessageContainer() {
    return this.lastAssistantMessageBubble;
  }

  clearMessages() {
    this.elements.chatMessages.innerHTML = "";
    this.lastAssistantMessageBubble = null;
    this.currentSkeletonLoader = null;
  }

  clearInput() {
    this.elements.chatInput.value = "";
    this.elements.chatInput.focus();
  }

  setActiveChat(chatId) {
    // This now targets the <li> wrapper for the history item
    document.querySelectorAll(".history-item.active").forEach(el => el.classList.remove("active"));
    if (chatId) {
      const el = document.querySelector(`.history-item[data-chat-id='${chatId}']`);
      if (el) el.classList.add("active");
    }
  }

  renderHistory(chats) {
    this.elements.historyList.innerHTML = "";
    if (!Array.isArray(chats)) return;

    chats.forEach(chat => {
      const id = String(chat.ID ?? "");
      const title = this.escapeHtml(String(chat.Title ?? "Untitled"));
      if (!id) return;

      const li = document.createElement("li");
      // This is the structure our CSS and JS expects now
      li.className = "history-item"; 
      li.setAttribute("data-chat-id", id);
      
      const anchor = document.createElement("a");
      anchor.href = `/chat?id=${id}`;
      anchor.textContent = title;
      // We are now styling the parent `li` for hover/active states

      // NOTE: Delete button functionality is not in the design brief,
      // but the old code had it. Let's keep it simple for now.
      // If you want it back, we can add a delete button element here.
      
      li.appendChild(anchor);
      this.elements.historyList.appendChild(li);
    });
  }

  renderMessages(messages) {
    this.clearMessages();
    if (!Array.isArray(messages)) return;
    messages.forEach(msg => this.displayMessage(msg.Content || "", msg.Role || "assistant"));
  }
  
  // NEW: Method to render quick action chips
  renderQuickActions(actions) {
      this.clearQuickActions();
      actions.forEach(actionText => {
          const chip = document.createElement('button');
          chip.className = 'quick-action-chip';
          chip.textContent = actionText;
          this.elements.quickActionsContainer.appendChild(chip);
      });
  }
  
  // NEW: Method to clear quick actions
  clearQuickActions() {
      this.elements.quickActionsContainer.innerHTML = '';
  }
  
  // NEW: Method to reset textarea height
  resetTextareaHeight() {
      this.elements.chatInput.style.height = 'auto';
  }
}