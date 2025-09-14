// File: web/static/js/admin.js
// REWRITTEN to match the new Tailwind CSS admin panel design.

document.addEventListener('DOMContentLoaded', () => {
    const userTableBody = document.getElementById('user-table-body');
    const searchInput = document.getElementById('user-search-input');

    if (!userTableBody || !searchInput) {
        console.error('Required admin elements not found!');
        return;
    }

    /**
     * Fetches all users from the API.
     */
    async function fetchUsers() {
        try {
            const response = await fetch('/api/admin/users');
            if (!response.ok) throw new Error(`HTTP error! Status: ${response.status}`);
            // Assuming the API response is { "users": [...] }
            const data = await response.json();
            return data.users || data || [];
        } catch (error) {
            console.error('Error fetching users:', error);
            userTableBody.innerHTML = `<tr><td colspan="4" class="px-6 py-4 text-center text-red-500">Error loading users. Please try again.</td></tr>`;
            return [];
        }
    }

    /**
     * Renders user data into the new Tailwind-styled table.
     * @param {Array} users - An array of user objects.
     */
    function renderUserTable(users) {
        userTableBody.innerHTML = ''; // Clear previous content
        if (users.length === 0) {
            userTableBody.innerHTML = `<tr><td colspan="4" class="px-6 py-4 text-center text-gray-500">No users found.</td></tr>`;
            return;
        }

        users.forEach(user => {
            const row = document.createElement('tr');
            // Add search terms to the row for easier filtering
            row.setAttribute('data-search-terms', `${user.Username.toLowerCase()} ${user.PhoneNumber.toLowerCase()}`);

            const roleBadge = user.IsAdmin
                ? `<span class="inline-flex items-center rounded-full bg-green-100 px-2.5 py-0.5 text-xs font-medium text-green-800">Admin</span>`
                : `<span class="inline-flex items-center rounded-full bg-blue-100 px-2.5 py-0.5 text-xs font-medium text-blue-800">User</span>`;

            row.innerHTML = `
                <td class="whitespace-nowrap px-6 py-4">
                    <div class="text-sm font-medium text-gray-900">${user.Username}</div>
                    <div class="text-sm text-gray-500">${user.PhoneNumber}</div>
                </td>
                <td class="whitespace-nowrap px-6 py-4">
                    <div class="text-sm text-gray-900">${user.Balance.toLocaleString()} / <span class="text-gray-500">${user.TotalBalance.toLocaleString()}</span></div>
                </td>
                <td class="whitespace-nowrap px-6 py-4 text-sm">${roleBadge}</td>
                <td class="whitespace-nowrap px-6 py-4 text-right text-sm font-medium">
                    <div class="action-container" data-user-id="${user.ID}">
                        <a href="#" class="add-credits-link text-primary-600 hover:text-primary-900">Add Credits</a>
                        <form class="top-up-form hidden flex items-center gap-2">
                            <input type="number" class="credits-input block w-24 rounded-md border-0 py-1.5 text-gray-900 ring-1 ring-inset ring-gray-300 placeholder:text-gray-400 focus:ring-2 focus:ring-inset focus:ring-primary-600 sm:text-sm" placeholder="Amount" min="1" required>
                            <button type="submit" class="rounded bg-primary-600 px-2 py-1 text-xs font-semibold text-white shadow-sm hover:bg-primary-500">Top-up</button>
                        </form>
                    </div>
                </td>
            `;
            userTableBody.appendChild(row);
        });
    }

    /**
     * A generic helper for making admin API POST calls.
     * @param {string} url - The API endpoint.
     * @param {object} body - The JSON body for the request.
     */
    async function apiCall(url, body) {
        try {
            const response = await fetch(url, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(body),
            });
            const result = await response.json();
            if (!response.ok) throw new Error(result.message || 'An API error occurred.');
            return result;
        } catch (error) {
            console.error(`API call to ${url} failed:`, error);
            alert(`Error: ${error.message}`);
            return null;
        }
    }

    /**
     * Handles search input to filter the table rows.
     */
    function handleSearch(event) {
        const searchTerm = event.target.value.toLowerCase();
        userTableBody.querySelectorAll('tr').forEach(row => {
            const searchTerms = row.dataset.searchTerms || '';
            row.style.display = searchTerms.includes(searchTerm) ? '' : 'none';
        });
    }

    /**
     * Handles clicks within the user table for actions.
     */
    async function handleTableClick(event) {
        const target = event.target;
        
        // Handle "Add Credits" link click
        if (target.classList.contains('add-credits-link')) {
            event.preventDefault();
            const form = target.nextElementSibling;
            if (form) {
                target.classList.add('hidden');
                form.classList.remove('hidden');
                form.querySelector('input').focus();
            }
        }

        // Handle "Top-up" form submission
        if (target.closest('.top-up-form')) {
            event.preventDefault();
            const form = target.closest('.top-up-form');
            const container = form.closest('.action-container');
            const userID = container.dataset.userId;
            const input = form.querySelector('.credits-input');
            const amount = parseInt(input.value, 10);

            if (userID && !isNaN(amount) && amount > 0) {
                const result = await apiCall('/api/admin/users/topup', { userID: parseInt(userID, 10), amountToAdd: amount });
                if (result) {
                    // Refresh the entire table to show new balance
                    const users = await fetchUsers();
                    renderUserTable(users);
                }
            } else {
                alert('Please enter a valid amount.');
            }
        }
    }
    
    // --- Initial Load and Event Binding ---
    async function init() {
        const users = await fetchUsers();
        renderUserTable(users);
        searchInput.addEventListener('input', handleSearch);
        userTableBody.addEventListener('click', handleTableClick);
    }

    init();
});