// G:\go_internist\web\static\js\chat\chat-ui.js
// Production-ready Chat UI with real-time markdown rendering

class ChatUI {
    constructor() {
        this.elements = {};
        this.messageCounter = 0;
        this.renderer = window.MarkdownRenderer;
        this.init();
    }

    init() {
        console.log('[ChatUI] Initializing UI elements...');
        
        this.elements = {
            chatList: document.getElementById('chat-list') || document.querySelector('.chat-list'),
            messagesContainer: document.getElementById('messages-container') || document.querySelector('.messages-container'),
            messageInput: document.getElementById('message-input') || document.querySelector('#message-input'),
            messageForm: document.getElementById('message-form') || document.querySelector('#message-form'),
            newChatBtn: document.getElementById('new-chat-btn') || document.querySelector('#new-chat-btn'),
            userBalance: document.getElementById('user-balance') || document.querySelector('#user-balance'),
            currentBalance: document.getElementById('current-balance') || document.querySelector('#current-balance')
        };

        this.ensureRequiredElements();
        console.log('✅ ChatUI elements initialized');
    }

    ensureRequiredElements() {
        if (!this.elements.messagesContainer) {
            console.warn('[ChatUI] Messages container not found, creating...');
            const container = document.createElement('div');
            container.id = 'messages-container';
            container.className = 'messages-container';
            document.querySelector('.chat-main')?.appendChild(container);
            this.elements.messagesContainer = container;
        }

        if (!this.elements.chatList) {
            console.warn('[ChatUI] Chat list not found, creating...');
            const list = document.createElement('div');
            list.id = 'chat-list';
            list.className = 'chat-list';
            document.querySelector('.chat-sidebar')?.appendChild(list);
            this.elements.chatList = list;
        }
    }

    // ✅ RENDER: Chat list with pagination info
    renderChatList(chats, pagination = null) {
        if (!this.elements.chatList) {
            console.error('[ChatUI] Chat list element not found');
            return;
        }

        console.log(`[ChatUI] Rendering ${chats.length} chats`);

        if (chats.length === 0) {
            this.elements.chatList.innerHTML = `
                <div class="no-chats">
                    <div class="no-chats-icon">💬</div>
                    <div class="no-chats-text">No medical consultations yet</div>
                    <button onclick="window.chatManager.createNewChat()" class="btn-primary">
                        Start New Chat
                    </button>
                </div>
            `;
            return;
        }

        let html = chats.map(chat => `
            <div class="chat-item" 
                 data-chat-id="${chat.id}" 
                 onclick="window.chatManager.selectChat(${chat.id})">
                <div class="chat-item-header">
                    <div class="chat-title" title="${this.escapeHtml(chat.title)}">
                        ${this.truncateText(chat.title, 40)}
                    </div>
                    <div class="chat-actions">
                        <button onclick="event.stopPropagation(); window.chatManager.deleteChat(${chat.id})" 
                                class="btn-delete" 
                                title="Delete chat">×</button>
                    </div>
                </div>
                <div class="chat-time">
                    ${this.formatTime(chat.updated_at || chat.created_at)}
                </div>
            </div>
        `).join('');

        // ✅ Add pagination info if available
        if (pagination && pagination.has_more) {
            html += `
                <div class="chat-list-pagination">
                    <div class="pagination-info">
                        Showing ${chats.length} of ${pagination.total} chats
                    </div>
                    <button onclick="window.chatManager.loadMoreChats()" class="btn-secondary">
                        Load More Chats
                    </button>
                </div>
            `;
        }

        this.elements.chatList.innerHTML = html;
        console.log('✅ Chat list rendered successfully');
    }

    // ✅ RENDER: Messages with enhanced markdown support
    renderMessages(messages) {
        if (!this.elements.messagesContainer) {
            console.error('[ChatUI] Messages container not found');
            return;
        }

        console.log(`[ChatUI] Rendering ${messages.length} messages`);

        if (messages.length === 0) {
            this.elements.messagesContainer.innerHTML = `
                <div class="no-messages">
                    <div class="no-messages-icon">🤖</div>
                    <div class="no-messages-text">Start your medical consultation</div>
                    <div class="no-messages-subtitle">Ask any medical question to get started</div>
                </div>
            `;
            return;
        }

        // Clear container
        this.elements.messagesContainer.innerHTML = '';

        // Render each message with enhanced formatting
        messages.forEach(message => {
            const messageElement = this.createMessageElement(message);
            this.elements.messagesContainer.appendChild(messageElement);
        });

        // Scroll to bottom
        this.scrollToBottom();
        console.log('✅ Messages rendered successfully');
    }

