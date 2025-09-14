// File: web/static/js/chat/chat-ui.js
// FINAL FIX: Correctly reads "messageType" from the API response.

export class ChatUI {
    constructor(elements) {
        this.elements = elements;
        this.lastAssistantMessageBubble = null;
        this.currentSkeletonLoader = null;
        this.isLoading = false;
    }

    // === CORE MESSAGE DISPLAY METHODS ===
    displayMessage(content, role, options = {}) {
        const li = document.createElement("li");
        li.className = `flex items-start gap-3 w-full ${role === 'user' ? 'justify-end' : ''}`;
        const messageContainer = document.createElement("div");
        messageContainer.className = `flex gap-3 max-w-4xl ${role === 'user' ? 'flex-row-reverse' : 'flex-row'}`;
        const avatarContainer = document.createElement("div");
        avatarContainer.className = "flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-gray-200";
        const avatarIcon = document.createElement("span");
        avatarIcon.className = "material-symbols-outlined text-[var(--text-secondary)]";
        avatarIcon.textContent = role === 'user' ? 'person' : 'smart_toy';
        avatarContainer.appendChild(avatarIcon);
        const contentWrapper = document.createElement("div");
        contentWrapper.className = `flex-1 ${role === 'user' ? 'text-right' : ''}`;
        const authorName = document.createElement("p");
        authorName.className = "text-sm font-medium text-[var(--text-secondary)]";
        authorName.textContent = role === 'user' ? 'You' : 'Internist AI';
        const messageBubble = document.createElement("div");
        messageBubble.className = `mt-1 rounded-lg p-3 text-base text-left ${
            role === 'user'
                ? 'rounded-tr-none bg-[var(--primary-color)] text-white'
                : 'rounded-tl-none bg-gray-100 text-[var(--text-primary)]'
        }`;
        messageBubble.className = `message-bubble mt-1 rounded-lg p-3 text-base text-left ${
        role === 'user'
            ? 'rounded-tr-none bg-[var(--primary-color)] text-white'
            : 'rounded-tl-none bg-gray-100 text-[var(--text-primary)]'
        }`;

        
        if (role === "assistant") {
            this.lastAssistantMessageBubble = messageBubble;
            if (options.showLoader === true) {
                messageBubble.innerHTML = this.createSkeletonLoader();
                this.currentSkeletonLoader = messageBubble.querySelector('.skeleton-loader');
            } else {
                this.renderMessageContent(messageBubble, content || "");
            }
        } else {
            messageBubble.textContent = content || "";
        }

        contentWrapper.appendChild(authorName);
        contentWrapper.appendChild(messageBubble);
        messageContainer.appendChild(avatarContainer);
        messageContainer.appendChild(contentWrapper);
        li.appendChild(messageContainer);
        this.elements.chatMessages.appendChild(li);
        this.scrollToBottom();
        return li;
    }
    
    renderMessageContent(container, content) {
        if (!container) return;
        if (!content || content.trim() === "") {
            container.innerHTML = '<em class="text-slate-500">No content</em>';
            return;
        }
        if (window.MarkdownRenderer) {
            window.MarkdownRenderer.render(container, content);
        } else {
            container.textContent = content;
        }
    }

    renderMessages(messages) {
        this.removeAllSkeletons();
        this.clearMessages();
        if (!Array.isArray(messages) || messages.length === 0) return;

        messages.forEach(msg => {
            const content = msg.content || "";
            // --- THIS IS THE FINAL FIX ---
            // The Go backend now sends "messageType" (camelCase). We look for that exact field.
            const role = (msg.messageType || "user").toLowerCase();
            
            this.displayMessage(content, role, { showLoader: false });
        });
        setTimeout(() => this.removeAllSkeletons(), 100);
    }
    
    // ... (The rest of the file remains the same) ...

