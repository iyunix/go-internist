// File: web/static/js/chat/chat-api.js
// REFACTORED: Centralized fetch logic and added CSRF protection to DELETE.

import { Utils } from '../utils.js';

export class ChatAPI {
  constructor() {
    this.baseUrl = '/api'; // Using a base URL for easier maintenance
  }

  // --- Public API Methods ---

  async fetchHistory() {
    return this._request('/chats');
  }

  async fetchMessages(chatId) {
    const response = await this._request(`/chats/${chatId}/messages`, {}, { handle404: true });
    if (response === null) { // Special handling for 404
        window.location.href = "/chat";
        return null;
    }
    return response;
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

  // --- Private Helper Methods ---

  /**
   * Centralized method for making API requests.
   * @param {string} endpoint - The API endpoint (e.g., '/chats').
   * @param {object} options - Options for the fetch call (method, body, etc.).
   * @param {object} config - Internal configuration for this helper.
   * @returns {Promise<any>} The JSON response.
   */
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
            return null; // Signal 404 to the caller
        }
        throw new Error(`API request failed: ${res.status} ${res.statusText}`);
      }
      
      // Handle responses that might not have a body (like DELETE)
      const contentType = res.headers.get("content-type");
      if (contentType && contentType.includes("application/json")) {
        return await res.json();
      }
      return true; // For successful non-JSON responses

    } catch (err) {
      console.error(`Failed to ${method} ${endpoint}:`, err);
      Utils.safeLog('error', `API call failed for ${method} ${endpoint}`, { 
        errorMessage: err.message, 
        stack: err.stack 
      });
      throw err;
    }
  }

  /**
   * Generates the required headers for an API request.
   * @param {string} method - The HTTP method.
   * @param {boolean} hasBody - Whether the request has a body.
   * @returns {HeadersInit} The headers object.
   */
  _getHeaders(method, hasBody = false) {
    const headers = {};
    if (hasBody) {
      headers['Content-Type'] = 'application/json';
    }

    // Add CSRF token to all state-changing methods
    if (['POST', 'PUT', 'DELETE', 'PATCH'].includes(method)) {
      const csrf = document.querySelector("input[name='csrf_token']")?.value;
      if (csrf) {
        headers['X-CSRF-Token'] = csrf;
      }
    }
    return headers;
  }
}