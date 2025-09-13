// File: web/static/js/chat/chat-api.js
// FIXED: Extract nested data from wrapped API responses

import { Utils } from '../utils.js';

export class ChatAPI {
  constructor() {
    this.baseUrl = '/api';
  }

  // --- FIXED API Methods ---

  async fetchHistory() {
    const response = await this._request('/chats');
    // FIXED: Extract chats array from wrapped response
    return response.chats || response || [];
  }

  async fetchMessages(chatId) {
    const response = await this._request(`/chats/${chatId}/messages`, {}, { handle404: true });
    if (response === null) {
        window.location.href = "/chat";
        return null;
    }
    // FIXED: Extract messages array from wrapped response
    return response.messages || response || [];
  }

  async createChat(title) {
    const newChat = await this._request('/chats', {
      method: 'POST',
      body: JSON.stringify({ title }),
    });
    const createdId = newChat.ID ?? newChat.id;
    if (!createdId) throw new Error("CreateChat response missing id");
    return String(createdId);
  }

  async deleteChat(chatId) {
    await this._request(`/chats/${chatId}`, { method: 'DELETE' });
    return true;
  }

  createStream(chatId, prompt) {
    const url = `${this.baseUrl}/chats/${chatId}/stream?q=${encodeURIComponent(prompt)}`;
    return new EventSource(url);
  }

  // --- Rest of your existing methods remain the same ---
  async _request(endpoint, options = {}, config = {}) {
    const url = this.baseUrl + endpoint;
    const method = options.method || 'GET';

    const fetchOptions = {
      ...options,
      method: method,
      headers: this._getHeaders(method, !!options.body),
      credentials: 'same-origin',
      cache: 'no-store'
    };

    try {
      const res = await fetch(url, fetchOptions);

      if (!res.ok) {
        if (res.status === 404 && config.handle404) {
            return null;
        }
        throw new Error(`API request failed: ${res.status} ${res.statusText}`);
      }
      
      const contentType = res.headers.get("content-type");
      if (contentType && contentType.includes("application/json")) {
        return await res.json();
      }
      return true;

    } catch (err) {
      console.error(`Failed to ${method} ${endpoint}:`, err);
      Utils.safeLog('error', `API call failed for ${method} ${endpoint}`, { 
        errorMessage: err.message, 
        stack: err.stack 
      });
      throw err;
    }
  }

  _getHeaders(method, hasBody = false) {
    const headers = {};
    if (hasBody) {
      headers['Content-Type'] = 'application/json';
    }

    if (['POST', 'PUT', 'DELETE', 'PATCH'].includes(method)) {
      const csrf = document.querySelector("input[name='csrf_token']")?.value;
      if (csrf) {
        headers['X-CSRF-Token'] = csrf;
      }
    }
    return headers;
  }
}
