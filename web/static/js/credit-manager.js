// File: web/static/js/credit-manager.js

class CreditManager {
  constructor() {
    this.currentBalance = 0;
    this.isLoading = false;
    this.currentBalanceElement = document.getElementById('current-balance');

    this.init();
  }

  init() {
    if (!this.currentBalanceElement) {
      console.warn('[CreditManager] Credit display element not found. Manager will not run.');
      return;
    }
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
        }
        throw new Error(`HTTP ${response.status}`);
      }

      const data = await response.json();
      // Attempt multiple structures for robustness
      let balance =
        typeof data.balance === 'number' ? data.balance :
        typeof data.credit === 'number' ? data.credit :
        (data.data && typeof data.data.balance === 'number') ? data.data.balance :
        (data.data && typeof data.data.credit === 'number') ? data.data.credit :
        null;

      if (balance === null) throw new Error('Invalid balance data received from API');
      this.updateBalance(balance);

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

// Initialize after DOM loads
document.addEventListener('DOMContentLoaded', () => {
  if (document.getElementById('current-balance')) {
    new CreditManager();
  }
});
