// ===== Global & Setup =====
const currentUsername = window.INTERNIST_DATA?.currentUsername || "Unknown User";
let activeChatID = window.INTERNIST_DATA?.activeChatID || 0;

if (typeof marked === "undefined" || typeof DOMPurify === "undefined") {
  console.error("CRITICAL ERROR: Libraries not loaded.");
}

// ===== DOM References =====
const messageContainer = document
  .getElementById("message-container")
  .querySelector(".space-y-8");

const chatForm = document.getElementById("chat-form");
const messageInput = document.getElementById("message-input");
const sendButton = document.getElementById("send-button");
const newChatBtn = document.getElementById("new-chat-btn");
const chatListContainer = document.getElementById("chat-list-container");
const deleteModal = document.getElementById("delete-confirm-modal");
const confirmDeleteBtn = document.getElementById("modal-confirm-delete-btn");
const cancelDeleteBtn = document.getElementById("modal-cancel-btn");

// ===== Sidebar References =====
const sidebar = document.getElementById("sidebar");
const sidebarToggle = document.getElementById("sidebar-toggle");

// ===== Scroll Helper =====
const mainContainer = document.querySelector("main.flex-1.flex-col");
const scrollToBottom = () => {
  if (mainContainer) mainContainer.scrollTop = mainContainer.scrollHeight;
};
window.onload = scrollToBottom;

// ===== Sidebar Collapse/Expand Logic =====
if (sidebar && sidebarToggle) {
  // Toggle button click
  sidebarToggle.addEventListener("click", () => {
    sidebar.classList.toggle("collapsed");
  });

  // Hover expand for desktop
  sidebar.addEventListener("mouseenter", () => {
    if (window.innerWidth >= 769 && sidebar.classList.contains("collapsed")) {
      sidebar.classList.add("hover-expanded");
    }
  });
  sidebar.addEventListener("mouseleave", () => {
    if (window.innerWidth >= 769) sidebar.classList.remove("hover-expanded");
  });
}

// ===== Infinite Scroll (Load Older Messages) =====
let messagePage = 1;
const messageLimit = 50;
let allMessagesLoaded = false;

const isTopVisible = () => messageContainer.scrollTop < 50;

messageContainer.addEventListener("scroll", async () => {
  if (isTopVisible() && !allMessagesLoaded) await loadOlderMessages();
});

async function loadOlderMessages() {
  if (allMessagesLoaded) return;
  messagePage += 1;

  const resp = await fetch(`/api/chats/${activeChatID}/messages?page=${messagePage}&limit=${messageLimit}`);
  if (!resp.ok) return;

  const data = await resp.json();
  if (!data.has_more || data.messages.length === 0) {
    allMessagesLoaded = true;
    return;
  }

  const prevHeight = messageContainer.scrollHeight;
  data.messages.reverse().forEach(renderMessageAtTop);
  messageContainer.scrollTop = messageContainer.scrollHeight - prevHeight + messageContainer.scrollTop;
}

