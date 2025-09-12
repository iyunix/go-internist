// G:\go_internist\web\static\js\chat\chat.js
// Main Chat Controller with real-time streaming and pagination

class ChatManager {
    constructor() {
        this.currentChatId = null;
        this.isInitialized = false;
        this.isStreaming = false;
        this.currentStream = null;
        this.streamingContent = '';
        this.currentPagination = null;

        console.log('[ChatManager] Instance created');
    }

    // ✅ INITIALIZATION: Auto-load with pagination support
    async initialize() {
        if (this.isInitialized) {
            console.log('[ChatManager] Already initialized, skipping...');
            return;
        }

        console.log('🚀 [ChatManager] Initializing...');

        try {
            // Show loading state
            ChatUI.showLoadingState('Loading your medical consultations...');

            // Initialize with pagination
            const history = await ChatAPI.initialize();

            if (history.hasHistory) {
                console.log(`✅ Found ${history.chats.length} chats, auto-selecting first chat`);

                // Display chat list with pagination info
                ChatUI.renderChatList(history.chats, history.pagination);
                this.currentPagination = history.pagination;

                // Auto-select and display first chat
                await this.selectChat(history.selectedChat.id, history.messages);
                console.log('✅ Chat history loaded successfully');

            } else {
                console.log('ℹ️ No chat history found, showing empty state');
                ChatUI.showEmptyState();
            }

            // Load user balance
            await this.updateUserBalance();

            // Set up event listeners
            this.setupEventListeners();

            this.isInitialized = true;
            console.log('🎉 ChatManager initialization complete');

        } catch (error) {
            console.error('❌ Chat initialization failed:', error);
            ChatUI.showError('Failed to load chat history. Please refresh the page.');
        }
    }

    // ✅ CHAT: Enhanced chat selection with message display
    async selectChat(chatId, preloadedMessages = null) {
        try {
            console.log(`[ChatManager] Selecting chat ${chatId}...`);

            // Update current chat ID
            this.currentChatId = chatId;

            // Update UI to show selected chat
            ChatUI.setActiveChat(chatId);

            // Load messages (use preloaded if available)
            let messages;
            if (preloadedMessages) {
                messages = preloadedMessages;
                console.log(`Using preloaded ${messages.length} messages`);
            } else {
                console.log('Loading messages for selected chat...');
                messages = await ChatAPI.getChatMessages(chatId);
            }

            // Display messages immediately
            ChatUI.renderMessages(messages);

            // Enable message input
            ChatUI.enableMessageInput();

            console.log(`✅ Chat ${chatId} selected with ${messages.length} messages`);

        } catch (error) {
            console.error(`❌ Failed to select chat ${chatId}:`, error);
            ChatUI.showError('Failed to load chat messages');
        }
    }

    // ✅ STREAMING: Enhanced message sending with real-time rendering
    async sendMessage(content) {
        if (!this.currentChatId || !content.trim()) {
            console.warn('Cannot send message: no chat selected or empty content');
            return;
        }

        if (this.isStreaming) {
            console.warn('Already streaming, ignoring new message');
            return;
        }

        try {
            console.log(`[ChatManager] Sending message to chat ${this.currentChatId}: "${content}"`);

            this.isStreaming = true;
            this.streamingContent = '';

            // Add user message to UI immediately
            ChatUI.addUserMessage(content);

            // Disable input during streaming
            ChatUI.disableMessageInput('AI is thinking...');

            // Show AI response placeholder
            const messageId = ChatUI.showAIResponsePlaceholder();

            // ✅ REAL-TIME STREAMING: Start streaming with enhanced callbacks
            this.currentStream = await ChatAPI.streamChatResponse(
                this.currentChatId,
                content,
                // onDelta - real-time markdown rendering
                (delta) => {
                    this.streamingContent += delta;
                    ChatUI.updateAIResponse(messageId, this.streamingContent);
                },
                // onSources - medical references
                (sources) => {
                    ChatUI.displaySources(sources);
                },
                // onComplete - streaming finished
                () => {
                    console.log('✅ Streaming completed');
                    this.isStreaming = false;
                    ChatUI.enableMessageInput();
                    ChatUI.finalizeAIResponse(messageId, this.streamingContent);
                    this.updateUserBalance(); // Refresh balance after message
                    this.streamingContent = '';
                },
                // onError - handle streaming errors
                (error) => {
                    console.error('❌ Streaming error:', error);
                    this.isStreaming = false;
                    ChatUI.enableMessageInput();
                    ChatUI.showError('Failed to get AI response. Please try again.');
                    this.streamingContent = '';
                }
            );

        } catch (error) {
            console.error('❌ Failed to send message:', error);
            this.isStreaming = false;
            ChatUI.enableMessageInput();
            ChatUI.showError('Failed to send message');
        }
    }

