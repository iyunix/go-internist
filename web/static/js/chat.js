// G:\go_internist\web\static\js\chat.js (Correct and Final Version)

// ===== Global & Setup =====
const currentUsername = window.INTERNIST_DATA?.currentUsername || "Unknown User";
let activeChatID = window.INTERNIST_DATA?.activeChatID || 0;
let chats = window.INTERNIST_DATA?.chats || [];
if (typeof marked === "undefined" || typeof DOMPurify === "undefined") {
  console.error("CRITICAL ERROR: Libraries not loaded.");
}

// ===== RTL Detection =====
function isRTL(text) {
  const rtlPattern = /[\u0600-\u06FF\u0750-\u077F\u08A0-\u08FF\u0590-\u05FF]/;
  return rtlPattern.test(text);
}

// ===== DOM References =====
const messageContainer = document.getElementById("message-container");
const chatForm = document.getElementById("chat-form");
const messageInput = document.getElementById("message-input");
const sendButton = document.getElementById("send-button");
const newChatBtn = document.getElementById("new-chat-btn");
const chatListContainer = document.getElementById("chat-list-container");
const deleteModal = document.getElementById("delete-confirm-modal");
const confirmDeleteBtn = document.getElementById("modal-confirm-delete-btn");
const cancelDeleteBtn = document.getElementById("modal-cancel-btn");
const sidebar = document.getElementById("sidebar");
const sidebarToggle = document.getElementById("sidebar-toggle");

// ===== Input RTL Direction =====
messageInput?.addEventListener("input", () => {
  messageInput.dir = isRTL(messageInput.value) ? "rtl" : "ltr";
});

// ===== Helpers =====
function enableInput(enable = true) {
  if (!messageInput || !sendButton) return;
  messageInput.disabled = !enable;
  sendButton.disabled = !enable;
  if (enable) messageInput.focus();
}
function sanitizeHTML(html) {
  return DOMPurify.sanitize(marked.parse(html || ""));
}

// ===== Sidebar Collapse/Expand Logic =====
if (sidebar && sidebarToggle) {
  sidebarToggle.addEventListener("click", () => sidebar.classList.toggle("collapsed"));
  sidebar.addEventListener("mouseenter", () => {
    if (window.innerWidth >= 769 && sidebar.classList.contains("collapsed")) {
      sidebar.classList.add("hover-expanded");
    }
  });
  sidebar.addEventListener("mouseleave", () => {
    if (window.innerWidth >= 769) sidebar.classList.remove("hover-expanded");
  });
}

// ===== Sidebar Chat Management =====
function insertOrUpdateChatInSidebar(chat) {
  if (!chatListContainer) return;
  const existing = chatListContainer.querySelector(`[data-chat-item-id='${chat.id}']`);
  const chatHTML = `
    <div data-chat-item-id="${chat.id}" class="group mt-1 flex items-center justify-between gap-3 rounded-md px-3 py-2 hover:bg-gray-100 ${chat.id === activeChatID ? "bg-gray-100" : ""}">
      <a class="flex-1 truncate" href="/chat?id=${chat.id}">${chat.title}</a>
      <button data-chat-id="${chat.id}" class="delete-chat-btn flex h-6 w-6 shrink-0 items-center justify-center rounded-md text-gray-500 opacity-0 group-hover:opacity-100 hover:bg-gray-200 hover:text-gray-800" title="Delete chat">
        <span class="material-symbols-outlined text-base">delete</span>
      </button>
    </div>`;
  if (existing) existing.outerHTML = chatHTML;
  else chatListContainer.insertAdjacentHTML("afterbegin", chatHTML);
  updateSidebarActiveHighlight();
}
function updateSidebarActiveHighlight() {
  if (!chatListContainer) return;
  chatListContainer.querySelectorAll("[data-chat-item-id]").forEach(el => {
    const chatId = parseInt(el.dataset.chatItemId, 10);
    el.classList.toggle("bg-gray-100", chatId === activeChatID);
  });
}
if (chatListContainer && chats.length > 0) chats.forEach(insertOrUpdateChatInSidebar);
if (chatListContainer) {
  chatListContainer.addEventListener("click", e => {
    const deleteButton = e.target.closest(".delete-chat-btn");
    if (deleteButton) {
      e.preventDefault();
      e.stopPropagation();
      confirmDeleteBtn.dataset.chatId = deleteButton.dataset.chatId;
      deleteModal.classList.remove("hidden");
    }
  });
}
confirmDeleteBtn?.addEventListener("click", () => {
  deleteModal.classList.add("hidden");
  deleteChat(confirmDeleteBtn.dataset.chatId);
});
cancelDeleteBtn?.addEventListener("click", () => {
  deleteModal.classList.add("hidden");
});
async function deleteChat(chatId) {
  try {
    const resp = await fetch(`/api/chats/${chatId}`, { method: "DELETE" });
    if (!resp.ok) throw new Error("Delete failed");
    document.querySelector(`div[data-chat-item-id='${chatId}']`)?.remove();
    if (parseInt(chatId) === activeChatID) {
      window.location.href = "/chat"; // Redirect to base chat page after deleting the active one.
    }
  } catch (err) {
    console.error(err);
    alert("Could not delete the chat. Please try again.");
  }
}
newChatBtn?.addEventListener("click", () => {
    window.location.href = "/chat"; // "New Chat" button simply navigates to the base page.
});

