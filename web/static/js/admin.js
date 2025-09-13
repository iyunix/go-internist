// File: web/static/js/admin.js
// REWRITTEN FOR SUBSCRIPTION PLAN MANAGEMENT

document.addEventListener('DOMContentLoaded', () => {
    const userTableBody = document.getElementById('user-table-body');
    const searchInput = document.getElementById('user-search-input');
    
    // For simplicity, we define the available plans here.
    // In a more advanced system, you might fetch these from an API.
    const availablePlans = ['basic', 'pro', 'premium'];

    if (!userTableBody || !searchInput) {
        console.error('Required admin elements not found!');
        return;
    }

    /**
     * Fetches all users from the API and renders them in the table.
     */
    async function fetchAndRenderUsers() {
        try {
            const response = await fetch('/api/admin/users');
            if (!response.ok) throw new Error(`Failed to fetch users: ${response.statusText}`);
            const users = await response.json();
            renderUserTable(users);
        } catch (error) {
            console.error('Error fetching users:', error);
            userTableBody.innerHTML = `<tr><td colspan="7" class="error-message">Error loading users.</td></tr>`;
        }
    }

    /**
     * Renders the user data into the HTML table with new action controls.
     * @param {Array} users - An array of user objects from the API.
     */
    function renderUserTable(users) {
        userTableBody.innerHTML = '';
        if (!users || users.length === 0) {
            userTableBody.innerHTML = `<tr><td colspan="7">No users found.</td></tr>`;
            return;
        }

        users.forEach(user => {
            const row = document.createElement('tr');
            
            // Generate the <option> elements for the plan selector dropdown
            const planOptions = availablePlans.map(plan => 
                `<option value="${plan}" ${user.subscription_plan === plan ? 'selected' : ''}>
                    ${plan.charAt(0).toUpperCase() + plan.slice(1)}
                </option>`
            ).join('');

            const isAdminBadge = user.IsAdmin ? '<span class="status-badge admin">Yes</span>' : '<span class="status-badge user">No</span>';

            // This HTML is new and more complex, with separate controls for each action.
            row.innerHTML = `
                <td>${user.id}</td>
                <td>${user.username}</td>
                <td>${user.phone_number}</td>
                <td>${user.character_balance.toLocaleString()}</td>
                <td>${user.total_character_balance.toLocaleString()}</td>
                <td>${isAdminBadge}</td>
                <td class="actions-column">
                    <div class="action-group plan-changer" data-user-id="${user.id}">
                        <select class="plan-select">${planOptions}</select>
                        <button class="renew-btn">Renew</button>
                    </div>
                    <form class="action-form top-up-form" data-user-id="${user.id}">
                        <input type="number" class="credits-input" placeholder="Top-up Amount" min="1">
                        <button type="submit">Top-up</button>
                    </form>
                </td>
            `;
            userTableBody.appendChild(row);
        });

        addEventListeners();
    }
    
    /**
     * Attaches event listeners to all the new controls in the table.
     */
    function addEventListeners() {
        // Listener for the plan change dropdowns
        document.querySelectorAll('.plan-select').forEach(select => {
            select.addEventListener('change', (event) => {
                const userID = event.target.closest('.plan-changer').dataset.userId;
                const newPlan = event.target.value;
                if (confirm(`Are you sure you want to change user ${userID} to the ${newPlan} plan?`)) {
                    changePlan(userID, newPlan);
                } else {
                    // Reset dropdown if cancelled
                    fetchAndRenderUsers(); 
                }
            });
        });

        // Listener for the "Renew" buttons
        document.querySelectorAll('.renew-btn').forEach(button => {
            button.addEventListener('click', (event) => {
                const userID = event.target.closest('.plan-changer').dataset.userId;
                if (confirm(`Are you sure you want to renew the subscription for user ${userID}? This will reset their balance.`)) {
                    renewSubscription(userID);
                }
            });
        });

        // Listener for the "Top-up" forms
        document.querySelectorAll('.top-up-form').forEach(form => {
            form.addEventListener('submit', (event) => {
                event.preventDefault();
                const userID = form.dataset.userId;
                const input = form.querySelector('.credits-input');
                const amountToAdd = parseInt(input.value, 10);
                if (!isNaN(amountToAdd) && amountToAdd > 0) {
                    topUpBalance(userID, amountToAdd);
                } else {
                    alert('Please enter a valid amount to top-up.');
                }
            });
        });
    }

    // --- NEW API CALLING FUNCTIONS ---

    async function changePlan(userID, newPlan) {
        await apiCall(`/api/admin/users/plan`, { userID: parseInt(userID, 10), newPlan });
    }

    async function renewSubscription(userID) {
        await apiCall(`/api/admin/users/renew`, { userID: parseInt(userID, 10) });
    }

    async function topUpBalance(userID, amountToAdd) {
        await apiCall(`/api/admin/users/topup`, { userID: parseInt(userID, 10), amountToAdd });
    }

    /**
     * A generic helper function for making API calls and refreshing the table.
     * @param {string} url - The API endpoint to call.
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
            
            await fetchAndRenderUsers(); // Refresh table on success
        } catch (error) {
            console.error(`API call to ${url} failed:`, error);
            alert(`Error: ${error.message}`);
        }
    }
    
    // --- Unchanged Functions ---
    function handleSearch() {
        const searchTerm = searchInput.value.toLowerCase();
        const rows = userTableBody.querySelectorAll('tr');
        rows.forEach(row => {
            const username = row.cells[1].textContent.toLowerCase();
            const phoneNumber = row.cells[2].textContent.toLowerCase();
            row.style.display = (username.includes(searchTerm) || phoneNumber.includes(searchTerm)) ? '' : 'none';
        });
    }

    fetchAndRenderUsers();
    searchInput.addEventListener('input', handleSearch);
});