    // ✅ PAGINATION: Load more chats
    async loadMoreChats() {
        if (!this.currentPagination || !this.currentPagination.has_more) {
            console.log('No more chats to load');
            return;
        }

        try {
            const nextPage = this.currentPagination.page + 1;
            console.log(`Loading more chats: page ${nextPage}`);

            const data = await ChatAPI.getChats(nextPage, this.currentPagination.limit);
            const newChats = data.chats || [];
            const newPagination = data.pagination || {};

            if (newChats.length > 0) {
                // Get current chats
                const currentChats = Array.from(document.querySelectorAll('.chat-item')).map(el => ({
                    id: el.dataset.chatId,
                    element: el
                }));

                // Combine and re-render
                const allChats = [...this.getCurrentChats(), ...newChats];
                ChatUI.renderChatList(allChats, newPagination);
                this.currentPagination = newPagination;

                console.log(`✅ Loaded ${newChats.length} more chats`);
            }

        } catch (error) {
            console.error('Failed to load more chats:', error);
            ChatUI.showError('Failed to load more chats');
        }
    }

    // ✅ CHAT: Create new chat
    async createNewChat() {
        try {
            const title = prompt('Enter chat title:', 'New Medical Consultation');
            if (!title || title.trim() === '') {
                return;
            }

            console.log(`Creating new chat: "${title}"`);
            ChatUI.showLoadingState('Creating new chat...');

            const newChat = await ChatAPI.createChat(title.trim());

            // Refresh chat list
            await this.refreshChatList();

            // Select the new chat
            await this.selectChat(newChat.id);

            console.log('✅ New chat created and selected');

        } catch (error) {
            console.error('❌ Failed to create new chat:', error);
            ChatUI.showError('Failed to create new chat');
        }
    }

    // ✅ CHAT: Delete chat
    async deleteChat(chatId) {
        if (!confirm('Are you sure you want to delete this medical consultation?')) {
            return;
        }

        try {
            console.log(`Deleting chat ${chatId}...`);

            await ChatAPI.deleteChat(chatId);

            // If deleted chat was selected, clear selection
            if (this.currentChatId === chatId) {
                this.currentChatId = null;
                ChatUI.showEmptyState();
            }

            // Refresh chat list
            await this.refreshChatList();

            console.log('✅ Chat deleted successfully');

        } catch (error) {
            console.error(`❌ Failed to delete chat ${chatId}:`, error);
            ChatUI.showError('Failed to delete chat');
        }
    }

    // ✅ REFRESH: Refresh chat list with pagination
    async refreshChatList() {
        try {
            console.log('Refreshing chat list...');
            const data = await ChatAPI.getChats();
            const chats = data.chats || [];
            const pagination = data.pagination || {};

            ChatUI.renderChatList(chats, pagination);
            this.currentPagination = pagination;

            // If no current chat selected and chats available, select first
            if (!this.currentChatId && chats.length > 0) {
                await this.selectChat(chats[0].id);
            }

        } catch (error) {
            console.error('❌ Failed to refresh chat list:', error);
        }
    }

