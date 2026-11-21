export default function RetrySettlementModal({
  settlement,
  actionLoading = false,
  onConfirm,
  onCancel,
}) {
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

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/40 backdrop-blur-[1px] transition-opacity"
        onClick={!actionLoading ? onCancel : undefined}
      />

      {/* Modal Card */}
      <div className="relative bg-white rounded-2xl shadow-2xl w-full max-w-[500px] overflow-hidden animate-in fade-in zoom-in-95 duration-150">
        <div className="p-4 sm:p-6">
          {/* Header */}
          <div className="flex items-start gap-4 mb-5">
            <div className="bg-[#eff6ff] text-[#2563eb] w-10 h-10 rounded-full flex items-center justify-center shrink-0 mt-0.5">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <path d="M21.5 2v6h-6M21.34 15.57a10 10 0 1 1-.57-8.38l5.67-5.67" />
              </svg>
            </div>
            <div>
              <h2 className="text-[#141b2b] text-xl font-bold leading-7">
                Retry Failed Settlement?
              </h2>
              <p className="text-[#3d4947] text-sm mt-1 leading-5">
                Retry payout processing of <span className="font-semibold text-[#141b2b]">{amountStr}</span> for settlement{" "}
                <span className="font-mono text-xs font-semibold">{settlement.id}</span>.
              </p>
            </div>
          </div>

          {/* Failure notice */}
          {settlement.failure_reason && (
            <div className="bg-[#fef2f2] border border-[#fecaca] rounded-xl p-3.5 mb-4 text-xs text-[#991b1b] space-y-1">
              <span className="font-semibold text-[#b91c1c] block">Previous Failure Reason:</span>
              <p className="text-[#7f1d1d] font-mono text-[11px] break-all">{settlement.failure_reason}</p>
            </div>
          )}

          {/* Process explanation */}
          <div className="bg-[#f0f9ff] border border-[#bae6fd] rounded-xl p-3.5 mb-5 text-xs text-[#0369a1] space-y-1.5">
            <div className="font-semibold text-[#0284c7] flex items-center gap-1.5">
              <svg width="15" height="15" viewBox="0 0 20 20" fill="currentColor">
                <path
                  fillRule="evenodd"
                  d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z"
                  clipRule="evenodd"
                />
              </svg>
              What happens during retry
            </div>
            <ul className="list-disc list-inside space-y-1 text-[#0369a1] pl-1">
              <li>Re-initiates payment dispatch with the payout provider.</li>
              <li>Validates that balance holds remain consistent.</li>
              <li>Upon success, completes journal entry and transitions settlement to <strong>COMPLETED</strong>.</li>
            </ul>
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
              type="button"
              onClick={onConfirm}
              disabled={actionLoading}
              className="w-full sm:w-auto bg-[#2563eb] text-white text-sm font-semibold px-5 py-2.5 sm:py-2 rounded-lg hover:bg-[#1d4ed8] transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2 cursor-pointer shadow-sm"
            >
              {actionLoading ? (
                <>
                  <svg className="animate-spin h-4 w-4 text-white" viewBox="0 0 24 24" fill="none">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                  </svg>
                  Retrying Payout...
                </>
              ) : (
                "Retry Payout"
              )}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