    // ✅ CREATE: Enhanced message element with markdown support
    createMessageElement(message) {
        const div = document.createElement('div');
        div.className = `message ${message.message_type}`;
        div.setAttribute('data-message-id', message.id);

        const time = this.formatTime(message.created_at);
        const isUser = message.message_type === 'user';
        const avatar = isUser ? '👤' : '🤖';
        const sender = isUser ? 'You' : 'Medical AI';
        const messageClass = isUser ? 'user-message' : 'ai-message';

        div.innerHTML = `
            <div class="message-content ${messageClass}">
                <div class="message-header">
                    <div class="message-avatar">${avatar}</div>
                    <div class="message-info">
                        <div class="message-sender">${sender}</div>
                        <div class="message-time">${time}</div>
                    </div>
                </div>
                <div class="message-text" data-message-content>
                    <!-- Content will be rendered here -->
                </div>
            </div>
        `;

        // ✅ ENHANCED: Render markdown content
        const contentElement = div.querySelector('[data-message-content]');
        if (this.renderer) {
            this.renderer.renderComplete(message.content, contentElement);
        } else {
            contentElement.innerHTML = this.formatMessageContent(message.content);
        }

        return div;
    }

    // ✅ STREAMING: Add user message immediately
    addUserMessage(content) {
        if (!this.elements.messagesContainer) return;

        const message = {
            id: `temp_user_${Date.now()}`,
            message_type: 'user',
            content: content,
            created_at: new Date().toISOString()
        };

        const messageElement = this.createMessageElement(message);
        this.elements.messagesContainer.appendChild(messageElement);
        this.scrollToBottom();

        console.log('[ChatUI] ✅ User message added to UI');
    }

    // ✅ STREAMING: Show AI response placeholder with real-time updates
    showAIResponsePlaceholder() {
        if (!this.elements.messagesContainer) return null;

        const messageId = `ai_stream_${++this.messageCounter}`;
        const div = document.createElement('div');
        div.className = 'message assistant streaming';
        div.setAttribute('data-message-id', messageId);
        
        div.innerHTML = `
            <div class="message-content ai-message">
                <div class="message-header">
                    <div class="message-avatar">🤖</div>
                    <div class="message-info">
                        <div class="message-sender">Medical AI</div>
                        <div class="message-time">Thinking...</div>
                    </div>
                </div>
                <div class="message-text" data-streaming-content>
                    <div class="typing-indicator">
                        <span></span><span></span><span></span>
                    </div>
                </div>
            </div>
        `;

        this.elements.messagesContainer.appendChild(div);
        this.scrollToBottom();

        return messageId;
    }

    // ✅ STREAMING: Update AI response with real-time markdown rendering
    updateAIResponse(messageId, content) {
        const messageElement = document.querySelector(`[data-message-id="${messageId}"]`);
        if (!messageElement) return;

        const contentElement = messageElement.querySelector('[data-streaming-content]');
        const timeElement = messageElement.querySelector('.message-time');

        if (contentElement) {
            // ✅ REAL-TIME: Render markdown progressively
            if (this.renderer) {
                this.renderer.renderStreaming(content, contentElement);
            } else {
                contentElement.innerHTML = this.formatMessageContent(content);
            }
        }

        if (timeElement) {
            timeElement.textContent = this.formatTime(new Date().toISOString());
        }

        // Remove streaming class and typing indicator
        messageElement.classList.remove('streaming');
        this.scrollToBottom();
    }

    // ✅ STREAMING: Finalize AI response
    finalizeAIResponse(messageId, finalContent) {
        const messageElement = document.querySelector(`[data-message-id="${messageId}"]`);
        if (!messageElement) return;

        const contentElement = messageElement.querySelector('[data-streaming-content]');
        if (contentElement && this.renderer) {
            // ✅ FINAL: Complete markdown rendering with all enhancements
            this.renderer.renderComplete(finalContent, contentElement);
        }

        messageElement.classList.remove('streaming');
        console.log(`[ChatUI] ✅ Finalized AI response ${messageId}`);
    }

    // ✅ SOURCES: Display medical sources with enhanced formatting
    displaySources(sources) {
        if (!sources || sources.length === 0) return;

        console.log(`[ChatUI] Displaying ${sources.length} medical sources`);

        const sourcesHtml = `
            <div class="medical-sources">
                <div class="sources-header">
                    <div class="sources-icon">📚</div>
                    <div class="sources-title">Medical References</div>
                    <div class="sources-count">${sources.length} source${sources.length !== 1 ? 's' : ''}</div>
                </div>
                <div class="sources-list">
                    ${sources.map((source, index) => `
                        <div class="source-item">
                            <div class="source-number">${index + 1}</div>
                            <div class="source-text">${this.escapeHtml(source)}</div>
                        </div>
                    `).join('')}
                </div>
            </div>
        `;

        // Add sources after the last message
        const lastMessage = this.elements.messagesContainer.lastElementChild;
        if (lastMessage) {
            const existingSources = lastMessage.querySelector('.medical-sources');
            if (existingSources) {
                existingSources.outerHTML = sourcesHtml;
            } else {
                lastMessage.insertAdjacentHTML('afterend', sourcesHtml);
            }
        }

        this.scrollToBottom();
    }

