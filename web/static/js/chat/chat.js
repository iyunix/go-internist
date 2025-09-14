// File: web/static/js/chat/chat.js

import { ChatUI } from './chat-ui.js';
import { ChatAPI } from './chat-api.js';
import { ChatStreamRenderer } from './chat-stream.js';
import { Utils } from '../utils.js';


function showConfirmationModal(message) {
  return new Promise((resolve) => {
    // remove any lingering modal
    const old = document.getElementById('confirmation-modal-overlay');
    if (old && old.parentNode) old.parentNode.removeChild(old);

    const overlay = document.createElement('div');
    overlay.id = 'confirmation-modal-overlay';
    overlay.className = 'fixed inset-0 flex items-center justify-center z-50 bg-black bg-opacity-50';

    overlay.innerHTML = `
      <div
        class="bg-white rounded-lg shadow-xl p-6 w-full max-w-sm mx-auto"
        role="alertdialog"
        aria-modal="true"
        aria-labelledby="confirm-title"
        aria-describedby="confirm-desc"
        tabindex="-1"
      >
        <div class="text-center">
          <div class="mx-auto flex items-center justify-center h-12 w-12 rounded-full bg-red-100 mb-2">
            <span class="material-symbols-outlined text-red-600">warning</span>
          </div>
          <h3 id="confirm-title" class="text-lg leading-6 font-medium text-gray-900 mt-2">Confirm Deletion</h3>
          <div class="mt-2">
            <p id="confirm-desc" class="text-sm text-gray-500">${message}</p>
          </div>
        </div>
        <div class="mt-5 sm:mt-6 grid grid-cols-2 gap-3">
          <button id="modal-cancel-btn" type="button"
            class="inline-flex justify-center w-full rounded-md border border-gray-300 px-4 py-2 bg-white text-base font-medium text-gray-700 shadow-sm hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 sm:text-sm">
            Cancel
          </button>
          <button id="modal-confirm-btn" type="button"
            class="inline-flex justify-center w-full rounded-md border border-transparent px-4 py-2 bg-red-600 text-base font-medium text-white shadow-sm hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500 sm:text-sm">
            Delete
          </button>
        </div>
      </div>
    `;

    document.body.appendChild(overlay);

    const dialog = overlay.querySelector('[role="alertdialog"]');
    const confirmBtn = overlay.querySelector('#modal-confirm-btn');
    const cancelBtn = overlay.querySelector('#modal-cancel-btn');
    const previouslyFocused = document.activeElement;

    // Focus helpers
    const focusableSelectors = [
      'button',
      '[href]',
      'input',
      'select',
      'textarea',
      '[tabindex]:not([tabindex="-1"])',
    ];
    const getFocusable = () =>
      [...dialog.querySelectorAll(focusableSelectors.join(','))].filter(
        (el) => !el.hasAttribute('disabled') && el.getAttribute('aria-hidden') !== 'true'
      );
    const setInitialFocus = () => {
      const els = getFocusable();
      (els[0] || dialog).focus();
    };
    const restoreFocus = () => {
      if (previouslyFocused && typeof previouslyFocused.focus === 'function') {
        previouslyFocused.focus();
      }
    };
    const closeModal = (result) => {
      overlay.classList.remove('opacity-100');
      setTimeout(() => {
        dialog.removeEventListener('keydown', onKeyDown);
        overlay.removeEventListener('click', onOverlayClick);
        if (overlay.parentNode) overlay.parentNode.removeChild(overlay);
        restoreFocus();
        resolve(result);
      }, 150); // for animation, can be 0 if no effect
    };
    const onOverlayClick = (e) => {
      if (e.target === overlay) closeModal(false);
    };
    const onKeyDown = (e) => {
      if (e.key === 'Escape') {
        e.preventDefault();
        closeModal(false);
        return;
      }
      if (e.key === 'Tab') {
        const els = getFocusable();
        if (els.length === 0) {
          e.preventDefault();
          return;
        }
        const first = els[0];
        const last = els[els.length - 1];
        if (e.shiftKey && document.activeElement === first) {
          e.preventDefault();
          last.focus();
        } else if (!e.shiftKey && document.activeElement === last) {
          e.preventDefault();
          first.focus();
        }
      }
    };
    confirmBtn.onclick = () => closeModal(true);
    cancelBtn.onclick = () => closeModal(false);
    overlay.addEventListener('click', onOverlayClick);
    dialog.addEventListener('keydown', onKeyDown);
    setTimeout(() => {
      overlay.classList.add('opacity-100');
      setInitialFocus();
    }, 10);
  });
}



