// File: web/static/js/chat/chat.js  
// UPDATED: Added auto-resizing textarea, enter-to-send, and quick actions.

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
      quickActionsContainer: document.getElementById("quickActions"), // NEW: Quick actions container
      submitButton: null
    };
    
    if (!this.elements.chatForm) {
      console.error("Required chat elements not found");
      return;
    }
    
    this.elements.submitButton = this.elements.chatForm.querySelector("button");
    this.activeChatId = Utils.getUrlParam("id");
    
    this.ui = new ChatUI(this.elements);
    this.api = new ChatAPI();
    this.isInitialized = false;
  }

  async init() {
    try {
      this.bindEvents();
      await this.loadHistory();
      
      if (this.activeChatId) {
        await this.loadMessages(this.activeChatId);
      } else {
        // NEW: If no active chat, show quick actions
        this.ui.renderQuickActions([
            "What is Go?", 
            "Explain project structure", 
            "Write a simple REST API"
        ]);
      }

      this.isInitialized = true;
      console.log("Chat app initialized successfully");
      
    } catch (err) {
      console.error("Failed to initialize chat app:", err);
    }
  }

  async loadHistory() {
    try {
      const chats = await this.api.fetchHistory();
      this.ui.renderHistory(chats);
      this.ui.setActiveChat(this.activeChatId);
    } catch (err) {
      console.error("Failed to load history:", err);
    }
  }

  async loadMessages(chatId) {
    this.ui.clearQuickActions(); // NEW: Clear quick actions when loading a chat
    if (!chatId) {
      this.ui.clearMessages();
      this.activeChatId = null;
      this.ui.setActiveChat(null);
      this.ui.renderQuickActions([ // NEW: Show quick actions on a new chat
        "What is Go?", 
        "Explain project structure", 
        "Write a simple REST API"
      ]);
      return;
    }

    try {
      const messages = await this.api.fetchMessages(chatId);
      if (messages === null) return;

      this.ui.renderMessages(messages);
      this.activeChatId = chatId;
      this.ui.setActiveChat(chatId);
    } catch (err) {
      console.error("Failed to load messages:", err);
    }
  }

  async handleSubmit(e) {
    e.preventDefault();
    
    const prompt = this.elements.chatInput.value.trim();
    if (!prompt) return;

    this.ui.clearQuickActions(); // NEW: Clear quick actions on first message
    this.ui.displayMessage(prompt, "user");
    this.ui.clearInput();
    this.ui.resetTextareaHeight(); // NEW: Reset textarea height after sending
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

    this.streamResponse(chatId, prompt);
  }

  streamResponse(chatId, prompt) {
    this.ui.displayMessage("", "assistant");
    this.ui.updateSkeletonStatus('searching', 'Searching knowledge base...');
    setTimeout(() => this.ui.updateSkeletonStatus('processing', 'Processing results...'), 800);
    setTimeout(() => this.ui.updateSkeletonStatus('thinking', 'AI is thinking...'), 1600);

    const streamRenderer = new ChatStreamRenderer(this.ui);
    const eventSource = this.api.createStream(chatId, prompt);

    eventSource.onmessage = (evt) => {
      this.ui.replaceSkeletonWithContent();
      streamRenderer.appendChunk(evt.data);
    };

    eventSource.addEventListener("metadata", (evt) => {
      try {
        const metadata = JSON.parse(evt.data);
        if (metadata.type === "sources") this.ui.setSources(metadata.sources);
        else if (metadata.type === "final_sources") {
          this.ui.setSources(metadata.sources);
          this.ui.addFootnote(metadata.sources);
        }
      } catch (err) {
        console.warn("Failed to parse metadata:", err);
      }
    });

    eventSource.addEventListener("done", () => {
      eventSource.close();
      streamRenderer.finalize();
      const sources = this.ui.getSources();
      if (sources && sources.length > 0) this.ui.addFootnote(sources);
      this.ui.toggleLoading(false);
    });

    eventSource.onerror = (err) => {
      console.error("Stream error:", err);
      eventSource.close();
      streamRenderer.destroy();
      this.ui.displayMessage("A streaming error occurred. Please try again.", "assistant");
      this.ui.toggleLoading(false);
    };
  }
  
  // NEW: Handle textarea auto-resize
  handleTextareaInput() {
      const textarea = this.elements.chatInput;
      textarea.style.height = 'auto'; // Reset height
      textarea.style.height = `${textarea.scrollHeight}px`; // Set to content height
  }
  
  // NEW: Handle "Enter to Send"
  handleTextareaKeydown(e) {
      if (e.key === 'Enter' && !e.shiftKey) {
          e.preventDefault();
          this.elements.chatForm.requestSubmit();
      }
  }
  
  // NEW: Handle clicking on a quick action chip
  handleQuickActionClick(e) {
      if (e.target.classList.contains('quick-action-chip')) {
          const prompt = e.target.textContent;
          this.elements.chatInput.value = prompt;
          this.elements.chatForm.requestSubmit();
      }
  }

  bindEvents() {
    this.elements.chatForm.addEventListener("submit", (e) => this.handleSubmit(e));
    this.elements.chatInput.addEventListener('input', () => this.handleTextareaInput()); // NEW
    this.elements.chatInput.addEventListener('keydown', (e) => this.handleTextareaKeydown(e)); // NEW

    this.elements.newChatBtn.addEventListener("click", (e) => {
      e.preventDefault();
      if (this.activeChatId || this.elements.chatMessages.children.length > 0) {
        window.history.pushState({}, "", "/chat");
        this.loadMessages(null);
      }
    });

    this.elements.historyList.addEventListener("click", async (e) => {
      const link = e.target.closest("a.history-item");
      if (!link) return;

      if (e.target.classList.contains("delete-chat-btn")) {
        await this.handleDeleteChat(e, link);
      } else {
        this.handleHistoryLink(e, link);
      }
    });
    
    // NEW: Add event listener for quick actions
    this.elements.quickActionsContainer.addEventListener('click', (e) => this.handleQuickActionClick(e));
  }

  async handleDeleteChat(e, item) {
    // UPDATED: Now receives the item directly
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
        console.error("Failed to delete chat:", err);
      }
    }
  }

  handleHistoryLink(e, item) {
    // UPDATED: Now receives the item directly
    e.preventDefault();
    
    const chatId = item?.getAttribute("data-chat-id");
    
    if (chatId && chatId !== this.activeChatId) {
      window.history.pushState({ chatId }, "", item.href);
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