    // ✅ UTILITY: Set active chat in sidebar
    setActiveChat(chatId) {
        // Remove active class from all chats
        this.elements.chatList?.querySelectorAll('.chat-item').forEach(item => {
            item.classList.remove('active', 'selected');
        });

        // Add active class to selected chat
        const selectedChat = this.elements.chatList?.querySelector(`[data-chat-id="${chatId}"]`);
        if (selectedChat) {
            selectedChat.classList.add('active', 'selected');
            console.log(`[ChatUI] ✅ Set chat ${chatId} as active`);
        }
    }

    // ✅ UTILITY: Update user balance display
    updateBalance(balance) {
        if (this.elements.userBalance) {
            this.elements.userBalance.textContent = balance.toLocaleString();
        }
        if (this.elements.currentBalance) {
            this.elements.currentBalance.textContent = balance.toLocaleString();
        }
        console.log(`[ChatUI] ✅ Balance updated: ${balance}`);
    }

    // ✅ STATE: Loading states
    showLoadingState(message = 'Loading...') {
        if (this.elements.messagesContainer) {
            this.elements.messagesContainer.innerHTML = `
                <div class="loading-state">
                    <div class="loading-spinner">
                        <div class="spinner"></div>
                    </div>
                    <div class="loading-text">${this.escapeHtml(message)}</div>
                </div>
            `;
        }
        console.log(`[ChatUI] Loading state: ${message}`);
    }

    showEmptyState() {
        if (this.elements.messagesContainer) {
            this.elements.messagesContainer.innerHTML = `
                <div class="empty-state">
                    <div class="empty-icon">🤖</div>
                    <div class="empty-title">Welcome to Medical AI</div>
                    <div class="empty-subtitle">Start a new consultation to get medical guidance</div>
                    <button onclick="window.chatManager.createNewChat()" class="btn-primary">
                        Start New Consultation
                    </button>
                </div>
            `;
        }
        console.log('[ChatUI] Showing empty state');
    }

    showError(message) {
        if (this.elements.messagesContainer) {
            this.elements.messagesContainer.innerHTML = `
                <div class="error-state">
                    <div class="error-icon">⚠️</div>
                    <div class="error-title">Something went wrong</div>
                    <div class="error-message">${this.escapeHtml(message)}</div>
                    <button onclick="location.reload()" class="btn-secondary">
                        Refresh Page
                    </button>
                </div>
            `;
        }
        console.error(`[ChatUI] Error state: ${message}`);
    }

    // ✅ INPUT: Message input management
    enableMessageInput(placeholder = 'Type your medical question...') {
        if (this.elements.messageInput) {
            this.elements.messageInput.disabled = false;
            this.elements.messageInput.placeholder = placeholder;
        }

        const submitBtn = document.getElementById('send-btn');
        if (submitBtn) {
            submitBtn.disabled = false;
        }
    }

    disableMessageInput(placeholder = 'AI is responding...') {
        if (this.elements.messageInput) {
            this.elements.messageInput.disabled = true;
            this.elements.messageInput.placeholder = placeholder;
        }

        const submitBtn = document.getElementById('send-btn');
        if (submitBtn) {
            submitBtn.disabled = true;
        }
    }

    // ✅ UTILITY: Helper functions
    scrollToBottom() {
        if (this.elements.messagesContainer) {
            this.elements.messagesContainer.scrollTop = this.elements.messagesContainer.scrollHeight;
        }
    }

    formatTime(dateString) {
        try {
            const date = new Date(dateString);
            const now = new Date();
            const diffHours = (now - date) / (1000 * 60 * 60);

            if (diffHours < 1) return 'Just now';
            if (diffHours < 24) return `${Math.floor(diffHours)}h ago`;
            if (diffHours < 168) return `${Math.floor(diffHours / 24)}d ago`;

            return date.toLocaleDateString('en-US', {
                month: 'short',
                day: 'numeric',
                hour: '2-digit',
                minute: '2-digit'
            });
        } catch (e) {
            return 'Recently';
        }
    }

    formatMessageContent(content) {
        if (!content) return '';

        let formatted = this.escapeHtml(content);

        // Basic markdown-like formatting if renderer not available
        formatted = formatted.replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>');
        formatted = formatted.replace(/\*(.*?)\*/g, '<em>$1</em>');
        formatted = formatted.replace(/\n/g, '<br>');
        formatted = formatted.replace(/(https?:\/\/[^\s]+)/g, '<a href="$1" target="_blank" rel="noopener">$1</a>');

        return formatted;
    }

    truncateText(text, maxLength) {
        if (!text) return '';
        if (text.length <= maxLength) return this.escapeHtml(text);
        return this.escapeHtml(text.substring(0, maxLength - 3)) + '...';
    }

    escapeHtml(text) {
        if (!text) return '';
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}

// ✅ Initialize and export
const chatUI = new ChatUI();
window.ChatUI = chatUI;

console.log('✅ ChatUI loaded and ready with real-time markdown support');