function renderMessageAtTop(msg) {
  const html =
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
             </div>
           </div>
         </div>`
      : `<div class="flex items-start justify-end gap-3">
           <div class="flex flex-col items-end">
             <p class="text-right text-sm font-medium text-[#64748b]">${currentUsername}</p>
             <div class="message-content mt-1 rounded-lg rounded-tr-none bg-[#13a4ec] p-3 text-base text-white">${msg.content}</div>
           </div>
         </div>`;

  const el = document.createElement("div");
  el.innerHTML = html;
  Array.from(el.childNodes).forEach(node => messageContainer.prepend(node));
}

// ===== Chat List + Delete Modal =====
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

if (confirmDeleteBtn) {
  confirmDeleteBtn.addEventListener("click", () => {
    deleteModal.classList.add("hidden");
    deleteChat(confirmDeleteBtn.dataset.chatId);
  });
}

if (cancelDeleteBtn) {
  cancelDeleteBtn.addEventListener("click", () => {
    deleteModal.classList.add("hidden");
  });
}

// ===== New Chat =====
if (newChatBtn) newChatBtn.addEventListener("click", createNewChat);

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
    window.location.href = `/chat?id=${newChat.id}`;
  } catch (err) {
    console.error("Error creating new chat:", err);
    alert("Could not create a new chat. Please try again.");
  }
}

// ===== Delete Chat =====
async function deleteChat(chatId) {
  try {
    const resp = await fetch(`/api/chats/${chatId}`, { method: "DELETE" });
    if (!resp.ok) throw new Error("Delete failed");
    document.querySelector(`div[data-chat-item-id='${chatId}']`)?.remove();
    if (parseInt(chatId) === activeChatID) window.location.href = "/chat";
  } catch (err) {
    console.error("Error deleting chat:", err);
    alert("Could not delete the chat. Please try again.");
  }
}

// ===== Send Message Flow =====
if (chatForm) {
  chatForm.addEventListener("submit", e => {
    e.preventDefault();
    const prompt = messageInput.value.trim();
    if (prompt) sendMessage(prompt);
  });
}

async function sendMessage(prompt) {
  const welcomeMsg = document.getElementById("welcome-message");
  if (welcomeMsg) welcomeMsg.remove();

  let currentChatId = activeChatID;
  if (currentChatId === 0) {
    currentChatId = await startNewChat(prompt);
    if (!currentChatId) return;
  }

  messageInput.value = "";
  enableInput(false);

  try {
    await fetch(`/api/chats/${currentChatId}/messages`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ content: prompt, messageType: "user" }),
    });
  } catch (err) {
    console.error("Error saving user message:", err);
    alert("Could not send message.");
    enableInput(true);
    return;
  }

  const safePrompt = DOMPurify.sanitize(prompt);
  messageContainer.insertAdjacentHTML(
    "beforeend",
    `<div class="flex items-start justify-end gap-3">
      <div class="flex flex-col items-end">
        <p class="text-right text-sm font-medium text-[#64748b]">${currentUsername}</p>
        <div class="message-content mt-1 rounded-lg rounded-tr-none bg-[#13a4ec] p-3 text-base text-white">${safePrompt}</div>
      </div>
    </div>`
  );

  scrollToBottom();
  await streamAssistantResponse(currentChatId, prompt);
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
    history.pushState({}, "", `/chat?id=${newChat.id}`);
    return newChat.id;
  } catch (err) {
    console.error("Error creating chat:", err);
    alert("Could not start a new chat session.");
    return null;
  }
}

function enableInput(enable = true) {
  messageInput.disabled = !enable;
  sendButton.disabled = !enable;
  if (enable) messageInput.focus();
}

// ===== Stream Assistant Response =====
async function streamAssistantResponse(chatId, prompt) {
  const wrapperId = "assistant-" + Date.now();
  const contentId = "assistant-content-" + Date.now();

  messageContainer.insertAdjacentHTML(
    "beforeend",
    `<div class="flex items-start gap-3 group" id="${wrapperId}">
      <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-gray-200">
        <span class="material-symbols-outlined text-[#64748b]">smart_toy</span>
      </div>
      <div class="flex-1">
        <p class="text-sm font-medium text-[#64748b]">Internist AI</p>
        <div class="relative mt-1">
          <div class="message-content rounded-lg rounded-tl-none bg-gray-100 p-3 text-base text-[#1e293b]">
            <div id="${contentId}" class="prose prose-lg">
              <div class="status-container space-y-2"></div>
            </div>
          </div>
        </div>
      </div>
    </div>`
  );
  scrollToBottom();

  const assistantContent = document.getElementById(contentId);
  const statusContainer = assistantContent.querySelector(".status-container");
  let fullResponse = "";
  let statuses = {};

  const updateStatusUI = () => {
    const steps = [
      { id: "understanding", text: "Understanding question..." },
      { id: "searching", text: "Searching UpToDate..." },
      { id: "thinking", text: "Generating response..." },
    ];
    statusContainer.innerHTML = steps
      .map(step => {
        if (!statuses[step.id]) return "";
        const icon =
          statuses[step.id] === "completed"
            ? '<span class="material-symbols-outlined text-green-500">check_circle</span>'
            : '<div class="spinner"></div>';
        return `<div class="status-item ${statuses[step.id]}">
                  <div class="status-icon">${icon}</div>
                  <span>${step.text}</span>
                </div>`;
      })
      .join("");
  };

  updateStatusUI();

  const es = new EventSource(`/api/chats/${chatId}/stream?q=${encodeURIComponent(prompt)}`);
  es.addEventListener("status", e => {
    const data = JSON.parse(e.data);
    for (const k in statuses) statuses[k] = "completed";
    statuses[data.status] = "in-progress";
    updateStatusUI();
  });

  let firstToken = true;
  es.onmessage = e => {
    if (firstToken) {
      assistantContent.innerHTML = "";
      firstToken = false;
    }
    const data = JSON.parse(e.data);
    fullResponse += data.content;
    assistantContent.innerHTML = DOMPurify.sanitize(marked.parse(fullResponse));
    scrollToBottom();
  };

  es.addEventListener("done", () => {
    es.close();
    const wrapper = document.getElementById(wrapperId);
    if (wrapper) {
      const relative = wrapper.querySelector(".relative");
      relative.insertAdjacentHTML(
        "beforeend",
        `<button onclick="exportMessageAsPDF(this.closest('.relative').querySelector('.message-content').innerHTML)" class="absolute top-1 right-1 opacity-0 group-hover:opacity-100 transition-opacity bg-white p-1 rounded-full shadow-sm hover:bg-gray-200"><span class="material-symbols-outlined text-base text-gray-600">picture_as_pdf</span></button>`
      );
    }
    enableInput(true);
  });

  es.onerror = () => {
    assistantContent.innerHTML = '<p class="text-red-500">Sorry, an error occurred.</p>';
    es.close();
    enableInput(true);
  };
}

// ===== PDF Export =====
async function exportMessageAsPDF(messageHTML) {
  try {
    const resp = await fetch("/static/pdf_template.html");
    if (!resp.ok) throw new Error("PDF template not found");
    const template = await resp.text();
    const win = window.open("", "_blank", "height=800,width=800");
    win.document.write(template);
    win.document.close();
    win.onload = () => {
      const dateEl = win.document.getElementById("generation-date");
      const contentEl = win.document.getElementById("ai-content-container");
      if (dateEl) dateEl.textContent = new Date().toLocaleString();
      if (contentEl) contentEl.innerHTML = DOMPurify.sanitize(messageHTML);
      setTimeout(() => {
        win.print();
        win.close();
      }, 250);
    };
  } catch (err) {
    console.error("Error exporting PDF:", err);
    alert("Could not generate PDF.");
  }
}
