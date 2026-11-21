import { useState, useEffect } from "react";
import { formatCurrency, formatDateTime } from "../../utils/formatters.js";
import notify from "../../utils/toast.js";

const settlementStatusBadgeStyles = {
  PENDING: "bg-[#fef9c3] text-[#854d0e] border border-[#fef08a]",
  APPROVED: "bg-[#e0f2fe] text-[#0369a1] border border-[#bae6fd]",
  PROCESSING: "bg-[#ede9fe] text-[#6d28d9] border border-[#ddd6fe]",
  COMPLETED: "bg-[#dcfce7] text-[#15803d] border border-[#bbf7d0]",
  REJECTED: "bg-[#fee2e2] text-[#b91c1c] border border-[#fecaca]",
  FAILED: "bg-[#ffedd5] text-[#c2410c] border border-[#fed7aa]",
};

export default function OrganizerSettlementDetailDrawer({ settlement, onClose }) {
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

  const handleCopy = (text, fieldName) => {
    if (!text) return;
    navigator.clipboard.writeText(text);
    setCopiedField(fieldName);
    notify.info(`Copied ${fieldName} to clipboard`);
    setTimeout(() => setCopiedField(null), 2000);
  };

  const status = (settlement.status || "PENDING").toUpperCase();
  const badgeClass = settlementStatusBadgeStyles[status] || "bg-[#f3f4f6] text-[#4b5563] border border-[#e5e7eb]";

  return (
    <div className="fixed inset-0 z-50 overflow-hidden" role="dialog" aria-modal="true">
      {/* Backdrop */}
      <div
        className="fixed inset-0 bg-black/40 backdrop-blur-xs transition-opacity duration-300"
        onClick={onClose}
      />

      <div className="fixed inset-y-0 right-0 max-w-full flex pl-10">
        <div className="w-screen max-w-lg bg-white shadow-2xl flex flex-col">
          {/* Header */}
          <div className="px-6 py-5 border-b border-[#e5e7eb] flex items-center justify-between bg-[#fbfcfd]">
            <div>
              <span className="text-xs font-semibold text-[#00685F] uppercase tracking-wider">
                Settlement Details
              </span>
              <h2 className="text-lg font-bold text-[#141b2b] mt-0.5">
                {formatCurrency(settlement.amount, settlement.currency)}
              </h2>
            </div>
            <button
              type="button"
              onClick={onClose}
              className="text-[#6b7280] hover:text-[#111827] p-2 rounded-lg hover:bg-gray-100 transition-colors cursor-pointer"
              aria-label="Close panel"
            >
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>

          {/* Body Content */}
          <div className="flex-1 overflow-y-auto p-6 space-y-6">
            {/* Status & Amount Card */}
            <div className="bg-[#f8fafc] border border-[#e2e8f0] rounded-xl p-5 space-y-3">
              <div className="flex items-center justify-between">
                <span className="text-sm text-[#64748b] font-medium">Lifecycle Status</span>
                <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-semibold ${badgeClass}`}>
                  {status}
                </span>
              </div>
              <div className="flex items-center justify-between pt-2 border-t border-[#e2e8f0]">
                <span className="text-sm text-[#64748b] font-medium">Disbursement Amount</span>
                <span className="text-base font-bold text-[#1e293b]">
                  {formatCurrency(settlement.amount, settlement.currency)}
                </span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-sm text-[#64748b] font-medium">Currency</span>
                <span className="text-sm font-semibold text-[#1e293b]">{settlement.currency || "INR"}</span>
              </div>
            </div>

            {/* Failure or Rejection Alert Box */}
            {status === "FAILED" && settlement.failure_reason && (
              <div className="bg-[#fff1f2] border border-[#fecdd3] rounded-xl p-4 text-[#9f1239]">
                <div className="flex items-start gap-2.5">
                  <svg className="w-5 h-5 shrink-0 text-[#e11d48] mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                  </svg>
                  <div>
                    <h4 className="text-sm font-bold">Disbursement Failed</h4>
                    <p className="text-xs mt-1 leading-relaxed">{settlement.failure_reason}</p>
                    <p className="text-xs mt-2 text-[#be123c]">
                      EventHub administrators have been notified and can retry the transfer once banking connectivity is restored.
                    </p>
                  </div>
                </div>
              </div>
            )}

            {status === "REJECTED" && settlement.rejection_reason && (
              <div className="bg-[#fff7ed] border border-[#fed7aa] rounded-xl p-4 text-[#9a3412]">
                <div className="flex items-start gap-2.5">
                  <svg className="w-5 h-5 shrink-0 text-[#ea580c] mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                  <div>
                    <h4 className="text-sm font-bold">Settlement Rejected by Admin</h4>
                    <p className="text-xs mt-1 leading-relaxed">{settlement.rejection_reason}</p>
                    <p className="text-xs mt-2 text-[#c2410c]">
                      The reserved hold has been released back into your available balance.
                    </p>
                  </div>
                </div>
              </div>
            )}

            {/* Scope & Identification */}
            <div className="space-y-4">
              <h3 className="text-xs font-semibold uppercase tracking-wider text-[#94a3b8]">
                Settlement Identifiers
              </h3>

              <div className="space-y-3">
                <div className="flex items-center justify-between text-sm py-1 border-b border-[#f1f5f9]">
                  <span className="text-[#64748b]">Settlement ID</span>
                  <div className="flex items-center gap-1.5 font-mono text-xs text-[#334155]">
                    <span>{settlement.id}</span>
                    <button
                      type="button"
                      onClick={() => handleCopy(settlement.id, "Settlement ID")}
                      className="p-1 hover:text-[#00685F] cursor-pointer inline-flex items-center gap-1"
                      title="Copy Settlement ID"
                    >
                      {copiedField === "Settlement ID" ? (
                        <span className="text-[11px] text-[#00685F] font-sans font-medium">Copied</span>
                      ) : (
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                          <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
                          <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
                        </svg>
                      )}
                    </button>
                  </div>
                </div>

                <div className="flex items-center justify-between text-sm py-1 border-b border-[#f1f5f9]">
                  <span className="text-[#64748b]">Scope</span>
                  <span className="text-xs font-medium text-[#1e293b]">
                    {settlement.event_id ? `Event: ${settlement.event_id}` : "General (All Completed Events)"}
                  </span>
                </div>

                {settlement.payout_reference && (
                  <div className="flex items-center justify-between text-sm py-1 border-b border-[#f1f5f9]">
                    <span className="text-[#64748b]">Payout Reference</span>
                    <div className="flex items-center gap-1.5 font-mono text-xs text-[#0f766e] font-semibold">
                      <span>{settlement.payout_reference}</span>
                      <button
                        type="button"
                        onClick={() => handleCopy(settlement.payout_reference, "Payout Reference")}
                        className="p-1 hover:text-[#00685F] cursor-pointer"
                        title="Copy Payout Reference"
                      >
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                          <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
                          <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
                        </svg>
                      </button>
                    </div>
                  </div>
                )}
              </div>
            </div>

            {/* Audit Timestamps */}
            <div className="space-y-4">
              <h3 className="text-xs font-semibold uppercase tracking-wider text-[#94a3b8]">
                Audit Timeline
              </h3>

              <div className="space-y-3">
                <div className="flex items-center justify-between text-sm py-1 border-b border-[#f1f5f9]">
                  <span className="text-[#64748b]">Requested Date</span>
                  <span className="text-xs text-[#334155]">{formatDateTime(settlement.created_at)}</span>
                </div>

                {settlement.approved_at && (
                  <div className="flex items-center justify-between text-sm py-1 border-b border-[#f1f5f9]">
                    <span className="text-[#64748b]">Approved Date</span>
                    <span className="text-xs text-[#334155]">{formatDateTime(settlement.approved_at)}</span>
                  </div>
                )}

                {settlement.rejected_at && (
                  <div className="flex items-center justify-between text-sm py-1 border-b border-[#f1f5f9]">
                    <span className="text-[#64748b]">Rejected Date</span>
                    <span className="text-xs text-[#9f1239]">{formatDateTime(settlement.rejected_at)}</span>
                  </div>
                )}

                {settlement.processed_at && (
                  <div className="flex items-center justify-between text-sm py-1 border-b border-[#f1f5f9]">
                    <span className="text-[#64748b]">Completed Date</span>
                    <span className="text-xs text-[#15803d] font-semibold">{formatDateTime(settlement.processed_at)}</span>
                  </div>
                )}
              </div>
            </div>
          </div>

          {/* Footer */}
          <div className="p-6 border-t border-[#e5e7eb] bg-[#fbfcfd]">
            <button
              type="button"
              onClick={onClose}
              className="w-full py-2.5 px-4 bg-white border border-[#cbd5e1] text-[#334155] hover:bg-[#f1f5f9] rounded-xl text-sm font-semibold transition-colors cursor-pointer"
            >
              Close
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
