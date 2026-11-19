import { useState, useMemo } from "react";
import { formatCurrency } from "../../utils/formatters.js";
import { requestOrganizerSettlement } from "../../api/organizerSettlementApi.js";
import notify from "../../utils/toast.js";

export default function RequestSettlementModal({
  balanceProjection,
  onClose,
  onSuccess,
}) {
  const [targetScope, setTargetScope] = useState("GENERAL");
  const [selectedEventId, setSelectedEventId] = useState("");
  const [amountStr, setAmountStr] = useState("");
  const [isConfirming, setIsConfirming] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [errorMessage, setErrorMessage] = useState("");

  const eligibleEvents = useMemo(() => {
    if (!balanceProjection?.event_balances) return [];
    return balanceProjection.event_balances.filter(
      (eb) => eb.is_eligible && parseFloat(eb.available_balance || 0) > 0
    );
  }, [balanceProjection]);

  // Determine current active available balance based on selected scope
  const activeAvailable = useMemo(() => {
    if (targetScope === "EVENT" && selectedEventId) {
      const found = eligibleEvents.find((e) => e.event_id === selectedEventId);
      return found ? parseFloat(found.available_balance || 0) : 0;
    }
    return parseFloat(balanceProjection?.available_balance || 0);
  }, [targetScope, selectedEventId, eligibleEvents, balanceProjection]);

  const numAmount = parseFloat(amountStr) || 0;
  const isAmountValid = numAmount > 0 && numAmount <= activeAvailable;

  const handleMaxClick = () => {
    if (activeAvailable > 0) {
      setAmountStr(activeAvailable.toFixed(2));
      setErrorMessage("");
    }
  };

  const handleProceedToConfirm = (e) => {
    e.preventDefault();
    setErrorMessage("");

    if (numAmount <= 0) {
      setErrorMessage("Please enter an amount greater than ₹0.00.");
      return;
    }
    if (numAmount > activeAvailable) {
      setErrorMessage(
        `Amount cannot exceed your available balance of ${formatCurrency(activeAvailable)}.`
      );
      return;
    }

    setIsConfirming(true);
  };

  const handleFinalSubmit = async () => {
    if (isSubmitting) return;
    setIsSubmitting(true);
    setErrorMessage("");

    try {
      const payload = {
        amount: numAmount,
      };
      if (targetScope === "EVENT" && selectedEventId) {
        payload.event_id = selectedEventId;
      }

      const res = await requestOrganizerSettlement(payload);
      notify.success("Settlement request submitted successfully.");
      if (onSuccess) {
        onSuccess(res?.data);
      }
      onClose();
    } catch (err) {
      const apiMsg =
        err?.response?.data?.error ||
        err?.response?.data?.message ||
        err?.message ||
        "Failed to submit settlement request.";
      setErrorMessage(apiMsg);
      setIsConfirming(false); // return to edit form so organizer can adjust
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/40 backdrop-blur-xs transition-opacity"
        onClick={!isSubmitting ? onClose : undefined}
      />

      {/* Modal Card */}
      <div className="relative bg-white rounded-2xl shadow-2xl w-full max-w-lg overflow-hidden animate-in fade-in zoom-in-95 duration-150">
        {/* Header */}
        <div className="px-6 py-5 border-b border-[#e5e7eb] flex items-center justify-between bg-[#fbfcfd]">
          <div>
            <span className="text-xs font-semibold text-[#00685F] uppercase tracking-wider">
              Organizer Payout
            </span>
            <h2 className="text-lg font-bold text-[#141b2b]">
              {isConfirming ? "Confirm Settlement Request" : "Request Settlement"}
            </h2>
          </div>
          {!isSubmitting && (
            <button
              type="button"
              onClick={onClose}
              className="text-[#6b7280] hover:text-[#111827] p-2 rounded-lg hover:bg-gray-100 transition-colors cursor-pointer"
            >
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          )}
        </div>

        {/* Content Body */}
        <div className="p-6 space-y-5">
          {errorMessage && (
            <div className="bg-[#fff1f2] border border-[#fecdd3] text-[#9f1239] text-xs font-medium p-3.5 rounded-xl flex items-start gap-2">
              <svg className="w-4 h-4 shrink-0 text-[#e11d48] mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
              </svg>
              <span>{errorMessage}</span>
            </div>
          )}

          {!isConfirming ? (
            <form onSubmit={handleProceedToConfirm} className="space-y-4">
              {/* Balance Summary Box */}
              <div className="bg-[#f0fdf4] border border-[#bbf7d0] rounded-xl p-4 flex items-center justify-between">
                <div>
                  <span className="text-xs text-[#166534] font-medium block">
                    {targetScope === "EVENT" && selectedEventId
                      ? "Selected Event Available Balance"
                      : "Total Available Balance"}
                  </span>
                  <span className="text-xl font-bold text-[#14532d]">
                    {formatCurrency(activeAvailable)}
                  </span>
                </div>
                <button
                  type="button"
                  onClick={handleMaxClick}
                  disabled={activeAvailable <= 0}
                  className="px-3 py-1.5 bg-[#dcfce7] hover:bg-[#bbf7d0] text-[#166534] text-xs font-bold rounded-lg border border-[#86efac] transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  Use Max
                </button>
              </div>

              {/* Settlement Scope Selection */}
              <div>
                <label className="block text-xs font-semibold text-[#475569] uppercase tracking-wider mb-1.5">
                  Disbursement Scope
                </label>
                <div className="grid grid-cols-2 gap-2">
                  <button
                    type="button"
                    onClick={() => {
                      setTargetScope("GENERAL");
                      setSelectedEventId("");
                      setErrorMessage("");
                    }}
                    className={`py-2 px-3 text-xs font-semibold rounded-lg border text-center transition-all cursor-pointer ${
                      targetScope === "GENERAL"
                        ? "border-[#00685F] bg-[#f0fdfa] text-[#00685F] shadow-xs"
                        : "border-[#e2e8f0] bg-white text-[#64748b] hover:bg-[#f8fafc]"
                    }`}
                  >
                    General Balance
                  </button>
                  <button
                    type="button"
                    onClick={() => {
                      setTargetScope("EVENT");
                      if (eligibleEvents.length > 0 && !selectedEventId) {
                        setSelectedEventId(eligibleEvents[0].event_id);
                      }
                      setErrorMessage("");
                    }}
                    disabled={eligibleEvents.length === 0}
                    className={`py-2 px-3 text-xs font-semibold rounded-lg border text-center transition-all cursor-pointer ${
                      targetScope === "EVENT"
                        ? "border-[#00685F] bg-[#f0fdfa] text-[#00685F] shadow-xs"
                        : "border-[#e2e8f0] bg-white text-[#64748b] hover:bg-[#f8fafc] disabled:opacity-50 disabled:cursor-not-allowed"
                    }`}
                  >
                    Specific Event {eligibleEvents.length > 0 ? `(${eligibleEvents.length})` : "(0)"}
                  </button>
                </div>
              </div>

              {/* Event Picker (if EVENT scope selected) */}
              {targetScope === "EVENT" && (
                <div>
                  <label htmlFor="settlement-event-select" className="block text-xs font-semibold text-[#475569] mb-1">
                    Select Event
                  </label>
                  <select
                    id="settlement-event-select"
                    value={selectedEventId}
                    onChange={(e) => {
                      setSelectedEventId(e.target.value);
                      setErrorMessage("");
                    }}
                    className="w-full text-xs py-2 px-3 bg-white border border-[#cbd5e1] rounded-lg text-[#1e293b] focus:outline-hidden focus:border-[#00685F] focus:ring-1 focus:ring-[#00685F]"
                  >
                    {eligibleEvents.map((evt) => (
                      <option key={evt.event_id} value={evt.event_id}>
                        {evt.event_title} — Avail: {formatCurrency(evt.available_balance)}
                      </option>
                    ))}
                  </select>
                </div>
              )}

              {/* Amount Input */}
              <div>
                <label htmlFor="settlement-amount-input" className="block text-xs font-semibold text-[#475569] uppercase tracking-wider mb-1.5">
                  Amount to Withdraw (INR)
                </label>
                <div className="relative">
                  <span className="absolute left-3.5 top-1/2 -translate-y-1/2 text-sm font-semibold text-[#94a3b8]">
                    ₹
                  </span>
                  <input
                    id="settlement-amount-input"
                    type="number"
                    step="0.01"
                    min="1"
                    max={activeAvailable}
                    value={amountStr}
                    onChange={(e) => {
                      setAmountStr(e.target.value);
                      setErrorMessage("");
                    }}
                    placeholder="0.00"
                    className="w-full pl-8 pr-4 py-2.5 bg-white border border-[#cbd5e1] rounded-xl text-sm font-semibold text-[#1e293b] placeholder-[#cbd5e1] focus:outline-hidden focus:border-[#00685F] focus:ring-2 focus:ring-[#00685F]/20"
                  />
                </div>
                <p className="text-[11px] text-[#64748b] mt-1.5">
                  Minimum ₹1.00. Funds will be reviewed by admin and disbursed to your registered bank account.
                </p>
              </div>

              {/* Form Action */}
              <div className="pt-2 flex items-center gap-3">
                <button
                  type="button"
                  onClick={onClose}
                  className="flex-1 py-2.5 px-4 bg-white border border-[#cbd5e1] text-[#475569] hover:bg-[#f8fafc] text-xs font-semibold rounded-xl transition-colors cursor-pointer"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={!isAmountValid}
                  className="flex-1 py-2.5 px-4 bg-[#00685F] hover:bg-[#00524B] text-white text-xs font-bold rounded-xl shadow-xs transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  Review Request
                </button>
              </div>
            </form>
          ) : (
            /* Confirmation Step */
            <div className="space-y-4">
              <div className="bg-[#f8fafc] border border-[#e2e8f0] rounded-xl p-4 space-y-2.5">
                <div className="flex justify-between text-xs">
                  <span className="text-[#64748b]">Disbursement Amount</span>
                  <span className="font-bold text-[#0f172a] text-sm">{formatCurrency(numAmount)}</span>
                </div>
                <div className="flex justify-between text-xs pt-2 border-t border-[#f1f5f9]">
                  <span className="text-[#64748b]">Scope</span>
                  <span className="font-medium text-[#1e293b]">
                    {targetScope === "EVENT" && selectedEventId
                      ? `Event Specific (${eligibleEvents.find((e) => e.event_id === selectedEventId)?.event_title || selectedEventId})`
                      : "General Available Balance"}
                  </span>
                </div>
                <div className="flex justify-between text-xs pt-2 border-t border-[#f1f5f9]">
                  <span className="text-[#64748b]">Remaining Balance After Hold</span>
                  <span className="font-semibold text-[#00685F]">
                    {formatCurrency(activeAvailable - numAmount)}
                  </span>
                </div>
              </div>

              <div className="bg-[#fffbeb] border border-[#fef3c7] rounded-xl p-3 text-xs text-[#92400e] leading-relaxed">
                <strong>Important:</strong> Upon submitting, {formatCurrency(numAmount)} will be placed in an
                <strong> In-Flight Settlement Hold</strong> to reserve your funds until administrative approval and bank transfer execution.
              </div>

              {/* Confirm Actions */}
              <div className="flex items-center gap-3 pt-2">
                <button
                  type="button"
                  onClick={() => setIsConfirming(false)}
                  disabled={isSubmitting}
                  className="flex-1 py-2.5 px-4 bg-white border border-[#cbd5e1] text-[#475569] hover:bg-[#f8fafc] text-xs font-semibold rounded-xl transition-colors cursor-pointer disabled:opacity-50"
                >
                  Back
                </button>
                <button
                  type="button"
                  onClick={handleFinalSubmit}
                  disabled={isSubmitting}
                  className="flex-1 py-2.5 px-4 bg-[#00685F] hover:bg-[#00524B] text-white text-xs font-bold rounded-xl shadow-xs transition-colors cursor-pointer flex items-center justify-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {isSubmitting ? (
                    <>
                      <svg className="animate-spin h-3.5 w-3.5 text-white" fill="none" viewBox="0 0 24 24">
                        <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                        <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z" />
                      </svg>
                      Submitting...
                    </>
                  ) : (
                    "Confirm & Request"
                  )}
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
