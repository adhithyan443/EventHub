/**
 * Utility formatters for currency, dates, and relative timestamps in EventHub.
 */

/**
 * Formats a numeric or decimal string value into Indian Rupee (INR) currency format.
 *
 * @param {number|string} amount
 * @param {string} [currency="INR"]
 * @returns {string} Formatted currency string or "—" if invalid
 */
export const formatCurrency = (amount, currency = "INR") => {
  if (amount === null || amount === undefined || amount === "") return "—";
  const num = typeof amount === "number" ? amount : parseFloat(amount);
  if (Number.isNaN(num)) return "—";
  return new Intl.NumberFormat("en-IN", {
    style: "currency",
    currency: currency || "INR",
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(num);
};

/**
 * Formats an ISO date/timestamp into a readable date and time string in Indian locale.
 *
 * @param {string|Date} dateVal
 * @returns {string} Formatted date time string or "—"
 */
export const formatDateTime = (dateVal) => {
  if (!dateVal) return "—";
  const d = new Date(dateVal);
  if (Number.isNaN(d.getTime())) return "—";
  return d.toLocaleDateString("en-IN", {
    day: "numeric",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
};

/**
 * Formats an ISO date into a short human-readable relative time string (e.g. "5m ago", "2h ago", "Yesterday").
 *
 * @param {string|Date} dateVal
 * @returns {string}
 */
export const formatRelativeTime = (dateVal) => {
  if (!dateVal) return "—";
  const d = new Date(dateVal);
  if (Number.isNaN(d.getTime())) return "—";
  const now = new Date();
  const diffSec = Math.floor((now.getTime() - d.getTime()) / 1000);

  if (diffSec < 0) {
    return d.toLocaleDateString("en-IN", {
      day: "numeric",
      month: "short",
    });
  }
  if (diffSec < 60) return "Just now";
  if (diffSec < 3600) return `${Math.floor(diffSec / 60)}m ago`;
  if (diffSec < 86400) return `${Math.floor(diffSec / 3600)}h ago`;
  if (diffSec < 172800) return "Yesterday";
  return d.toLocaleDateString("en-IN", {
    day: "numeric",
    month: "short",
  });
};
