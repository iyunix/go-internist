// --- DOM Elements ---
const searchInput = document.getElementById('search-input');
const userTableBody = document.getElementById('user-table-body');
const paginationContainer = document.getElementById('pagination-container');
let currentSearch = '';
let debounceTimer;

// --- Core Function to Fetch and Render Users ---
async function fetchAndRenderUsers(page = 1, search = '') {
    try {
        const response = await fetch(`/api/admin/users?page=${page}&search=${search}`);
        if (!response.ok) throw new Error('Failed to fetch users');
        const data = await response.json();
        renderTable(data.users);
        renderPagination(data.total, data.page, data.limit);
    } catch (error) {
        console.error("Error fetching users:", error);
        userTableBody.innerHTML = `<tr><td colspan="6" class="text-center py-4 text-red-500">Failed to load users.</td></tr>`;
    }
}

// --- Render Functions ---
function renderTable(users) {
    userTableBody.innerHTML = '';
    if (!users || users.length === 0) {
        userTableBody.innerHTML = `<tr><td colspan="6" class="text-center py-4 text-gray-500">No users found.</td></tr>`;
        return;
    }
    users.forEach(user => {
        const row = `
            <tr>
                <td class="whitespace-nowrap px-6 py-4 text-sm font-medium text-gray-900">${user.username}</td>
                <td class="whitespace-nowrap px-6 py-4 text-sm text-gray-500">${user.phone_number}</td>
                <td class="whitespace-nowrap px-6 py-4 text-sm text-gray-500">${user.character_balance}</td>
                <td class="whitespace-nowrap px-6 py-4 text-sm text-gray-500">
                    ${user.status === 'active' 
                        ? `<span class="inline-flex items-center rounded-full bg-green-100 px-2.5 py-0.5 text-xs font-medium text-green-800">Active</span>`
                        : `<span class="inline-flex items-center rounded-full bg-yellow-100 px-2.5 py-0.5 text-xs font-medium text-yellow-800">Pending</span>`
                    }
                </td>
                <td class="whitespace-nowrap px-6 py-4 text-sm">
                    ${user.isAdmin 
                        ? `<span class="inline-flex items-center rounded-full bg-red-100 px-2.5 py-0.5 text-xs font-medium text-red-800">Admin</span>`
                        : `<span class="inline-flex items-center rounded-full bg-blue-100 px-2.5 py-0.5 text-xs font-medium text-blue-800">User</span>`
                    }
                </td>
                <td class="whitespace-nowrap px-6 py-4 text-right text-sm font-medium">
                    <button class="font-medium text-primary-600 hover:text-primary-900" 
                            data-action="add-credits" 
                            data-userid="${user.id}" 
                            data-username="${user.username}">
                        Add Credits
                    </button>
                </td>
            </tr>
        `;
        userTableBody.innerHTML += row;
    });
}

function renderPagination(total, page, limit) {
    const totalPages = Math.ceil(total / limit);
    paginationContainer.innerHTML = `
        <div>
            <p class="text-sm text-gray-700">
                Showing <span class="font-medium">${(page - 1) * limit + 1}</span>
                to <span class="font-medium">${Math.min(page * limit, total)}</span>
                of <span class="font-medium">${total}</span> results
            </p>
        </div>
        <div>
             <p class="text-sm text-gray-700">Page ${page} of ${totalPages}</p>
        </div>
    `;
}

// --- Event Listeners ---
searchInput.addEventListener('input', (e) => {
    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(() => {
        currentSearch = e.target.value;
        fetchAndRenderUsers(1, currentSearch);
    }, 300);
});

userTableBody.addEventListener('click', (e) => {
    const target = e.target;
    if (target.dataset.action === 'add-credits') {
        const userId = target.dataset.userid;
        const username = target.dataset.username;
        handleAddCredits(userId, username);
    }
});

async function handleAddCredits(userId, username) {
    const amountStr = prompt(`How many credits would you like to add to ${username}?`);
    if (amountStr) {
        const amount = parseInt(amountStr, 10);
        if (isNaN(amount) || amount <= 0) {
            alert('Please enter a valid positive number.');
            return;
        }
        try {
            const response = await fetch('/api/admin/users/topup', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ userID: parseInt(userId), amount: amount })
            });
            if (!response.ok) {
                const errorData = await response.json();
                throw new Error(errorData.error || 'Failed to add credits');
            }
            alert(`Successfully added ${amount} credits to ${username}.`);
            fetchAndRenderUsers(1, currentSearch);
        } catch (error) {
            console.error('Error adding credits:', error);
            alert(`Error: ${error.message}`);
        }
    }
}
