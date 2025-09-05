// File: web/static/js/chat/chat-api.js
// Handles API calls and data fetching

import { Utils } from '../utils.js';

export class ChatAPI {
  constructor() {
    this.baseUrl = '';
  }

  // Fetch chat history
  async fetchHistory() {
    try {
      const res = await fetch("/api/chats", {
        method: "GET",
        credentials: "same-origin",
        cache: "no-store"
      });
      
      if (!res.ok) throw new Error(`Failed to load history: ${res.statusText}`);
      return await res.json();
    } catch (err) {
      console.error("Failed to load history:", err);
      Utils.safeLog('error', 'Failed to load chat history', { 
        errorMessage: err.message, 
        stack: err.stack 
      });
      throw err;
    }
  }

  // Fetch messages for a specific chat
  async fetchMessages(chatId) {
    try {
      const res = await fetch(`/api/chats/${chatId}/messages`, {
        method: "GET",
        credentials: "same-origin",
        cache: "no-store"
      });
      
      if (!res.ok) {
        if (res.status === 404) {
          window.location.href = "/chat";
          return null;
        }
        throw new Error(`Failed to load messages: ${res.statusText}`);
      }
      
      return await res.json();
    } catch (err) {
      console.error("Failed to load messages:", err);
      Utils.safeLog('error', 'Failed to load messages for chat', { 
        chatId: chatId, 
        errorMessage: err.message, 
        stack: err.stack 
      });
      throw err;
    }
  }

  // Create a new chat
  async createChat(title) {
    try {
      const csrf = document.querySelector("input[name='csrf_token']")?.value || "";
      const res = await fetch("/api/chats", {
        method: "POST",
        credentials: "same-origin",
        cache: "no-store",
        headers: {
          "Content-Type": "application/json",
          ...(csrf ? { "X-CSRF-Token": csrf } : {})
        },
        body: JSON.stringify({ title }),
      });
      
      if (!res.ok) throw new Error(`Failed to create chat: ${res.statusText}`);
      
      const newChat = await res.json();
      const createdId = newChat.ID ?? newChat.id;
      if (!createdId) throw new Error("CreateChat response missing id");
      
      return String(createdId);
    } catch (err) {
      console.error("Failed to create chat:", err);
      Utils.safeLog('error', 'Failed to create new chat', { 
        errorMessage: err.message, 
        stack: err.stack 
      });
      throw err;
    }
  }

  // Delete a chat
  async deleteChat(chatId) {
    try {
      const res = await fetch(`/api/chats/${chatId}`, { 
        method: "DELETE", 
        credentials: "same-origin", 
        cache: "no-store" 
      });
      
      if (!res.ok) throw new Error(`Failed to delete chat: ${res.statusText}`);
      return true;
    } catch (err) {
      console.error("Failed to delete chat:", err);
      Utils.safeLog('error', 'Failed to delete chat', { 
        chatId: chatId, 
        errorMessage: err.message, 
        stack: err.stack 
      });
      throw err;
    }
  }

  // Create EventSource for streaming
  createStream(chatId, prompt) {
    const url = `/api/chats/${chatId}/stream?q=${encodeURIComponent(prompt)}`;
    return new EventSource(url);
  }
}
