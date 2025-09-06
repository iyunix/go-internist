// File: web/static/js/chat/chat.js
// FIXED: Targeted skeleton removal that preserves message content
// UPDATED: Added credit balance integration

import { ChatUI } from './chat-ui.js';
import { ChatAPI } from './chat-api.js';
import { ChatStreamRenderer } from './chat-stream.js';
import { Utils } from '../utils.js';

class ChatApp {
  constructor() {
    this.elements = {
      chatForm: document.getElementById("chatForm"),
      chatInput: document.getElementById("chatInput"),
      chatMessages: document.getElementById("chatMessages"),
      newChatBtn: document.getElementById("newChatBtn"),
      historyList: document.getElementById("historyList"),
      quickActionsContainer: document.getElementById("quickActions"),
      submitButton: null
    };

    if (!this.elements.chatForm) {
      console.error("[ChatApp] Required chat elements not found");
      return;
    }

    this.elements.submitButton = this.elements.chatForm.querySelector("button");
    this.activeChatId = Utils.getUrlParam("id");

    this.ui = new ChatUI(this.elements);
    this.api = new ChatAPI();
    this.isInitialized = false;
  }

  /**
   * Remove skeleton loaders without removing actual messages
   */
  removeSkeletonLoaders() {
    const skeletonLoaders = document.querySelectorAll('.skeleton-loader, .skeleton-container');
    skeletonLoaders.forEach(el => el.remove());

    const skeletonStatus = document.querySelectorAll('.skeleton-status, .skeleton-lines');
    skeletonStatus.forEach(el => el.remove());

    const loadingInputs = document.querySelectorAll('input.loading, button.loading, form.loading');
    loadingInputs.forEach(el => el.classList.remove('loading'));
  }

  async init() {
    try {
      this.bindEvents();
      await this.loadHistory();

      if (this.activeChatId) {
        await this.loadMessages(this.activeChatId);
      } else {
        this.renderDefaultQuickActions();
      }

      this.isInitialized = true;
      console.log("[ChatApp] Initialized successfully");
    } catch (err) {
      console.error("[ChatApp] Failed to initialize:", err);
    }
  }

  /**
   * Render default quick actions
   */
  renderDefaultQuickActions() {
    this.ui.renderQuickActions([
      "What are the symptoms of diabetes?",
      "How to treat high blood pressure?",
      "Common causes of headaches"
    ]);
  }

  async loadHistory() {
    try {
      const chats = await this.api.fetchHistory();
      this.ui.renderHistory(chats);
      this.ui.setActiveChat(this.activeChatId);
    } catch (err) {
      console.error("[ChatApp] Failed to load history:", err);
    }
  }

  async loadMessages(chatId) {
    this.ui.clearQuickActions();

    if (!chatId) {
      this.ui.clearMessages();
      this.activeChatId = null;
      this.ui.setActiveChat(null);
      this.renderDefaultQuickActions();
      return;
    }

    try {
      console.log("[ChatApp] Loading messages for chat:", chatId);
      const messages = await this.api.fetchMessages(chatId);

      if (!messages) {
        console.warn("[ChatApp] No messages for chat:", chatId);
        return;
      }

      this.ui.renderMessages(messages);
      this.activeChatId = chatId;
      this.ui.setActiveChat(chatId);
    } catch (err) {
      console.error("[ChatApp] Failed to load messages:", err);
      this.ui.displayMessage("Failed to load chat messages. Please try again.", "assistant");
    }
  }

  /**
   * Handle form submit: validate balance and question length
   */
  async handleSubmit(e) {
    e.preventDefault();
    const prompt = this.elements.chatInput.value.trim();
    if (!prompt) return;

    // Check credit balance
    if (window.creditManager && !window.creditManager.canAskQuestion(prompt.length)) {
      const currentBalance = window.creditManager.getCurrentBalance();
      const chargeAmount = Math.max(100, prompt.length);

      this.ui.displayMessage(
        `Insufficient credits. You need ${chargeAmount}, but only have ${currentBalance}.`,
        "assistant"
      );
      return;
    }

    // Check length limit
    if (prompt.length > 1000) {
      this.ui.displayMessage("Question too long. Please limit to 1000 characters.", "assistant");
      return;
    }

    this.ui.clearQuickActions();
    this.ui.displayMessage(prompt, "user");
    this.ui.clearInput();
    this.ui.resetTextareaHeight?.();
    this.ui.toggleLoading(true);

    let chatId = this.activeChatId;
    if (!chatId) {
      try {
        chatId = await this.api.createChat(prompt);
        window.history.pushState({ chatId }, "", `/chat?id=${chatId}`);
        this.activeChatId = chatId;
        await this.loadHistory();
        this.ui.setActiveChat(chatId);
      } catch (err) {
        this.ui.displayMessage("Error: Could not create a new chat session.", "assistant");
        this.ui.toggleLoading(false);
        return;
      }
    }

    // Deduct credits immediately
    if (window.creditManager) {
      const chargeAmount = Math.max(100, prompt.length);
      window.creditManager.onQuestionAsked(chargeAmount);
      console.log(`[ChatApp] Deducted ${chargeAmount} credits`);
    }

    this.streamResponse(chatId, prompt);
  }

