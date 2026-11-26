import { useState, useEffect } from "react";
import notify from "../../utils/toast";
import { formatCurrency, formatDateTime } from "../../utils/formatters";

const bookingStatusBadgeStyles = {
  CONFIRMED: "bg-[#dcfce7] text-[#15803d] border border-[#bbf7d0]",
  PENDING: "bg-[#fef9c3] text-[#854d0e] border border-[#fef08a]",
  CANCELLED: "bg-[#fee2e2] text-[#b91c1c] border border-[#fecaca]",
  PARTIALLY_CANCELLED: "bg-[#ffedd5] text-[#c2410c] border border-[#fed7aa]",
};

const paymentStatusBadgeStyles = {
  SUCCESS: "bg-[#dcfce7] text-[#15803d] border border-[#bbf7d0]",
  PENDING: "bg-[#fef9c3] text-[#854d0e] border border-[#fef08a]",
  FAILED: "bg-[#fee2e2] text-[#b91c1c] border border-[#fecaca]",
  REFUND_REQUIRED: "bg-[#ffedd5] text-[#c2410c] border border-[#fed7aa]",
};

const refundStatusBadgeStyles = {
  FULL: "bg-[#fae8ff] text-[#86198f] border border-[#f5d0fe]",
  PARTIAL: "bg-[#f3e8ff] text-[#7e22ce] border border-[#e9d5ff]",
  PENDING: "bg-[#fef9c3] text-[#854d0e] border border-[#fef08a]",
  FAILED: "bg-[#fee2e2] text-[#b91c1c] border border-[#fecaca]",
};

const ticketStatusBadgeStyles = {
  VALID: "bg-[#fef9c3] text-[#854d0e] border border-[#fef08a]",
  USED: "bg-[#dcfce7] text-[#15803d] border border-[#bbf7d0]",
  CANCELLED: "bg-[#fee2e2] text-[#b91c1c] border border-[#fecaca]",
  REFUNDED: "bg-[#f3e8ff] text-[#7e22ce] border border-[#e9d5ff]",
};

