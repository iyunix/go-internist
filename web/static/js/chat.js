document.addEventListener("DOMContentLoaded", function () {
    const chatForm = document.getElementById("chatForm");
    const chatInput = document.getElementById("chatInput");
    const chatMessages = document.getElementById("chatMessages");
    const newChatBtn = document.getElementById("newChatBtn");
    const historyList = document.getElementById("historyList");

    let currentChatID = null;

    async function loadHistory() {
        try {
            const response = await fetch("/api/chats");
            if (!response.ok) return;
            const chats = await response.json();
            historyList.innerHTML = "";
            if (chats) {
                chats.forEach(chat => createHistoryItemDOM(chat.Title, chat.ID));
            }
        } catch (error) {
            console.error("Failed to load history:", error);
        }
    }

    async function loadChat(chatID) {
        if (!chatID) {
            chatMessages.innerHTML = "";
            currentChatID = null;
            document.querySelectorAll(".history-item.active").forEach(el => el.classList.remove("active"));
            return;
        }

        try {
            const response = await fetch(`/api/chats/${chatID}/messages`);
            if (!response.ok) {
                window.location.href = "/chat";
                return;
            }
            const messages = await response.json();
            chatMessages.innerHTML = "";
            if (messages) {
                // --- THIS IS THE FIX ---
                // Handle both capitalized (msg.Content) and lowercase (msg.content) properties
                // to make the rendering robust.
                messages.forEach(msg => {
                    const content = msg.Content || msg.content;
                    const role = msg.Role || msg.role;
                    appendMessage(content, role);
                });
            }
            currentChatID = chatID;
            document.querySelectorAll(".history-item.active").forEach(el => el.classList.remove("active"));
            const activeItem = document.querySelector(`.history-item[data-chat-id='${chatID}']`);
            if (activeItem) {
                activeItem.classList.add("active");
            }
        } catch (error) {
            console.error("Failed to load chat:", error);
        }
    }

    function appendMessage(content, author) {
        const messageItem = document.createElement("li");
        messageItem.className = `msg ${author}`;
        const avatar = document.createElement("span");
        avatar.className = "avatar";
        avatar.textContent = author === "user" ? "You" : "AI";
        const text = document.createElement("p");
        text.textContent = content;
        messageItem.appendChild(avatar);
        messageItem.appendChild(text);
        chatMessages.appendChild(messageItem);
        chatMessages.scrollTop = chatMessages.scrollHeight;
    }

    function createHistoryItemDOM(title, chatID) {
        const wrapper = document.createElement("div");
        wrapper.className = "history-item";
        wrapper.setAttribute("data-chat-id", chatID);

        const link = document.createElement("a");
        link.href = `/chat?id=${chatID}`;
        link.textContent = title;

        const deleteBtn = document.createElement("button");
        deleteBtn.className = "delete-chat-btn";
        deleteBtn.innerHTML = "&times;";
        deleteBtn.title = "Delete chat";
        deleteBtn.onclick = (e) => {
            e.preventDefault();
            e.stopPropagation();
            if (confirm("Are you sure you want to delete this chat?")) {
                deleteChat(chatID, wrapper);
            }
        };

        wrapper.appendChild(link);
        wrapper.appendChild(deleteBtn);
        historyList.prepend(wrapper);
    }

    async function deleteChat(chatID, elementToRemove) {
        try {
            const response = await fetch(`/api/chats/${chatID}`, { method: 'DELETE' });
            if (response.ok) {
                elementToRemove.remove();
                if (currentChatID == chatID) {
                    newChatBtn.click();
                }
            } else {
                alert("Failed to delete chat.");
            }
        } catch (error) {
            console.error("Failed to delete chat:", error);
        }
    }
    
    function setLoading(isLoading) {
        const loadingIndicator = document.getElementById("loadingIndicator");
        if (isLoading) {
            chatInput.disabled = true;
            chatForm.querySelector("button").disabled = true;
            if (!loadingIndicator) {
                const indicator = document.createElement("div");
                indicator.id = "loadingIndicator";
                indicator.className = "msg assistant";
                indicator.innerHTML = `<span class="avatar">AI</span><p><span class="spinner"></span></p>`;
                chatMessages.appendChild(indicator);
                chatMessages.scrollTop = chatMessages.scrollHeight;
            }
        } else {
            chatInput.disabled = false;
            chatForm.querySelector("button").disabled = false;
            if (loadingIndicator) {
                loadingIndicator.remove();
            }
            chatInput.focus();
        }
    }

    chatForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        const msg = chatInput.value.trim();
        if (!msg) return;


        appendMessage(msg, "user");
        chatInput.value = "";
        chatInput.dir = 'ltr';
        chatInput.style.height = "auto";
        setLoading(true);

        const endpoint = currentChatID ? `/api/chats/${currentChatID}/messages` : "/api/chats";

        try {
            let bodyObj = { content: msg };
            if (currentChatID !== null && currentChatID !== undefined) bodyObj.chat_id = currentChatID;
            const response = await fetch(endpoint, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify(bodyObj)
            });

            setLoading(false);
            const data = await response.json();
            
            if (data.reply) {
                appendMessage(data.reply, "assistant");
            } else if (data.error) {
                appendMessage(data.error, "assistant");
            }
            
            if (data.chat && !currentChatID) {
                await loadHistory();
                window.history.pushState({}, "", `/chat?id=${data.chat.ID}`);
                loadChat(data.chat.ID);
            }
        } catch (err) {
            setLoading(false);
            appendMessage("Sorry, a server error occurred.", "assistant");
        }
    });

    historyList.addEventListener("click", function(e) {
        e.preventDefault();
        const item = e.target.closest(".history-item");
        if (item) {
            const chatID = item.getAttribute("data-chat-id");
            if (chatID !== currentChatID) {
                window.history.pushState({}, "", `/chat?id=${chatID}`);
                loadChat(chatID);
            }
        }
    });

    newChatBtn.addEventListener("click", (e) => {
        e.preventDefault();
        if (currentChatID !== null) {
            window.history.pushState({}, "", "/chat");
            loadChat(null);
        }
    });

    chatInput.addEventListener('input', () => {
        const rtlRegex = /[\u0600-\u06FF\u0750-\u077F]/;
        chatInput.dir = rtlRegex.test(chatInput.value) ? 'rtl' : 'ltr';
        chatInput.style.height = "auto";
        chatInput.style.height = (chatInput.scrollHeight) + "px";
    });

    const initialChatID = new URLSearchParams(window.location.search).get("id");
    loadHistory().then(() => {
        if (initialChatID) {
            loadChat(initialChatID);
        }
    });
});
