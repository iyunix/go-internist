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
    return function() {
      const args = arguments;
      const context = this;
      if (!inThrottle) {
        func.apply(context, args);
        inThrottle = true;
        setTimeout(() => inThrottle = false, limit);
      }
    };
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
