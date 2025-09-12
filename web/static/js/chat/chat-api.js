// G:\go_internist\web\static\js\chat\chat-api.js
// Production-ready Chat API with pagination support

class ChatAPI {
    constructor() {
        this.baseURL = '/api';
        this.defaultTimeout = 10000;
    }

    // ✅ FIXED: Handle new pagination response format
    async getChats(page = 1, limit = 20) {
        try {
            console.log(`[ChatAPI] Fetching chats: page=${page}, limit=${limit}`);
            
            const response = await fetch(`${this.baseURL}/chats?page=${page}&limit=${limit}`, {
                method: 'GET',
                headers: {
                    'Accept': 'application/json',
                    'Content-Type': 'application/json'
                },
                timeout: this.defaultTimeout
            });

            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }

            const data = await response.json();
            console.log(`[ChatAPI] Raw response:`, data);

            // ✅ Handle both old format (array) and new format (object with pagination)
            let chats, pagination;
            
            if (Array.isArray(data)) {
                // Old format: just array of chats
                chats = data;
                pagination = { total: data.length, page: 1, limit: data.length, has_more: false };
            } else {
                // New format: object with pagination metadata
                chats = data.chats || [];
                pagination = {
                    total: data.total || 0,
                    page: data.page || 1,
                    limit: data.limit || 20,
                    has_more: data.has_more || false
                };
            }

            console.log(`[ChatAPI] Processed ${chats.length} chats (total: ${pagination.total})`);
            return { chats, pagination };

        } catch (error) {
            console.error('[ChatAPI] Failed to fetch chats:', error);
            throw new Error(`Failed to load chats: ${error.message}`);
        }
    }

    async getChatMessages(chatId) {
        try {
            console.log(`[ChatAPI] Fetching messages for chat ${chatId}`);
            
            const response = await fetch(`${this.baseURL}/chats/${chatId}/messages`, {
                method: 'GET',
                headers: {
                    'Accept': 'application/json',
                    'Content-Type': 'application/json'
                }
            });

            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }

            const data = await response.json();
            
            // Handle both direct array and object with messages property
            const messages = Array.isArray(data) ? data : (data.messages || []);
            
            console.log(`[ChatAPI] Retrieved ${messages.length} messages for chat ${chatId}`);
            return messages;

        } catch (error) {
            console.error(`[ChatAPI] Failed to fetch messages for chat ${chatId}:`, error);
            throw new Error(`Failed to load messages: ${error.message}`);
        }
    }

    // ✅ ENHANCED: Auto-initialization with pagination
    async initialize() {
        try {
            console.log('[ChatAPI] Initializing chat system...');
            
            const data = await this.getChats(1, 50); // Load first 50 chats
            const chats = data.chats || [];
            const pagination = data.pagination || {};
            
            if (chats.length > 0) {
                const firstChat = chats[0];
                const messages = await this.getChatMessages(firstChat.id);
                
                return {
                    chats,
                    selectedChat: firstChat,
                    messages,
                    hasHistory: true,
                    pagination
                };
            }
            
            return {
                chats: [],
                selectedChat: null,
                messages: [],
                hasHistory: false,
                pagination: { total: 0, page: 1, limit: 50, has_more: false }
            };

        } catch (error) {
            console.error('[ChatAPI] Initialization failed:', error);
            return {
                chats: [],
                selectedChat: null,
                messages: [],
                hasHistory: false,
                error: error.message
            };
        }
    }

    async createChat(title) {
        try {
            const response = await fetch(`${this.baseURL}/chats`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Accept': 'application/json'
                },
                body: JSON.stringify({ title })
            });

            if (!response.ok) {
                throw new Error(`Failed to create chat: ${response.statusText}`);
            }

            const newChat = await response.json();
            console.log('[ChatAPI] Created new chat:', newChat);
            return newChat;

        } catch (error) {
            console.error('[ChatAPI] Failed to create chat:', error);
            throw error;
        }
    }

    async deleteChat(chatId) {
        try {
            const response = await fetch(`${this.baseURL}/chats/${chatId}`, {
                method: 'DELETE'
            });

            if (!response.ok) {
                throw new Error(`Failed to delete chat: ${response.statusText}`);
            }

            console.log(`[ChatAPI] Deleted chat ${chatId}`);
            return true;

        } catch (error) {
            console.error(`[ChatAPI] Failed to delete chat ${chatId}:`, error);
            throw error;
        }
    }

    async getUserBalance() {
        try {
            const response = await fetch(`${this.baseURL}/user/balance`);
            
            if (!response.ok) {
                throw new Error(`Failed to get balance: ${response.statusText}`);
            }

            const data = await response.json();
            return data.balance || data;

        } catch (error) {
            console.error('[ChatAPI] Failed to get user balance:', error);
            return 0;
        }
    }

    // ✅ STREAMING: Enhanced SSE connection for real-time responses
    async streamChatResponse(chatId, prompt, onDelta, onSources, onComplete, onError) {
        try {
            const url = `${this.baseURL}/chats/${chatId}/stream?q=${encodeURIComponent(prompt)}`;
            console.log(`[ChatAPI] Starting stream for chat ${chatId}`);

            const eventSource = new EventSource(url);
            let isCompleted = false;

            eventSource.onmessage = (event) => {
                if (event.data && event.data.trim() && onDelta) {
                    onDelta(event.data);
                }
            };

            eventSource.addEventListener('metadata', (event) => {
                try {
                    const data = JSON.parse(event.data);
                    if (data.type === 'medical_sources' && data.sources && onSources) {
                        onSources(data.sources);
                    }
                } catch (error) {
                    console.warn('[ChatAPI] Failed to parse metadata:', error);
                }
            });

            eventSource.addEventListener('complete', (event) => {
                console.log('[ChatAPI] Stream completed');
                isCompleted = true;
                eventSource.close();
                if (onComplete) onComplete();
            });

            eventSource.addEventListener('done', (event) => {
                if (!isCompleted) {
                    console.log('[ChatAPI] Stream finished');
                    eventSource.close();
                    if (onComplete) onComplete();
                }
            });

            eventSource.addEventListener('error', (event) => {
                console.error('[ChatAPI] Stream error:', event);
                eventSource.close();
                if (onError) onError(new Error('Stream connection failed'));
            });

            eventSource.onerror = (error) => {
                console.error('[ChatAPI] EventSource error:', error);
                eventSource.close();
                if (onError) onError(new Error('Connection error'));
            };

            return eventSource;

        } catch (error) {
            console.error('[ChatAPI] Stream setup failed:', error);
            if (onError) onError(error);
            throw error;
        }
    }
}

// ✅ Export singleton instance
const chatAPI = new ChatAPI();
window.ChatAPI = chatAPI;

console.log('✅ ChatAPI loaded and ready');
