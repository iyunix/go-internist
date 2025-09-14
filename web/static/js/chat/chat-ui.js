// File: web/static/js/chat/chat-ui.js
// REWRITTEN: Clean classNames, robust role normalization, ARIA live-region for chat,
// safe Markdown rendering for assistant messages, and unchanged external API.

export class ChatUI {
  constructor(elements) {
    this.elements = elements;
    this.lastAssistantMessageBubble = null;
    this.currentSkeletonLoader = null;
    this.isLoading = false;

    this._initA11y();
  }

  // Initialize ARIA live log region for chat messages (polite announcements)
  _initA11y() {
    const container = this.elements?.chatMessages;
    if (!container) return;

    // Apply defaults only if not already set in HTML
    if (!container.hasAttribute('role')) container.setAttribute('role', 'log');
    if (!container.hasAttribute('aria-live')) container.setAttribute('aria-live', 'polite');
    if (!container.hasAttribute('aria-relevant')) container.setAttribute('aria-relevant', 'additions text');

    // A log requires an accessible name
    if (!container.hasAttribute('aria-label') && !container.hasAttribute('aria-labelledby')) {
      container.setAttribute('aria-label', 'Chat messages');
    }
  }

  // Normalize message role into 'user' | 'assistant'
  normalizeRole(type) {
    const r = String(type || 'user').toLowerCase();
    if (r === 'assistant' || r === 'system' || r === 'ai' || r === 'model') return 'assistant';
    if (r === 'user' || r === 'human') return 'user';
    return 'assistant';
  }

  // === CORE MESSAGE DISPLAY METHODS ===
  displayMessage(content, role, options = {}) {
    const normalizedRole = this.normalizeRole(role);

    const li = document.createElement('li');
    li.className = `flex items-start gap-3 w-full ${normalizedRole === 'user' ? 'justify-end user-message' : 'assistant-message'}`;

    const messageContainer = document.createElement('div');
    messageContainer.className = `flex gap-3 max-w-4xl ${normalizedRole === 'user' ? 'flex-row-reverse' : 'flex-row'}`;

    const avatarContainer = document.createElement('div');
    avatarContainer.className = 'flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-gray-200';

    const avatarIcon = document.createElement('span');
    avatarIcon.className = 'material-symbols-outlined text-[var(--text-secondary)]';
    avatarIcon.textContent = normalizedRole === 'user' ? 'person' : 'smart_toy';
    avatarContainer.appendChild(avatarIcon);

    const contentWrapper = document.createElement('div');
    contentWrapper.className = `flex-1 ${normalizedRole === 'user' ? 'text-right' : ''}`;

    const authorName = document.createElement('p');
    authorName.className = 'text-sm font-medium text-[var(--text-secondary)]';
    authorName.textContent = normalizedRole === 'user' ? 'You' : 'Internist AI';

    const messageBubble = document.createElement('div');
    messageBubble.className =
      `message-bubble mt-1 rounded-lg p-3 text-base text-left ` +
      (normalizedRole === 'user'
        ? 'rounded-tr-none bg-[var(--primary-color)] text-white'
        : 'rounded-tl-none bg-gray-100 text-[var(--text-primary)]');

    if (normalizedRole === 'assistant') {
      this.lastAssistantMessageBubble = messageBubble;
      if (options.showLoader === true) {
        messageBubble.innerHTML = this.createSkeletonLoader();
        this.currentSkeletonLoader = messageBubble.querySelector('.skeleton-loader');
      } else {
        this.renderMessageContent(messageBubble, content || '');
      }
    } else {
      // For user messages, always plain text
      messageBubble.textContent = content || '';
    }

    contentWrapper.appendChild(authorName);
    contentWrapper.appendChild(messageBubble);
    messageContainer.appendChild(avatarContainer);
    messageContainer.appendChild(contentWrapper);
    li.appendChild(messageContainer);
    this.elements.chatMessages.appendChild(li);

    this.scrollToBottom();
    return li;
  }

  renderMessageContent(container, content) {
    if (!container) return;
    if (!content || String(content).trim() === '') {
      container.innerHTML = '<em class="text-slate-500">No content</em>';
      return;
    }
    // Prefer safe Markdown + sanitization if available
    if (window.MarkdownRenderer && typeof window.MarkdownRenderer.render === 'function') {
      window.MarkdownRenderer.render(container, String(content));
    } else {
      // Fallback: plain text
      container.textContent = String(content);
    }
  }

  renderMessages(messages) {
    this.removeAllSkeletons();
    this.clearMessages();
    if (!Array.isArray(messages) || messages.length === 0) return;

    messages.forEach((msg) => {
      const content = msg?.content ?? '';
      const role = this.normalizeRole(msg?.messageType ?? 'user');
      this.displayMessage(content, role, { showLoader: false });
    });

    setTimeout(() => this.removeAllSkeletons(), 100);
  }