// ===== START: Rendering Logic =====
function createMessageHTML(msg) {
  if (msg.messageType === "assistant") {
    return `
      <div class="flex items-start gap-3 group" id="message-${msg.id}">
        <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-gray-200">
          <span class="material-symbols-outlined text-[#64748b]">smart_toy</span>
        </div>
        <div class="flex-1">
          <p class="text-sm font-medium text-[#64748b]">Internist AI</p>
          <div class="relative mt-1">
            <div class="message-content rounded-lg rounded-tl-none bg-gray-100 p-3 text-base text-[#1e293b]">
              <div class="prose prose-lg">${sanitizeHTML(msg.content)}</div>
            </div>
            <button onclick="exportMessageAsPDF(this.parentElement.querySelector('.message-content').innerHTML)"
                    class="absolute top-1 right-1 opacity-0 group-hover:opacity-100 transition-opacity bg-white p-1 rounded-full shadow-sm hover:bg-gray-200"
                    title="Export as PDF">
              <span class="material-symbols-outlined text-base text-gray-600">picture_as_pdf</span>
            </button>
          </div>
        </div>
      </div>`;
  } else { // User message
    return `
      <div class="flex items-start justify-end gap-3" id="message-${msg.id}">
        <div class="flex flex-col items-end">
          <p class="text-right text-sm font-medium text-[#64748b]">${currentUsername}</p>
          <div class="message-content mt-1 rounded-lg rounded-tr-none bg-[#13a4ec] p-3 text-base text-white" dir="${isRTL(msg.content) ? "rtl" : "ltr"}">
            ${msg.content}
          </div>
        </div>
      </div>`;
  }
}

function appendMessageToDOM(msg) {
    if (!messageContainer) return;
    const html = createMessageHTML(msg);
    const container = messageContainer.querySelector('.space-y-8');
    if (container) {
        container.insertAdjacentHTML('beforeend', html);
        messageContainer.scrollTop = messageContainer.scrollHeight;
    }
}

async function loadInitialMessages() {
    if (!activeChatID || !messageContainer) return;
    try {
        const resp = await fetch(`/api/chats/${activeChatID}/messages?page=1&limit=50`);
        if (!resp.ok) throw new Error("Failed to load messages");
        const data = await resp.json();
        const messages = data.messages || [];
        const container = messageContainer.querySelector('.space-y-8');
        if (container) {
            container.innerHTML = '';
            messages.forEach(msg => {
                container.insertAdjacentHTML('beforeend', createMessageHTML(msg));
            });
            messageContainer.scrollTop = messageContainer.scrollHeight;
        }
    } catch (err) {
        console.error(err);
    }
}
// ===== END: Rendering Logic =====

// ===== Send & Stream Messages =====
chatForm?.addEventListener("submit", e => {
  e.preventDefault();
  const prompt = messageInput.value.trim();
  if (prompt) sendMessage(prompt);
});

let currentEventSource = null;

async function sendMessage(prompt) {
  if (!messageContainer) return;
  document.getElementById("welcome-message")?.remove();
  let chatId = activeChatID;
  
  // If this is the very first message in a new session, create the chat first.
  if (chatId === 0) {
    chatId = await startNewChat(prompt);
    if (!chatId) { // If chat creation failed, stop.
        enableInput(true);
        return;
    }
  }
  
  messageInput.value = "";
  enableInput(false);
  
  // Optimistically append user message directly to DOM
  const userMsg = {
    id: Date.now().toString() + "-user",
    messageType: "user",
    content: prompt, // Rendered as plain text
  };
  appendMessageToDOM(userMsg);
  
  if (currentEventSource) currentEventSource.close();
  
  await streamAssistantResponse(chatId, prompt);
}

// MODIFIED: This function now ONLY creates the chat and updates the state. It does NOT reload the page.
async function startNewChat(prompt) {
  try {
    const title = prompt.length > 50 ? prompt.slice(0, 50) + "..." : prompt;
    const resp = await fetch("/api/chats", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ title }),
    });
    if (!resp.ok) throw new Error("Server failed to create chat");
    const newChat = await resp.json();
    
    // Update the state WITHOUT reloading the page
    activeChatID = newChat.id;
    history.pushState({ chatId: newChat.id }, "", `/chat?id=${newChat.id}`);
    insertOrUpdateChatInSidebar(newChat);
    updateSidebarActiveHighlight();
    
    return newChat.id; // Return the new ID to sendMessage
  } catch (err) {
    console.error(err);
    alert("Could not start a new chat session.");
    return null;
  }
}

