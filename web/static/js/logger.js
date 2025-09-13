/**
 * Sends a log message to the backend server.
 * @param {string} level The log level (e.g., 'info', 'warn', 'error').
 * @param {string} message The primary log message.
 * @param {object} [context={}] Optional additional data or context.
 */
async function logToServer(level, message, context = {}) {
  try {
    const payload = JSON.stringify({ level, message, context });
    
    // Use navigator.sendBeacon if available. It's a reliable way
    // to send data even when a page is closing.
    if (navigator.sendBeacon) {
      navigator.sendBeacon('/api/log', payload);
    } else {
      // Fallback to fetch for older browsers.
      await fetch('/api/log', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: payload,
        keepalive: true // Important for requests during page unload
      });
    }
  } catch (error) {
    // If logging fails, log to the console as a last resort.
    console.error('Failed to send log to server:', error);
  }
}

/**
 * Global error handler to catch all unhandled JavaScript exceptions.
 * This is the key to finding and fixing unexpected bugs.
 */
window.onerror = function(message, source, lineno, colno, error) {
  logToServer('error', 'Unhandled Exception', {
    message: message,
    source: source,
    lineno: lineno,
    colno: colno,
    // Include the stack trace if the browser provides it.
    stack: error ? error.stack : 'N/A',
  });
  // Return true to prevent the browser's default error handling (e.g., logging to console).
  return true;
};