export default function AdminBookingDetailDrawer({ booking, onClose }) {
  const [activeTab, setActiveTab] = useState("overview");
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

  if (!booking) return null;

  const copyToClipboard = async (text, label) => {
    if (!text) return;
    try {
      await navigator.clipboard.writeText(text);
      setCopiedField(label);
      notify.info(`Copied ${label} to clipboard`);
      setTimeout(() => setCopiedField(null), 2000);
    } catch {
      notify.error("Failed to copy to clipboard");
    }
  };

  const bookingStatus = booking.booking_status?.toUpperCase() || "CONFIRMED";
  const bookingBadge =
    bookingStatusBadgeStyles[bookingStatus] ||
    "bg-[#f3f4f6] text-[#4b5563] border border-[#e5e7eb]";

  const paymentStatus = booking.payment_status?.toUpperCase() || "SUCCESS";
  const paymentBadge =
    paymentStatusBadgeStyles[paymentStatus] ||
    "bg-[#f3f4f6] text-[#4b5563] border border-[#e5e7eb]";

  const refundStatus = booking.refund_status?.toUpperCase();
  const refundBadge = refundStatus ? refundStatusBadgeStyles[refundStatus] : null;

  const financials = booking.financials || {
    original_booking_total: booking.original_booking_total || "0.00",
    successful_captured_amount: booking.original_booking_total || "0.00",
    total_refunded_amount: "0.00",
    net_captured_after_refunds: booking.original_booking_total || "0.00",
    currency: booking.currency || "INR",
    subtotal: booking.original_booking_total || "0.00",
    platform_fee: "0.00",
    tax: "0.00",
    discount: "0.00",
  };

  const currency = financials.currency || booking.currency || "INR";
  const items = booking.items || [];
  const tickets = booking.tickets || [];
  const payments = booking.payments || [];
  const refunds = booking.refunds || [];
  const cancellation = booking.cancellation_policy || null;
  const organizer = booking.organizer || null;

  return (
    <div
      className="fixed inset-0 z-50 overflow-hidden"
      role="dialog"
      aria-modal="true"
      aria-labelledby="admin-booking-drawer-title"
    >
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/40 backdrop-blur-xs transition-opacity"
        onClick={onClose}
        aria-hidden="true"
      />

      <div className="fixed inset-y-0 right-0 max-w-full flex pl-6 sm:pl-10">
        <div className="w-screen max-w-2xl bg-white shadow-2xl flex flex-col h-full overflow-hidden">
          {/* Drawer Top Header */}
          <div className="px-6 py-5 border-b border-[#dce2f7] bg-[#f9f9ff] flex items-center justify-between shrink-0">
            <div>
              <div className="flex flex-wrap items-center gap-2">
                <h2 id="admin-booking-drawer-title" className="text-[#141b2b] text-lg font-bold">
                  Platform Booking Details
                </h2>
                <span className={`text-[11px] font-bold px-2 py-0.5 rounded-full ${bookingBadge}`}>
                  {bookingStatus}
                </span>
                <span className={`text-[11px] font-bold px-2 py-0.5 rounded-full ${paymentBadge}`}>
                  PAYMENT: {paymentStatus}
                </span>
                {refundBadge && (
                  <span className={`text-[11px] font-bold px-2 py-0.5 rounded-full ${refundBadge}`}>
                    REFUND: {refundStatus}
                  </span>
                )}
              </div>
              <div className="flex items-center gap-2 mt-1">
                <span className="text-xs font-mono font-semibold text-[#00685f]">
                  {booking.booking_number}
                </span>
                <button
                  type="button"
                  onClick={() => copyToClipboard(booking.booking_number, "Booking Reference")}
                  className="text-[#6d7a77] hover:text-[#00685f] transition-colors p-0.5 cursor-pointer"
                  title="Copy reference"
                >
                  <svg className="w-3.5 h-3.5" viewBox="0 0 20 20" fill="currentColor">
                    <path d="M8 3a1 1 0 011-1h2a1 1 0 110 2H9a1 1 0 01-1-1z" />
                    <path d="M6 3a2 2 0 00-2 2v11a2 2 0 002 2h8a2 2 0 002-2V5a2 2 0 00-2-2 3 3 0 01-3 3H9a3 3 0 01-3-3z" />
                  </svg>
                </button>
                {copiedField === "Booking Reference" && (
                  <span className="text-[10px] text-[#00685f] font-semibold">Copied!</span>
                )}
              </div>
            </div>

            <button
              type="button"
              onClick={onClose}
              className="p-2 rounded-lg text-[#6d7a77] hover:bg-[#dce2f7]/50 hover:text-[#141b2b] transition-colors cursor-pointer"
              aria-label="Close drawer"
            >
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>

          {/* Drawer Navigation Tabs */}
          <div className="flex border-b border-[#dce2f7] bg-white px-6 shrink-0 overflow-x-auto">
            <button
              type="button"
              onClick={() => setActiveTab("overview")}
              className={`py-3 px-3 text-xs font-bold border-b-2 transition-colors cursor-pointer whitespace-nowrap ${
                activeTab === "overview"
                  ? "border-[#00685f] text-[#00685f]"
                  : "border-transparent text-[#6d7a77] hover:text-[#141b2b]"
              }`}
            >
              Overview
            </button>
            <button
              type="button"
              onClick={() => setActiveTab("items")}
              className={`py-3 px-3 text-xs font-bold border-b-2 transition-colors cursor-pointer whitespace-nowrap ${
                activeTab === "items"
                  ? "border-[#00685f] text-[#00685f]"
                  : "border-transparent text-[#6d7a77] hover:text-[#141b2b]"
              }`}
            >
              Items ({items.length})
            </button>
            <button
              type="button"
              onClick={() => setActiveTab("tickets")}
              className={`py-3 px-3 text-xs font-bold border-b-2 transition-colors cursor-pointer whitespace-nowrap ${
                activeTab === "tickets"
                  ? "border-[#00685f] text-[#00685f]"
                  : "border-transparent text-[#6d7a77] hover:text-[#141b2b]"
              }`}
            >
              Tickets ({tickets.length})
            </button>
            <button
              type="button"
              onClick={() => setActiveTab("financials")}
              className={`py-3 px-3 text-xs font-bold border-b-2 transition-colors cursor-pointer whitespace-nowrap ${
                activeTab === "financials"
                  ? "border-[#00685f] text-[#00685f]"
                  : "border-transparent text-[#6d7a77] hover:text-[#141b2b]"
              }`}
            >
              Financials
            </button>
            <button
              type="button"
              onClick={() => setActiveTab("payments")}
              className={`py-3 px-3 text-xs font-bold border-b-2 transition-colors cursor-pointer whitespace-nowrap ${
                activeTab === "payments"
                  ? "border-[#00685f] text-[#00685f]"
                  : "border-transparent text-[#6d7a77] hover:text-[#141b2b]"
              }`}
            >
              Payments & Refunds ({payments.length + refunds.length})
            </button>
          </div>

          {/* Drawer Scrollable Content */}
          <div className="flex-1 overflow-y-auto p-6 space-y-6">
            {/* OVERVIEW TAB */}
            {activeTab === "overview" && (
              <div className="space-y-6">
                {/* Event & Organizer Card */}
                <div className="border border-[#dce2f7] rounded-xl p-4 bg-[#f9f9ff] space-y-3">
                  <div className="flex items-center justify-between">
                    <h3 className="text-xs font-bold text-[#3d4947] uppercase tracking-wider">
                      Event & Organizer Details
                    </h3>
                    <span className="text-[11px] font-semibold text-[#00685f] bg-[#00685f]/10 px-2 py-0.5 rounded">
                      {booking.event?.event_type || "Event"}
                    </span>
                  </div>
                  <div>
                    <h4 className="text-sm font-bold text-[#141b2b]">
                      {booking.event?.title || "Untitled Event"}
                    </h4>
                    {booking.event?.venue_name && (
                      <p className="text-xs text-[#6d7a77] mt-0.5 flex items-center gap-1">
                        <span>📍</span> {booking.event.venue_name}
                      </p>
                    )}
                  </div>
                  {organizer && (
                    <div className="pt-2 border-t border-[#dce2f7] grid grid-cols-1 sm:grid-cols-2 gap-2 text-xs">
                      <div>
                        <span className="text-[#6d7a77] block text-[11px]">Organizer Business:</span>
                        <span className="font-semibold text-[#141b2b]">
                          {organizer.business_name || "—"}
                        </span>
                      </div>
                      {organizer.email && (
                        <div>
                          <span className="text-[#6d7a77] block text-[11px]">Organizer Email:</span>
                          <span className="font-mono text-[#141b2b]">{organizer.email}</span>
                        </div>
                      )}
                    </div>
                  )}
                </div>

                {/* Customer Info Card */}
                <div className="border border-[#dce2f7] rounded-xl p-4 bg-white space-y-3">
                  <h3 className="text-xs font-bold text-[#3d4947] uppercase tracking-wider">
                    Customer Information
                  </h3>
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 text-xs">
                    <div>
                      <span className="text-[#6d7a77] block">Purchaser Name:</span>
                      <span className="font-semibold text-[#141b2b]">
                        {booking.customer?.full_name || "Guest Customer"}
                      </span>
                    </div>
                    <div>
                      <span className="text-[#6d7a77] block">Purchaser Email:</span>
                      <span className="font-mono text-[#141b2b] break-all">
                        {booking.customer?.email || "—"}
                      </span>
                    </div>
                  </div>
                </div>

                {/* Booking Order Info Card */}
                <div className="border border-[#dce2f7] rounded-xl p-4 bg-white space-y-3">
                  <h3 className="text-xs font-bold text-[#3d4947] uppercase tracking-wider">
                    Order Information
                  </h3>
                  <div className="grid grid-cols-2 sm:grid-cols-3 gap-3 text-xs">
                    <div>
                      <span className="text-[#6d7a77] block">Order Placed:</span>
                      <span className="font-medium text-[#141b2b]">
                        {formatDateTime(booking.created_at)}
                      </span>
                    </div>
                    <div>
                      <span className="text-[#6d7a77] block">Total Amount:</span>
                      <span className="font-bold font-mono text-[#141b2b]">
                        {formatCurrency(financials.original_booking_total, currency)}
                      </span>
                    </div>
                    <div>
                      <span className="text-[#6d7a77] block">Tickets Purchased:</span>
                      <span className="font-semibold text-[#00685f]">
                        {booking.ticket_count || tickets.length}
                      </span>
                    </div>
                  </div>
                </div>

                {/* Cancellation Policy Snapshot */}
                {cancellation && (
                  <div className="border border-[#dce2f7] rounded-xl p-4 bg-[#f9f9ff] space-y-2 text-xs">
                    <h3 className="text-xs font-bold text-[#3d4947] uppercase tracking-wider">
                      Cancellation Policy Snapshot
                    </h3>
                    <div className="grid grid-cols-2 sm:grid-cols-3 gap-2">
                      <div>
                        <span className="text-[#6d7a77] block">Cancellation Allowed:</span>
                        <span className="font-semibold text-[#141b2b]">
                          {cancellation.cancellation_allowed ? "Yes" : "No"}
                        </span>
                      </div>
                      <div>
                        <span className="text-[#6d7a77] block">Deadline:</span>
                        <span className="font-semibold text-[#141b2b]">
                          {cancellation.cancellation_deadline_hours} hours prior
                        </span>
                      </div>
                      <div>
                        <span className="text-[#6d7a77] block">Refund Policy:</span>
                        <span className="font-semibold text-[#141b2b]">
                          {cancellation.refund_policy || "STANDARD"} ({cancellation.refund_percentage}%)
                        </span>
                      </div>
                    </div>
                  </div>
                )}
              </div>
            )}

            {/* ITEMS TAB */}
            {activeTab === "items" && (
              <div className="space-y-4">
                <div className="border border-[#dce2f7] rounded-xl overflow-hidden bg-white">
                  <div className="overflow-x-auto">
                    <table className="w-full text-left text-xs">
                      <thead className="bg-[#f9f9ff] border-b border-[#dce2f7] text-[#3d4947]">
                        <tr>
                          <th className="py-2.5 px-3 font-semibold">Line Item</th>
                          <th className="py-2.5 px-3 font-semibold">Seat</th>
                          <th className="py-2.5 px-3 font-semibold text-center">Qty</th>
                          <th className="py-2.5 px-3 font-semibold text-right">Unit Price</th>
                          <th className="py-2.5 px-3 font-semibold text-right">Total</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-[#f1f3ff]">
                        {items.length === 0 ? (
                          <tr>
                            <td colSpan={5} className="py-6 text-center text-[#6d7a77]">
                              No line items found for this booking.
                            </td>
                          </tr>
                        ) : (
                          items.map((item, idx) => (
                            <tr key={item.id || idx} className="hover:bg-[#f9f9ff]">
                              <td className="py-2.5 px-3 font-medium text-[#141b2b]">
                                {item.ticket_type_name || "Ticket"}
                              </td>
                              <td className="py-2.5 px-3 text-[#6d7a77]">
                                {item.seat_label || "General"}
                              </td>
                              <td className="py-2.5 px-3 text-center font-semibold text-[#141b2b]">
                                {item.quantity}
                              </td>
                              <td className="py-2.5 px-3 text-right font-mono text-[#3d4947]">
                                {formatCurrency(item.unit_price, currency)}
                              </td>
                              <td className="py-2.5 px-3 text-right font-mono font-bold text-[#141b2b]">
                                {formatCurrency(item.total_price, currency)}
                              </td>
                            </tr>
                          ))
                        )}
                      </tbody>
                    </table>
                  </div>
                </div>
              </div>
            )}

            {/* TICKETS TAB */}
            {activeTab === "tickets" && (
              <div className="space-y-3">
                {tickets.length === 0 ? (
                  <div className="border border-[#dce2f7] rounded-xl p-8 text-center text-xs text-[#6d7a77] bg-white">
                    No individual ticket passes issued for this booking.
                  </div>
                ) : (
                  tickets.map((tck, idx) => {
                    const tStatus = tck.status?.toUpperCase() || "VALID";
                    const tBadge = ticketStatusBadgeStyles[tStatus] || "bg-[#f3f4f6] text-[#4b5563]";
                    return (
                      <div
                        key={tck.id || idx}
                        className="border border-[#dce2f7] rounded-xl p-3.5 bg-white flex flex-col sm:flex-row sm:items-center justify-between gap-3 text-xs"
                      >
                        <div className="space-y-1">
                          <div className="flex items-center gap-2">
                            <span className="font-mono font-bold text-[#00685f]">
                              {tck.ticket_code}
                            </span>
                            <button
                              type="button"
                              onClick={() => copyToClipboard(tck.ticket_code, "Ticket Code")}
                              className="text-[#6d7a77] hover:text-[#00685f] p-0.5 cursor-pointer"
                              title="Copy code"
                            >
                              <svg className="w-3.5 h-3.5" viewBox="0 0 20 20" fill="currentColor">
                                <path d="M8 3a1 1 0 011-1h2a1 1 0 110 2H9a1 1 0 01-1-1z" />
                                <path d="M6 3a2 2 0 00-2 2v11a2 2 0 002 2h8a2 2 0 002-2V5a2 2 0 00-2-2 3 3 0 01-3 3H9a3 3 0 01-3-3z" />
                              </svg>
                            </button>
                            <span className="text-[#6d7a77] text-[11px]">
                              Pass #{tck.admission_number}
                            </span>
                          </div>
                          <div className="text-[#141b2b] font-medium">
                            {tck.ticket_type_name}
                            {tck.seat_label && (
                              <span className="text-[#6d7a77] font-normal ml-1">
                                • {tck.seat_label}
                              </span>
                            )}
                          </div>
                        </div>

                        <div className="flex items-center gap-3">
                          <div className="text-right">
                            <span className={`text-[10px] font-bold px-2 py-0.5 rounded-full ${tBadge}`}>
                              {tStatus}
                            </span>
                            {tck.used_at && (
                              <span className="block text-[10px] text-[#6d7a77] mt-0.5">
                                Used: {formatDateTime(tck.used_at)}
                              </span>
                            )}
                          </div>
                          <div className="font-mono font-bold text-right text-[#141b2b] min-w-[70px]">
                            {formatCurrency(tck.price, currency)}
                          </div>
                        </div>
                      </div>
                    );
                  })
                )}
              </div>
            )}

            {/* FINANCIALS TAB */}
            {activeTab === "financials" && (
              <div className="space-y-4">
                {/* Captured Revenue Summary Card */}
                <div className="border border-[#dce2f7] rounded-xl p-4 bg-[#f9f9ff] space-y-3">
                  <h3 className="text-xs font-bold text-[#3d4947] uppercase tracking-wider">
                    Monetary Reconciliation
                  </h3>
                  <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
                    <div className="bg-white p-3 rounded-lg border border-[#dce2f7]">
                      <span className="text-[11px] text-[#6d7a77] block">Gross Captured</span>
                      <span className="text-base font-bold font-mono text-[#00685f]">
                        {formatCurrency(financials.successful_captured_amount, currency)}
                      </span>
                    </div>
                    <div className="bg-white p-3 rounded-lg border border-[#dce2f7]">
                      <span className="text-[11px] text-[#6d7a77] block">Total Refunded</span>
                      <span className="text-base font-bold font-mono text-[#ba1a1a]">
                        {formatCurrency(financials.total_refunded_amount, currency)}
                      </span>
                    </div>
                    <div className="bg-white p-3 rounded-lg border border-[#dce2f7]">
                      <span className="text-[11px] text-[#6d7a77] block">Net Captured</span>
                      <span className="text-base font-bold font-mono text-[#141b2b]">
                        {formatCurrency(financials.net_captured_after_refunds, currency)}
                      </span>
                    </div>
                  </div>
                </div>

                {/* Price Breakdown Breakdown */}
                <div className="border border-[#dce2f7] rounded-xl p-4 bg-white space-y-2.5 text-xs">
                  <h3 className="text-xs font-bold text-[#3d4947] uppercase tracking-wider mb-2">
                    Order Price Breakdown
                  </h3>
                  <div className="flex justify-between py-1 border-b border-[#f1f3ff]">
                    <span className="text-[#6d7a77]">Tickets Subtotal:</span>
                    <span className="font-mono font-medium text-[#141b2b]">
                      {formatCurrency(financials.subtotal, currency)}
                    </span>
                  </div>
                  <div className="flex justify-between py-1 border-b border-[#f1f3ff]">
                    <span className="text-[#6d7a77]">Platform Fee:</span>
                    <span className="font-mono font-medium text-[#141b2b]">
                      {formatCurrency(financials.platform_fee, currency)}
                    </span>
                  </div>
                  <div className="flex justify-between py-1 border-b border-[#f1f3ff]">
                    <span className="text-[#6d7a77]">Taxes:</span>
                    <span className="font-mono font-medium text-[#141b2b]">
                      {formatCurrency(financials.tax, currency)}
                    </span>
                  </div>
                  {Number(financials.discount) > 0 && (
                    <div className="flex justify-between py-1 border-b border-[#f1f3ff] text-[#ba1a1a]">
                      <span>Discount Applied:</span>
                      <span className="font-mono font-medium">
                        -{formatCurrency(financials.discount, currency)}
                      </span>
                    </div>
                  )}
                  <div className="flex justify-between py-2 text-sm font-bold border-t border-[#dce2f7]">
                    <span className="text-[#141b2b]">Original Booking Total:</span>
                    <span className="font-mono text-[#00685f]">
                      {formatCurrency(financials.original_booking_total, currency)}
                    </span>
                  </div>
                </div>
              </div>
            )}

            {/* PAYMENTS & REFUNDS TAB */}
            {activeTab === "payments" && (
              <div className="space-y-6">
                {/* Payments Section */}
                <div className="space-y-3">
                  <h3 className="text-xs font-bold text-[#3d4947] uppercase tracking-wider">
                    Payment Gateway Attempts ({payments.length})
                  </h3>
                  {payments.length === 0 ? (
                    <div className="border border-[#dce2f7] rounded-xl p-6 text-center text-xs text-[#6d7a77] bg-white">
                      No payment transaction records found.
                    </div>
                  ) : (
                    payments.map((p, idx) => (
                      <div
                        key={p.id || idx}
                        className="border border-[#dce2f7] rounded-xl p-3.5 bg-white space-y-2 text-xs"
                      >
                        <div className="flex items-center justify-between">
                          <span className="font-mono font-bold text-[#141b2b]">
                            {p.provider} • {p.payment_method || "Online"}
                          </span>
                          <span
                            className={`text-[10px] font-bold px-2 py-0.5 rounded-full ${
                              paymentStatusBadgeStyles[p.status?.toUpperCase()] || "bg-[#f3f4f6]"
                            }`}
                          >
                            {p.status}
                          </span>
                        </div>
                        <div className="grid grid-cols-2 sm:grid-cols-3 gap-2 text-[11px] text-[#6d7a77]">
                          <div>
                            <span>Amount: </span>
                            <span className="font-mono font-bold text-[#141b2b]">
                              {formatCurrency(p.amount, currency)}
                            </span>
                          </div>
                          <div>
                            <span>Transaction ID: </span>
                            <span className="font-mono text-[#141b2b]">{p.transaction_id || "—"}</span>
                          </div>
                          <div>
                            <span>Paid At: </span>
                            <span className="text-[#141b2b]">
                              {p.paid_at ? formatDateTime(p.paid_at) : "Pending"}
                            </span>
                          </div>
                        </div>
                      </div>
                    ))
                  )}
                </div>

                {/* Refunds Section */}
                <div className="space-y-3">
                  <h3 className="text-xs font-bold text-[#3d4947] uppercase tracking-wider">
                    Gateway Refunds ({refunds.length})
                  </h3>
                  {refunds.length === 0 ? (
                    <div className="border border-[#dce2f7] rounded-xl p-6 text-center text-xs text-[#6d7a77] bg-white">
                      No refund records recorded for this booking.
                    </div>
                  ) : (
                    refunds.map((r, idx) => (
                      <div
                        key={r.id || idx}
                        className="border border-[#dce2f7] rounded-xl p-3.5 bg-white space-y-2 text-xs"
                      >
                        <div className="flex items-center justify-between">
                          <span className="font-mono font-bold text-[#ba1a1a]">
                            Refund {formatCurrency(r.amount, currency)}
                          </span>
                          <span
                            className={`text-[10px] font-bold px-2 py-0.5 rounded-full ${
                              refundStatusBadgeStyles[r.status?.toUpperCase()] || "bg-[#f3f4f6]"
                            }`}
                          >
                            {r.status}
                          </span>
                        </div>
                        <div className="grid grid-cols-2 sm:grid-cols-3 gap-2 text-[11px] text-[#6d7a77]">
                          <div>
                            <span>Reason: </span>
                            <span className="text-[#141b2b]">{r.reason || "Customer cancellation"}</span>
                          </div>
                          <div>
                            <span>Gateway ID: </span>
                            <span className="font-mono text-[#141b2b]">{r.gateway_refund_id || "—"}</span>
                          </div>
                          <div>
                            <span>Processed: </span>
                            <span className="text-[#141b2b]">
                              {r.processed_at ? formatDateTime(r.processed_at) : formatDateTime(r.created_at)}
                            </span>
                          </div>
                        </div>
                      </div>
                    ))
                  )}
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
