import { useState, useEffect } from "react";
import notify from "../../utils/toast";

const paymentStatusBadgeStyles = {
  SUCCESS: "bg-[#dcfce7] text-[#15803d] border border-[#bbf7d0]",
  PENDING: "bg-[#fef9c3] text-[#854d0e] border border-[#fef08a]",
  FAILED: "bg-[#fee2e2] text-[#b91c1c] border border-[#fecaca]",
  REFUND_REQUIRED: "bg-[#ffedd5] text-[#c2410c] border border-[#fed7aa]",
  REFUNDED: "bg-[#f3e8ff] text-[#7e22ce] border border-[#e9d5ff]",
};

const allocationBadgeStyles = {
  ACCOUNTED: "bg-[#ccfbf1] text-[#0f766e] border border-[#99f6e4]",
  UNALLOCATED: "bg-[#fef3c7] text-[#b45309] border border-[#fde68a]",
  NOT_APPLICABLE: "bg-[#f3f4f6] text-[#4b5563] border border-[#e5e7eb]",
};

const formatCurrency = (amount, currency = "INR") => {
  if (amount === null || amount === undefined) return "—";
  const num = typeof amount === "number" ? amount : parseFloat(amount);
  if (isNaN(num)) return "—";
  return new Intl.NumberFormat("en-IN", {
    style: "currency",
    currency: currency || "INR",
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(num);
};

const formatDateTime = (dateStr) => {
  if (!dateStr) return "—";
  const d = new Date(dateStr);
  if (isNaN(d.getTime())) return "—";
  return d.toLocaleDateString("en-IN", {
    day: "numeric",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
};

export default function PaymentDetailDrawer({ payment, onClose }) {
  const [copiedField, setCopiedField] = useState(null);

  useEffect(() => {
    const handleKeyDown = (e) => {
      if (e.key === "Escape") {
        onClose();
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [onClose]);

  if (!payment) return null;

  const copyToClipboard = async (text, fieldName) => {
    if (!text) return;
    try {
      await navigator.clipboard.writeText(text);
      setCopiedField(fieldName);
      notify.info(`Copied ${fieldName} to clipboard`);
      setTimeout(() => {
        setCopiedField(null);
      }, 2000);
    } catch {
      notify.error("Failed to copy to clipboard");
    }
  };

  const statusKey = payment.status?.toUpperCase() || "PENDING";
  const statusBadge =
    paymentStatusBadgeStyles[statusKey] ||
    "bg-[#f3f4f6] text-[#4b5563] border border-[#e5e7eb]";

  const allocationKey = payment.allocation_status?.toUpperCase() || "NOT_APPLICABLE";
  const allocationBadge =
    allocationBadgeStyles[allocationKey] ||
    "bg-[#f3f4f6] text-[#4b5563] border border-[#e5e7eb]";

  const allocationExplanation = {
    ACCOUNTED:
      "This payment has been captured and successfully allocated to the double-entry financial ledger (Debited Razorpay Clearing, Credited Organizer Payable & Platform Revenue).",
    UNALLOCATED:
      "This payment was captured by Razorpay, but the booking was expired or cancelled prior to final confirmation. Funds are held in clearing and require refund processing or allocation reconciliation.",
    NOT_APPLICABLE:
      "Ledger revenue allocation is not applicable for this record (e.g. zero-value free bookings, uncaptured pending payments, or failed attempts).",
  }[allocationKey] || "Ledger allocation status information not available.";

  return (
    <div className="fixed inset-0 z-50 overflow-hidden">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/40 backdrop-blur-xs transition-opacity"
        onClick={onClose}
      />

      <div className="fixed inset-y-0 right-0 max-w-full flex pl-10">
        <div className="w-screen max-w-xl bg-white shadow-2xl flex flex-col">
          {/* Drawer Header */}
          <div className="px-6 py-5 border-b border-[#dce2f7] bg-[#f9f9ff] flex items-center justify-between">
            <div>
              <div className="flex items-center gap-2">
                <h2 className="text-[#141b2b] text-lg font-bold">Payment Details</h2>
                <span
                  className={`text-[10px] font-bold px-2 py-0.5 rounded-full ${statusBadge}`}
                >
                  {statusKey}
                </span>
              </div>
              <p className="text-[#6d7a77] text-xs font-mono mt-0.5 select-all">
                {payment.id}
              </p>
            </div>
            <button
              type="button"
              onClick={onClose}
              className="p-1.5 rounded-lg text-[#6d7a77] hover:text-[#141b2b] hover:bg-[#eef0fb] transition-colors cursor-pointer"
              title="Close drawer"
            >
              <svg width="18" height="18" viewBox="0 0 20 20" fill="currentColor">
                <path
                  fillRule="evenodd"
                  d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
                  clipRule="evenodd"
                />
              </svg>
            </button>
          </div>

          {/* Drawer Content */}
          <div className="flex-1 overflow-y-auto p-6 space-y-6">
            {/* Amount & Status Card */}
            <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-5 shadow-[0px_1px_1.5px_rgba(0,0,0,0.1)]">
              <div className="text-xs font-medium text-[#6d7a77] uppercase tracking-wide">
                Payment Amount
              </div>
              <div className="text-3xl font-extrabold text-[#141b2b] mt-1 tracking-tight">
                {formatCurrency(payment.amount, payment.currency)}
              </div>
              <div className="flex flex-wrap items-center gap-2 mt-3 pt-3 border-t border-[rgba(188,201,198,0.3)]">
                <div className="flex items-center gap-1.5">
                  <span className="text-xs text-[#6d7a77] font-medium">Status:</span>
                  <span
                    className={`text-xs font-bold px-2 py-0.5 rounded-full ${statusBadge}`}
                  >
                    {statusKey}
                  </span>
                </div>
                <span className="text-[#bcc9c6]">•</span>
                <div className="flex items-center gap-1.5">
                  <span className="text-xs text-[#6d7a77] font-medium">Ledger Allocation:</span>
                  <span
                    className={`text-xs font-bold px-2 py-0.5 rounded-full ${allocationBadge}`}
                  >
                    {allocationKey}
                  </span>
                </div>
                <span className="text-[#bcc9c6]">•</span>
                <div className="flex items-center gap-1.5">
                  <span className="text-xs text-[#6d7a77] font-medium">Gateway:</span>
                  <span
                    className={`text-xs font-semibold px-2 py-0.5 rounded-full ${
                      payment.has_gateway_capture
                        ? "bg-[#dcfce7] text-[#15803d]"
                        : "bg-[#f3f4f6] text-[#6b7280]"
                    }`}
                  >
                    {payment.has_gateway_capture ? "Captured" : "Not Captured"}
                  </span>
                </div>
              </div>
            </div>

            {/* Allocation Context Explainer */}
            <div
              className={`rounded-xl p-4 border text-xs leading-relaxed ${
                allocationKey === "ACCOUNTED"
                  ? "bg-[#f0fdfa] border-[#99f6e4] text-[#0f766e]"
                  : allocationKey === "UNALLOCATED"
                  ? "bg-[#fffbeb] border-[#fde68a] text-[#92400e]"
                  : "bg-[#f9fafb] border-[#e5e7eb] text-[#4b5563]"
              }`}
            >
              <div className="font-bold flex items-center gap-1.5 mb-1">
                <svg width="14" height="14" viewBox="0 0 20 20" fill="currentColor">
                  <path
                    fillRule="evenodd"
                    d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z"
                    clipRule="evenodd"
                  />
                </svg>
                Financial Ledger Allocation Context ({allocationKey})
              </div>
              <p>{allocationExplanation}</p>
            </div>

            {/* Identifiers & Provider Reference */}
            <div className="bg-white border border-[#dce2f7] rounded-xl overflow-hidden shadow-xs">
              <div className="bg-[#f1f3ff] px-4 py-2.5 border-b border-[#dce2f7] text-xs font-bold text-[#141b2b] uppercase tracking-wide">
                Gateway & Transaction References
              </div>
              <div className="p-4 space-y-3">
                <div>
                  <div className="flex items-center justify-between text-xs text-[#6d7a77] mb-1">
                    <span>Transaction ID</span>
                    {payment.transaction_id && (
                      <button
                        type="button"
                        onClick={() =>
                          copyToClipboard(payment.transaction_id, "Transaction ID")
                        }
                        className="text-[#00685f] text-[11px] font-semibold hover:underline cursor-pointer"
                      >
                        {copiedField === "Transaction ID" ? "Copied!" : "Copy"}
                      </button>
                    )}
                  </div>
                  <div className="text-xs font-mono bg-[#f9f9ff] p-2 rounded border border-[#bcc9c6]/40 text-[#141b2b] select-all break-all">
                    {payment.transaction_id || "— (No Gateway Transaction ID)"}
                  </div>
                </div>

                <div>
                  <div className="flex items-center justify-between text-xs text-[#6d7a77] mb-1">
                    <span>Provider Order ID</span>
                    {payment.provider_order_id && (
                      <button
                        type="button"
                        onClick={() =>
                          copyToClipboard(payment.provider_order_id, "Order ID")
                        }
                        className="text-[#00685f] text-[11px] font-semibold hover:underline cursor-pointer"
                      >
                        {copiedField === "Order ID" ? "Copied!" : "Copy"}
                      </button>
                    )}
                  </div>
                  <div className="text-xs font-mono bg-[#f9f9ff] p-2 rounded border border-[#bcc9c6]/40 text-[#141b2b] select-all break-all">
                    {payment.provider_order_id || "—"}
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-3 pt-2">
                  <div>
                    <div className="text-xs text-[#6d7a77]">Provider</div>
                    <div className="text-sm font-semibold text-[#141b2b] mt-0.5">
                      {payment.provider || "RAZORPAY"}
                    </div>
                  </div>
                  <div>
                    <div className="text-xs text-[#6d7a77]">Payment Method</div>
                    <div className="text-sm font-semibold text-[#141b2b] mt-0.5">
                      {payment.method || "—"}
                    </div>
                  </div>
                </div>
              </div>
            </div>

            {/* Booking Information */}
            <div className="bg-white border border-[#dce2f7] rounded-xl overflow-hidden shadow-xs">
              <div className="bg-[#f1f3ff] px-4 py-2.5 border-b border-[#dce2f7] text-xs font-bold text-[#141b2b] uppercase tracking-wide">
                Booking Reference
              </div>
              <div className="p-4 space-y-3">
                <div className="flex items-center justify-between">
                  <div>
                    <div className="text-xs text-[#6d7a77]">Booking Number</div>
                    <div className="text-sm font-bold text-[#141b2b] mt-0.5">
                      {payment.booking_number || "—"}
                    </div>
                  </div>
                  {payment.booking_number && (
                    <button
                      type="button"
                      onClick={() =>
                        copyToClipboard(payment.booking_number, "Booking Number")
                      }
                      className="text-[#00685f] text-xs font-semibold hover:underline cursor-pointer"
                    >
                      {copiedField === "Booking Number" ? "Copied!" : "Copy Number"}
                    </button>
                  )}
                </div>

                <div>
                  <div className="flex items-center justify-between text-xs text-[#6d7a77] mb-1">
                    <span>Booking ID</span>
                    {payment.booking_id && (
                      <button
                        type="button"
                        onClick={() => copyToClipboard(payment.booking_id, "Booking ID")}
                        className="text-[#00685f] text-[11px] font-semibold hover:underline cursor-pointer"
                      >
                        {copiedField === "Booking ID" ? "Copied!" : "Copy"}
                      </button>
                    )}
                  </div>
                  <div className="text-xs font-mono bg-[#f9f9ff] p-2 rounded border border-[#bcc9c6]/40 text-[#141b2b] select-all break-all">
                    {payment.booking_id || "—"}
                  </div>
                </div>
              </div>
            </div>

            {/* Parties: Customer, Event, Organizer */}
            <div className="bg-white border border-[#dce2f7] rounded-xl overflow-hidden shadow-xs">
              <div className="bg-[#f1f3ff] px-4 py-2.5 border-b border-[#dce2f7] text-xs font-bold text-[#141b2b] uppercase tracking-wide">
                Customer & Event Details
              </div>
              <div className="p-4 space-y-4">
                {/* Customer */}
                <div>
                  <div className="text-xs font-semibold text-[#6d7a77] uppercase tracking-wide mb-1">
                    Customer
                  </div>
                  <div className="bg-[#f9f9ff] p-3 rounded-lg border border-[#bcc9c6]/40">
                    <div className="font-bold text-sm text-[#141b2b]">
                      {payment.customer?.name || "Anonymous Customer"}
                    </div>
                    <div className="text-xs text-[#3d4947] mt-0.5">
                      {payment.customer?.email || "—"}
                    </div>
                    {payment.customer?.id && (
                      <div className="text-[10px] text-[#6d7a77] font-mono mt-1 select-all">
                        ID: {payment.customer.id}
                      </div>
                    )}
                  </div>
                </div>

                {/* Event */}
                <div>
                  <div className="text-xs font-semibold text-[#6d7a77] uppercase tracking-wide mb-1">
                    Event
                  </div>
                  <div className="bg-[#f9f9ff] p-3 rounded-lg border border-[#bcc9c6]/40">
                    <div className="font-bold text-sm text-[#141b2b]">
                      {payment.event?.title || "—"}
                    </div>
                    {payment.event?.id && (
                      <div className="text-[10px] text-[#6d7a77] font-mono mt-1 select-all">
                        ID: {payment.event.id}
                      </div>
                    )}
                  </div>
                </div>

                {/* Organizer */}
                <div>
                  <div className="text-xs font-semibold text-[#6d7a77] uppercase tracking-wide mb-1">
                    Organizer
                  </div>
                  <div className="bg-[#f9f9ff] p-3 rounded-lg border border-[#bcc9c6]/40">
                    <div className="font-bold text-sm text-[#141b2b]">
                      {payment.organizer?.business_name || "—"}
                    </div>
                    {payment.organizer?.id && (
                      <div className="text-[10px] text-[#6d7a77] font-mono mt-1 select-all">
                        ID: {payment.organizer.id}
                      </div>
                    )}
                  </div>
                </div>
              </div>
            </div>

            {/* Timestamps */}
            <div className="bg-white border border-[#dce2f7] rounded-xl overflow-hidden shadow-xs">
              <div className="bg-[#f1f3ff] px-4 py-2.5 border-b border-[#dce2f7] text-xs font-bold text-[#141b2b] uppercase tracking-wide">
                Timestamps
              </div>
              <div className="p-4 grid grid-cols-1 sm:grid-cols-3 gap-3">
                <div>
                  <div className="text-xs text-[#6d7a77]">Paid At</div>
                  <div className="text-xs font-semibold text-[#141b2b] mt-1">
                    {formatDateTime(payment.paid_at)}
                  </div>
                </div>
                <div>
                  <div className="text-xs text-[#6d7a77]">Created At</div>
                  <div className="text-xs font-semibold text-[#141b2b] mt-1">
                    {formatDateTime(payment.created_at)}
                  </div>
                </div>
                <div>
                  <div className="text-xs text-[#6d7a77]">Updated At</div>
                  <div className="text-xs font-semibold text-[#141b2b] mt-1">
                    {formatDateTime(payment.updated_at)}
                  </div>
                </div>
              </div>
            </div>
          </div>

          {/* Drawer Footer */}
          <div className="px-6 py-4 border-t border-[#dce2f7] bg-[#f9f9ff] flex items-center justify-end">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 bg-white border border-[#bcc9c6] text-[#141b2b] text-xs font-bold rounded-lg hover:bg-[#eef0fb] transition-colors cursor-pointer"
            >
              Close
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
