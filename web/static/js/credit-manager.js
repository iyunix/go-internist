// File: web/static/js/credit-manager.js
// CORRECTED VERSION

class CreditManager {
    constructor() {
        this.currentBalance = 0;
        this.totalCredits = 0;
        this.isLoading = false;
        this.retryCount = 0;
        this.maxRetries = 5;
        this.refreshInterval = null;
        this.lowCreditNotified = false;

        // DOM Elements
        this.progressFill = document.getElementById('credit-progress-fill');
        this.progressText = document.getElementById('credit-percentage');
        this.currentBalanceElement = document.getElementById('current-balance');
        this.totalBalanceElement = document.querySelector('.total-balance'); // <-- This is a correct addition
        this.creditStatus = document.getElementById('credit-status');
        this.statusText = this.creditStatus?.querySelector('.status-text');

        this.init();
    }

    init() {
        console.log('[CreditManager] Initializing...');
        this.loadBalance();

        this.refreshInterval = setInterval(() => {
            if (!this.isLoading) this.loadBalance();
        }, 30000);
    }

    async loadBalance() {
        if (this.isLoading) return;
        this.isLoading = true;

        try {
            // FIXED: Reverted URL to the correct one your backend is using.
            const response = await fetch('/api/user/balance', {
                method: 'GET',
                credentials: 'include',
                headers: { 'Accept': 'application/json' }
            });

            if (!response.ok) {
                if (response.status === 401) {
                    this.showError('Please log in again');
                    setTimeout(() => (window.location.href = '/login'), 3000);
                    return;
                }
                throw new Error(`HTTP ${response.status}`);
            }

            const data = await response.json();
            // FIXED: Reverted this check to use 'balance' as per your original code.
            if (typeof data.balance !== 'number' || typeof data.totalCredits !== 'number') {
                throw new Error('Invalid balance data');
            }

            this.retryCount = 0; // reset on success
            this.updateBalance(data.balance, data.totalCredits);

        } catch (err) {
            console.error('[CreditManager] Error:', err);
            this.showError('Failed to load balance');
            this.retryWithBackoff();
        } finally {
            this.isLoading = false;
        }
    }

    retryWithBackoff() {
        if (this.retryCount >= this.maxRetries) return;
        const delay = Math.min(1000 * 2 ** this.retryCount, 30000);
        this.retryCount++;
        console.log(`[CreditManager] Retrying in ${delay / 1000}s`);
        setTimeout(() => this.loadBalance(), delay);
    }

    updateBalance(newBalance, totalCredits) {
        const oldBalance = this.currentBalance;
        this.currentBalance = Math.max(0, newBalance);
        this.totalCredits = totalCredits;

        // This block updates the total credits display in the UI.
        if (this.totalBalanceElement) {
            this.totalBalanceElement.textContent = totalCredits.toLocaleString();
        }

        const percentage = totalCredits > 0 ? Math.max(0, Math.min(100, (this.currentBalance / this.totalCredits) * 100)) : 0;

        this.animateProgressBar(percentage);
        this.updateBalanceDisplay(oldBalance, this.currentBalance);
        this.updateStatus(percentage);

        if (percentage <= 10 && percentage > 0 && !this.lowCreditNotified) {
            this.lowCreditNotified = true;
            this.showNotification('⚠️', 'Low credit balance!', 'warning');
        }

        if (this.currentBalance === 0) {
            this.disableChatInput();
            this.showNotification('❌', 'No credits left!', 'danger');
        } else {
            this.enableChatInput();
        }

        this.dispatchBalanceEvent(this.currentBalance, percentage);
    }

    // --- The rest of your file remains unchanged ---

    animateProgressBar(percentage) {
        if (!this.progressFill || !this.progressText) return;
        this.progressFill.style.width = `${percentage}%`;
        this.progressText.textContent = `${Math.round(percentage)}%`;

        this.progressFill.classList.remove('warning', 'danger');
        if (percentage <= 5) this.progressFill.classList.add('danger');
        else if (percentage <= 20) this.progressFill.classList.add('warning');
    }

