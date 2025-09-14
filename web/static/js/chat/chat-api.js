// File: web/static/js/chat/chat-api.js
// IMPROVED: Consistent JSON handling, robust envelope unwrapping, Accept header,
// safe SSE URL construction, and API-layer purity (no UI redirects).

import { Utils } from '../utils.js';

export class ChatAPI {
  constructor() {
    this.baseUrl = '/api';
  }

  // --- Chat APIs ---

  async fetchHistory() {
    const response = await this._request('/chats');
    const payload = this._unwrap(response);
    return payload.chats ?? payload ?? [];
  }

  async fetchMessages(chatId) {
    if (!chatId) return [];
    const response = await this._request(`/chats/${encodeURIComponent(chatId)}/messages`, {}, { handle404: true });
    if (response === null) {
      // Let the UI decide what to do (e.g., navigate away or show a message)
      return null;
    }
    const payload = this._unwrap(response);
    return payload.messages ?? payload ?? [];
  }

  async createChat(title) {
    const newChat = await this._request('/chats', {
      method: 'POST',
      body: JSON.stringify({ title }),
    });
    const unwrapped = this._unwrap(newChat);
    const createdId =
      unwrapped.ID ?? unwrapped.id ?? unwrapped.data?.ID ?? unwrapped.data?.id;
    if (!createdId) throw new Error('CreateChat response missing id');
    return String(createdId);
  }

  async deleteChat(chatId) {
    await this._request(`/chats/${encodeURIComponent(chatId)}`, { method: 'DELETE' });
    return true;
  }

  // Server-Sent Events stream; note: EventSource cannot set custom headers,
  // so auth/CSRF must rely on same-origin cookies.
  createStream(chatId, prompt) {
    const url = new URL(`${this.baseUrl}/chats/${encodeURIComponent(chatId)}/stream`, window.location.origin);
    url.search = new URLSearchParams({ q: String(prompt ?? '') }).toString();
    // For cross-origin SSE with cookies, pass { withCredentials: true }
    // return new EventSource(url.toString(), { withCredentials: true });
    return new EventSource(url.toString());
  }

  // --- Core request helper ---

  async _request(endpoint, options = {}, config = {}) {
    const url = this.baseUrl + endpoint;
    const method = options.method || 'GET';

    const fetchOptions = {
      ...options,
      method,
      headers: this._getHeaders(method, !!options.body),
      credentials: 'same-origin',
      cache: 'no-store',
    };

    try {
      const res = await fetch(url, fetchOptions);

      if (!res.ok) {
        if (res.status === 404 && config.handle404) {
          return null;
        }
        throw new Error(`API request failed: ${res.status} ${res.statusText}`);
      }

      // 204 No Content short-circuit
      if (res.status === 204) return true;

      const contentType = res.headers.get('content-type') || '';
      if (contentType.toLowerCase().includes('json')) {
        return await res.json();
      }
      // For non-JSON responses we consider success true (e.g., DELETE 200)
      return true;
    } catch (err) {
      console.error(`Failed to ${method} ${endpoint}:`, err);
      Utils.safeLog('error', `API call failed for ${method} ${endpoint}`, {
        errorMessage: err.message,
        stack: err.stack,
      });
      throw err;
    }
  }

  _getHeaders(method, hasBody = false) {
    const headers = {
      Accept: 'application/json',
    };
    if (hasBody) {
      headers['Content-Type'] = 'application/json';
    }

    // Attach CSRF token for state-changing requests
    if (['POST', 'PUT', 'DELETE', 'PATCH'].includes(method)) {
      const csrf = document.querySelector("input[name='csrf_token']")?.value;
      if (csrf) {
        headers['X-CSRF-Token'] = csrf;
      }
    }
    return headers;
  }

  // Normalize various envelope shapes
  _unwrap(obj) {
    if (!obj || typeof obj !== 'object') return obj;
    if (obj.data && typeof obj.data === 'object') return obj.data;
    return obj;
  }
}
