// File: web/static/js/chat.js
document.addEventListener("DOMContentLoaded", function () {
    const chatForm = document.getElementById("chatForm");
    const chatInput = document.getElementById("chatInput");
    const chatMessages = document.getElementById("chatMessages");
    const newChatBtn = document.getElementById("newChatBtn");
    const historyList = document.getElementById("historyList");

    let activeChatId = new URLSearchParams(window.location.search).get("id");

    // --- Main Submit Handler ---
    chatForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        const prompt = chatInput.value.trim();
        if (!prompt) return;

        displayMessage(prompt, "user");
        chatInput.value = "";
        toggleLoading(true);

        let chatId = activeChatId;

        // STEP 1: If it's a new chat, create it first to get an ID.
        if (!chatId) {
            try {
                const res = await fetch("/api/chats", {
                    method: "POST",
                    headers: { "Content-Type": "application/json" },
                    body: JSON.stringify({ title: prompt }),
                });
                if (!res.ok) throw new Error(`Failed to create chat: ${res.statusText}`);
                
                const newChat = await res.json();
                chatId = newChat.ID;
                
                window.history.pushState({ chatId }, "", `/chat?id=${chatId}`);
                await loadHistory();
                setActiveChat(chatId);

            } catch (err) {
                console.error(err);
                displayMessage("Error: Could not create a new chat session.", "assistant");
                toggleLoading(false);
                return;
            }
        }

        // STEP 2: Start the stream now that we have a valid chat ID.
        const aiMsgElement = displayMessage("", "assistant", true);
        const eventSource = new EventSource(`/api/chats/${chatId}/stream?q=${encodeURIComponent(prompt)}`);

        eventSource.onmessage = (event) => {
            aiMsgElement.querySelector("p").textContent += event.data;
            chatMessages.scrollTop = chatMessages.scrollHeight;
        };

        eventSource.onerror = (err) => {
            // This will now only fire on a genuine network error,
            // not when the stream closes normally.
            console.error("EventSource failed:", err);
            displayMessage("A streaming error occurred. Please try again.", "assistant");
            eventSource.close();
            toggleLoading(false);
        };

        // ADDED: This listens for our custom "done" event from the server.
        eventSource.addEventListener('done', () => {
            console.log("Stream successfully completed by server.");
            eventSource.close();
            toggleLoading(false);
        });
    });

    // --- Helper Functions ---
    function toggleLoading(isLoading) {
        chatInput.disabled = isLoading;
        chatForm.querySelector("button").disabled = isLoading;
    }

    function displayMessage(content, role, returnElement = false) {
        const li = document.createElement("li");
        li.className = `msg ${role}`;
        
        const avatar = document.createElement("span");
        avatar.className = "avatar";
        avatar.textContent = role === "user" ? "You" : "AI";
        
        const p = document.createElement("p");
        p.textContent = content;
        
        li.appendChild(avatar);
        li.appendChild(p);
        chatMessages.appendChild(li);
        
        chatMessages.scrollTop = chatMessages.scrollHeight;
        
        if (returnElement) {
            return li;
        }
    }
    
    async function loadHistory() {
        try {
            const res = await fetch("/api/chats");
            if (!res.ok) return;
            const chats = await res.json();
            historyList.innerHTML = "";
            if (chats) {
                chats.forEach(chat => {
                    const div = document.createElement("div");
                    div.className = "history-item";
                    div.setAttribute("data-chat-id", chat.ID);
                    div.innerHTML = `<a href="/chat?id=${chat.ID}">${chat.Title}</a><button class="delete-chat-btn" title="Delete chat">&times;</button>`;
                    historyList.prepend(div);
                });
                setActiveChat(activeChatId);
            }
        } catch (err) {
            console.error("Failed to load history:", err);
        }
    }
    
    async function loadMessages(chatId) {
        if (!chatId) {
            chatMessages.innerHTML = "";
            activeChatId = null;
            return;
        }
        try {
            const res = await fetch(`/api/chats/${chatId}/messages`);
            if (!res.ok) {
                 window.location.href = "/chat";
                 return;
            }
            const messages = await res.json();
            chatMessages.innerHTML = "";
            if(messages) {
                messages.forEach(msg => displayMessage(msg.Content, msg.Role));
            }
            activeChatId = chatId;
            setActiveChat(chatId);
        } catch (err) {
            console.error("Failed to load messages:", err);
        }
    }

    function setActiveChat(chatId) {
        document.querySelectorAll(".history-item.active").forEach(el => el.classList.remove("active"));
        if(chatId) {
            const el = document.querySelector(`.history-item[data-chat-id='${chatId}']`);
            if(el) el.classList.add("active");
        }
    }

    // --- Event Listeners ---
    newChatBtn.addEventListener("click", (e) => {
        e.preventDefault();
        if (activeChatId) {
            window.history.pushState({}, "", "/chat");
            loadMessages(null);
        }
    });

    historyList.addEventListener("click", async (e) => {
        if(e.target.classList.contains("delete-chat-btn")) {
            e.preventDefault();
            const item = e.target.closest(".history-item");
            const chatId = item.getAttribute("data-chat-id");
            if(confirm("Are you sure you want to delete this chat?")) {
                try {
                    const res = await fetch(`/api/chats/${chatId}`, { method: "DELETE" });
                    if(res.ok) {
                        item.remove();
                        if (chatId === activeChatId) newChatBtn.click();
                    }
                } catch(err) {
                    console.error("Failed to delete chat", err);
                }
            }
        } else if (e.target.tagName === 'A') {
            e.preventDefault();
            const item = e.target.closest(".history-item");
            const chatId = item.getAttribute("data-chat-id");
            if (chatId !== activeChatId) {
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
    });
});