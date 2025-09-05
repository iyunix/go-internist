// File: web/static/js/chat/chat.js  
// FIXED: Uses new ChatStreamRenderer properly

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
    if (!chatId) {
      this.ui.clearMessages();
      this.activeChatId = null;
      this.ui.setActiveChat(null);
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

    this.ui.displayMessage(prompt, "user");
    this.ui.clearInput();
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

  // FIXED: Uses new ChatStreamRenderer
  streamResponse(chatId, prompt) {
    // Create empty assistant message first
    this.ui.displayMessage("", "assistant");
    
    // Create stream renderer that updates this message
    const streamRenderer = new ChatStreamRenderer(this.ui);
    const eventSource = this.api.createStream(chatId, prompt);

    eventSource.onmessage = (evt) => {
      streamRenderer.appendChunk(evt.data);
    };

    eventSource.addEventListener("done", () => {
      eventSource.close();
      streamRenderer.finalize();
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

  bindEvents() {
    this.elements.chatForm.addEventListener("submit", (e) => this.handleSubmit(e));

    this.elements.newChatBtn.addEventListener("click", (e) => {
      e.preventDefault();
      if (this.activeChatId) {
        window.history.pushState({}, "", "/chat");
        this.loadMessages(null);
      }
    });

    this.elements.historyList.addEventListener("click", async (e) => {
      if (e.target.classList.contains("delete-chat-btn")) {
        await this.handleDeleteChat(e);
      } else if (e.target.tagName === 'A') {
        this.handleHistoryLink(e);
      }
    });
  }

  async handleDeleteChat(e) {
    e.preventDefault();
    e.stopPropagation();
    
    const item = e.target.closest("li.history-item");
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

  handleHistoryLink(e) {
    e.preventDefault();
    
    const item = e.target.closest("li.history-item");
    const chatId = item?.getAttribute("data-chat-id");
    
    if (chatId && chatId !== this.activeChatId) {
      window.history.pushState({ chatId }, "", e.target.href);
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
