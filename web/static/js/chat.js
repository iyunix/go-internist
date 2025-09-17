// G:\go_internist\web\static\js\chat.js
const currentUsername = window.INTERNIST_DATA?.currentUsername || "Unknown User";
let activeChatID = window.INTERNIST_DATA?.activeChatID || 0;
if (typeof marked === 'undefined' || typeof DOMPurify === 'undefined') { 
    console.error("CRITICAL ERROR: Libraries not loaded."); 
}

let messagePage = 1;
const messageLimit = 50; // Or whatever page size you want
let allMessagesLoaded = false;

const loadOlderBtn = document.getElementById('load-older-btn');
if (loadOlderBtn && activeChatID) {
  loadOlderBtn.addEventListener('click', () => {
    loadOlderMessages();
  });
}

async function loadOlderMessages() {
  if (allMessagesLoaded) return;

  try {
    // Increment page for older messages (or decrement if you're loading backwards)
    messagePage += 1;
    const response = await fetch(`/api/chats/${activeChatID}/messages?page=${messagePage}&limit=${messageLimit}`);
    if (!response.ok) throw new Error("Failed to load messages.");

    const data = await response.json();
    if (data.messages.length === 0) {
      allMessagesLoaded = true;
      loadOlderBtn.disabled = true;
      loadOlderBtn.textContent = "No more messages.";
      return;
    }

    // Insert at the top of your message list (reverse order if needed)
    data.messages.forEach(msg => {
      const el = document.createElement('div');
      el.innerHTML = `<div class="chat-message">${DOMPurify.sanitize(msg.content)}</div>`;
      messageContainer.prepend(el.firstChild);
    });

    if (!data.has_more) {
      allMessagesLoaded = true;
      loadOlderBtn.disabled = true;
      loadOlderBtn.textContent = "No more messages.";
    }
  } catch (err) {
    console.error("Error loading older messages:", err);
    alert("Failed to load older messages.");
  }
}


const chatForm = document.getElementById('chat-form');
const messageInput = document.getElementById('message-input');
const sendButton = document.getElementById('send-button');
const messageContainer = document.getElementById('message-container').querySelector('.space-y-8');
const newChatBtn = document.getElementById('new-chat-btn');
const welcomeNewChatBtn = document.getElementById('welcome-new-chat-btn');
const mainChatArea = document.querySelector('main.flex-1.flex-col');
const chatListContainer = document.getElementById('chat-list-container');
const deleteModal = document.getElementById('delete-confirm-modal');
const confirmDeleteBtn = document.getElementById('modal-confirm-delete-btn');
const cancelDeleteBtn = document.getElementById('modal-cancel-btn');

function scrollToBottom() { 
    if (mainChatArea) { 
        mainChatArea.scrollTop = mainChatArea.scrollHeight; 
    } 
}

window.onload = scrollToBottom;

if (chatListContainer) {
    chatListContainer.addEventListener('click', (e) => {
        const deleteButton = e.target.closest('.delete-chat-btn');
        if (deleteButton) {
            e.preventDefault(); 
            e.stopPropagation();
            const chatId = deleteButton.dataset.chatId;
            confirmDeleteBtn.dataset.chatId = chatId;
            deleteModal.classList.remove('hidden');
        }
    });
}

if (confirmDeleteBtn) {
    confirmDeleteBtn.addEventListener('click', () => {
        const chatId = confirmDeleteBtn.dataset.chatId;
        deleteModal.classList.add('hidden');
        deleteChat(chatId);
    });
}

if (cancelDeleteBtn) {
    cancelDeleteBtn.addEventListener('click', () => {
        deleteModal.classList.add('hidden');
    });
}

if (newChatBtn) {
    newChatBtn.addEventListener('click', async () => {
        // No more prompt - use temporary title
        const chatTitle = "New Chat";
        
        try {
            const response = await fetch('/api/chats', { 
                method: 'POST', 
                headers: { 'Content-Type': 'application/json' }, 
                body: JSON.stringify({ title: chatTitle.trim() }) 
            });
            if (!response.ok) throw new Error('Failed to create chat');

            const newChat = await response.json();
            activeChatID = newChat.id;
            window.INTERNIST_DATA.activeChatID = newChat.id;

            // Add new chat to the chat list UI
            if (chatListContainer) {
                const newChatHTML = `
                    <div class="group mt-1 flex items-center justify-between gap-3 rounded-md px-3 py-2 bg-gray-100 hover:bg-gray-100" data-chat-item-id="${newChat.id}">
                        <a class="flex items-center gap-3 truncate w-full" href="/chat?id=${newChat.id}">
                            <span class="material-symbols-outlined text-lg text-[#64748b]">chat_bubble</span>
                            <span class="truncate">${newChat.title}</span>
                        </a>
                        <button class="delete-chat-btn flex h-6 w-6 shrink-0 items-center justify-center rounded-md text-gray-500 opacity-0 group-hover:opacity-100 hover:bg-gray-200 hover:text-gray-800" data-chat-id="${newChat.id}" title="Delete chat">
                            <span class="material-symbols-outlined text-base">delete</span>
                        </button>
                    </div>
                `;
                chatListContainer.querySelector('h2')?.insertAdjacentHTML('afterend', newChatHTML);
            }

            // Redirect to the new chat page
            window.location.href = `/chat?id=${newChat.id}`;
        } catch (error) {
            console.error('Error creating new chat:', error);
            alert('Could not create a new chat. Please try again.');
        }
    });
}


