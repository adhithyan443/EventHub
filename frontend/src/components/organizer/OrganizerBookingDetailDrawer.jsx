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

export default function OrganizerBookingDetailDrawer({ booking, onClose }) {
  const [activeTab, setActiveTab] = useState("overview"); // overview, items, tickets, financials, payments
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

  return (
    <div
      className="fixed inset-0 z-50 overflow-hidden"
      role="dialog"
      aria-modal="true"
      aria-labelledby="booking-drawer-title"
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
                <h2 id="booking-drawer-title" className="text-[#141b2b] text-lg font-bold">
                  Booking Details
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
              className="text-[#6d7a77] hover:text-[#141b2b] p-2 rounded-lg hover:bg-gray-100 transition-colors cursor-pointer"
              aria-label="Close drawer"
            >
              <svg className="w-5 h-5" viewBox="0 0 20 20" fill="currentColor">
                <path
                  fillRule="evenodd"
                  d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
                  clipRule="evenodd"
                />
              </svg>
            </button>
          </div>

          {/* Navigation Tabs */}
          <div className="flex border-b border-[#dce2f7] bg-white px-6 gap-6 text-xs font-semibold overflow-x-auto shrink-0">
            {[
              { id: "overview", label: "Overview" },
              { id: "items", label: `Items (${items.length || booking.ticket_count || 1})` },
              { id: "tickets", label: `Tickets (${tickets.length || booking.ticket_count || 1})` },
              { id: "financials", label: "Financials" },
              { id: "payments", label: `Payments & Refunds (${payments.length + refunds.length})` },
            ].map((tab) => (
              <button
                key={tab.id}
                type="button"
                onClick={() => setActiveTab(tab.id)}
                className={`py-3 border-b-2 font-bold cursor-pointer transition-all whitespace-nowrap ${
                  activeTab === tab.id
                    ? "border-[#00685f] text-[#00685f]"
                    : "border-transparent text-[#6d7a77] hover:text-[#141b2b]"
                }`}
              >
                {tab.label}
              </button>
            ))}
          </div>

          {/* Drawer Body Content */}
          <div className="flex-1 overflow-y-auto p-6 space-y-6">
            {/* TAB 1: OVERVIEW */}
            {activeTab === "overview" && (
              <div className="space-y-5">
                {/* Customer Information Card */}
                <div className="bg-[#f9f9ff] border border-[#bcc9c6]/50 rounded-xl p-4">
                  <h3 className="text-xs font-bold uppercase tracking-wider text-[#3d4947] mb-3 flex items-center gap-1.5">
                    <svg className="w-4 h-4 text-[#00685f]" viewBox="0 0 20 20" fill="currentColor">
                      <path
                        fillRule="evenodd"
                        d="M10 9a3 3 0 100-6 3 3 0 000 6zm-7 9a7 7 0 1114 0H3z"
                        clipRule="evenodd"
                      />
                    </svg>
                    Customer Details
                  </h3>
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 text-xs">
                    <div>
                      <span className="text-[#6d7a77] block text-[11px]">Full Name</span>
                      <span className="font-semibold text-[#141b2b] text-sm">
                        {booking.customer?.full_name || "—"}
                      </span>
                    </div>
                    <div>
                      <span className="text-[#6d7a77] block text-[11px]">Email Address</span>
                      <span className="font-semibold text-[#141b2b]">
                        {booking.customer?.email || "—"}
                      </span>
                    </div>
                    {booking.customer?.phone && (
                      <div>
                        <span className="text-[#6d7a77] block text-[11px]">Phone Number</span>
                        <span className="font-medium text-[#141b2b]">{booking.customer.phone}</span>
                      </div>
                    )}
                    <div>
                      <span className="text-[#6d7a77] block text-[11px]">Booking Date</span>
                      <span className="font-medium text-[#141b2b]">
                        {formatDateTime(booking.created_at)}
                      </span>
                    </div>
                  </div>
                </div>

                {/* Event Information Card */}
                <div className="bg-[#f9f9ff] border border-[#bcc9c6]/50 rounded-xl p-4">
                  <h3 className="text-xs font-bold uppercase tracking-wider text-[#3d4947] mb-3 flex items-center gap-1.5">
                    <svg className="w-4 h-4 text-[#00685f]" viewBox="0 0 20 20" fill="currentColor">
                      <path
                        fillRule="evenodd"
                        d="M6 2a1 1 0 00-1 1v1H4a2 2 0 00-2 2v10a2 2 0 002 2h12a2 2 0 002-2V6a2 2 0 00-2-2h-1V3a1 1 0 10-2 0v1H7V3a1 1 0 00-1-1zm0 5a1 1 0 000 2h8a1 1 0 100-2H6z"
                        clipRule="evenodd"
                      />
                    </svg>
                    Event Information
                  </h3>
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 text-xs">
                    <div className="sm:col-span-2">
                      <span className="text-[#6d7a77] block text-[11px]">Event Title</span>
                      <span className="font-bold text-[#141b2b] text-sm">
                        {booking.event?.title || "—"}
                      </span>
                    </div>
                    <div>
                      <span className="text-[#6d7a77] block text-[11px]">Venue / Location</span>
                      <span className="font-medium text-[#141b2b]">
                        {booking.event?.venue_name || "Online / To be announced"}
                      </span>
                    </div>
                    <div>
                      <span className="text-[#6d7a77] block text-[11px]">Event Type</span>
                      <span className="font-medium text-[#141b2b]">
                        {booking.event?.event_type || "Standard Admission"}
                      </span>
                    </div>
                  </div>
                </div>

                {/* Financial Summary Highlight */}
                <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
                  <div className="bg-white border border-[#bcc9c6]/50 rounded-xl p-3 shadow-xs">
                    <span className="text-[10px] uppercase font-bold text-[#6d7a77] block">
                      Original Total
                    </span>
                    <span className="text-sm font-bold text-[#141b2b] mt-0.5 block">
                      {formatCurrency(financials.original_booking_total, currency)}
                    </span>
                  </div>
                  <div className="bg-white border border-[#bcc9c6]/50 rounded-xl p-3 shadow-xs">
                    <span className="text-[10px] uppercase font-bold text-[#6d7a77] block">
                      Gross Captured
                    </span>
                    <span className="text-sm font-bold text-[#15803d] mt-0.5 block">
                      {formatCurrency(financials.successful_captured_amount, currency)}
                    </span>
                  </div>
                  <div className="bg-white border border-[#bcc9c6]/50 rounded-xl p-3 shadow-xs">
                    <span className="text-[10px] uppercase font-bold text-[#6d7a77] block">
                      Refunded
                    </span>
                    <span className="text-sm font-bold text-[#7e22ce] mt-0.5 block">
                      {formatCurrency(financials.total_refunded_amount, currency)}
                    </span>
                  </div>
                  <div className="bg-white border border-[#bcc9c6]/50 rounded-xl p-3 shadow-xs">
                    <span className="text-[10px] uppercase font-bold text-[#6d7a77] block">
                      Net Captured
                    </span>
                    <span className="text-sm font-bold text-[#00685f] mt-0.5 block">
                      {formatCurrency(financials.net_captured_after_refunds, currency)}
                    </span>
                  </div>
                </div>

                {/* Cancellation Policy Snapshot (if applicable) */}
                {cancellation && (
                  <div className="bg-[#fef9c3]/40 border border-[#fef08a] rounded-xl p-4 text-xs">
                    <h4 className="font-bold text-[#854d0e] mb-1">Cancellation Policy Snapshot</h4>
                    <p className="text-[#713f12] text-[11px] leading-relaxed">
                      Cancellation Allowed: <strong>{(cancellation.cancellation_allowed ?? cancellation.allowed) ? "Yes" : "No"}</strong>
                      {(cancellation.cancellation_allowed ?? cancellation.allowed) && (
                        <>
                          {" "}
                          • Deadline: <strong>{cancellation.cancellation_deadline_hours ?? cancellation.deadline_hours} hours</strong> before
                          event • Refund: <strong>{cancellation.refund_percentage}%</strong> (
                          {cancellation.refund_policy || "Standard Policy"})
                        </>
                      )}
                    </p>
                  </div>
                )}
              </div>
            )}

            {/* TAB 2: ORDER ITEMS */}
            {activeTab === "items" && (
              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <h3 className="text-xs font-bold uppercase tracking-wider text-[#3d4947]">
                    Line Items & Ticket Allocations
                  </h3>
                  <span className="text-xs text-[#6d7a77] font-medium">
                    {items.length} item{items.length === 1 ? "" : "s"}
                  </span>
                </div>

                {items.length === 0 ? (
                  <div className="p-6 text-center text-xs text-[#6d7a77] bg-[#f9f9ff] rounded-xl border border-dashed border-[#bcc9c6]">
                    No line item breakdown available for this booking. Total tickets:{" "}
                    <strong>{booking.ticket_count || 1}</strong>.
                  </div>
                ) : (
                  <div className="border border-[#bcc9c6]/50 rounded-xl overflow-hidden bg-white shadow-xs">
                    <table className="w-full text-left text-xs">
                      <thead className="bg-[#f9f9ff] border-b border-[#bcc9c6]/30 text-[#6d7a77] text-[11px] uppercase">
                        <tr>
                          <th className="py-2.5 px-3 font-semibold">Item / Ticket Type</th>
                          <th className="py-2.5 px-3 font-semibold text-center">Seat</th>
                          <th className="py-2.5 px-3 font-semibold text-center">Qty</th>
                          <th className="py-2.5 px-3 font-semibold text-right">Unit Price</th>
                          <th className="py-2.5 px-3 font-semibold text-right">Subtotal</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-[#bcc9c6]/20">
                        {items.map((item, idx) => (
                          <tr key={item.id || idx} className="hover:bg-[#f9f9ff]">
                            <td className="py-2.5 px-3 font-bold text-[#141b2b]">
                              {item.ticket_type_name || "General Admission"}
                            </td>
                            <td className="py-2.5 px-3 text-center text-[#3d4947]">
                              {item.seat_label ? (
                                <span className="bg-[#eef0fb] text-[#141b2b] px-2 py-0.5 rounded font-mono font-medium">
                                  {item.seat_label}
                                </span>
                              ) : (
                                "—"
                              )}
                            </td>
                            <td className="py-2.5 px-3 text-center font-bold text-[#141b2b]">
                              {item.quantity}
                            </td>
                            <td className="py-2.5 px-3 text-right text-[#6d7a77]">
                              {formatCurrency(item.unit_price, currency)}
                            </td>
                            <td className="py-2.5 px-3 text-right font-bold text-[#141b2b]">
                              {formatCurrency(item.total_price, currency)}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </div>
            )}

            {/* TAB 3: TICKETS / ADMISSIONS */}
            {activeTab === "tickets" && (
              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <h3 className="text-xs font-bold uppercase tracking-wider text-[#3d4947]">
                    Digital Ticket Passes
                  </h3>
                  <span className="text-xs text-[#6d7a77] font-medium">
                    {tickets.length} admission pass{tickets.length === 1 ? "" : "es"}
                  </span>
                </div>

                {tickets.length === 0 ? (
                  <div className="p-6 text-center text-xs text-[#6d7a77] bg-[#f9f9ff] rounded-xl border border-dashed border-[#bcc9c6]">
                    No individual ticket codes generated yet for this booking order.
                  </div>
                ) : (
                  <div className="space-y-2.5">
                    {tickets.map((tck, idx) => {
                      const tStatus = tck.status?.toUpperCase() || "VALID";
                      const tBadge =
                        ticketStatusBadgeStyles[tStatus] ||
                        "bg-[#f3f4f6] text-[#4b5563] border border-[#e5e7eb]";
                      return (
                        <div
                          key={tck.id || idx}
                          className="bg-white border border-[#bcc9c6]/50 rounded-xl p-3.5 shadow-xs flex flex-col sm:flex-row sm:items-center justify-between gap-3"
                        >
                          <div>
                            <div className="flex items-center gap-2">
                              <span className="font-mono font-bold text-xs text-[#00685f]">
                                {tck.ticket_code}
                              </span>
                              <span
                                className={`text-[10px] font-bold px-2 py-0.5 rounded-full ${tBadge}`}
                              >
                                {tStatus}
                              </span>
                              {tck.admission_number && (
                                <span className="text-[10px] text-[#6d7a77] font-medium bg-[#f9f9ff] px-1.5 py-0.5 rounded border border-[#bcc9c6]/30">
                                  #{tck.admission_number}
                                </span>
                              )}
                            </div>
                            <div className="text-xs text-[#3d4947] font-medium mt-1">
                              {tck.ticket_type_name || "General Admission"}
                              {tck.seat_label && ` • Seat: ${tck.seat_label}`}
                            </div>
                            <div className="text-[11px] text-[#6d7a77] mt-0.5">
                              Issued: {formatDateTime(tck.issued_at || booking.created_at)}
                              {tck.used_at && (
                                <span className="text-[#0369a1] font-semibold ml-2">
                                  • Checked in: {formatDateTime(tck.used_at)}
                                </span>
                              )}
                            </div>
                          </div>

                          <div className="text-right shrink-0">
                            <span className="text-xs font-bold text-[#141b2b]">
                              {formatCurrency(tck.price, currency)}
                            </span>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                )}
              </div>
            )}

            {/* TAB 4: FINANCIALS */}
            {activeTab === "financials" && (
              <div className="space-y-4">
                <h3 className="text-xs font-bold uppercase tracking-wider text-[#3d4947]">
                  Complete Financial Breakdown
                </h3>

                <div className="bg-[#f9f9ff] border border-[#bcc9c6]/50 rounded-xl p-5 space-y-3 text-xs">
                  <div className="flex justify-between py-1 border-b border-[#bcc9c6]/20">
                    <span className="text-[#6d7a77]">Items Subtotal</span>
                    <span className="font-bold text-[#141b2b]">
                      {formatCurrency(financials.subtotal, currency)}
                    </span>
                  </div>
                  {parseFloat(financials.platform_fee || "0") > 0 && (
                    <div className="flex justify-between py-1 border-b border-[#bcc9c6]/20">
                      <span className="text-[#6d7a77]">Platform Fee</span>
                      <span className="font-medium text-[#141b2b]">
                        {formatCurrency(financials.platform_fee, currency)}
                      </span>
                    </div>
                  )}
                  {parseFloat(financials.tax || "0") > 0 && (
                    <div className="flex justify-between py-1 border-b border-[#bcc9c6]/20">
                      <span className="text-[#6d7a77]">Taxes & Surcharges</span>
                      <span className="font-medium text-[#141b2b]">
                        {formatCurrency(financials.tax, currency)}
                      </span>
                    </div>
                  )}
                  {parseFloat(financials.discount || "0") > 0 && (
                    <div className="flex justify-between py-1 border-b border-[#bcc9c6]/20">
                      <span className="text-[#15803d]">Coupon Discount</span>
                      <span className="font-bold text-[#15803d]">
                        -{formatCurrency(financials.discount, currency)}
                      </span>
                    </div>
                  )}
                  <div className="flex justify-between py-1.5 border-b border-[#141b2b]/20 text-sm">
                    <span className="font-bold text-[#141b2b]">Original Booking Amount</span>
                    <span className="font-extrabold text-[#141b2b]">
                      {formatCurrency(financials.original_booking_total, currency)}
                    </span>
                  </div>

                  <div className="pt-2 space-y-2">
                    <div className="flex justify-between py-1">
                      <span className="text-[#15803d] font-semibold">Successful Captured Payment</span>
                      <span className="font-bold text-[#15803d]">
                        {formatCurrency(financials.successful_captured_amount, currency)}
                      </span>
                    </div>
                    <div className="flex justify-between py-1">
                      <span className="text-[#7e22ce] font-semibold">Refunded Amount</span>
                      <span className="font-bold text-[#7e22ce]">
                        -{formatCurrency(financials.total_refunded_amount, currency)}
                      </span>
                    </div>
                    <div className="flex justify-between py-2 border-t-2 border-[#00685f]/30 text-sm bg-white p-3 rounded-lg">
                      <div>
                        <span className="font-bold text-[#00685f] block">
                          Net Captured Amount
                        </span>
                        <span className="text-[11px] text-[#6d7a77]">
                          Authoritative captured payment minus refunds (before settlement/fees)
                        </span>
                      </div>
                      <span className="font-extrabold text-[#00685f] text-base self-center">
                        {formatCurrency(financials.net_captured_after_refunds, currency)}
                      </span>
                    </div>
                  </div>
                </div>
              </div>
            )}

            {/* TAB 5: PAYMENTS & REFUNDS */}
            {activeTab === "payments" && (
              <div className="space-y-5">
                {/* Payment Attempts */}
                <div>
                  <h3 className="text-xs font-bold uppercase tracking-wider text-[#3d4947] mb-2.5">
                    Payment Gateway Transactions ({payments.length})
                  </h3>

                  {payments.length === 0 ? (
                    <div className="p-4 text-center text-xs text-[#6d7a77] bg-[#f9f9ff] rounded-xl border border-dashed border-[#bcc9c6]">
                      No gateway transaction log records associated.
                    </div>
                  ) : (
                    <div className="space-y-2">
                      {payments.map((p, idx) => {
                        const pStatus = p.status?.toUpperCase() || "PENDING";
                        const pBadge =
                          paymentStatusBadgeStyles[pStatus] ||
                          "bg-[#f3f4f6] text-[#4b5563] border border-[#e5e7eb]";
                        return (
                          <div
                            key={p.id || idx}
                            className="bg-white border border-[#bcc9c6]/50 rounded-xl p-3 shadow-xs text-xs"
                          >
                            <div className="flex items-center justify-between">
                              <span className="font-bold text-[#141b2b] uppercase">
                                {p.provider || "Gateway"} {p.payment_method ? `• ${p.payment_method}` : ""}
                              </span>
                              <span className={`text-[10px] font-bold px-2 py-0.5 rounded-full ${pBadge}`}>
                                {pStatus}
                              </span>
                            </div>
                            <div className="mt-1.5 flex flex-wrap justify-between text-[#6d7a77] text-[11px]">
                              <div>
                                Transaction ID:{" "}
                                <span className="font-mono text-[#141b2b]">
                                  {p.transaction_id || p.id || "—"}
                                </span>
                              </div>
                              <div className="font-bold text-[#141b2b]">
                                {formatCurrency(p.amount, currency)}
                              </div>
                            </div>
                            {p.paid_at && (
                              <div className="text-[10px] text-[#6d7a77] mt-1">
                                Paid At: {formatDateTime(p.paid_at)}
                              </div>
                            )}
                          </div>
                        );
                      })}
                    </div>
                  )}
                </div>

                {/* Refund Records */}
                <div>
                  <h3 className="text-xs font-bold uppercase tracking-wider text-[#3d4947] mb-2.5">
                    Refund Records ({refunds.length})
                  </h3>

                  {refunds.length === 0 ? (
                    <div className="p-4 text-center text-xs text-[#6d7a77] bg-[#f9f9ff] rounded-xl border border-dashed border-[#bcc9c6]">
                      No refund records initiated or settled for this booking.
                    </div>
                  ) : (
                    <div className="space-y-2">
                      {refunds.map((ref, idx) => {
                        const rStatus = ref.status?.toUpperCase() || "SUCCESS";
                        const rBadge =
                          refundStatusBadgeStyles[rStatus] ||
                          "bg-[#f3f4f6] text-[#4b5563] border border-[#e5e7eb]";
                        return (
                          <div
                            key={ref.id || idx}
                            className="bg-[#faf5ff] border border-[#f3e8ff] rounded-xl p-3 shadow-xs text-xs"
                          >
                            <div className="flex items-center justify-between">
                              <span className="font-bold text-[#86198f]">
                                Refund Amount: {formatCurrency(ref.amount, currency)}
                              </span>
                              <span className={`text-[10px] font-bold px-2 py-0.5 rounded-full ${rBadge}`}>
                                {rStatus}
                              </span>
                            </div>
                            {ref.reason && (
                              <div className="text-[11px] text-[#7e22ce] mt-1">
                                Reason: <em>{ref.reason}</em>
                              </div>
                            )}
                            <div className="flex justify-between text-[10px] text-[#9333ea] mt-1">
                              <span>Ref ID: {ref.gateway_refund_id || ref.id || "—"}</span>
                              <span>Date: {formatDateTime(ref.created_at)}</span>
                            </div>
                          </div>
                        );
                      })}
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>

          {/* Drawer Footer */}
          <div className="px-6 py-4 border-t border-[#dce2f7] bg-[#f9f9ff] flex items-center justify-between shrink-0 text-xs">
            <span className="text-[#6d7a77]">
              Order ID: <span className="font-mono">{booking.id}</span>
            </span>
            <button
              type="button"
              onClick={onClose}
              className="bg-[#00685f] hover:bg-[#005149] text-white px-4 py-2 rounded-lg font-bold transition-colors cursor-pointer"
            >
              Close
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