class ChatApp {
  constructor() {
    this.elements = {
      chatForm: document.getElementById('chatForm'),
      chatInput: document.getElementById('chatInput'),
      chatMessages: document.getElementById('chatMessages'),
      newChatBtn: document.getElementById('newChatBtn'),
      historyList: document.getElementById('historyList'),
      submitButton: null,
    };

    // If there is no chat form on this page, do nothing
    if (!this.elements.chatForm) return;

    this.elements.submitButton = this.elements.chatForm.querySelector("button[type='submit']");
    this.activeChatId = Utils.getUrlParam('id');
    this.ui = new ChatUI(this.elements);
    this.api = new ChatAPI();
    this.isInitialized = false;

    // Track the current streaming session to avoid duplicates and leaks
    this.currentStream = {
      eventSource: null,
      renderer: null,
    };
  }

  async init() {
    try {
      this.bindEvents();
      await this.loadHistory();

      if (this.activeChatId) {
        await this.loadMessages(this.activeChatId);
      } else {
        this.ui.clearMessages();
      }
      this.isInitialized = true;
    } catch (err) {
      console.error('[ChatApp] Failed to initialize:', err);
    }
  }

  async loadHistory() {
    try {
      const chats = await this.api.fetchHistory();
      this.ui.renderHistory(chats);
      this.ui.setActiveChat(this.activeChatId);
    } catch (err) {
      console.error('[ChatApp] Failed to load history:', err);
    }
  }

  async loadMessages(chatId) {
    // If navigating to a blank state, clear messages and selection
    if (!chatId) {
      this.ui.clearMessages();
      this.activeChatId = null;
      this.ui.setActiveChat(null);
      return;
    }

    // Cancel any in-flight stream when switching threads
    this._cancelCurrentStream();

    try {
      const messages = await this.api.fetchMessages(chatId);
      if (!messages) return;
      this.ui.renderMessages(messages);
      this.activeChatId = chatId;
      this.ui.setActiveChat(chatId);
    } catch (err) {
      console.error('[ChatApp] Failed to load messages:', err);
      this.ui.displayMessage('Failed to load chat messages. Please try again.', 'assistant');
    }
  }

  async handleSubmit(e) {
    e.preventDefault();
    const prompt = this.elements.chatInput.value.trim();
    if (!prompt || this.ui.isLoading) return;

    if (prompt.length > 4000) {
      this.ui.displayMessage('Question too long. Please limit to 4000 characters.', 'assistant');
      return;
    }

    // Render user message immediately
    this.ui.displayMessage(prompt, 'user', { showLoader: false });
    this.ui.clearInput();
    this.ui.toggleLoading(true);

    // Ensure a chat exists
    let chatId = this.activeChatId;
    if (!chatId) {
      try {
        chatId = await this.api.createChat(prompt.substring(0, 50));
        window.history.pushState({ chatId }, '', `/chat?id=${chatId}`);
        this.activeChatId = chatId;
        await this.loadHistory();
      } catch (err) {
        this.ui.displayMessage('Error: Could not create a new chat session.', 'assistant');
        this.ui.toggleLoading(false);
        return;
      }
    }

    this.streamResponse(chatId, prompt);
  }