  streamResponse(chatId, prompt) {
    this.ui.displayMessage("", "assistant", { showLoader: true });
    this.ui.updateSkeletonStatus?.('searching', 'Searching knowledge base...');
    setTimeout(() => this.ui.updateSkeletonStatus?.('processing', 'Processing results...'), 800);
    setTimeout(() => this.ui.updateSkeletonStatus?.('thinking', 'AI is thinking...'), 1600);

    const streamRenderer = new ChatStreamRenderer(this.ui);
    const eventSource = this.api.createStream(chatId, prompt);

    eventSource.onmessage = (evt) => {
      this.ui.replaceSkeletonWithContent?.();
      streamRenderer.appendChunk(evt.data);
    };

    eventSource.addEventListener("metadata", (evt) => {
      try {
        const metadata = JSON.parse(evt.data);
        if (metadata.type === "sources") this.ui.setSources?.(metadata.sources);
        else if (metadata.type === "final_sources") {
          this.ui.setSources?.(metadata.sources);
          this.ui.addFootnote?.(metadata.sources);
        }
      } catch (err) {
        console.warn("[ChatApp] Failed to parse metadata:", err);
      }
    });

    eventSource.addEventListener("done", () => {
      if (eventSource.readyState !== EventSource.CLOSED) eventSource.close();
      streamRenderer.finalize();
      const sources = this.ui.getSources?.();
      if (sources?.length) this.ui.addFootnote?.(sources);
      this.ui.toggleLoading(false);
      this.removeSkeletonLoaders();
    });

    eventSource.onerror = (err) => {
      console.error("[ChatApp] Stream error:", err);
      if (eventSource.readyState !== EventSource.CLOSED) eventSource.close();
      streamRenderer.destroy();

      // Refresh balance after error
      if (window.creditManager && !window.creditManager.isLoading) {
        setTimeout(() => window.creditManager.loadBalance(), 1000);
      }

      this.ui.displayMessage("A streaming error occurred. Please try again.", "assistant");
      this.ui.toggleLoading(false);
      this.removeSkeletonLoaders();
    };
  }

  handleTextareaInput() {
    const textarea = this.elements.chatInput;
    textarea.style.height = 'auto';
    textarea.style.height = `${textarea.scrollHeight}px`;
  }

  handleTextareaKeydown(e) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      this.elements.chatForm.requestSubmit();
    }
  }

  handleQuickActionClick(e) {
    if (e.target.classList.contains('quick-action-chip')) {
      const prompt = e.target.textContent;
      this.elements.chatInput.value = prompt;
      this.elements.chatForm.requestSubmit();
    }
  }

  bindEvents() {
    this.elements.chatForm.addEventListener("submit", (e) => this.handleSubmit(e));
    this.elements.chatInput.addEventListener('input', () => this.handleTextareaInput());
    this.elements.chatInput.addEventListener('keydown', (e) => this.handleTextareaKeydown(e));

    this.elements.newChatBtn.addEventListener("click", (e) => {
      e.preventDefault();
      if (this.activeChatId || this.elements.chatMessages.children.length > 0) {
        window.history.pushState({}, "", "/chat");
        this.loadMessages(null);
      }
    });

    this.elements.historyList.addEventListener("click", async (e) => {
      const link = e.target.closest("a");
      if (!link) return;

      const historyItem = link.parentElement;
      if (!historyItem || !historyItem.classList.contains("history-item")) return;

      if (e.target.classList.contains("delete-chat-btn")) {
        await this.handleDeleteChat(e, historyItem);
      } else {
        this.handleHistoryLink(e, historyItem);
      }
    });

    if (this.elements.quickActionsContainer) {
      this.elements.quickActionsContainer.addEventListener('click', (e) => this.handleQuickActionClick(e));
    }

    // Handle credit balance updates
    document.addEventListener('balanceUpdated', (event) => {
      const { percentage } = event.detail;
      if (percentage === 0) {
        this.elements.chatInput.disabled = true;
        this.elements.submitButton.disabled = true;
        this.elements.chatInput.placeholder = "No credits remaining - unable to ask questions";
        this.ui.clearQuickActions();
      } else {
        this.elements.chatInput.disabled = false;
        this.elements.submitButton.disabled = false;
        this.elements.chatInput.placeholder = "Ask me anything…";
      }
    });
  }

  async handleDeleteChat(e, item) {
    e.preventDefault();
    e.stopPropagation();

    const chatId = item?.getAttribute("data-chat-id");
    if (!chatId) return;

    if (confirm("Are you sure you want to delete this chat?")) {
      try {
        await this.api.deleteChat(chatId);
        item.remove();
        if (chatId === this.activeChatId) {
          this.elements.newChatBtn.click();
        }
      } catch (err) {
        console.error("[ChatApp] Failed to delete chat:", err);
      }
    }
  }

  handleHistoryLink(e, item) {
    e.preventDefault();
    const chatId = item?.getAttribute("data-chat-id");

    if (chatId && chatId !== this.activeChatId) {
      console.log("[ChatApp] Navigating to chat:", chatId);
      window.history.pushState({ chatId }, "", `/chat?id=${chatId}`);
      this.loadMessages(chatId);
    }
  }
}

document.addEventListener("DOMContentLoaded", () => {
  const app = new ChatApp();
  if (app.elements.chatForm) {
    app.init();
  }
});
