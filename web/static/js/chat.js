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
const vScrollContainer = document.getElementById("message-container");
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
const mainContainer = document.querySelector("main.flex-1.flex-col");

// ===== Virtual Scroll State =====
let virtualMessages = []; // [{id, messageType, content,...}]
let vTopIndex = 0;
let vVisibleCount = 30;
let vMessageHeight = 90;
let vScrollBuffer = 6;

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
  return DOMPurify.sanitize(marked.parse(html));
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
// Sidebar Chat Click/Delete Event
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
      const firstChat = chatListContainer.querySelector("[data-chat-item-id]");
      if (firstChat) {
        activeChatID = parseInt(firstChat.dataset.chatItemId, 10);
        updateSidebarActiveHighlight();
        window.location.href = `/chat?id=${activeChatID}`;
      } else {
        activeChatID = 0;
        window.location.href = "/chat";
      }
    }
  } catch (err) {
    console.error(err);
    alert("Could not delete the chat. Please try again.");
  }
}
newChatBtn?.addEventListener("click", createNewChat);
async function createNewChat() {
  try {
    const response = await fetch("/api/chats", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ title: "New Chat" }),
    });
    if (!response.ok) throw new Error("Failed to create chat");
    const newChat = await response.json();
    activeChatID = newChat.id;
    insertOrUpdateChatInSidebar(newChat);
    updateSidebarActiveHighlight();
    window.location.href = `/chat?id=${newChat.id}`;
  } catch (err) {
    console.error(err);
    alert("Could not create a new chat. Please try again.");
  }
}

// ===== Virtual Message Rendering =====
function renderVirtualMessages(force = false) {
  if (!vScrollContainer) return;
  const scrollTop = vScrollContainer.scrollTop;
  const containerHeight = vScrollContainer.clientHeight;
  vVisibleCount = Math.ceil(containerHeight / vMessageHeight) + vScrollBuffer;
  vTopIndex = Math.max(0, Math.floor(scrollTop / vMessageHeight) - Math.floor(vScrollBuffer / 2));
  const vBottomIndex = Math.min(vTopIndex + vVisibleCount, virtualMessages.length);
  const topSpacer = document.createElement("div");
  topSpacer.style.height = vTopIndex * vMessageHeight + "px";
  const bottomSpacer = document.createElement("div");
  bottomSpacer.style.height = (virtualMessages.length - vBottomIndex) * vMessageHeight + "px";
  let visibleHTML = "";
  for (let i = vTopIndex; i < vBottomIndex; ++i) {
    const msg = virtualMessages[i];
    visibleHTML +=
      msg.messageType === "assistant"
        ? `<div class="flex items-start gap-3 group">
            <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-gray-200">
              <span class="material-symbols-outlined text-[#64748b]">smart_toy</span>
            </div>
            <div class="flex-1">
              <p class="text-sm font-medium text-[#64748b]">Internist AI</p>
              <div class="relative mt-1">
                <div class="message-content rounded-lg rounded-tl-none bg-gray-100 p-3 text-base text-[#1e293b]">
                  <div class="prose prose-lg">${msg.content}</div>
                </div>
                <button onclick="exportMessageAsPDF(this.parentElement.querySelector('.message-content').innerHTML)"
                        class="absolute top-1 right-1 transition-opacity bg-white p-1 rounded-full shadow-sm hover:bg-gray-200 z-10"
                        title="Export as PDF">
                  <span class="material-symbols-outlined text-base text-gray-600">picture_as_pdf</span>
                </button>
              </div>
            </div>
          </div>`

        : `<div class="flex items-start justify-end gap-3">
             <div class="flex flex-col items-end">
               <p class="text-right text-sm font-medium text-[#64748b]">${currentUsername}</p>
               <div class="message-content mt-1 rounded-lg rounded-tr-none bg-[#13a4ec] p-3 text-base text-white" dir="${isRTL(msg.content) ? "rtl" : "ltr"}">${msg.content}</div>
             </div>
           </div>`;
  }
  vScrollContainer.innerHTML = "";
  vScrollContainer.appendChild(topSpacer);
  const messagesDiv = document.createElement("div");
  messagesDiv.innerHTML = visibleHTML;
  while (messagesDiv.firstChild) {
    vScrollContainer.appendChild(messagesDiv.firstChild);
  }
  vScrollContainer.appendChild(bottomSpacer);
}