    // === SKELETON LOADER METHODS ===
    createSkeletonLoader() {
        return `
            <div class="skeleton-loader space-y-3">
                <div class="flex items-center gap-2">
                    <div class="skeleton-status-icon h-4 w-4 rounded-full bg-gray-300 animate-pulse"></div>
                    <span class="skeleton-status-text text-sm text-gray-500">Thinking...</span>
                </div>
                <div class="space-y-2">
                    <div class="h-4 w-5/6 rounded bg-gray-300 animate-pulse"></div>
                    <div class="h-4 w-full rounded bg-gray-300 animate-pulse"></div>
                    <div class="h-4 w-3/4 rounded bg-gray-300 animate-pulse"></div>
                </div>
            </div>
        `;
    }
    updateSkeletonStatus(status, message) {
        if (!this.currentSkeletonLoader) return;
        const text = this.currentSkeletonLoader.querySelector('.skeleton-status-text');
        if (text) text.textContent = message;
    }
    replaceSkeletonWithContent() {
        if (this.lastAssistantMessageBubble && this.currentSkeletonLoader) {
            this.lastAssistantMessageBubble.innerHTML = '';
            this.currentSkeletonLoader = null;
        }
    }
    removeAllSkeletons() {
        document.querySelectorAll('.skeleton-loader').forEach(el => el.remove());
        this.currentSkeletonLoader = null;
    }

    // === UI STATE MANAGEMENT ===
    toggleLoading(isLoading) {
        this.isLoading = isLoading;
        if (this.elements.chatInput) this.elements.chatInput.disabled = isLoading;
        if (this.elements.submitButton) this.elements.submitButton.disabled = isLoading;
    }
    clearMessages() {
        this.elements.chatMessages.innerHTML = "";
        this.lastAssistantMessageBubble = null;
        this.currentSkeletonLoader = null;
    }
    clearInput() {
        if (this.elements.chatInput) {
            this.elements.chatInput.value = "";
            this.elements.chatInput.focus();
            this.resetTextareaHeight();
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
        document.querySelectorAll("#historyList a").forEach(el => {
            el.classList.remove("bg-gray-100", "font-semibold");
            el.classList.add("text-[var(--text-primary)]", "hover:bg-gray-100");
        });
        if (chatId) {
            const activeElement = document.querySelector(`#historyList a[data-chat-id='${chatId}']`);
            if (activeElement) {
                activeElement.classList.add("bg-gray-100", "font-semibold");
                activeElement.classList.remove("hover:bg-gray-100");
            }
        }
    }
    renderHistory(chats) {
        this.elements.historyList.innerHTML = "";
        if (!Array.isArray(chats)) return;
        chats.forEach(chat => {
            const id = String(chat.ID ?? "");
            const title = this.escapeHtml(String(chat.Title ?? "Untitled"));
            if (!id) return;
            const li = document.createElement("li");
            li.className = "flex items-center justify-between group";
            const anchor = document.createElement("a");
            anchor.href = `/chat?id=${id}`;
            anchor.setAttribute("data-chat-id", id);
            anchor.className = "flex flex-grow items-center gap-3 rounded-md px-3 py-2 text-sm font-medium text-[var(--text-primary)] hover:bg-gray-100 truncate";
            const icon = document.createElement("span");
            icon.className = "material-symbols-outlined text-lg text-[var(--text-secondary)]";
            icon.textContent = "chat_bubble";
            const text = document.createElement("span");
            text.className = "truncate";
            text.textContent = title;
            anchor.appendChild(icon);
            anchor.appendChild(text);
            const deleteBtn = document.createElement("button");
            deleteBtn.className = "delete-chat-btn p-1 rounded-md text-gray-400 hover:bg-gray-200 hover:text-gray-600 opacity-0 group-hover:opacity-100 transition-opacity";
            deleteBtn.innerHTML = `<span class="material-symbols-outlined text-lg">delete</span>`;
            deleteBtn.setAttribute("data-chat-id", id);
            deleteBtn.setAttribute("aria-label", "Delete chat");
            li.appendChild(anchor);
            li.appendChild(deleteBtn);
            this.elements.historyList.appendChild(li);
        });
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
}