async function streamAssistantResponse(chatId, prompt) {
  if (!messageContainer) return;

  const wrapperId = "assistant-stream-" + Date.now();
  const contentId = "assistant-stream-content-" + Date.now();

  const assistantMsgHTML = `
    <div class="flex items-start gap-3 group" id="${wrapperId}">
      <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-gray-200">
        <span class="material-symbols-outlined text-[#64748b]">smart_toy</span>
      </div>
      <div class="flex-1">
        <p class="text-sm font-medium text-[#64748b]">Internist AI</p>
        <div class="relative mt-1">
          <div class="message-content rounded-lg rounded-tl-none bg-gray-100 p-3 text-base text-[#1e293b]">
            <div class="prose prose-lg" id="${contentId}">
              <div class="status-container space-y-2"></div>
            </div>
          </div>
        </div>
      </div>
    </div>`;
  const container = messageContainer.querySelector('.space-y-8');
  if(container) {
    container.insertAdjacentHTML('beforeend', assistantMsgHTML);
    messageContainer.scrollTop = messageContainer.scrollHeight;
  }
  
  const assistantContent = () => document.getElementById(contentId);
  const statusContainer = () => assistantContent()?.querySelector(".status-container");

  if (!document.getElementById("spinner-style")) {
    const style = document.createElement("style");
    style.id = "spinner-style";
    style.textContent = `
      .spinner { display:inline-block;width:1em;height:1em;border:2px solid #ccc;border-top:2px solid #13a4ec;border-radius:50%;animation:spin 0.6s linear infinite;margin-right:0.33em }
      @keyframes spin { to { transform:rotate(360deg);} }
    `;
    document.head.appendChild(style);
  }

  let statuses = {
    retrieving_context: "pending",
    processing: "pending", 
    searching: "pending",
    thinking: "pending",
  };
  const updateStatusUI = () => {
    const steps = [
      { id: "retrieving_context", text: "Analyzing conversation history..." },
      { id: "processing", text: "Processing your question..." },
      { id: "searching", text: "Searching UpToDate..." }, 
      { id: "thinking", text: "Generating response..." },
    ];

    if (!statusContainer()) return;
    statusContainer().innerHTML = steps
      .map(step => {
        if (statuses[step.id] === "pending") return "";
        const icon =
          statuses[step.id] === "completed"
            ? '<span class="material-symbols-outlined text-green-500">check_circle</span>'
            : '<div class="spinner"></div>';
        return `<div class="status-item ${statuses[step.id]}"><div class="status-icon">${icon}</div><span>${step.text}</span></div>`;
      })
      .join("");
  };
  updateStatusUI();
  
  if (currentEventSource) currentEventSource.close();
  
  currentEventSource = new EventSource(`/api/chats/${chatId}/stream?q=${encodeURIComponent(prompt)}`);
  
  currentEventSource.addEventListener("status", e => {
    const data = JSON.parse(e.data);
    Object.keys(statuses).forEach(k => { if (statuses[k] !== "pending") statuses[k] = "completed"; });
    if (statuses.hasOwnProperty(data.status)) statuses[data.status] = "in-progress";
    updateStatusUI();
  });

  let firstChunk = true;
  let fullContent = "";

  currentEventSource.onmessage = e => {
    const data = JSON.parse(e.data);
    const contentDiv = assistantContent();
    if(contentDiv){
        if (firstChunk) {
            contentDiv.innerHTML = "";
            firstChunk = false;
        }
        fullContent += data.content;
        contentDiv.innerHTML = sanitizeHTML(fullContent);
        messageContainer.scrollTop = messageContainer.scrollHeight;
    }
  };

  currentEventSource.addEventListener("done", () => {
    if (currentEventSource) currentEventSource.close();
    enableInput(true);
  });

  currentEventSource.onerror = () => {
    if (currentEventSource) currentEventSource.close();
    if (assistantContent()) assistantContent().innerHTML = '<p class="text-red-500">Sorry, an error occurred.</p>';
    enableInput(true);
  };
}

// ===== PDF Export (unchanged) =====
async function exportMessageAsPDF(messageHTML) {
  try {
    const resp = await fetch("/static/pdf_template.html");
    if (!resp.ok) throw new Error("PDF template not found");
    const template = await resp.text();
    const win = window.open("", "_blank", "height=800,width=800");
    if (!win) return alert("Please allow popups for PDF export");
    win.document.write(template);
    win.document.close();
    win.onload = () => {
      const dateEl = win.document.getElementById("generation-date");
      const contentEl = win.document.getElementById("ai-content-container");
      if (dateEl) dateEl.textContent = new Date().toLocaleString();
      if (contentEl) contentEl.innerHTML = DOMPurify.sanitize(messageHTML);
      setTimeout(() => { win.print(); win.close(); }, 250);
    };
  } catch (err) {
    console.error(err);
    alert("Could not generate PDF.");
  }
}

// ===== Initial Load =====
if (messageContainer) loadInitialMessages();