  streamResponse(chatId, prompt) {
    // Ensure no overlapping streams
    this._cancelCurrentStream();

    // Create assistant bubble with loader
    this.ui.displayMessage('', 'assistant', { showLoader: true });

    const renderer = new ChatStreamRenderer(this.ui);
    const eventSource = this.api.createStream(chatId, prompt);

    this.currentStream = { eventSource, renderer };
    let firstChunkReceived = false;

    eventSource.onmessage = (evt) => {
      if (!firstChunkReceived) {
        this.ui.replaceSkeletonWithContent?.();
        firstChunkReceived = true;
      }

      try {
        const rawData = evt.data ?? '';
        let content = '';
        if (rawData.startsWith('[') && rawData.endsWith(']')) {
          const chunks = JSON.parse(rawData);
          for (const part of chunks) {
            if (part && typeof part.content === 'string') content += part.content;
          }
        } else if (rawData.startsWith('{') && rawData.endsWith('}')) {
          const obj = JSON.parse(rawData);
          if (obj && typeof obj.content === 'string') content = obj.content;
        } else {
          content = rawData;
        }
        if (content) renderer.appendChunk(content);
      } catch (err) {
        console.warn('[ChatApp] Could not parse stream data as JSON, using raw text.', err);
        renderer.appendChunk(evt.data || '');
      }
    };

    eventSource.addEventListener('done', () => {
      eventSource.close();
      renderer.finalize();
      this.ui.toggleLoading(false);
      this.currentStream = { eventSource: null, renderer: null };
    });
    eventSource.addEventListener('complete', () => {
      eventSource.close();
      renderer.finalize();
      this.ui.toggleLoading(false);
      this.currentStream = { eventSource: null, renderer: null };
    });

    eventSource.onerror = (err) => {
      console.error('[ChatApp] Stream error:', err);
      eventSource.close();
      renderer.destroy();
      this.ui.displayMessage('A streaming error occurred. Please try again.', 'assistant');
      this.ui.toggleLoading(false);
      this.currentStream = { eventSource: null, renderer: null };
    };
  }

  handleTextareaInput() {
    const textarea = this.elements.chatInput;
    textarea.style.height = 'auto';
    const maxHeight = 200;
    textarea.style.height = `${Math.min(textarea.scrollHeight, maxHeight)}px`;
  }

  handleTextareaKeydown(e) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      const form = this.elements.chatForm;
      if (form && typeof form.requestSubmit === 'function') {
        form.requestSubmit();
      } else if (form) {
        form.submit();
      }
    }
  }

  bindEvents() {
    this.elements.chatForm.addEventListener('submit', (e) => this.handleSubmit(e));
    this.elements.chatInput.addEventListener('input', () => this.handleTextareaInput());
    this.elements.chatInput.addEventListener('keydown', (e) => this.handleTextareaKeydown(e));
    this.elements.newChatBtn.addEventListener('click', (e) => {
      e.preventDefault();
      if (this.ui.isLoading) return;
      window.history.pushState({}, '', '/chat');
      this.loadMessages(null);
    });
    this.elements.historyList.addEventListener('click', async (e) => {
      if (this.ui.isLoading) return;
      const target = e.target;
      const deleteBtn = target.closest('.delete-chat-btn');
      if (deleteBtn) {
        e.preventDefault();
        e.stopPropagation();
        await this.handleDeleteChat(deleteBtn);
        return;
      }
      const link = target.closest('a');
      if (link) {
        e.preventDefault();
        const chatId = link.getAttribute('data-chat-id');
        if (chatId && chatId !== this.activeChatId) {
          window.history.pushState({ chatId }, '', `/chat?id=${chatId}`);
          this.loadMessages(chatId);
        }
      }
    });
    window.addEventListener('popstate', (e) => {
      if (this.ui.isLoading) return;
      const stateChatId = e.state && e.state.chatId ? String(e.state.chatId) : null;
      const urlChatId = Utils.getUrlParam('id') || null;
      const nextId = stateChatId || urlChatId || null;
      this.loadMessages(nextId);
    });
    window.addEventListener('beforeunload', () => this._cancelCurrentStream());
  }

  async handleDeleteChat(button) {
    const chatId = button.getAttribute('data-chat-id');
    if (!chatId) return;

    const confirmed = await showConfirmationModal('Are you sure you want to permanently delete this chat?');
    if (!confirmed) return;

    try {
      await this.api.deleteChat(chatId);
      const li = button.closest('li');
      if (li && li.parentElement) li.parentElement.removeChild(li);
      if (chatId === this.activeChatId) {
        this.elements.newChatBtn.click();
      }
    } catch (err) {
      console.error('[ChatApp] Failed to delete chat:', err);
      alert('Failed to delete chat.');
    }
  }

  _cancelCurrentStream() {
    if (this.currentStream.eventSource) {
      try {
        this.currentStream.eventSource.close();
      } catch { }
    }
    if (this.currentStream.renderer) {
      try {
        this.currentStream.renderer.destroy();
      } catch { }
    }
    this.currentStream = { eventSource: null, renderer: null };
    this.ui.toggleLoading(false);
  }
}

document.addEventListener('DOMContentLoaded', () => {
  const app = new ChatApp();
  if (app.elements.chatForm) {
    app.init();
  }
});
