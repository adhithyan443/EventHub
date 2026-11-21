import { useState } from "react";

export default function RejectSettlementModal({
  settlement,
  actionLoading = false,
  onConfirm,
  onCancel,
}) {
  const [reason, setReason] = useState("");
  const [error, setError] = useState("");

  if (!settlement) return null;

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

  const amountStr = formatCurrency(settlement.amount, settlement.currency);

  const handleSubmit = (e) => {
    e.preventDefault();
    const trimmed = reason.trim();
    if (!trimmed) {
      setError("A rejection reason is required.");
      return;
    }
    setError("");
    onConfirm(trimmed);
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/40 backdrop-blur-[1px] transition-opacity"
        onClick={!actionLoading ? onCancel : undefined}
      />

      {/* Modal Card */}
      <div className="relative bg-white rounded-2xl shadow-2xl w-full max-w-[500px] overflow-hidden animate-in fade-in zoom-in-95 duration-150">
        <form onSubmit={handleSubmit} className="p-4 sm:p-6">
          {/* Header */}
          <div className="flex items-start gap-4 mb-5">
            <div className="bg-[#fef2f2] text-[#b91c1c] w-10 h-10 rounded-full flex items-center justify-center shrink-0 mt-0.5">
              <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
                <path
                  d="M15 5L5 15M5 5L15 15"
                  stroke="#b91c1c"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                />
              </svg>
            </div>
            <div>
              <h2 className="text-[#141b2b] text-xl font-bold leading-7">
                Reject Settlement Request?
              </h2>
              <p className="text-[#3d4947] text-sm mt-1 leading-5">
                You are about to reject settlement <span className="font-mono text-xs font-semibold">{settlement.id}</span> of{" "}
                <span className="font-semibold text-[#141b2b]">{amountStr}</span>.
              </p>
            </div>
          </div>

          {/* Details & Impact Notice */}
          <div className="bg-[#fffbeb] border border-[#fef3c7] rounded-xl p-3.5 mb-4 text-xs text-[#92400e] space-y-1.5">
            <div className="font-semibold text-[#b45309] flex items-center gap-1.5">
              <svg width="15" height="15" viewBox="0 0 20 20" fill="currentColor">
                <path
                  fillRule="evenodd"
                  d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
                  clipRule="evenodd"
                />
              </svg>
              Consequence of Rejection
            </div>
            <p className="text-[#78350f]">
              The in-flight settlement hold on the organizer&apos;s payable balance will be <strong>released</strong> back into their available balance immediately. No funds will be transferred.
            </p>
          </div>

          {/* Reason Input */}
          <div className="mb-5">
            <label className="block text-sm font-semibold text-[#141b2b] mb-1.5">
              Rejection Reason <span className="text-[#dc2626]">*</span>
            </label>
            <textarea
              value={reason}
              onChange={(e) => {
                setReason(e.target.value);
                if (error) setError("");
              }}
              placeholder="Explain why this settlement request is being rejected (e.g. invalid tax documents, pending dispute)..."
              rows={4}
              required
              disabled={actionLoading}
              className={`w-full border rounded-lg px-3 py-2.5 text-sm text-[#141b2b] placeholder:text-[#94a3b8] resize-none focus:outline-none focus:ring-2 focus:border-transparent transition-colors disabled:bg-gray-50 disabled:cursor-not-allowed ${
                error
                  ? "border-[#ef4444] focus:ring-[#ef4444]/30"
                  : "border-[#cbd5e1] focus:ring-[#00685f]/30 focus:border-[#00685f]"
              }`}
            />
            {error && <p className="text-xs text-[#dc2626] mt-1 font-medium">{error}</p>}
          </div>

          {/* Actions */}
          <div className="flex flex-col-reverse sm:flex-row items-center justify-end gap-2 sm:gap-3">
            <button
              type="button"
              onClick={onCancel}
              disabled={actionLoading}
              className="w-full sm:w-auto border border-[#bcc9c6] bg-white text-[#141b2b] text-sm font-medium px-5 py-2.5 sm:py-2 rounded-lg hover:bg-[#f9f9ff] transition-colors cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={!reason.trim() || actionLoading}
              className="w-full sm:w-auto bg-[#dc2626] text-white text-sm font-semibold px-5 py-2.5 sm:py-2 rounded-lg hover:bg-[#b91c1c] transition-colors disabled:opacity-40 disabled:cursor-not-allowed flex items-center justify-center gap-2 cursor-pointer shadow-sm"
            >
              {actionLoading ? (
                <>
                  <svg className="animate-spin h-4 w-4 text-white" viewBox="0 0 24 24" fill="none">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                  </svg>
                  Rejecting...
                </>
              ) : (
                "Confirm Rejection"
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