  // === SKELETON LOADER METHODS ===
  createSkeletonLoader() {
    return `
      <div class="skeleton-loader space-y-3">
        <div class="flex items-center gap-2">
          <div class="skeleton-status-icon h-4 w-4 rounded-full bg-gray-300 animate-pulse"></div>
          <span class="skeleton-status-text text-sm text-gray-500">Thinking...</span>
        </div>
        <div class="space-y-2">
          <div class="h-4 w-5/6 rounded bg-gray-300 animate-pulse"></div>
          <div class="h-4 w-full rounded bg-gray-300 animate-pulse"></div>
          <div class="h-4 w-3/4 rounded bg-gray-300 animate-pulse"></div>
        </div>
      </div>
    `;
  }

  updateSkeletonStatus(_status, message) {
    if (!this.currentSkeletonLoader) return;
    const text = this.currentSkeletonLoader.querySelector('.skeleton-status-text');
    if (text) text.textContent = message;
  }

  replaceSkeletonWithContent() {
    if (this.lastAssistantMessageBubble && this.currentSkeletonLoader) {
      this.lastAssistantMessageBubble.innerHTML = '';
      this.currentSkeletonLoader = null;
    }
  }

  removeAllSkeletons() {
    document.querySelectorAll('.skeleton-loader').forEach((el) => el.remove());
    this.currentSkeletonLoader = null;
  }

  // === UI STATE MANAGEMENT ===
  toggleLoading(isLoading) {
    this.isLoading = isLoading;
    if (this.elements.chatInput) this.elements.chatInput.disabled = isLoading;
    if (this.elements.submitButton) this.elements.submitButton.disabled = isLoading;
  }

  clearMessages() {
    this.elements.chatMessages.innerHTML = '';
    this.lastAssistantMessageBubble = null;
    this.currentSkeletonLoader = null;
  }

  clearInput() {
    if (this.elements.chatInput) {
      this.elements.chatInput.value = '';
      this.elements.chatInput.focus();
      this.resetTextareaHeight();
    }
  }

  resetTextareaHeight() {
    if (this.elements.chatInput) {
      this.elements.chatInput.style.height = 'auto';
    }
  }

  scrollToBottom() {
    if (this.elements.chatMessages) {
      this.elements.chatMessages.scrollTop = this.elements.chatMessages.scrollHeight;
    }
  }

  // === HISTORY AND NAVIGATION ===
  setActiveChat(chatId) {
    document.querySelectorAll('#historyList a').forEach((el) => {
      el.classList.remove('bg-gray-100', 'font-semibold');
      el.classList.add('text-[var(--text-primary)]', 'hover:bg-gray-100');
    });
    if (chatId) {
      const activeElement = document.querySelector(`#historyList a[data-chat-id='${chatId}']`);
      if (activeElement) {
        activeElement.classList.add('bg-gray-100', 'font-semibold');
        activeElement.classList.remove('hover:bg-gray-100');
      }
    }
  }

  renderHistory(chats) {
    this.elements.historyList.innerHTML = '';
    if (!Array.isArray(chats)) return;

    chats.forEach((chat) => {
      const id = String(chat.ID ?? '');
      const title = this.escapeHtml(String(chat.Title ?? 'Untitled'));
      if (!id) return;

      const li = document.createElement('li');
      li.className = 'flex items-center justify-between group';

      const anchor = document.createElement('a');
      anchor.href = `/chat?id=${id}`;
      anchor.setAttribute('data-chat-id', id);
      anchor.className =
        'flex flex-grow items-center gap-3 rounded-md px-3 py-2 text-sm font-medium text-[var(--text-primary)] hover:bg-gray-100 truncate';

      const icon = document.createElement('span');
      icon.className = 'material-symbols-outlined text-lg text-[var(--text-secondary)]';
      icon.textContent = 'chat_bubble';

      const text = document.createElement('span');
      text.className = 'truncate';
      text.textContent = title;

      anchor.appendChild(icon);
      anchor.appendChild(text);

      const deleteBtn = document.createElement('button');
      deleteBtn.className =
        'delete-chat-btn p-1 rounded-md text-gray-400 hover:bg-gray-200 hover:text-gray-600 opacity-0 group-hover:opacity-100 transition-opacity';
      deleteBtn.innerHTML = `<span class="material-symbols-outlined text-lg">delete</span>`;
      deleteBtn.setAttribute('data-chat-id', id);
      deleteBtn.setAttribute('aria-label', 'Delete chat');

      li.appendChild(anchor);
      li.appendChild(deleteBtn);
      this.elements.historyList.appendChild(li);
    });
  }

  // === UTILITY METHODS ===
  escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text || '';
    return div.innerHTML;
  }

  getLastAssistantMessageContainer() {
    return this.lastAssistantMessageBubble;
  }
}
