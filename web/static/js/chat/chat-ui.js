// File: web/static/js/chat/chat-ui.js
// Complete rewrite with skeleton loader fixes and enhanced readability

import { Utils } from '../utils.js';

export class ChatUI {
    constructor(elements) {
        this.elements = elements;
        this.lastAssistantMessageBubble = null;
        this.currentSkeletonLoader = null;
        this.currentSources = [];
        this.isLoading = false;
    }

    // === CORE MESSAGE DISPLAY METHODS ===

    displayMessage(content, role, options = {}) {
        console.log(`[ChatUI] Displaying message - Role: ${role}, Content length: ${content?.length || 0}, ShowLoader: ${options.showLoader}`);
        
        const li = document.createElement("li");
        li.className = `message-item ${role}`;

        // Create avatar
        const avatar = document.createElement("img");
        avatar.className = "avatar";
        avatar.src = role === "user" ? "/static/img/user-avatar.png" : "/static/img/ai-avatar.png";
        avatar.alt = `${role} avatar`;

        // Create message bubble
        const messageBubble = document.createElement("div");
        messageBubble.className = `message-bubble ${role}`;

        if (role === "assistant") {
            this.lastAssistantMessageBubble = messageBubble;

            // CRITICAL: Only show loader for NEW streaming messages
            if (options.showLoader === true) {
                console.log("[ChatUI] Creating skeleton loader for streaming message");
                messageBubble.innerHTML = this.createSkeletonLoader();
                this.currentSkeletonLoader = messageBubble.querySelector('.skeleton-loader');
            } else {
                // CRITICAL: Render historical message content immediately
                console.log("[ChatUI] Rendering historical message content");
                this.renderMessageContent(messageBubble, content || "");
            }
        } else {
            // User message - always plain text
            messageBubble.textContent = content || "";
        }

        li.appendChild(avatar);
        li.appendChild(messageBubble);
        this.elements.chatMessages.appendChild(li);
        this.scrollToBottom();

        console.log(`[ChatUI] Message displayed successfully - Role: ${role}`);
        return li;
    }

    renderMessageContent(container, content) {
        if (!container) {
            console.error("[ChatUI] No container provided for message content");
            return;
        }

        if (!content || content.trim() === "") {
            console.warn("[ChatUI] Empty content provided");
            container.innerHTML = '<em style="color: #6b7280;">No content</em>';
            return;
        }

        try {
            // Use markdown renderer if available
            if (window.MarkdownRenderer && typeof window.MarkdownRenderer.render === 'function') {
                console.log("[ChatUI] Using MarkdownRenderer");
                window.MarkdownRenderer.render(container, content);
            } else {
                console.warn("[ChatUI] MarkdownRenderer not available, using plain text");
                container.innerHTML = content.replace(/\n/g, '<br>');
            }
        } catch (error) {
            console.error("[ChatUI] Error rendering message content:", error);
            container.textContent = content; // Fallback to plain text
        }
    }

    renderMessages(messages) {
        console.log(`[ChatUI] Rendering ${messages?.length || 0} messages`);
        
        // CRITICAL: Clear everything first
        this.removeAllSkeletons();
        this.clearMessages();
        
        if (!Array.isArray(messages)) {
            console.warn("[ChatUI] Messages is not an array:", typeof messages);
            return;
        }

        if (messages.length === 0) {
            console.log("[ChatUI] No messages to render");
            return;
        }

        // Render each message
        messages.forEach((msg, index) => {
            const content = msg.Content || msg.content || "";
            const role = msg.Role || msg.role || "assistant";
            
            console.log(`[ChatUI] Message ${index + 1}: ${role} - ${content.substring(0, 50)}...`);
            
            // CRITICAL: Historical messages should NEVER show skeleton loader
            this.displayMessage(content, role, { showLoader: false });
        });

        // CRITICAL: Final cleanup to ensure no skeletons remain
        setTimeout(() => this.removeAllSkeletons(), 100);
        console.log("[ChatUI] All messages rendered successfully");
    }

    // === SKELETON LOADER METHODS ===

