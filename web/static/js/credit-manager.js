// File: web/static/js/credit-manager.js
// MODIFIED to work with the simplified Tailwind CSS UI in chat.html

class CreditManager {
    constructor() {
        this.currentBalance = 0;
        this.isLoading = false;
        
        // --- SIMPLIFIED DOM Elements ---
        // The only element we need to update in the new UI.
        this.currentBalanceElement = document.getElementById('current-balance');

        this.init();
    }

    init() {
        if (!this.currentBalanceElement) {
            console.warn('[CreditManager] Credit display element not found. Manager will not run.');
            return;
        }
        console.log('[CreditManager] Initializing...');
        this.loadBalance();
    }

    async loadBalance() {
        if (this.isLoading) return;
        this.isLoading = true;

        try {
            const response = await fetch('/api/user/balance', {
                method: 'GET',
                credentials: 'include',
                headers: { 'Accept': 'application/json' }
            });

            if (!response.ok) {
                if (response.status === 401) {
                    console.error('[CreditManager] User not authenticated.');
                    this.updateBalanceDisplay('--');
                    // Optional: redirect to login
                    // window.location.href = '/login';
                }
                throw new Error(`HTTP ${response.status}`);
            }

            const data = await response.json();
            if (typeof data.balance !== 'number') {
                throw new Error('Invalid balance data received from API');
            }
            
            this.updateBalance(data.balance);

        } catch (err) {
            console.error('[CreditManager] Error loading balance:', err);
            this.updateBalanceDisplay('Error');
        } finally {
            this.isLoading = false;
        }
    }

    updateBalance(newBalance) {
        this.currentBalance = Math.max(0, newBalance);
        this.updateBalanceDisplay(this.currentBalance.toLocaleString());
        
        if (this.currentBalance === 0) {
            this.disableChatInput();
        } else {
            this.enableChatInput();
        }

        // Dispatch an event so other parts of the app can react
        document.dispatchEvent(new CustomEvent('balanceUpdated', {
            detail: { balance: this.currentBalance }
        }));
    }
    
    updateBalanceDisplay(text) {
        if (this.currentBalanceElement) {
            this.currentBalanceElement.textContent = text;
        }
    }

    disableChatInput() {
        const input = document.getElementById('chatInput');
        const btn = document.querySelector('#chatForm button[type="submit"]');
        if (input && btn) {
            input.disabled = true;
            btn.disabled = true;
            input.placeholder = 'No credits remaining';
        }
    }

    enableChatInput() {
        const input = document.getElementById('chatInput');
        const btn = document.querySelector('#chatForm button[type="submit"]');
        if (input && btn) {
            input.disabled = false;
            btn.disabled = false;
            input.placeholder = 'Ask me anything…';
        }
    }
}

// Initialize the credit manager when the page loads
document.addEventListener('DOMContentLoaded', () => {
    // Only initialize if we are on a page that has the balance element
    if (document.getElementById('current-balance')) {
        new CreditManager();
    }
});