import { useState, useEffect } from "react";
import notify from "../../utils/toast";

const statusBadgeStyles = {
  PENDING: "bg-[#fef9c3] text-[#854d0e] border border-[#fef08a]",
  APPROVED: "bg-[#eff6ff] text-[#1d4ed8] border border-[#bfdbfe]",
  PROCESSING: "bg-[#e0e7ff] text-[#4338ca] border border-[#c7d2fe]",
  COMPLETED: "bg-[#dcfce7] text-[#15803d] border border-[#bbf7d0]",
  REJECTED: "bg-[#fee2e2] text-[#b91c1c] border border-[#fecaca]",
  FAILED: "bg-[#fff1f2] text-[#be123c] border border-[#fecdd3]",
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

export default function AdminSettlementDetailDrawer({
  settlement,
  onClose,
  onApprove,
  onReject,
  onRetry,
}) {
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

  if (!settlement) return null;

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

  const statusKey = settlement.status?.toUpperCase() || "PENDING";
  const badgeClass =
    statusBadgeStyles[statusKey] || "bg-[#f3f4f6] text-[#4b5563] border border-[#e5e7eb]";

  const isPending = statusKey === "PENDING";
  const isFailed = statusKey === "FAILED";

  return (
    <div className="fixed inset-0 z-50 overflow-hidden">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/40 backdrop-blur-xs transition-opacity"
        onClick={onClose}
      />

      <div className="fixed inset-y-0 right-0 max-w-full flex pl-10">
        <div className="w-screen max-w-md bg-white shadow-2xl flex flex-col h-full animate-in slide-in-from-right duration-200">
          {/* Header */}
          <div className="p-5 border-b border-[#e2e8f0] flex items-center justify-between bg-white shrink-0">
            <div>
              <span className="text-[11px] font-semibold text-[#00685f] tracking-wider uppercase">
                Settlement Details
              </span>
              <div className="flex items-center gap-2 mt-0.5">
                <h2 className="text-lg font-bold text-[#141b2b] font-mono">
                  {settlement.id ? `${settlement.id.slice(0, 8)}...${settlement.id.slice(-4)}` : "—"}
                </h2>
                {settlement.id && (
                  <button
                    type="button"
                    onClick={() => copyToClipboard(settlement.id, "Settlement ID")}
                    title="Copy full settlement ID"
                    className="text-[#64748b] hover:text-[#00685f] p-1 rounded transition-colors"
                  >
                    {copiedField === "Settlement ID" ? (
                      <span className="text-xs text-[#00685f] font-sans font-medium">Copied!</span>
                    ) : (
                      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                        <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
                        <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
                      </svg>
                    )}
                  </button>
                )}
              </div>
            </div>
            <button
              type="button"
              onClick={onClose}
              className="text-[#64748b] hover:text-[#141b2b] p-2 rounded-lg hover:bg-[#f1f5f9] transition-colors"
            >
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <line x1="18" y1="6" x2="6" y2="18" />
                <line x1="6" y1="6" x2="18" y2="18" />
              </svg>
            </button>
          </div>

          {/* Body Content */}
          <div className="flex-1 overflow-y-auto p-5 space-y-5 text-sm text-[#334155]">
            {/* Status & Amount Card */}
            <div className="bg-[#f8fafc] border border-[#e2e8f0] rounded-xl p-4 flex items-center justify-between">
              <div>
                <span className="text-xs text-[#64748b] block font-medium">Settlement Amount</span>
                <span className="text-2xl font-bold text-[#141b2b]">
                  {formatCurrency(settlement.amount, settlement.currency)}
                </span>
                <span className="text-xs text-[#94a3b8] block mt-0.5">Currency: {settlement.currency || "INR"}</span>
              </div>
              <span className={`inline-flex items-center px-2.5 py-1 rounded-full text-xs font-semibold ${badgeClass}`}>
                {statusKey}
              </span>
            </div>

            {/* Scope & Organizer Section */}
            <div className="space-y-3">
              <h3 className="text-xs font-semibold uppercase tracking-wider text-[#64748b]">Organizer & Scope</h3>
              <div className="bg-white border border-[#e2e8f0] rounded-xl p-3.5 space-y-2.5 text-xs">
                <div className="flex justify-between items-center">
                  <span className="text-[#64748b]">Organizer UUID</span>
                  <div className="flex items-center gap-1.5 font-mono">
                    <span className="font-medium text-[#1e293b]">
                      {settlement.organizer_id ? `${settlement.organizer_id.slice(0, 8)}...${settlement.organizer_id.slice(-4)}` : "—"}
                    </span>
                    {settlement.organizer_id && (
                      <button
                        type="button"
                        onClick={() => copyToClipboard(settlement.organizer_id, "Organizer ID")}
                        className="text-[#64748b] hover:text-[#00685f]"
                        title="Copy Organizer UUID"
                      >
                        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                          <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
                          <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
                        </svg>
                      </button>
                    )}
                  </div>
                </div>

                <div className="flex justify-between items-center">
                  <span className="text-[#64748b]">Settlement Scope</span>
                  <span className="font-medium text-[#1e293b]">
                    {settlement.event_id ? "Event-Specific" : "General Account Balance"}
                  </span>
                </div>

                {settlement.event_id && (
                  <div className="flex justify-between items-center">
                    <span className="text-[#64748b]">Event ID</span>
                    <div className="flex items-center gap-1.5 font-mono">
                      <span className="font-medium text-[#1e293b]">
                        {`${settlement.event_id.slice(0, 8)}...${settlement.event_id.slice(-4)}`}
                      </span>
                      <button
                        type="button"
                        onClick={() => copyToClipboard(settlement.event_id, "Event ID")}
                        className="text-[#64748b] hover:text-[#00685f]"
                        title="Copy Event ID"
                      >
                        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                          <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
                          <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
                        </svg>
                      </button>
                    </div>
                  </div>
                )}
              </div>
            </div>

            {/* Operational & Failure Details */}
            {(settlement.payout_reference || settlement.failure_reason || settlement.rejection_reason) && (
              <div className="space-y-3">
                <h3 className="text-xs font-semibold uppercase tracking-wider text-[#64748b]">Disbursement Audit</h3>
                <div className="bg-white border border-[#e2e8f0] rounded-xl p-3.5 space-y-2.5 text-xs">
                  {settlement.payout_reference && (
                    <div className="flex justify-between items-center">
                      <span className="text-[#64748b]">Payout Reference</span>
                      <div className="flex items-center gap-1.5 font-mono font-medium text-[#0f766e]">
                        <span>{settlement.payout_reference}</span>
                        <button
                          type="button"
                          onClick={() => copyToClipboard(settlement.payout_reference, "Payout Reference")}
                          className="text-[#64748b] hover:text-[#00685f]"
                        >
                          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                            <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
                            <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
                          </svg>
                        </button>
                      </div>
                    </div>
                  )}

                  {settlement.rejection_reason && (
                    <div className="pt-1">
                      <span className="text-[#dc2626] font-semibold block mb-1">Rejection Reason:</span>
                      <div className="p-2.5 bg-[#fef2f2] text-[#991b1b] rounded-lg font-mono text-[11px] leading-relaxed">
                        {settlement.rejection_reason}
                      </div>
                    </div>
                  )}

                  {settlement.failure_reason && (
                    <div className="pt-1">
                      <span className="text-[#b91c1c] font-semibold block mb-1">Payout Failure Reason:</span>
                      <div className="p-2.5 bg-[#fff1f2] text-[#9f1239] rounded-lg font-mono text-[11px] leading-relaxed break-all">
                        {settlement.failure_reason}
                      </div>
                    </div>
                  )}
                </div>
              </div>
            )}

            {/* Lifecycle Timestamps */}
            <div className="space-y-3">
              <h3 className="text-xs font-semibold uppercase tracking-wider text-[#64748b]">Lifecycle Timeline</h3>
              <div className="bg-white border border-[#e2e8f0] rounded-xl p-3.5 space-y-2 text-xs">
                <div className="flex justify-between">
                  <span className="text-[#64748b]">Requested At</span>
                  <span className="font-medium text-[#1e293b]">{formatDateTime(settlement.created_at)}</span>
                </div>
                {settlement.approved_at && (
                  <div className="flex justify-between">
                    <span className="text-[#64748b]">Approved At</span>
                    <span className="font-medium text-[#1e293b]">{formatDateTime(settlement.approved_at)}</span>
                  </div>
                )}
                {settlement.processed_at && (
                  <div className="flex justify-between">
                    <span className="text-[#64748b]">Processed At</span>
                    <span className="font-medium text-[#1e293b]">{formatDateTime(settlement.processed_at)}</span>
                  </div>
                )}
                {settlement.rejected_at && (
                  <div className="flex justify-between">
                    <span className="text-[#64748b]">Rejected At</span>
                    <span className="font-medium text-[#1e293b]">{formatDateTime(settlement.rejected_at)}</span>
                  </div>
                )}
                <div className="flex justify-between">
                  <span className="text-[#64748b]">Last Updated</span>
                  <span className="font-medium text-[#1e293b]">{formatDateTime(settlement.updated_at)}</span>
                </div>
              </div>
            </div>
          </div>

          {/* Drawer Actions Footer */}
          {(isPending || isFailed) && (
            <div className="p-4 border-t border-[#e2e8f0] bg-[#f8fafc] shrink-0 flex items-center justify-end gap-2.5">
              {isPending && (
                <>
                  <button
                    type="button"
                    onClick={() => onReject(settlement)}
                    className="border border-[#fca5a5] text-[#b91c1c] hover:bg-[#fef2f2] px-4 py-2 rounded-lg text-sm font-semibold transition-colors cursor-pointer"
                  >
                    Reject Request
                  </button>
                  <button
                    type="button"
                    onClick={() => onApprove(settlement)}
                    className="bg-[#00685f] hover:bg-[#005a52] text-white px-4 py-2 rounded-lg text-sm font-semibold transition-colors cursor-pointer shadow-sm"
                  >
                    Approve & Payout
                  </button>
                </>
              )}
              {isFailed && (
                <button
                  type="button"
                  onClick={() => onRetry(settlement)}
                  className="bg-[#2563eb] hover:bg-[#1d4ed8] text-white px-4 py-2 rounded-lg text-sm font-semibold transition-colors cursor-pointer shadow-sm flex items-center gap-2"
                >
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                    <path d="M21.5 2v6h-6M21.34 15.57a10 10 0 1 1-.57-8.38l5.67-5.67" />
                  </svg>
                  Retry Payout
                </button>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
