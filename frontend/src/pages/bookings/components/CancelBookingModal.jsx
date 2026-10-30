import { useState } from "react";
import { cancelBooking } from "../../../api/bookingApi";
import {
  calculateExpectedRefund,
  calculateCancellationDeadline,
  formatDeadlineDateTime,
} from "../../../utils/cancellation";
import {
  CloseIcon,
  AlertTriangleIcon,
  CheckCircleIcon,
  RefundUndoIcon,
} from "./BookingIcons";

export default function CancelBookingModal({
  booking,
  event = null,
  onClose,
  onSuccess,
}) {
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState(null);
  const [result, setResult] = useState(null);

  if (!booking) return null;

  const bookingId = booking.ID || booking.id;
  const bookingNumber = booking.BookingNumber || booking.booking_number || bookingId;
  const totalPaid = Number(booking.Total ?? booking.total ?? 0);

  const eventTitle =
    event?.title ||
    booking.EventTitle ||
    booking.event_title ||
    `Event Booking #${bookingNumber}`;

  const deadlineHours = Number(
    booking.CancellationDeadlineHours ??
    booking.cancellation_deadline_hours ??
    0
  );

  const deadlineDate = event?.schedule
    ? calculateCancellationDeadline(event.schedule, deadlineHours)
    : null;

  const { expectedRefund, refundPolicy, refundPercentage } =
    calculateExpectedRefund(booking);

  const assignedSeats = booking.Seats || booking.seats || [];

  const handleConfirmCancel = async () => {
    if (isSubmitting) return;

    setIsSubmitting(true);
    setError(null);

    try {
      const response = await cancelBooking(bookingId);
      const output = response?.data || {
        booking_id: bookingId,
        booking_number: bookingNumber,
        status: "CANCELLED",
        refund_amount: expectedRefund,
        refund_status: expectedRefund > 0 ? "SUCCESS" : "NO_REFUND",
        cancelled_at: new Date().toISOString(),
      };

      setResult(output);
    } catch (err) {
      const backendMsg =
        err?.response?.data?.error ||
        err?.response?.data?.message ||
        err?.message ||
        "Failed to cancel booking. Please try again or contact support.";
      setError(backendMsg);
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleFinish = () => {
    if (result && typeof onSuccess === "function") {
      onSuccess(result);
    }
    onClose();
  };

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="cancel-modal-title"
      className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-xs animate-in fade-in duration-200"
    >
      <div className="relative w-full max-w-lg rounded-2xl bg-white p-6 sm:p-7 shadow-xl border border-slate-200 max-h-[90vh] overflow-y-auto">
        {/* Close Button */}
        <button
          type="button"
          onClick={result ? handleFinish : onClose}
          disabled={isSubmitting}
          className="absolute right-4 top-4 p-1.5 text-slate-400 hover:text-slate-700 rounded-lg transition-colors cursor-pointer disabled:opacity-40"
          aria-label="Close modal"
        >
          <CloseIcon className="w-5 h-5" />
        </button>

        {/* Phase A: Confirmation View */}
        {!result && (
          <div>
            {/* Header */}
            <div className="flex items-start gap-3.5 mb-5">
              <div className="size-11 rounded-full bg-rose-100 text-rose-600 flex items-center justify-center shrink-0">
                <AlertTriangleIcon className="w-6 h-6" />
              </div>
              <div className="min-w-0 pr-6">
                <h3
                  id="cancel-modal-title"
                  className="font-display font-bold text-lg sm:text-xl text-ink"
                >
                  Cancel Booking?
                </h3>
                <p className="mt-0.5 text-xs sm:text-sm text-[#565e74] truncate">
                  {eventTitle}
                </p>
                <p className="text-[11px] font-mono text-slate-500 mt-0.5">
                  Ref: {bookingNumber}
                </p>
              </div>
            </div>

            {/* Error Message if API fails */}
            {error && (
              <div className="mb-5 rounded-xl border border-red-200 bg-red-50/80 p-3.5 text-xs text-red-700 flex items-start gap-2.5">
                <span className="font-bold text-red-600 text-sm">⚠</span>
                <div className="flex-1">
                  <strong className="block font-semibold">Cancellation Error</strong>
                  <span>{error}</span>
                </div>
              </div>
            )}

            {/* Policy & Refund Calculation Card */}
            <div className="rounded-xl bg-[#f8fafc] border border-slate-200 p-4 space-y-3 text-xs sm:text-sm mb-5">
              <div className="flex items-center justify-between">
                <span className="text-[#565e74]">Cancellation Policy</span>
                <span className="font-semibold text-slate-900">
                  {refundPolicy === "FULL"
                    ? "Full Refund (100%)"
                    : refundPolicy === "PARTIAL"
                    ? `Partial Refund (${refundPercentage}%)`
                    : "No Refund"}
                </span>
              </div>

              <div className="flex items-center justify-between">
                <span className="text-[#565e74]">Total Amount Paid</span>
                <span className="font-medium text-slate-800">
                  ₹{totalPaid.toLocaleString("en-IN")}
                </span>
              </div>

              <div className="flex items-center justify-between border-t border-slate-200/80 pt-2.5">
                <span className="font-semibold text-slate-800">Expected Refund</span>
                <div className="text-right">
                  <strong
                    className={`font-display text-base font-bold ${
                      expectedRefund > 0 ? "text-emerald-700" : "text-slate-600"
                    }`}
                  >
                    ₹{expectedRefund.toLocaleString("en-IN")}
                  </strong>
                  {expectedRefund > 0 ? (
                    <span className="block text-[10px] text-emerald-600 font-medium">
                      Credited to original payment method
                    </span>
                  ) : (
                    <span className="block text-[10px] text-slate-500">
                      Non-refundable per policy
                    </span>
                  )}
                </div>
              </div>

              {deadlineDate && (
                <div className="flex items-start justify-between gap-2 border-t border-slate-200/80 pt-2.5 text-[11px] text-[#565e74]">
                  <span>Cancellation Window</span>
                  <span className="font-medium text-slate-700 text-right">
                    Allowed until {formatDeadlineDateTime(deadlineDate)}
                  </span>
                </div>
              )}
            </div>

            {/* Warning Disclaimer Box */}
            <div className="rounded-xl border border-amber-200 bg-amber-50/60 p-3.5 text-xs text-amber-900 mb-6 space-y-1.5">
              <p className="font-semibold flex items-center gap-1.5 text-amber-950">
                <span>⚠</span>
                <span>Important Consequences:</span>
              </p>
              <ul className="list-disc list-inside space-y-1 text-amber-800 text-[11px] sm:text-xs">
                <li>
                  All digital passes and QR codes in this booking will be{" "}
                  <strong>permanently invalidated</strong>.
                </li>
                {assignedSeats.length > 0 ? (
                  <li>
                    Assigned seats ({assignedSeats.length}) will be released
                    immediately to open event inventory.
                  </li>
                ) : (
                  <li>
                    Reserved admission tickets will be returned to event inventory.
                  </li>
                )}
                <li>This cancellation cannot be undone or reactivated.</li>
              </ul>
            </div>

            {/* Modal Actions */}
            <div className="flex flex-col-reverse sm:flex-row items-center justify-end gap-3 pt-2">
              <button
                type="button"
                onClick={onClose}
                disabled={isSubmitting}
                className="w-full sm:w-auto px-4 py-2.5 rounded-lg border border-[#dfe5e3] bg-white text-xs sm:text-sm font-semibold text-[#565e74] hover:bg-slate-50 transition-colors cursor-pointer disabled:opacity-50"
              >
                Keep Booking
              </button>

              <button
                type="button"
                onClick={handleConfirmCancel}
                disabled={isSubmitting}
                className="w-full sm:w-auto px-5 py-2.5 rounded-lg bg-[#ba1a1a] hover:bg-[#93000a] text-white text-xs sm:text-sm font-semibold shadow-xs transition-colors cursor-pointer flex items-center justify-center gap-2 disabled:opacity-60"
              >
                {isSubmitting ? (
                  <>
                    <div className="size-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                    <span>Processing Cancellation...</span>
                  </>
                ) : (
                  <span>Confirm Cancellation</span>
                )}
              </button>
            </div>
          </div>
        )}

        {/* Phase B: Cancellation Result View */}
        {result && (
          <div>
            {/* Header */}
            <div className="flex items-start gap-3.5 mb-5">
              <div className="size-11 rounded-full bg-emerald-100 text-emerald-700 flex items-center justify-center shrink-0">
                <CheckCircleIcon className="w-6 h-6" />
              </div>
              <div className="min-w-0 pr-6">
                <h3
                  id="cancel-modal-title"
                  className="font-display font-bold text-lg sm:text-xl text-ink"
                >
                  Booking Successfully Cancelled
                </h3>
                <p className="mt-0.5 text-xs sm:text-sm text-[#565e74] truncate">
                  {eventTitle}
                </p>
                <p className="text-[11px] font-mono text-slate-500 mt-0.5">
                  Ref: {result.booking_number || bookingNumber}
                </p>
              </div>
            </div>

            {/* Outcome Details Card */}
            <div className="rounded-xl bg-[#f8fafc] border border-slate-200 p-4 space-y-3 text-xs sm:text-sm mb-5">
              <div className="flex items-center justify-between">
                <span className="text-[#565e74]">Booking Status</span>
                <span className="rounded-full bg-red-100 px-2.5 py-0.5 text-xs font-bold text-red-700 uppercase">
                  CANCELLED
                </span>
              </div>

              <div className="flex items-center justify-between">
                <span className="text-[#565e74]">Refund Status</span>
                <span
                  className={`rounded-full px-2.5 py-0.5 text-xs font-semibold ${
                    result.refund_status === "SUCCESS"
                      ? "bg-emerald-100 text-emerald-800"
                      : result.refund_status === "PENDING"
                      ? "bg-amber-100 text-amber-800"
                      : "bg-slate-100 text-slate-700"
                  }`}
                >
                  {result.refund_status || "COMPLETED"}
                </span>
              </div>

              <div className="flex items-center justify-between border-t border-slate-200/80 pt-2.5">
                <span className="text-[#565e74]">Refund Amount</span>
                <strong className="font-display text-base font-bold text-slate-900">
                  ₹{Number(result.refund_amount || 0).toLocaleString("en-IN")}
                </strong>
              </div>

              {result.refund_id && (
                <div className="flex items-center justify-between border-t border-slate-200/80 pt-2.5 text-[11px]">
                  <span className="text-[#565e74]">Refund Transaction ID</span>
                  <code className="font-mono text-slate-800 font-semibold bg-white px-2 py-0.5 rounded border border-slate-200">
                    {result.refund_id}
                  </code>
                </div>
              )}
            </div>

            {/* Informative Guidance Note */}
            <div className="rounded-xl bg-[#eef1ff] p-4 text-xs text-[#3d4947] mb-6 flex items-start gap-3">
              <RefundUndoIcon className="w-5 h-5 text-[#00685f] shrink-0 mt-0.5" />
              <div className="space-y-1">
                {result.refund_status === "SUCCESS" && (
                  <p>
                    A refund of{" "}
                    <strong>
                      ₹{Number(result.refund_amount || 0).toLocaleString("en-IN")}
                    </strong>{" "}
                    has been initiated. The funds will reflect in your original
                    payment method within <strong>3–5 business days</strong>.
                  </p>
                )}
                {result.refund_status === "PENDING" && (
                  <p>
                    Your refund request is currently being processed by the
                    payment gateway. It will be credited to your original payment
                    account shortly.
                  </p>
                )}
                {(result.refund_status === "NO_REFUND" ||
                  Number(result.refund_amount || 0) === 0) && (
                  <p>
                    Your booking is now cancelled. As per the applicable event
                    cancellation policy, no refund is eligible for this booking.
                  </p>
                )}
                <p className="text-[11px] text-[#565e74]">
                  All ticket passes for this booking are now void for event entry.
                </p>
              </div>
            </div>

            {/* Close Button */}
            <div className="flex justify-end pt-2">
              <button
                type="button"
                onClick={handleFinish}
                className="w-full sm:w-auto px-6 py-2.5 rounded-lg bg-[#00685f] hover:bg-[#00544c] text-white text-xs sm:text-sm font-semibold shadow-xs transition-colors cursor-pointer"
              >
                Done
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