    // ✅ BALANCE: Update user balance display
    async updateUserBalance() {
        try {
            const balance = await ChatAPI.getUserBalance();
            ChatUI.updateBalance(balance);
        } catch (error) {
            console.warn('Failed to update balance:', error);
        }
    }

    // ✅ EVENTS: Set up event listeners
    setupEventListeners() {
        console.log('[ChatManager] Setting up event listeners...');

        // Message form submission
        const messageForm = document.getElementById('message-form');
        const messageInput = document.getElementById('message-input');

        if (messageForm && messageInput) {
            messageForm.addEventListener('submit', (e) => {
                e.preventDefault();
                const content = messageInput.value.trim();
                if (content && !this.isStreaming) {
                    this.sendMessage(content);
                    messageInput.value = '';
                }
            });

            // Auto-resize textarea
            messageInput.addEventListener('input', () => {
                messageInput.style.height = 'auto';
                messageInput.style.height = messageInput.scrollHeight + 'px';
            });

            // Enter key handling
            messageInput.addEventListener('keydown', (e) => {
                if (e.key === 'Enter' && !e.shiftKey) {
                    e.preventDefault();
                    messageForm.dispatchEvent(new Event('submit'));
                }
            });
        }

        // New chat button
        const newChatBtn = document.getElementById('new-chat-btn');
        if (newChatBtn) {
            newChatBtn.addEventListener('click', () => this.createNewChat());
        }

        // Handle page visibility changes
        document.addEventListener('visibilitychange', () => {
            if (!document.hidden && this.isInitialized) {
                console.log('[ChatManager] Page became visible, refreshing data...');
                this.updateUserBalance();
            }
        });

        console.log('✅ Event listeners set up successfully');
    }

    // ✅ UTILITY: Get current chats from DOM
    getCurrentChats() {
        const chatElements = document.querySelectorAll('.chat-item');
        return Array.from(chatElements).map(el => {
            const titleEl = el.querySelector('.chat-title');
            const timeEl = el.querySelector('.chat-time');
            return {
                id: parseInt(el.dataset.chatId),
                title: titleEl ? titleEl.textContent : 'Unknown',
                updated_at: timeEl ? timeEl.textContent : new Date().toISOString()
            };
        });
    }

    // ✅ CLEANUP: Cleanup method
    destroy() {
        console.log('[ChatManager] Cleaning up...');

        if (this.currentStream) {
            this.currentStream.close();
        }

        this.isInitialized = false;
        this.currentChatId = null;
        this.isStreaming = false;
    }
}

// ✅ Initialize on page load
const chatManager = new ChatManager();
window.chatManager = chatManager;

// Auto-initialization
document.addEventListener('DOMContentLoaded', async function() {
    console.log('🚀 Chat page loaded, starting initialization...');
    try {
        await chatManager.initialize();
    } catch (error) {
        console.error('❌ Failed to initialize chat on page load:', error);
        ChatUI.showError('Failed to initialize chat. Please refresh the page.');
    }
});

// Handle page reload/refresh
window.addEventListener('load', async function() {
    if (!chatManager.isInitialized) {
        console.log('🔄 Window loaded, ensuring chat initialization...');
        await chatManager.initialize();
    }
});

// Handle page focus
window.addEventListener('focus', function() {
    if (chatManager.isInitialized && !chatManager.isStreaming) {
        console.log('[ChatManager] Window focused, refreshing data...');
        chatManager.updateUserBalance();
    }
});

// Cleanup on page unload
window.addEventListener('beforeunload', function() {
    chatManager.destroy();
});

// ✅ Global functions for UI interactions
window.selectChat = (chatId) => chatManager.selectChat(chatId);
window.deleteChat = (chatId) => chatManager.deleteChat(chatId);
window.createNewChat = () => chatManager.createNewChat();

console.log('✅ Chat Manager loaded and ready with real-time streaming support');