    createSkeletonLoader() {
        return `
            <div class="skeleton-loader">
                <div class="skeleton-status">
                    <div class="skeleton-status-icon searching"></div>
                    <span class="skeleton-status-text">Searching knowledge base...</span>
                </div>
                <div class="skeleton-lines">
                    <div class="skeleton-line skeleton-line-long"></div>
                    <div class="skeleton-line skeleton-line-medium"></div>
                    <div class="skeleton-line skeleton-line-short"></div>
                </div>
            </div>
        `;
    }

    updateSkeletonStatus(status, message) {
        if (!this.currentSkeletonLoader) {
            console.warn("[ChatUI] No current skeleton loader to update");
            return;
        }
        
        const icon = this.currentSkeletonLoader.querySelector('.skeleton-status-icon');
        const text = this.currentSkeletonLoader.querySelector('.skeleton-status-text');
        
        if (icon) {
            icon.className = `skeleton-status-icon ${status}`;
        }
        if (text) {
            text.textContent = message;
        }
        
        console.log(`[ChatUI] Skeleton status updated: ${status} - ${message}`);
    }

    replaceSkeletonWithContent() {
        if (this.lastAssistantMessageBubble && this.currentSkeletonLoader) {
            console.log("[ChatUI] Replacing skeleton with streaming content");
            this.lastAssistantMessageBubble.innerHTML = '';
            this.currentSkeletonLoader = null;
        }
    }

    removeAllSkeletons() {
        console.log("[ChatUI] Removing all skeleton loaders");
        
        // Remove all skeleton elements
        const skeletonSelectors = [
            '.skeleton-loader',
            '.skeleton-container', 
            '.skeleton-message',
            '[class*="skeleton"]'
        ];
        
        skeletonSelectors.forEach(selector => {
            const elements = document.querySelectorAll(selector);
            elements.forEach(element => {
                // Only remove if it's actually a skeleton, not a message with content
                if (element.classList.contains('skeleton-loader') || 
                    element.classList.contains('skeleton-container') ||
                    element.classList.contains('skeleton-message')) {
                    element.remove();
                }
            });
        });
        
        // Reset skeleton state
        this.currentSkeletonLoader = null;
        console.log("[ChatUI] All skeletons removed");
    }

    // === UI STATE MANAGEMENT ===

    toggleLoading(isLoading) {
        this.isLoading = isLoading;
        
        if (this.elements.chatInput) {
            this.elements.chatInput.disabled = isLoading;
        }
        if (this.elements.submitButton) {
            this.elements.submitButton.disabled = isLoading;
        }
        
        console.log(`[ChatUI] Loading state: ${isLoading}`);
    }

    clearMessages() {
        console.log("[ChatUI] Clearing all messages");
        this.elements.chatMessages.innerHTML = "";
        this.lastAssistantMessageBubble = null;
        this.currentSkeletonLoader = null;
    }

    clearInput() {
        if (this.elements.chatInput) {
            this.elements.chatInput.value = "";
            this.elements.chatInput.focus();
        }
    }

    resetTextareaHeight() {
        if (this.elements.chatInput) {
            this.elements.chatInput.style.height = 'auto';
        }
    }

    scrollToBottom() {
        if (this.elements.chatMessages) {
            this.elements.chatMessages.scrollTop = this.elements.chatMessages.scrollHeight;
        }
    }

    // === HISTORY AND NAVIGATION ===

    setActiveChat(chatId) {
        // Remove active class from all history items
        document.querySelectorAll(".history-item.active").forEach(el => {
            el.classList.remove("active");
        });
        
        // Add active class to current chat
        if (chatId) {
            const activeElement = document.querySelector(`.history-item[data-chat-id='${chatId}']`);
            if (activeElement) {
                activeElement.classList.add("active");
                console.log(`[ChatUI] Set active chat: ${chatId}`);
            }
        }
    }