if (welcomeNewChatBtn) {
    welcomeNewChatBtn.addEventListener('click', () => {
        newChatBtn?.click();
    });
}


if (chatForm) {
    chatForm.addEventListener('submit', (e) => {
        e.preventDefault();
        const prompt = messageInput.value.trim();
        if (prompt) { 
            sendMessage(prompt); 
        }
    });
}

async function createNewChat(title) {
    try {
        const response = await fetch('/api/chats', { 
            method: 'POST', 
            headers: { 'Content-Type': 'application/json' }, 
            body: JSON.stringify({ title: title }) 
        });
        if (!response.ok) throw new Error('Failed to create chat');
        const newChat = await response.json();
        window.location.href = `/chat?id=${newChat.id}`;
    } catch (error) {
        console.error('Error creating new chat:', error);
        alert('Could not create a new chat.');
    }
}

async function deleteChat(chatId) {
    try {
        const response = await fetch(`/api/chats/${chatId}`, { method: 'DELETE' });
        if (!response.ok) throw new Error('Failed to delete chat on server.');
        document.querySelector(`div[data-chat-item-id='${chatId}']`).remove();
        if (activeChatID && parseInt(chatId) === activeChatID) {
            window.location.href = '/chat';
        }
    } catch (error) {
        console.error('Error deleting chat:', error);
        alert('Could not delete the chat. Please try again.');
    }
}

