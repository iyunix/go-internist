// File: web/static/js/chat/chat.js
// MODIFIED: Replaced the default confirm() dialog with a custom, modern modal.

import { ChatUI } from './chat-ui.js';
import { ChatAPI } from './chat-api.js';
import { ChatStreamRenderer } from './chat-stream.js';
import { Utils } from '../utils.js';

// NEW FUNCTION: Creates and manages a custom confirmation modal.
function showConfirmationModal(message) {
    return new Promise((resolve) => {
        // Create modal overlay
        const overlay = document.createElement('div');
        overlay.className = 'fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 transition-opacity duration-300 opacity-0';
        overlay.id = 'confirmation-modal-overlay';

        // Create modal content
        overlay.innerHTML = `
            <div class="bg-white rounded-lg shadow-xl p-6 w-full max-w-sm transform transition-all duration-300 scale-95 opacity-0">
                <div class="text-center">
                    <div class="mx-auto flex items-center justify-center h-12 w-12 rounded-full bg-red-100">
                        <span class="material-symbols-outlined text-red-600">warning</span>
                    </div>
                    <h3 class="text-lg leading-6 font-medium text-gray-900 mt-4">Confirm Deletion</h3>
                    <div class="mt-2">
                        <p class="text-sm text-gray-500">${message}</p>
                    </div>
                </div>
                <div class="mt-5 sm:mt-6 grid grid-cols-2 gap-3">
                    <button id="modal-cancel-btn" type="button" class="inline-flex justify-center w-full rounded-md border border-gray-300 px-4 py-2 bg-white text-base font-medium text-gray-700 shadow-sm hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 sm:text-sm">
                        Cancel
                    </button>
                    <button id="modal-confirm-btn" type="button" class="inline-flex justify-center w-full rounded-md border border-transparent px-4 py-2 bg-red-600 text-base font-medium text-white shadow-sm hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500 sm:text-sm">
                        Delete
                    </button>
                </div>
            </div>
        `;

        document.body.appendChild(overlay);

        // Get elements
        const modalContent = overlay.querySelector('div > div');
        const confirmBtn = document.getElementById('modal-confirm-btn');
        const cancelBtn = document.getElementById('modal-cancel-btn');

        // Function to close the modal
        const closeModal = (result) => {
            overlay.classList.remove('opacity-100');
            modalContent.classList.remove('scale-100', 'opacity-100');
            setTimeout(() => {
                document.body.removeChild(overlay);
                resolve(result);
            }, 300); // Wait for animation to finish
        };

        // Add event listeners
        confirmBtn.onclick = () => closeModal(true);
        cancelBtn.onclick = () => closeModal(false);
        overlay.onclick = (e) => {
            if (e.target.id === 'confirmation-modal-overlay') {
                closeModal(false);
            }
        };

        // Trigger fade-in animation
        setTimeout(() => {
            overlay.classList.add('opacity-100');
            modalContent.classList.add('scale-100', 'opacity-100');
        }, 10); // Short delay to ensure transition triggers
    });
}


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

        if (!this.elements.chatForm) return;

        this.elements.submitButton = this.elements.chatForm.querySelector("button[type='submit']");
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
            } else {
                this.ui.clearMessages();
            }
            this.isInitialized = true;
        } catch (err) {
            console.error("[ChatApp] Failed to initialize:", err);
        }
    }

    // ... other methods (loadHistory, loadMessages, handleSubmit, etc.) remain the same ...
    async loadHistory() {
        try {
            const chats = await this.api.fetchHistory();
            this.ui.renderHistory(chats);
            this.ui.setActiveChat(this.activeChatId);
        } catch (err) {
            console.error("[ChatApp] Failed to load history:", err);
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
            if (!messages) return;
            this.ui.renderMessages(messages);
            this.activeChatId = chatId;
            this.ui.setActiveChat(chatId);
        } catch (err) {
            console.error("[ChatApp] Failed to load messages:", err);
            this.ui.displayMessage("Failed to load chat messages. Please try again.", "assistant");
        }
    }

    async handleSubmit(e) {
        e.preventDefault();
        const prompt = this.elements.chatInput.value.trim();
        if (!prompt || this.ui.isLoading) return;

        if (prompt.length > 4000) {
            this.ui.displayMessage("Question too long. Please limit to 4000 characters.", "assistant");
            return;
        }

        this.ui.displayMessage(prompt, "user");
        this.ui.clearInput();
        this.ui.toggleLoading(true);

        let chatId = this.activeChatId;
        if (!chatId) {
            try {
                chatId = await this.api.createChat(prompt.substring(0, 50));
                window.history.pushState({ chatId }, "", `/chat?id=${chatId}`);
                this.activeChatId = chatId;
                await this.loadHistory();
            } catch (err) {
                this.ui.displayMessage("Error: Could not create a new chat session.", "assistant");
                this.ui.toggleLoading(false);
                return;
            }
        }
        this.streamResponse(chatId, prompt);
    }

    streamResponse(chatId, prompt) {
        this.ui.displayMessage("", "assistant", { showLoader: true });
        
        const streamRenderer = new ChatStreamRenderer(this.ui);
        const eventSource = this.api.createStream(chatId, prompt);
        let firstChunkReceived = false;

        eventSource.onmessage = (evt) => {
            if (!firstChunkReceived) {
                this.ui.replaceSkeletonWithContent?.();
                firstChunkReceived = true;
            }
            
            try {
                const rawData = evt.data;
                let content = '';
                if (rawData.startsWith('[') && rawData.endsWith(']')) {
                    const chunks = JSON.parse(rawData);
                    for (const chunk of chunks) {
                        if (chunk && chunk.content) {
                            content += chunk.content;
                        }
                    }
                } else if (rawData.startsWith('{') && rawData.endsWith('}')) {
                    const chunk = JSON.parse(rawData);
                    if (chunk && chunk.content) {
                        content = chunk.content;
                    }
                } else {
                    content = rawData;
                }
                if (content) {
                    streamRenderer.appendChunk(content);
                }
            } catch (err) {
                console.warn("[ChatApp] Could not parse stream data as JSON, using raw text.", err);
                streamRenderer.appendChunk(evt.data);
            }
        };
        
        eventSource.addEventListener("done", () => {
            eventSource.close();
            streamRenderer.finalize();
            this.ui.toggleLoading(false);
        });

        eventSource.onerror = (err) => {
            console.error("[ChatApp] Stream error:", err);
            eventSource.close();
            streamRenderer.destroy();
            this.ui.displayMessage("A streaming error occurred. Please try again.", "assistant");
            this.ui.toggleLoading(false);
        };
    }

    handleTextareaInput() {
        const textarea = this.elements.chatInput;
        textarea.style.height = 'auto';
        const maxHeight = 200;
        textarea.style.height = `${Math.min(textarea.scrollHeight, maxHeight)}px`;
    }

    handleTextareaKeydown(e) {
        if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            this.elements.chatForm.requestSubmit();
        }
    }

    bindEvents() {
        this.elements.chatForm.addEventListener("submit", (e) => this.handleSubmit(e));
        this.elements.chatInput.addEventListener('input', () => this.handleTextareaInput());
        this.elements.chatInput.addEventListener('keydown', (e) => this.handleTextareaKeydown(e));

        this.elements.newChatBtn.addEventListener("click", (e) => {
            e.preventDefault();
            if (this.ui.isLoading) return;
            window.history.pushState({}, "", "/chat");
            this.loadMessages(null);
        });

        this.elements.historyList.addEventListener("click", async (e) => {
            if (this.ui.isLoading) return;
            const target = e.target;
            
            const deleteBtn = target.closest(".delete-chat-btn");
            if (deleteBtn) {
                e.preventDefault();
                e.stopPropagation();
                await this.handleDeleteChat(deleteBtn);
                return;
            }

            const link = target.closest("a");
            if (link) {
                e.preventDefault();
                const chatId = link.getAttribute("data-chat-id");
                if (chatId && chatId !== this.activeChatId) {
                    window.history.pushState({ chatId }, "", `/chat?id=${chatId}`);
                    this.loadMessages(chatId);
                }
            }
        });
    }

    // MODIFIED SECTION: This function now uses the custom modal.
    async handleDeleteChat(button) {
        const chatId = button.getAttribute("data-chat-id");
        if (!chatId) return;

        const confirmed = await showConfirmationModal("Are you sure you want to permanently delete this chat?");

        if (confirmed) {
            try {
                await this.api.deleteChat(chatId);
                button.parentElement.remove(); // Remove the entire LI
                if (chatId === this.activeChatId) {
                    this.elements.newChatBtn.click();
                }
            } catch (err) {
                console.error("[ChatApp] Failed to delete chat:", err);
                // We could show another modal for errors here if desired
                alert("Failed to delete chat.");
            }
        }
    }
}

document.addEventListener("DOMContentLoaded", () => {
    const app = new ChatApp();
    if (app.elements.chatForm) {
        app.init();
    }
});

