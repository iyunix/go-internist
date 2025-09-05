// File: web/static/js/utils.js
// Common utility functions

export const Utils = {
  // Debounce function calls
  debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
      const later = () => {
        clearTimeout(timeout);
        func(...args);
      };
      clearTimeout(timeout);
      timeout = setTimeout(later, wait);
    };
  },

  // Throttle function calls
  throttle(func, limit) {
    let inThrottle;
    return function(...args) {
      if (!inThrottle) {
        func.apply(this, args);
        inThrottle = true;
        setTimeout(() => inThrottle = false, limit);
      }
    };
  },

  // NEW: Centralized function to escape HTML and prevent XSS
  escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = String(text); // Ensure input is treated as a string
    return div.innerHTML;
  },

  // Safe logging to server
  safeLog(level, message, payload) {
    try {
      if (typeof window.logToServer === "function") {
        window.logToServer(level, message, payload);
      }
    } catch (_) {
      // Fail silently
    }
  },

  // Auto-scroll to bottom of container
  scrollToBottom(container, smooth = true) {
    if (!container) return;
    if (smooth) {
      container.scrollTo({
        top: container.scrollHeight,
        behavior: 'smooth'
      });
    } else {
      container.scrollTop = container.scrollHeight;
    }
  },

  // Get URL parameter
  getUrlParam(name) {
    return new URLSearchParams(window.location.search).get(name);
  }
};