    renderHistory(chats) {
        console.log(`[ChatUI] Rendering ${chats?.length || 0} chat history items`);
        
        this.elements.historyList.innerHTML = "";
        
        if (!Array.isArray(chats)) {
            console.warn("[ChatUI] Chat history is not an array");
            return;
        }

        chats.forEach(chat => {
            const id = String(chat.ID ?? "");
            const title = this.escapeHtml(String(chat.Title ?? "Untitled"));
            
            if (!id) {
                console.warn("[ChatUI] Chat missing ID:", chat);
                return;
            }

            const li = document.createElement("li");
            li.className = "history-item";
            li.setAttribute("data-chat-id", id);

            const anchor = document.createElement("a");
            anchor.href = `/chat?id=${id}`;
            anchor.className = "history-link";
            anchor.textContent = title;

            li.appendChild(anchor);
            this.elements.historyList.appendChild(li);
        });

        console.log(`[ChatUI] Rendered ${chats.length} history items`);
    }

    // === QUICK ACTIONS ===

    renderQuickActions(actions) {
        console.log(`[ChatUI] Rendering ${actions?.length || 0} quick actions`);
        
        this.clearQuickActions();
        
        if (!Array.isArray(actions)) {
            return;
        }

        // Predefined answers for Internist medical assistant
        const predefinedAnswers = {
            'What is Go?': 'Go (Golang) is a programming language, but as a medical assistant, I focus on health-related topics. Please ask about symptoms, medications, or medical advice.',
            'Explain project structure': 'This project is Internist, a medical assistant. The structure includes modules for chat, authentication, and AI-powered medical support.',
            'Write a simple REST API': 'As a medical assistant, I can help with health questions, medication info, and symptom analysis. For technical help, please consult a developer resource.'
        };
        actions.forEach(actionText => {
            const chip = document.createElement('button');
            chip.className = 'quick-action-chip';
            chip.textContent = actionText;
            chip.setAttribute('type', 'button');
            chip.addEventListener('click', () => {
                // Show predefined answer only, do not call LLM or backend
                this.displayMessage(predefinedAnswers[actionText] || 'No predefined answer available.', 'assistant');
            });
            this.elements.quickActionsContainer.appendChild(chip);
        });
    }

    clearQuickActions() {
        if (this.elements.quickActionsContainer) {
            this.elements.quickActionsContainer.innerHTML = '';
        }
    }

    // === SOURCES AND FOOTNOTES ===

    setSources(sources) {
        this.currentSources = sources || [];
        console.log(`[ChatUI] Sources set: ${this.currentSources.length}`);
    }

    getSources() {
        return this.currentSources;
    }

    createFootnote(sources) {
        if (!sources || sources.length === 0) {
            return '';
        }
        
        const sourceItems = sources.map(source => 
            `<li>${this.escapeHtml(source)}</li>`
        ).join('');
        
        return `
            <div class="message-footnote">
                <h6>Sources</h6>
                <ul>${sourceItems}</ul>
            </div>
        `;
    }

    addFootnote(sources) {
        if (!this.lastAssistantMessageBubble || !sources || sources.length === 0) {
            return;
        }

        const messageItem = this.lastAssistantMessageBubble.parentNode;
        if (!messageItem) return;

        // Remove existing footnote
        const existingFootnote = messageItem.querySelector('.message-footnote');
        if (existingFootnote) {
            existingFootnote.remove();
        }

        // Add new footnote
        messageItem.insertAdjacentHTML('beforeend', this.createFootnote(sources));
        this.scrollToBottom();
        
        console.log(`[ChatUI] Added footnote with ${sources.length} sources`);
    }

    // === UTILITY METHODS ===

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text || "";
        return div.innerHTML;
    }

    getLastAssistantMessageContainer() {
        return this.lastAssistantMessageBubble;
    }

    // === DEBUG AND MONITORING ===

    debugState() {
        console.log("[ChatUI] Debug State:", {
            hasElements: !!this.elements,
            messagesContainer: !!this.elements?.chatMessages,
            lastMessageBubble: !!this.lastAssistantMessageBubble,
            currentSkeleton: !!this.currentSkeletonLoader,
            sourcesCount: this.currentSources.length,
            isLoading: this.isLoading
        });
    }
}
