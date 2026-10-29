import { toast } from "react-toastify";

/**
 * Strips tokens, passwords, OTPs, database details, stack traces,
 * and internal server jargon to protect security and provide clean UX.
 */
function sanitizeMessage(message) {
  if (!message || typeof message !== "string") {
    return "An unexpected error occurred. Please try again.";
  }

  const lower = message.toLowerCase();

  // Redact security credentials and internal diagnostics
  if (
    lower.includes("bearer ") ||
    lower.includes("jwt") ||
    lower.includes("refresh_token") ||
    lower.includes("access_token") ||
    lower.includes("password") ||
    lower.includes("otp") ||
    lower.includes("client_secret") ||
    lower.includes("sql") ||
    lower.includes("query error") ||
    lower.includes("database error") ||
    lower.includes("syntax error") ||
    lower.includes("panic:") ||
    lower.includes("stack trace") ||
    lower.includes("internal server error")
  ) {
    return "A secure server error occurred. Please try again or contact support.";
  }

  return message.trim();
}

/**
 * Maps known raw API response errors to clear, friendly user messages.
 */
export function getErrorMessage(error, fallback = "An unexpected error occurred. Please try again.") {
  if (!error) return fallback;
  if (typeof error === "string") return sanitizeMessage(error);

  // Network connection error
  if (error.code === "ERR_NETWORK" || error.message === "Network Error") {
    return "Unable to connect to the server. Please check your internet connection.";
  }

  // Request timeout
  if (error.code === "ECONNABORTED" || error.message?.includes("timeout")) {
    return "Request timed out. Please try again.";
  }

  // HTTP status-specific safe defaults
  const status = error.response?.status;
  const serverMsg =
    error.response?.data?.message ||
    error.response?.data?.error ||
    error.message;

  if (status === 401) {
    return "Your session has expired. Please sign in again.";
  }
  if (status === 403) {
    return "You do not have permission to perform this action.";
  }
  if (status === 404) {
    return serverMsg ? sanitizeMessage(serverMsg) : "Requested resource was not found.";
  }
  if (status === 429) {
    return "Too many requests. Please slow down and try again shortly.";
  }
  if (status >= 500) {
    return "Server encountered an issue. Our team has been notified.";
  }

  return sanitizeMessage(serverMsg || fallback);
}

/**
 * Generates a stable deduplication ID from message content or options.
 */
function getToastId(message, options = {}) {
  if (options.toastId) return options.toastId;
  if (typeof message === "string") {
    // Hash-like short string from message to prevent identical stacking
    return message.slice(0, 60);
  }
  return undefined;
}

export const notify = {
  success: (message, options = {}) => {
    const text = sanitizeMessage(message);
    const toastId = getToastId(text, options);
    return toast.success(text, {
      toastId,
      autoClose: 4000,
      ...options,
    });
  },

  error: (errorOrMessage, options = {}) => {
    const text =
      typeof errorOrMessage === "string"
        ? sanitizeMessage(errorOrMessage)
        : getErrorMessage(errorOrMessage, options.fallback);
    const toastId = getToastId(text, options);
    return toast.error(text, {
      toastId,
      autoClose: 6000,
      ...options,
    });
  },

  warning: (message, options = {}) => {
    const text = sanitizeMessage(message);
    const toastId = getToastId(text, options);
    return toast.warning(text, {
      toastId,
      autoClose: 5000,
      ...options,
    });
  },

  info: (message, options = {}) => {
    const text = sanitizeMessage(message);
    const toastId = getToastId(text, options);
    return toast.info(text, {
      toastId,
      autoClose: 4000,
      ...options,
    });
  },

  dismiss: (id) => toast.dismiss(id),
  clearWaitingQueue: () => toast.clearWaitingQueue(),
  isActive: (id) => toast.isActive(id),
};

export default notify;
