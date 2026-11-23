import { useState, useEffect } from "react";
import StatusBadge from "./StatusBadge";
import { formatCurrency } from "../../../utils/formatters";
import {
  getOrganizerEventCancellationPreview,
  cancelOrganizerEvent,
} from "../../../api/organizerApi";
import notify from "../../../utils/toast";

export default function CancelEventModal({
  isOpen,
  onClose,
  eventId,
  eventTitle = "",
  onSuccess,
}) {
  const [loadingPreview, setLoadingPreview] = useState(false);
  const [previewError, setPreviewError] = useState("");
  const [previewData, setPreviewData] = useState(null);

  const [reason, setReason] = useState("");
  const [hasConfirmed, setHasConfirmed] = useState(false);
  const [isCancelling, setIsCancelling] = useState(false);
  const [cancelError, setCancelError] = useState("");

  const handleClose = () => {
    if (isCancelling) return;
    setPreviewData(null);
    setPreviewError("");
    setReason("");
    setHasConfirmed(false);
    setCancelError("");
    onClose();
  };

  useEffect(() => {
    if (!isOpen || !eventId) return;

    let isMounted = true;
    const timer = setTimeout(() => {
      setLoadingPreview(true);
      setPreviewError("");
      setCancelError("");

      getOrganizerEventCancellationPreview(eventId)
        .then((response) => {
          if (!isMounted) return;
          const data = response?.data || response;
          setPreviewData(data);
        })
        .catch((err) => {
          if (!isMounted) return;
          console.error("Failed to fetch event cancellation preview:", err);
          const errMsg =
            err?.response?.data?.message ||
            err?.response?.data?.error ||
            err?.message ||
            "Unable to load cancellation preview. Please try again.";
          setPreviewError(errMsg);
        })
        .finally(() => {
          if (isMounted) {
            setLoadingPreview(false);
          }
        });
    }, 0);

    return () => {
      isMounted = false;
      clearTimeout(timer);
    };
  }, [isOpen, eventId]);

  if (!isOpen) return null;

  const handleConfirmCancellation = async () => {
    if (!eventId || isCancelling) return;

    const trimmedReason = reason.trim();
    if (!trimmedReason) {
      setCancelError("A reason for event cancellation is required.");
      return;
    }

    if (trimmedReason.length > 500) {
      setCancelError("Cancellation reason cannot exceed 500 characters.");
      return;
    }

    if (!hasConfirmed) {
      setCancelError("Please check the confirmation box to proceed.");
      return;
    }

    try {
      setIsCancelling(true);
      setCancelError("");

      const response = await cancelOrganizerEvent(eventId, trimmedReason);
      const summary = response?.data || response;

      notify.success(
        response?.message ||
          `Event "${eventTitle || "Event"}" was cancelled successfully.`
      );

      if (typeof onSuccess === "function") {
        onSuccess(summary);
      }

      handleClose();
    } catch (err) {
      console.error("Failed to cancel organizer event:", err);
      const errMsg =
        err?.response?.data?.message ||
        err?.response?.data?.error ||
        err?.message ||
        "Failed to cancel event. Please try again.";
      setCancelError(errMsg);
      notify.error(errMsg);
    } finally {
      setIsCancelling(false);
    }
  };

  const isCancellable = Boolean(previewData?.eligibility?.is_cancellable);
  const blockingReasons = Array.isArray(previewData?.eligibility?.blocking_reasons)
    ? previewData.eligibility.blocking_reasons
    : [];

  const settlementWarning =
    previewData?.settlement_impact?.settlement_warning ||
    (previewData?.settlement_impact?.pending_settlements_count > 0
      ? `${previewData.settlement_impact.pending_settlements_count} pending settlement request(s) will be automatically rejected upon cancellation.`
      : "");

  const opImpact = previewData?.operational_impact || {};
  const finImpact = previewData?.financial_impact || {};
  const currency = finImpact?.currency || "INR";

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center p-3 sm:p-4 bg-slate-900/60 backdrop-blur-xs animate-fade-in"
      onClick={() => {
        if (!isCancelling && !loadingPreview) handleClose();
      }}
    >
      <div
        className="bg-white rounded-2xl shadow-2xl border border-[#bcc9c6] max-w-2xl w-full max-h-[92vh] flex flex-col overflow-hidden"
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
        aria-labelledby="cancel-event-modal-title"
      >
        {/* Modal Header */}
        <div className="p-5 sm:p-6 border-b border-[#bcc9c6]/40 flex items-start justify-between gap-4 shrink-0 bg-white">
          <div className="flex items-start gap-3.5">
            <div className="size-11 rounded-full bg-red-50 border border-red-100 flex items-center justify-center shrink-0 text-red-600">
              <svg
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
                className="size-6 text-red-600"
              >
                <circle cx="12" cy="12" r="10" />
                <line x1="4.93" y1="4.93" x2="19.07" y2="19.07" />
              </svg>
            </div>

            <div>
              <h3
                id="cancel-event-modal-title"
                className="text-lg sm:text-xl font-bold text-[#141b2b]"
              >
                Cancel Event
              </h3>
              <p className="mt-1 text-xs sm:text-sm text-[#565e74]">
                Review the operational and financial impact before confirming event cancellation.
              </p>
            </div>
          </div>

          <button
            type="button"
            onClick={handleClose}
            disabled={isCancelling}
            className="text-[#565e74] hover:text-[#141b2b] p-1.5 rounded-lg hover:bg-[#f1f3ff] transition-colors disabled:opacity-40 disabled:cursor-not-allowed cursor-pointer"
            aria-label="Close modal"
          >
            <svg
              className="size-5"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth="2"
                d="M6 18L18 6M6 6l12 12"
              />
            </svg>
          </button>
        </div>

        {/* Modal Body */}
        <div className="p-5 sm:p-6 overflow-y-auto flex-1 space-y-5">
          {/* Loading State */}
          {loadingPreview && (
            <div className="flex flex-col items-center justify-center py-14 gap-3 text-center">
              <div className="size-10 border-3 border-red-600 border-t-transparent rounded-full animate-spin" />
              <span className="text-sm font-semibold text-[#141b2b]">
                Evaluating cancellation impact...
              </span>
              <span className="text-xs text-[#565e74] max-w-sm">
                Retrieving live bookings, tickets, reservations, settlement safeties, and refund calculations.
              </span>
            </div>
          )}

          {/* Preview Fetch Error */}
          {!loadingPreview && previewError && (
            <div className="p-4 bg-red-50 border border-red-200 rounded-xl space-y-3 text-center">
              <div className="text-sm font-semibold text-red-800">
                Unable to load cancellation preview
              </div>
              <div className="text-xs text-red-700">{previewError}</div>
              <button
                type="button"
                onClick={() => {
                  setPreviewError("");
                  setLoadingPreview(true);
                  getOrganizerEventCancellationPreview(eventId)
                    .then((res) => setPreviewData(res?.data || res))
                    .catch((err) =>
                      setPreviewError(
                        err?.response?.data?.message ||
                          err?.message ||
                          "Failed to load preview."
                      )
                    )
                    .finally(() => setLoadingPreview(false));
                }}
                className="inline-flex items-center gap-1.5 px-3.5 py-1.5 bg-red-600 hover:bg-red-700 text-white text-xs font-semibold rounded-lg transition-colors cursor-pointer"
              >
                Retry Preview
              </button>
            </div>
          )}

          {/* Preview Loaded Content */}
          {!loadingPreview && !previewError && previewData && (
            <>
              {/* Event Information Header Card */}
              <div className="bg-[#f1f3ff]/50 border border-[#bcc9c6]/50 rounded-xl p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
                <div className="min-w-0">
                  <div className="text-xs text-[#565e74] font-medium uppercase tracking-wider">
                    Target Event
                  </div>
                  <div className="text-base font-bold text-[#141b2b] truncate mt-0.5">
                    {previewData?.event?.title || eventTitle || "Event"}
                  </div>
                  <div className="text-xs text-[#565e74] mt-0.5">
                    Scheduled Date:{" "}
                    <span className="font-medium text-[#141b2b]">
                      {previewData?.event?.scheduled_date || "—"}
                    </span>
                  </div>
                </div>

                <div className="shrink-0 self-start sm:self-auto">
                  <StatusBadge
                    status={previewData?.event?.status || "PUBLISHED"}
                  />
                </div>
              </div>

              {/* Eligibility Banners */}
              {!isCancellable ? (
                <div className="p-4 bg-red-50 border border-red-200 rounded-xl space-y-2">
                  <div className="flex items-center gap-2 text-sm font-bold text-red-800">
                    <svg
                      className="size-5 text-red-600 shrink-0"
                      viewBox="0 0 20 20"
                      fill="currentColor"
                    >
                      <path
                        fillRule="evenodd"
                        d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
                        clipRule="evenodd"
                      />
                    </svg>
                    <span>Cancellation Blocked</span>
                  </div>
                  <p className="text-xs text-red-700 leading-relaxed">
                    This event cannot be cancelled due to the following business rules:
                  </p>
                  <ul className="list-disc list-inside space-y-1 text-xs text-red-800 font-medium pl-1">
                    {blockingReasons.length > 0 ? (
                      blockingReasons.map((reasonText, idx) => (
                        <li key={idx}>{reasonText}</li>
                      ))
                    ) : (
                      <li>Event status does not permit cancellation.</li>
                    )}
                  </ul>
                </div>
              ) : (
                <div className="p-3.5 bg-emerald-50 border border-emerald-200 rounded-xl flex items-center gap-2.5 text-xs text-emerald-800 font-medium">
                  <svg
                    className="size-4 text-emerald-600 shrink-0"
                    viewBox="0 0 20 20"
                    fill="currentColor"
                  >
                    <path
                      fillRule="evenodd"
                      d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
                      clipRule="evenodd"
                    />
                  </svg>
                  <span>
                    This event is eligible for cancellation. All ticket holders will be refunded.
                  </span>
                </div>
              )}

              {/* Settlement Warning Banner */}
              {settlementWarning && (
                <div className="p-3.5 bg-amber-50 border border-amber-200 rounded-xl space-y-1 text-amber-900">
                  <div className="flex items-center gap-2 text-xs font-bold text-amber-800">
                    <svg
                      className="size-4 text-amber-600 shrink-0"
                      viewBox="0 0 20 20"
                      fill="currentColor"
                    >
                      <path
                        fillRule="evenodd"
                        d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
                        clipRule="evenodd"
                      />
                    </svg>
                    <span>Settlement Safety Notice</span>
                  </div>
                  <p className="text-xs text-amber-800 leading-relaxed">
                    {settlementWarning}
                  </p>
                </div>
              )}

              {/* Operational Impact Grid */}
              <div className="space-y-2.5">
                <div className="text-xs font-bold text-[#141b2b] uppercase tracking-wider">
                  Operational Impact Summary
                </div>
                <div className="grid grid-cols-2 sm:grid-cols-3 gap-2.5">
                  <div className="bg-[#f1f3ff]/40 border border-[#bcc9c6]/40 rounded-xl p-3">
                    <div className="text-[11px] text-[#565e74] font-medium">
                      Bookings Affected
                    </div>
                    <div className="text-lg font-bold text-[#141b2b] mt-0.5">
                      {opImpact.affected_bookings ?? 0}
                      <span className="text-xs font-normal text-[#565e74]">
                        {" "}
                        / {opImpact.total_bookings ?? 0} total
                      </span>
                    </div>
                  </div>

                  <div className="bg-[#f1f3ff]/40 border border-[#bcc9c6]/40 rounded-xl p-3">
                    <div className="text-[11px] text-[#565e74] font-medium">
                      Tickets Cancelled
                    </div>
                    <div className="text-lg font-bold text-[#141b2b] mt-0.5">
                      {opImpact.affected_tickets ?? 0}
                      <span className="text-xs font-normal text-[#565e74]">
                        {" "}
                        / {opImpact.total_tickets_sold ?? 0} sold
                      </span>
                    </div>
                  </div>

                  <div className="bg-[#f1f3ff]/40 border border-[#bcc9c6]/40 rounded-xl p-3">
                    <div className="text-[11px] text-[#565e74] font-medium">
                      Active Holds Released
                    </div>
                    <div className="text-lg font-bold text-[#141b2b] mt-0.5">
                      {opImpact.active_reservations_released ?? 0}
                    </div>
                  </div>

                  <div className="bg-[#f1f3ff]/40 border border-[#bcc9c6]/40 rounded-xl p-3">
                    <div className="text-[11px] text-[#565e74] font-medium">
                      Reserved Seats Freed
                    </div>
                    <div className="text-lg font-bold text-[#141b2b] mt-0.5">
                      {opImpact.reserved_seats_released ?? 0}
                    </div>
                  </div>

                  <div className="bg-[#f1f3ff]/40 border border-[#bcc9c6]/40 rounded-xl p-3">
                    <div className="text-[11px] text-[#565e74] font-medium">
                      GA Quantity Restored
                    </div>
                    <div className="text-lg font-bold text-[#141b2b] mt-0.5">
                      {opImpact.general_admission_quantity_released ?? 0}
                    </div>
                  </div>

                  <div className="bg-[#f1f3ff]/40 border border-[#bcc9c6]/40 rounded-xl p-3">
                    <div className="text-[11px] text-[#565e74] font-medium">
                      Active Checkouts Expired
                    </div>
                    <div className="text-lg font-bold text-[#141b2b] mt-0.5">
                      {opImpact.active_checkouts_expired ?? 0}
                    </div>
                  </div>
                </div>
              </div>

              {/* Financial Impact Breakdown */}
              <div className="space-y-2.5">
                <div className="text-xs font-bold text-[#141b2b] uppercase tracking-wider">
                  Financial Impact & Attendee Refunds
                </div>

                {/* Highlight Refund Card */}
                <div className="bg-rose-50/70 border border-rose-200 rounded-xl p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
                  <div>
                    <div className="text-xs font-semibold text-rose-800">
                      Total Customer Refund Obligation
                    </div>
                    <div className="text-2xl font-extrabold text-rose-700 mt-1">
                      {formatCurrency(finImpact.expected_refund_amount, currency)}
                    </div>
                    <p className="text-[11px] text-rose-600 mt-0.5">
                      Includes 100% ticket price, platform fee, and tax refunds to buyers.
                    </p>
                  </div>

                  <div className="sm:text-right text-xs text-[#565e74] shrink-0 border-t sm:border-t-0 pt-2 sm:pt-0 border-rose-200">
                    <div>Gross Collected: {formatCurrency(finImpact.gross_collected_amount, currency)}</div>
                    <div>Already Refunded: {formatCurrency(finImpact.already_refunded_amount, currency)}</div>
                  </div>
                </div>

                {/* Detailed Ledger Reversals Table */}
                <div className="border border-[#bcc9c6]/50 rounded-xl overflow-hidden text-xs">
                  <div className="bg-gray-50/80 px-3.5 py-2 font-semibold text-[#141b2b] border-b border-[#bcc9c6]/30 flex justify-between">
                    <span>Ledger Accounting Breakdown</span>
                    <span>Reversal Amount</span>
                  </div>

                  <div className="divide-y divide-[#bcc9c6]/20 bg-white">
                    <div className="px-3.5 py-2.5 flex justify-between items-center">
                      <div>
                        <span className="font-medium text-[#141b2b]">
                          Organizer Payable Revocation
                        </span>
                        <div className="text-[11px] text-[#565e74]">
                          Organizer net revenue deducted from EventHub ledger
                        </div>
                      </div>
                      <span className="font-bold text-red-600">
                        {formatCurrency(finImpact.organizer_payable_revocation, currency)}
                      </span>
                    </div>

                    <div className="px-3.5 py-2.5 flex justify-between items-center">
                      <div>
                        <span className="font-medium text-[#141b2b]">
                          Platform Fee Refund Reversal
                        </span>
                        <div className="text-[11px] text-[#565e74]">
                          Platform fee waived and refunded to attendees
                        </div>
                      </div>
                      <span className="font-medium text-[#141b2b]">
                        {formatCurrency(finImpact.platform_fee_refund, currency)}
                      </span>
                    </div>

                    <div className="px-3.5 py-2.5 flex justify-between items-center">
                      <div>
                        <span className="font-medium text-[#141b2b]">
                          Tax Refund Reversal
                        </span>
                        <div className="text-[11px] text-[#565e74]">
                          GST/Sales tax refunded to attendees
                        </div>
                      </div>
                      <span className="font-medium text-[#141b2b]">
                        {formatCurrency(finImpact.tax_refund, currency)}
                      </span>
                    </div>
                  </div>
                </div>
              </div>

              {/* Form Input for Reason (Eligible Events Only) */}
              {isCancellable && (
                <div className="space-y-4 pt-2 border-t border-[#bcc9c6]/40">
                  <div className="space-y-1.5">
                    <div className="flex justify-between items-center">
                      <label
                        htmlFor="cancellation-reason-input"
                        className="text-xs sm:text-sm font-bold text-[#141b2b]"
                      >
                        Cancellation Reason <span className="text-red-500">*</span>
                      </label>
                      <span
                        className={`text-[11px] font-medium ${
                          reason.trim().length > 500
                            ? "text-red-600"
                            : "text-[#565e74]"
                        }`}
                      >
                        {reason.trim().length}/500
                      </span>
                    </div>

                    <textarea
                      id="cancellation-reason-input"
                      rows={3}
                      value={reason}
                      onChange={(e) => {
                        setReason(e.target.value);
                        if (cancelError) setCancelError("");
                      }}
                      disabled={isCancelling}
                      placeholder="Explain why this event is being cancelled. This reason will be included in email notifications sent to all registered attendees..."
                      className="w-full text-xs sm:text-sm border border-[#bcc9c6] rounded-xl p-3 focus:outline-none focus:ring-2 focus:ring-red-500/20 focus:border-red-500 transition-all placeholder:text-[#565e74]/60 resize-none disabled:bg-gray-50 disabled:cursor-not-allowed"
                    />
                  </div>

                  {/* Confirmation Checkbox */}
                  <label className="flex items-start gap-2.5 cursor-pointer select-none">
                    <input
                      type="checkbox"
                      checked={hasConfirmed}
                      onChange={(e) => {
                        setHasConfirmed(e.target.checked);
                        if (cancelError) setCancelError("");
                      }}
                      disabled={isCancelling}
                      className="mt-0.5 size-4 rounded text-red-600 border-[#bcc9c6] focus:ring-red-500 cursor-pointer disabled:cursor-not-allowed"
                    />
                    <span className="text-xs text-[#141b2b] leading-relaxed">
                      I understand that cancelling this event is{" "}
                      <strong className="text-red-600 font-semibold">
                        permanent and cannot be undone
                      </strong>
                      . All valid tickets will be invalidated and attendee refunds will be immediately initiated.
                    </span>
                  </label>
                </div>
              )}

              {/* Execution Error Banner */}
              {cancelError && (
                <div className="p-3 bg-red-100 border border-red-200 rounded-xl text-xs text-red-800 font-medium">
                  {cancelError}
                </div>
              )}
            </>
          )}
        </div>

        {/* Modal Footer */}
        <div className="bg-[#f1f3ff]/60 px-5 sm:px-6 py-4 border-t border-[#bcc9c6]/40 flex items-center justify-end gap-3 shrink-0">
          <button
            type="button"
            onClick={handleClose}
            disabled={isCancelling}
            className="px-4 py-2 text-xs sm:text-sm font-medium text-[#565e74] hover:text-[#141b2b] bg-white border border-[#bcc9c6] hover:bg-gray-50 rounded-lg transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isCancellable ? "Keep Event" : "Close"}
          </button>

          {isCancellable && (
            <button
              type="button"
              onClick={handleConfirmCancellation}
              disabled={
                loadingPreview ||
                !isCancellable ||
                !reason.trim() ||
                reason.trim().length > 500 ||
                !hasConfirmed ||
                isCancelling
              }
              className="px-4 py-2 text-xs sm:text-sm font-medium text-white bg-red-600 hover:bg-red-700 rounded-lg transition-colors shadow-2xs cursor-pointer flex items-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {isCancelling ? (
                <>
                  <svg
                    className="animate-spin size-4 text-white"
                    xmlns="http://www.w3.org/2000/svg"
                    fill="none"
                    viewBox="0 0 24 24"
                  >
                    <circle
                      className="opacity-25"
                      cx="12"
                      cy="12"
                      r="10"
                      stroke="currentColor"
                      strokeWidth="4"
                    />
                    <path
                      className="opacity-75"
                      fill="currentColor"
                      d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                    />
                  </svg>
                  <span>Cancelling Event...</span>
                </>
              ) : (
                <span>Confirm Event Cancellation</span>
              )}
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