// ===== Virtual Scroll Infinite Scroll Control =====
let messagePage = 1;
const messageLimit = 50;
let allMessagesLoaded = false;
async function loadVirtualOlderMessages() {
  if (allMessagesLoaded || !vScrollContainer) return;
  messagePage += 1;
  try {
    const resp = await fetch(`/api/chats/${activeChatID}/messages?page=${messagePage}&limit=${messageLimit}`);
    if (!resp.ok) throw new Error("Failed to load messages");
    const data = await resp.json();
    if (!data.has_more || !Array.isArray(data.messages) || data.messages.length === 0) {
      allMessagesLoaded = true;
      return;
    }
    virtualMessages = data.messages.concat(virtualMessages);
    renderVirtualMessages(true);
  } catch (err) {
    console.error(err);
  }
}
async function loadInitialMessages() {
  messagePage = 1;
  allMessagesLoaded = false;
  try {
    const resp = await fetch(`/api/chats/${activeChatID}/messages?page=${messagePage}&limit=${messageLimit}`);
    if (!resp.ok) throw new Error("Failed to load messages");
    const data = await resp.json();
    virtualMessages = Array.isArray(data.messages) ? data.messages : [];
    renderVirtualMessages(true);
    vScrollContainer.scrollTop = vScrollContainer.scrollHeight;
  } catch (err) {
    console.error(err);
  }
}
vScrollContainer?.addEventListener("scroll", () => {
  if (vScrollContainer.scrollTop < 50 && !allMessagesLoaded) {
    loadVirtualOlderMessages();
  }
  renderVirtualMessages();
});
window.addEventListener("resize", () => renderVirtualMessages(true));

// ===== Send & Stream Messages =====
chatForm?.addEventListener("submit", e => {
  e.preventDefault();
  const prompt = messageInput.value.trim();
  if (prompt) sendMessage(prompt);
});
let currentEventSource = null;
async function sendMessage(prompt) {
  if (!vScrollContainer) return;
  document.getElementById("welcome-message")?.remove();
  let chatId = activeChatID;
  if (chatId === 0) chatId = await startNewChat(prompt);
  if (!chatId) return;
  messageInput.value = "";
  enableInput(false);
  // Optimistically push user message to virtual list
  const userMsg = {
    id: Date.now().toString() + "-user",
    messageType: "user",
    content: sanitizeHTML(prompt),
  };
  virtualMessages.push(userMsg);
  if (currentEventSource) currentEventSource.close();
  renderVirtualMessages(true);
  vScrollContainer.scrollTop = vScrollContainer.scrollHeight;
  try {
    await fetch(`/api/chats/${chatId}/messages`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ content: prompt, messageType: "user" }),
    });
  } catch {
    alert("Error sending message.");
    enableInput(true);
    return;
  }
  await streamAssistantResponse(chatId);
}
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
    activeChatID = newChat.id;
    insertOrUpdateChatInSidebar(newChat);
    updateSidebarActiveHighlight();
    history.pushState({}, "", `/chat?id=${newChat.id}`);
    await loadInitialMessages();
    return newChat.id;
  } catch {
    alert("Could not start a new chat session.");
    return null;
  }
}
async function streamAssistantResponse(chatId) {
  if (!vScrollContainer) return;
  const vIdx = virtualMessages.length;
  const wrapperId = "assistant-stream-" + Date.now();
  const contentId = "assistant-stream-content-" + Date.now();
  const msgObj = {
    id: wrapperId,
    messageType: "assistant",
    content: `<div id="${contentId}"><div class="status-container space-y-2"></div></div>`,
  };
  virtualMessages.push(msgObj);
  renderVirtualMessages(true);
  vScrollContainer.scrollTop = vScrollContainer.scrollHeight;
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
    understanding: "pending",
    searching: "pending",
    thinking: "pending",
  };
  const updateStatusUI = () => {
    const steps = [
      { id: "understanding", text: "Understanding question..." },
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
  currentEventSource = new EventSource(`/api/chats/${chatId}/stream`);
  currentEventSource.addEventListener("status", e => {
    const data = JSON.parse(e.data);
    Object.keys(statuses).forEach(k => { if (statuses[k] !== "pending") statuses[k] = "completed"; });
    if (statuses.hasOwnProperty(data.status)) statuses[data.status] = "in-progress";
    updateStatusUI();
  });
  let firstChunk = true;
  currentEventSource.onmessage = e => {
    const data = JSON.parse(e.data);
    if (firstChunk) {
      assistantContent().innerHTML = "";
      firstChunk = false;
    }
    assistantContent().insertAdjacentHTML("beforeend", sanitizeHTML(data.content));
    vScrollContainer.scrollTop = vScrollContainer.scrollHeight;
  };
  function finishStream() {
    if (currentEventSource) currentEventSource.close();
    msgObj.content = assistantContent()?.innerHTML || "";
    renderVirtualMessages(true);
    enableInput(true);
  }
  currentEventSource.addEventListener("done", finishStream);
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

// ===== Initial Load (call this on page ready!) =====
if (vScrollContainer) loadInitialMessages();