async function sendMessage(prompt) {
    const welcomeMsg = document.getElementById('welcome-message');
    if (welcomeMsg) { welcomeMsg.remove(); }
    let currentChatId = activeChatID;
    
    if (currentChatId === 0) {
        try {
            const title = prompt.length > 50 ? prompt.substring(0, 50) + '...' : prompt;
            const resp = await fetch('/api/chats', { 
                method: 'POST', 
                headers: { 'Content-Type': 'application/json' }, 
                body: JSON.stringify({ title }) 
            });
            if (!resp.ok) throw new Error('Server failed to create chat.');
            const newChat = await resp.json();
            currentChatId = newChat.id; 
            activeChatID = newChat.id;
            history.pushState({}, '', `/chat?id=${currentChatId}`);
            const newChatHTML = `<div class="group mt-1 flex items-center justify-between gap-3 rounded-md px-3 py-2 bg-gray-100 hover:bg-gray-100" data-chat-item-id="${newChat.id}"><a class="flex items-center gap-3 truncate w-full" href="/chat?id=${newChat.id}"><span class="material-symbols-outlined text-lg text-[#64748b]">chat_bubble</span><span class="truncate">${newChat.title}</span></a><button class="delete-chat-btn flex h-6 w-6 shrink-0 items-center justify-center rounded-md text-gray-500 opacity-0 group-hover:opacity-100 hover:bg-gray-200 hover:text-gray-800" data-chat-id="${newChat.id}" title="Delete chat"><span class="material-symbols-outlined text-base">delete</span></button></div>`;
            chatListContainer.querySelector('h2').insertAdjacentHTML('afterend', newChatHTML);
        } catch (err) { 
            console.error('Error creating chat:', err); 
            alert('Could not start a new chat session.'); 
            return; 
        }
    }

    messageInput.value = ''; 
    messageInput.disabled = true; 
    sendButton.disabled = true;
    
    try {
        const response = await fetch(`/api/chats/${currentChatId}/messages`, { 
            method: 'POST', 
            headers: { 'Content-Type': 'application/json' }, 
            body: JSON.stringify({ content: prompt, messageType: 'user' }) 
        });
        if (!response.ok) throw new Error("Failed to save user message.");
    } catch (error) { 
        console.error("Error saving user message:", error); 
        alert("Could not send message."); 
        messageInput.disabled = false; 
        sendButton.disabled = false; 
        return; 
    }

    const safePrompt = prompt.replace(/</g, '&lt;').replace(/>/g, '&gt;');
    messageContainer.insertAdjacentHTML('beforeend', `<div class="flex items-start justify-end gap-3"><div class="flex-1"><p class="text-right text-sm font-medium text-[#64748b]">${currentUsername}</p><div class="mt-1 rounded-lg rounded-tr-none bg-[#13a4ec] p-3 text-base text-white">${safePrompt}</div></div></div>`);
    scrollToBottom();

    const assistantMessageId = 'assistant-response-' + Date.now();
    const assistantMessageHTML = `<div class="flex items-start gap-3 group"><div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-gray-200"><span class="material-symbols-outlined text-[#64748b]">smart_toy</span></div><div class="flex-1"><p class="text-sm font-medium text-[#64748b]">Internist AI</p><div id="${assistantMessageId}" class="message-content mt-1 text-base text-[#1e293b]"><div class="status-container space-y-2 rounded-lg bg-gray-100 p-3"></div></div></div></div>`;
    messageContainer.insertAdjacentHTML('beforeend', assistantMessageHTML);
    scrollToBottom();

    const assistantResponseElement = document.getElementById(assistantMessageId);
    const statusContainer = assistantResponseElement.querySelector('.status-container');
    let fullResponseText = ''; 
    let statuses = {};
    
    function updateStatusUI() {
        const steps = [
            { id: 'understanding', text: 'Understanding question...' }, 
            { id: 'searching', text: 'Searching UpToDate...' }, 
            { id: 'thinking', text: 'Generating response...' }
        ];
        let html = '';
        for (const step of steps) { 
            if (statuses[step.id]) { 
                const icon = statuses[step.id] === 'completed' ? '<span class="material-symbols-outlined text-green-500">check_circle</span>' : '<div class="spinner"></div>'; 
                html += `<div class="status-item ${statuses[step.id]}"><div class="status-icon">${icon}</div><span>${step.text}</span></div>`; 
            } 
        }
        statusContainer.innerHTML = html;
    }
    
    updateStatusUI();

    const eventSource = new EventSource(`/api/chats/${currentChatId}/stream?q=${encodeURIComponent(prompt)}`);
    
    eventSource.addEventListener('status', (e) => { 
        const data = JSON.parse(e.data); 
        for (const key in statuses) { 
            statuses[key] = 'completed'; 
        } 
        statuses[data.status] = 'in-progress'; 
        updateStatusUI(); 
    });

    let isFirstToken = true;
    eventSource.onmessage = (e) => {
        if (isFirstToken) { 
            assistantResponseElement.classList.add('prose', 'prose-lg', 'rounded-lg', 'bg-gray-100', 'p-3'); 
            assistantResponseElement.innerHTML = ''; 
            isFirstToken = false; 
        }
        const data = JSON.parse(e.data); 
        fullResponseText += data.content;
        assistantResponseElement.innerHTML = DOMPurify.sanitize(marked.parse(fullResponseText)); 
        scrollToBottom();
    };

    eventSource.addEventListener('done', () => {
        eventSource.close();
        const exportButtonHTML = `<button onclick="exportMessageAsPDF(this.parentElement.querySelector('.message-content').innerHTML)" class="absolute top-1 right-1 opacity-0 group-hover:opacity-100 transition-opacity bg-white p-1 rounded-full shadow-sm hover:bg-gray-200"><span class="material-symbols-outlined text-base text-gray-600">picture_as_pdf</span></button>`;
        assistantResponseElement.parentElement.classList.add('relative');
        assistantResponseElement.insertAdjacentHTML('afterend', exportButtonHTML);
        messageInput.disabled = false; 
        sendButton.disabled = false; 
        messageInput.focus();
    });

    eventSource.onerror = () => { 
        assistantResponseElement.innerHTML = '<p class="text-red-500">Sorry, an error occurred.</p>'; 
        eventSource.close(); 
        messageInput.disabled = false; 
        sendButton.disabled = false; 
        messageInput.focus(); 
    };
}

async function exportMessageAsPDF(messageHTML) {
    try {
        const response = await fetch('/static/pdf_template.html');
        if (!response.ok) throw new Error('PDF template not found.');
        const templateHTML = await response.text();
        const printWindow = window.open('', '_blank', 'height=800,width=800');
        printWindow.document.write(templateHTML);
        printWindow.document.close();
        printWindow.onload = function() {
            const dateElement = printWindow.document.getElementById('generation-date');
            const contentElement = printWindow.document.getElementById('ai-content-container');
            if (dateElement) dateElement.textContent = new Date().toLocaleString();
            if (contentElement) contentElement.innerHTML = DOMPurify.sanitize(messageHTML);
            setTimeout(() => { printWindow.print(); printWindow.close(); }, 250);
        };
    } catch (error) { 
        console.error('Error exporting to PDF:', error); 
        alert('Could not generate PDF.'); 
    }
}


// Sidebar toggle logic for mobile
document.addEventListener('DOMContentLoaded', function() {
  const sidebar = document.getElementById('sidebar');
  const sidebarToggle = document.getElementById('sidebar-toggle');
  const sidebarOverlay = document.getElementById('sidebar-overlay');

  sidebarToggle?.addEventListener('click', () => {
    sidebar.classList.toggle('-translate-x-full');
    sidebarOverlay.classList.toggle('hidden');
  });

  sidebarOverlay?.addEventListener('click', () => {
    sidebar.classList.add('-translate-x-full');
    sidebarOverlay.classList.add('hidden');
  });
});