    updateBalanceDisplay(fromBalance, toBalance) {
        if (!this.currentBalanceElement) return;
        const duration = 800, start = performance.now(), diff = toBalance - fromBalance;

        const animate = (time) => {
            const progress = Math.min((time - start) / duration, 1);
            const easeOut = 1 - Math.pow(1 - progress, 3);
            const current = Math.round(fromBalance + diff * easeOut);
            this.currentBalanceElement.textContent = current.toLocaleString();
            if (progress < 1) requestAnimationFrame(animate);
        };
        requestAnimationFrame(animate);
    }

    updateStatus(percentage) {
        if (!this.statusText) return;
        this.statusText.className = 'status-text';

        if (percentage > 50) {
            this.statusText.textContent = 'Good standing';
            this.statusText.classList.add('good');
        } else if (percentage > 20) {
            this.statusText.textContent = 'Running low';
            this.statusText.classList.add('warning');
        } else if (percentage > 0) {
            this.statusText.textContent = 'Critical low';
            this.statusText.classList.add('danger');
        } else {
            this.statusText.textContent = 'No credits';
            this.statusText.classList.add('danger');
        }
    }

    onQuestionAsked(charsUsed) {
        console.log(`[CreditManager] Deducting ${charsUsed} credits`);
        const newBalance = Math.max(0, this.currentBalance - charsUsed);
        this.updateBalance(newBalance, this.totalCredits);

        setTimeout(() => {
            if (!this.isLoading) this.loadBalance();
        }, 2000);
    }

    showNotification(icon, text, type) {
        if (document.querySelector('.credit-notification')) return;

        const n = document.createElement('div');
        n.className = `credit-notification ${type}`;
        n.innerHTML = `<div class="notification-content">
            <span class="notification-icon">${icon}</span>
            <span class="notification-text">${text}</span>
        </div>`;
        document.body.appendChild(n);

        setTimeout(() => {
            n.style.animation = 'slideOutNotification 0.3s ease-in forwards';
            setTimeout(() => n.remove(), 300);
        }, 4000);
    }

    disableChatInput() {
        const input = document.getElementById('chatInput');
        const btn = document.querySelector('#chatForm button');
        if (input && btn) {
            input.disabled = true;
            btn.disabled = true;
            input.placeholder = 'No credits remaining';
        }
    }

    enableChatInput() {
        const input = document.getElementById('chatInput');
        const btn = document.querySelector('#chatForm button');
        if (input && btn) {
            input.disabled = false;
            btn.disabled = false;
            input.placeholder = 'Ask me anything…';
        }
    }

    showError(message) {
        console.error(`[CreditManager] ${message}`);
        if (this.progressText) this.progressText.textContent = 'Error';
        if (this.statusText) {
            this.statusText.textContent = message;
            this.statusText.className = 'status-text danger';
        }
        if (this.currentBalanceElement) this.currentBalanceElement.textContent = '--';
    }

    dispatchBalanceEvent(balance, percentage) {
        document.dispatchEvent(new CustomEvent('balanceUpdated', {
            detail: { balance, percentage, totalCredits: this.totalCredits }
        }));
    }

    canAskQuestion(len) {
        const minCharge = 100;
        return this.currentBalance >= Math.max(minCharge, len);
    }

    destroy() {
        if (this.refreshInterval) clearInterval(this.refreshInterval);
    }
}

// CSS animations (rest of file is the same)
const style = document.createElement('style');
style.textContent = `
@keyframes slideOutNotification { from{opacity:1;} to{opacity:0; transform:translateX(100%);} }
.credit-notification {
    position:fixed; top:20px; right:20px; z-index:1000;
    padding:12px 16px; border-radius:8px; font-size:14px;
    font-weight:500; max-width:300px; box-shadow:0 4px 12px rgba(0,0,0,0.15);
    animation:slideInNotification 0.3s ease-out;
}
.credit-notification.warning { background:#fff3cd; color:#856404; border-left:4px solid #ffc107; }
.credit-notification.danger { background:#f8d7da; color:#721c24; border-left:4px solid #dc3545; }
`;
document.head.appendChild(style);

// Initialize
let creditManager = null;
function initializeCreditManager() {
    if (creditManager) creditManager.destroy();
    creditManager = new CreditManager();
    window.creditManager = creditManager;
}
document.readyState === 'loading'
    ? document.addEventListener('DOMContentLoaded', initializeCreditManager)
    : initializeCreditManager();