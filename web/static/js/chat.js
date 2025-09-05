// File: web/static/js/chat.js
// Chat interface with Markdown rendering for assistant responses
// Dependencies: marked.js, DOMPurify, MarkdownRenderer helper must be loaded first

document.addEventListener("DOMContentLoaded", function () {
  // DOM elements
  const chatForm = document.getElementById("chatForm");
  const chatInput = document.getElementById("chatInput");
  const chatMessages = document.getElementById("chatMessages");
  const newChatBtn = document.getElementById("newChatBtn");
  const historyList = document.getElementById("historyList");
  
  // State
  let activeChatId = new URLSearchParams(window.location.search).get("id");

  // --- Helper Functions ---
  function toggleLoading(isLoading) {
    chatInput.disabled = isLoading;
    chatForm.querySelector("button").disabled = isLoading;
  }

  // Display message: user as plain text, assistant as rendered Markdown HTML
  function displayMessage(content, role, returnElement = false) {
    const li = document.createElement("li");
    li.className = `msg ${role}`;

    const avatar = document.createElement("span");
    avatar.className = "avatar";
    avatar.textContent = role === "user" ? "You" : "AI";

    // Use div for assistant (allows block elements), p for user (inline only)
    const container = document.createElement(role === "assistant" ? "div" : "p");

    if (role === "assistant") {
      try {
        // Render Markdown to safe HTML
        window.MarkdownRenderer?.render(container, content || "");
      } catch (e) {
        // Fallback to plain text if renderer unavailable
        container.textContent = content || "";
      }
    } else {
      container.textContent = content || "";
    }

    li.appendChild(avatar);
    li.appendChild(container);
    chatMessages.appendChild(li);
    chatMessages.scrollTop = chatMessages.scrollHeight;

    if (returnElement) return li;
  }

  async function loadHistory() {
    try {
      const res = await fetch("/api/chats", {
        method: "GET",
        credentials: "same-origin",
        cache: "no-store"
      });
      if (!res.ok) throw new Error(`Failed to load history: ${res.statusText}`);

      const chats = await res.json();
      historyList.innerHTML = "";

      if (Array.isArray(chats)) {
        chats.forEach(chat => {
          const id = String(chat.ID ?? chat.id ?? "");
          const title = String(chat.Title ?? chat.title ?? "Untitled");
          if (!id) return;

          // Create proper li element for ul container
          const li = document.createElement("li");
          li.className = "history-item";
          li.setAttribute("data-chat-id", id);
          li.innerHTML = `<a href="/chat?id=${id}">${title}</a><button class="delete-chat-btn" title="Delete chat" aria-label="Delete chat">&times;</button>`;
          historyList.appendChild(li);
        });
        setActiveChat(activeChatId);
      }
    } catch (err) {
      console.error("Failed to load history:", err);
      safeLog('error', 'Failed to load chat history', { errorMessage: err.message, stack: err.stack });
    }
  }

  async function loadMessages(chatId) {
    if (!chatId) {
      chatMessages.innerHTML = "";
      activeChatId = null;
      setActiveChat(null);
      return;
    }

    try {
      const res = await fetch(`/api/chats/${chatId}/messages`, {
        method: "GET",
        credentials: "same-origin",
        cache: "no-store"
      });
      if (!res.ok) {
        window.location.href = "/chat";
        return;
      }

      const messages = await res.json();
      chatMessages.innerHTML = "";

      if (Array.isArray(messages) && messages.length > 0) {
        messages.forEach(msg => {
          const role = msg.Role ?? msg.role ?? "assistant";
          const content = msg.Content ?? msg.content ?? "";
          const li = displayMessage("", role, true);
          const container = li.querySelector(role === "assistant" ? "div" : "p");
          
          if (role === "assistant") {
            try {
              window.MarkdownRenderer?.render(container, content);
            } catch (e) {
              container.textContent = content;
            }
          } else {
            container.textContent = content;
          }
        });
      }
      
      activeChatId = chatId;
      setActiveChat(chatId);
    } catch (err) {
      console.error("Failed to load messages:", err);
      safeLog('error', 'Failed to load messages for chat', { chatId: chatId, errorMessage: err.message, stack: err.stack });
    }
  }

  function setActiveChat(chatId) {
    document.querySelectorAll(".history-item.active").forEach(el => el.classList.remove("active"));
    if (chatId) {
      const el = document.querySelector(`.history-item[data-chat-id='${chatId}']`);
      if (el) el.classList.add("active");
    }
  }

  function safeLog(level, message, payload) {
    try {
      if (typeof window.logToServer === "function") {
        window.logToServer(level, message, payload);
      }
    } catch (_) {}
  }

  // --- Event Handlers ---
  chatForm.addEventListener("submit", async function (e) {
    e.preventDefault();
    const prompt = chatInput.value.trim();
    if (!prompt) return;

    displayMessage(prompt, "user");
    chatInput.value = "";
    toggleLoading(true);

    let chatId = activeChatId;

    // Create new chat if none active
    if (!chatId) {
      try {
        const csrf = document.querySelector("input[name='csrf_token']")?.value || "";
        const res = await fetch("/api/chats", {
          method: "POST",
          credentials: "same-origin",
          cache: "no-store",
          headers: {
            "Content-Type": "application/json",
            ...(csrf ? { "X-CSRF-Token": csrf } : {})
          },
          body: JSON.stringify({ title: prompt }),
        });
        if (!res.ok) throw new Error(`Failed to create chat: ${res.statusText}`);

        const newChat = await res.json();
        const createdId = newChat.ID ?? newChat.id;
        if (!createdId) throw new Error("CreateChat response missing id");
        chatId = String(createdId);

        window.history.pushState({ chatId }, "", `/chat?id=${chatId}`);
        await loadHistory();
        setActiveChat(chatId);
      } catch (err) {
        console.error(err);
        safeLog('error', 'Failed to create new chat', { errorMessage: err.message, stack: err.stack });
        displayMessage("Error: Could not create a new chat session.", "assistant");
        toggleLoading(false);
        return;
      }
    }

    // Stream assistant response with Markdown rendering
    const aiMsgElement = displayMessage("", "assistant", true);
    const container = aiMsgElement.querySelector("div");

    let stream;
    try {
      stream = window.MarkdownRenderer?.createStream(container);
    } catch (e) {
      // Fallback if MarkdownRenderer not available
      stream = {
        buffer: "",
        append(chunk) { 
          this.buffer += String(chunk); 
          container.textContent = this.buffer; 
        },
        flush() {}
      };
    }

    const eventSource = new EventSource(`/api/chats/${chatId}/stream?q=${encodeURIComponent(prompt)}`);
    eventSource.onmessage = (evt) => {
      stream.append(evt.data);
      chatMessages.scrollTop = chatMessages.scrollHeight;
    };
    eventSource.addEventListener('done', () => {
      eventSource.close();
      stream.flush?.();
      toggleLoading(false);
    });
    eventSource.onerror = (err) => {
      console.error("EventSource failed:", err);
      safeLog('error', 'EventSource streaming failed', { chatId: chatId });
      displayMessage("A streaming error occurred. Please try again.", "assistant");
      eventSource.close();
      toggleLoading(false);
    };
  });

  newChatBtn.addEventListener("click", (e) => {
    e.preventDefault();
    if (activeChatId) {
      window.history.pushState({}, "", "/chat");
      loadMessages(null);
    }
  });

  historyList.addEventListener("click", async (e) => {
    // Delete chat button
    if (e.target.classList.contains("delete-chat-btn")) {
      e.preventDefault();
      const item = e.target.closest("li.history-item");
      const chatId = item?.getAttribute("data-chat-id");
      if (!chatId) return;

      if (confirm("Are you sure you want to delete this chat?")) {
        try {
          const res = await fetch(`/api/chats/${chatId}`, { 
            method: "DELETE", 
            credentials: "same-origin", 
            cache: "no-store" 
          });
          if (res.ok) {
            item.remove();
            if (chatId === activeChatId) newChatBtn.click();
          } else {
            throw new Error(`Failed to delete chat: ${res.statusText}`);
          }
        } catch (err) {
          console.error("Failed to delete chat", err);
          safeLog('error', 'Failed to delete chat', { chatId: chatId, errorMessage: err.message, stack: err.stack });
        }
      }
      return;
    }

    // Chat history link
    if (e.target.tagName === 'A') {
      e.preventDefault();
      const item = e.target.closest("li.history-item");
      const chatId = item?.getAttribute("data-chat-id");
      if (chatId && chatId !== activeChatId) {
        window.history.pushState({ chatId }, "", e.target.href);
        loadMessages(chatId);
      }
    }
  });

  // --- Initial Load ---
  loadHistory().then(() => {
    if (activeChatId) {
      loadMessages(activeChatId);
    }
  }).catch(err => {
    console.error("Error during initial page load:", err);
    safeLog('error', 'Error during initial page load sequence', { errorMessage: err.message, stack: err.stack });
  });
});
