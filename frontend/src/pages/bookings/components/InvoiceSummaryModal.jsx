import { useState, useEffect } from "react";
import {
  CloseIcon,
  DownloadIcon,
  TagReceiptIcon,
  CalendarIcon,
  LocationPinIcon,
  ClockIcon,
  VerifiedBadgeIcon,
  InfoCircleIcon,
} from "./BookingIcons";
import { getBookingInvoice, downloadBookingInvoicePdf } from "../../../api/bookingApi";

export default function InvoiceSummaryModal({ bookingId, isOpen, onClose }) {
  const [invoice, setInvoice] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [isDownloading, setIsDownloading] = useState(false);
  const [downloadError, setDownloadError] = useState(null);

  useEffect(() => {
    if (!isOpen || !bookingId) return;

    let isMounted = true;
    setLoading(true);
    setError(null);
    setDownloadError(null);

    getBookingInvoice(bookingId)
      .then((res) => {
        if (!isMounted) return;
        if (res?.data) {
          setInvoice(res.data);
        } else {
          setError("Invoice data not found.");
        }
      })
      .catch((err) => {
        if (!isMounted) return;
        const msg =
          err?.response?.data?.error ||
          err?.response?.data?.message ||
          err?.message ||
          "Failed to load booking receipt.";
        setError(msg);
      })
      .finally(() => {
        if (isMounted) setLoading(false);
      });

    return () => {
      isMounted = false;
    };
  }, [isOpen, bookingId]);

  if (!isOpen) return null;

  const handleDownload = async () => {
    if (!bookingId) return;
    setIsDownloading(true);
    setDownloadError(null);
    try {
      await downloadBookingInvoicePdf(bookingId);
    } catch (err) {
      setDownloadError(err?.message || "Failed to download receipt PDF.");
    } finally {
      setIsDownloading(false);
    }
  };

  const formatDate = (dateStr) => {
    if (!dateStr) return "N/A";
    try {
      const d = new Date(dateStr);
      if (isNaN(d.getTime())) return dateStr;
      return d.toLocaleDateString("en-IN", {
        day: "2-digit",
        month: "short",
        year: "numeric",
      });
    } catch {
      return dateStr;
    }
  };

  const formatDateTime = (dateStr) => {
    if (!dateStr) return "N/A";
    try {
      const d = new Date(dateStr);
      if (isNaN(d.getTime())) return dateStr;
      return d.toLocaleString("en-IN", {
        day: "2-digit",
        month: "short",
        year: "numeric",
        hour: "2-digit",
        minute: "2-digit",
        hour12: true,
      });
    } catch {
      return dateStr;
    }
  };

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="invoice-modal-title"
      className="fixed inset-0 z-50 flex items-center justify-center p-3 sm:p-4 bg-black/50 backdrop-blur-xs animate-in fade-in duration-200"
    >
      <div className="relative w-full max-w-2xl max-h-[90vh] flex flex-col rounded-2xl bg-white shadow-2xl border border-slate-200 overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-slate-200/80 px-6 py-4 bg-slate-50/70">
          <div className="flex items-center gap-3">
            <div className="h-10 w-10 rounded-xl bg-teal-50 text-[#00796d] border border-teal-100 flex items-center justify-center">
              <TagReceiptIcon className="w-5 h-5" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h3 id="invoice-modal-title" className="font-display font-bold text-lg text-slate-900">
                  Booking Receipt
                </h3>
                {invoice?.status && (
                  <span
                    className={`text-[10px] uppercase font-bold px-2 py-0.5 rounded-full ${
                      invoice.status === "ISSUED"
                        ? "bg-emerald-100 text-emerald-800 border border-emerald-200"
                        : "bg-rose-100 text-rose-800 border border-rose-200"
                    }`}
                  >
                    {invoice.status}
                  </span>
                )}
              </div>
              <p className="text-xs text-slate-500 font-mono mt-0.5">
                {invoice?.receipt_number || "Loading receipt..."}
              </p>
            </div>
          </div>

          <button
            type="button"
            onClick={onClose}
            className="p-1.5 text-slate-400 hover:text-slate-700 hover:bg-slate-200/50 rounded-lg transition-colors cursor-pointer"
            aria-label="Close modal"
          >
            <CloseIcon className="w-5 h-5" />
          </button>
        </div>

        {/* Modal Body */}
        <div className="p-6 overflow-y-auto space-y-6 text-sm">
          {loading && (
            <div className="py-12 flex flex-col items-center justify-center text-slate-500 gap-3">
              <div className="w-8 h-8 border-3 border-teal-600 border-t-transparent rounded-full animate-spin" />
              <p className="text-xs font-medium">Fetching receipt summary...</p>
            </div>
          )}

          {error && !loading && (
            <div className="rounded-xl border border-rose-200 bg-rose-50/80 p-5 text-center text-rose-800">
              <p className="font-semibold text-sm">{error}</p>
              <button
                type="button"
                onClick={() => {
                  setLoading(true);
                  setError(null);
                  getBookingInvoice(bookingId)
                    .then((res) => {
                      if (res?.data) setInvoice(res.data);
                    })
                    .catch((err) => setError(err?.message || "Failed to load receipt."))
                    .finally(() => setLoading(false));
                }}
                className="mt-3 inline-flex items-center justify-center rounded-lg bg-rose-600 px-4 py-1.5 text-xs font-semibold text-white hover:bg-rose-700 transition-colors cursor-pointer"
              >
                Try Again
              </button>
            </div>
          )}

          {!loading && !error && invoice && (
            <>
              {/* Metadata Grid */}
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 bg-slate-50 border border-slate-200/80 rounded-xl p-4 text-xs">
                <div className="space-y-1.5">
                  <div className="flex justify-between sm:justify-start sm:gap-3">
                    <span className="text-slate-500">Booking Ref:</span>
                    <strong className="font-mono text-slate-800 font-bold">
                      {invoice.booking?.booking_number || "—"}
                    </strong>
                  </div>
                  <div className="flex justify-between sm:justify-start sm:gap-3">
                    <span className="text-slate-500">Receipt Date:</span>
                    <span className="text-slate-700 font-medium">
                      {formatDateTime(invoice.issued_at)}
                    </span>
                  </div>
                  <div className="flex justify-between sm:justify-start sm:gap-3">
                    <span className="text-slate-500">Payment ID:</span>
                    <span className="font-mono text-slate-700">
                      {invoice.payment?.transaction_id || "Completed"}
                    </span>
                  </div>
                  <div className="flex justify-between sm:justify-start sm:gap-3">
                    <span className="text-slate-500">Method:</span>
                    <span className="text-slate-700 font-medium">
                      {invoice.payment?.method || "ONLINE"}
                    </span>
                  </div>
                </div>

                <div className="space-y-1.5 border-t sm:border-t-0 sm:border-l border-slate-200/80 pt-2 sm:pt-0 sm:pl-4">
                  <div className="flex justify-between sm:justify-start sm:gap-3">
                    <span className="text-slate-500">Billed To:</span>
                    <strong className="text-slate-800 font-semibold truncate">
                      {invoice.customer?.name || "Customer"}
                    </strong>
                  </div>
                  <div className="flex justify-between sm:justify-start sm:gap-3">
                    <span className="text-slate-500">Email:</span>
                    <span className="text-slate-700 truncate">
                      {invoice.customer?.email || "—"}
                    </span>
                  </div>
                  {invoice.customer?.phone && (
                    <div className="flex justify-between sm:justify-start sm:gap-3">
                      <span className="text-slate-500">Phone:</span>
                      <span className="text-slate-700">{invoice.customer.phone}</span>
                    </div>
                  )}
                  <div className="flex justify-between sm:justify-start sm:gap-3">
                    <span className="text-slate-500">Organizer:</span>
                    <span className="text-slate-700 font-medium truncate">
                      {invoice.issuer?.organizer_name || invoice.issuer?.issuer_name || "EventHub"}
                    </span>
                  </div>
                </div>
              </div>

              {/* Event Card */}
              <div className="rounded-xl border border-teal-100 bg-teal-50/40 p-4">
                <h4 className="font-display font-bold text-slate-900 text-sm sm:text-base">
                  {invoice.event?.title || "Event Details"}
                </h4>
                <div className="mt-2.5 flex flex-wrap gap-y-2 gap-x-4 text-xs text-slate-600">
                  <span className="inline-flex items-center gap-1.5 font-medium">
                    <CalendarIcon className="w-3.5 h-3.5 text-[#00796d]" />
                    {formatDate(invoice.event?.date)}
                  </span>
                  {invoice.event?.time && (
                    <span className="inline-flex items-center gap-1.5">
                      <ClockIcon className="w-3.5 h-3.5 text-[#00796d]" />
                      {invoice.event.time}
                    </span>
                  )}
                  <span className="inline-flex items-center gap-1.5">
                    <LocationPinIcon className="w-3.5 h-3.5 text-[#00796d]" />
                    {invoice.event?.is_online
                      ? "Online Event"
                      : [invoice.event?.venue_name, invoice.event?.venue_address]
                          .filter(Boolean)
                          .join(", ") || "Venue Details TBA"}
                  </span>
                </div>
              </div>

              {/* Line Items Table */}
              <div>
                <h5 className="font-semibold text-xs text-slate-500 uppercase tracking-wider mb-2">
                  Items Breakdown
                </h5>
                <div className="border border-slate-200 rounded-xl overflow-hidden shadow-2xs">
                  <table className="w-full text-left border-collapse text-xs">
                    <thead>
                      <tr className="bg-slate-100/80 border-b border-slate-200 text-slate-600 font-semibold">
                        <th className="py-2.5 px-3">Description</th>
                        <th className="py-2.5 px-3">Seat</th>
                        <th className="py-2.5 px-3 text-center">Qty</th>
                        <th className="py-2.5 px-3 text-right">Unit Price</th>
                        <th className="py-2.5 px-3 text-right">Total</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-100">
                      {invoice.items && invoice.items.length > 0 ? (
                        invoice.items.map((it, idx) => (
                          <tr key={it.id || idx} className="hover:bg-slate-50/50">
                            <td className="py-2.5 px-3 font-medium text-slate-900">
                              {it.category_name || "General Admission"}
                            </td>
                            <td className="py-2.5 px-3 text-slate-600 font-mono">
                              {it.seat_label || "—"}
                            </td>
                            <td className="py-2.5 px-3 text-center text-slate-700">
                              {it.quantity}
                            </td>
                            <td className="py-2.5 px-3 text-right text-slate-700">
                              ₹{Number(it.unit_price || 0).toLocaleString("en-IN")}
                            </td>
                            <td className="py-2.5 px-3 text-right font-semibold text-slate-900">
                              ₹{Number(it.total_price || 0).toLocaleString("en-IN")}
                            </td>
                          </tr>
                        ))
                      ) : (
                        <tr>
                          <td colSpan={5} className="py-3 px-3 text-center text-slate-400">
                            General Admission (×1)
                          </td>
                        </tr>
                      )}
                    </tbody>
                  </table>
                </div>
              </div>

              {/* Pricing Breakdown */}
              <div className="rounded-xl bg-slate-50 border border-slate-200/80 p-4 space-y-2 text-xs">
                <div className="flex justify-between text-slate-600">
                  <span>Subtotal</span>
                  <span className="font-medium text-slate-900">
                    ₹{Number(invoice.pricing?.subtotal || 0).toLocaleString("en-IN")}
                  </span>
                </div>
                <div className="flex justify-between text-slate-600">
                  <span>Platform Fee</span>
                  <span className="font-medium text-slate-900">
                    ₹{Number(invoice.pricing?.platform_fee || 0).toLocaleString("en-IN")}
                  </span>
                </div>
                <div className="flex justify-between text-slate-600">
                  <span>Taxes (GST)</span>
                  <span className="font-medium text-slate-900">
                    ₹{Number(invoice.pricing?.tax_amount || 0).toLocaleString("en-IN")}
                  </span>
                </div>
                {Number(invoice.pricing?.discount_amount || 0) > 0 && (
                  <div className="flex justify-between text-emerald-700 font-medium">
                    <span>Discount</span>
                    <span>-₹{Number(invoice.pricing.discount_amount).toLocaleString("en-IN")}</span>
                  </div>
                )}
                <div className="border-t border-slate-200 pt-2.5 mt-2 flex justify-between items-center text-sm font-bold">
                  <span className="text-slate-900">Total Paid</span>
                  <span className="font-display text-base sm:text-lg text-[#00796d]">
                    ₹{Number(invoice.pricing?.paid_amount || invoice.pricing?.total_amount || 0).toLocaleString("en-IN")}
                  </span>
                </div>
              </div>

              {/* Explanatory Legal Notice */}
              <div className="flex items-start gap-2.5 rounded-lg bg-slate-100/90 border border-slate-200/60 p-3 text-slate-500 text-[11px] leading-relaxed">
                <InfoCircleIcon className="w-4 h-4 shrink-0 text-slate-400 mt-0.5" />
                <span>
                  {invoice.issuer?.disclaimer ||
                    "This document is an official booking and payment receipt confirming your reservation and payment receipt. Not a GST tax invoice."}
                </span>
              </div>

              {downloadError && (
                <p className="text-xs text-rose-600 text-center font-medium">
                  {downloadError}
                </p>
              )}
            </>
          )}
        </div>

        {/* Modal Footer */}
        <div className="flex flex-wrap items-center justify-between gap-3 border-t border-slate-200/80 px-6 py-4 bg-slate-50/70">
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg border border-slate-300 bg-white px-4 py-2 text-xs font-semibold text-slate-700 hover:bg-slate-100 transition-colors cursor-pointer"
          >
            Close
          </button>

          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => window.print()}
              disabled={loading || !!error}
              className="hidden sm:inline-flex items-center justify-center rounded-lg border border-slate-300 bg-white px-4 py-2 text-xs font-semibold text-slate-700 hover:bg-slate-100 transition-colors cursor-pointer disabled:opacity-50"
            >
              Print
            </button>
            <button
              type="button"
              onClick={handleDownload}
              disabled={loading || !!error || isDownloading}
              className="inline-flex items-center justify-center gap-2 rounded-lg bg-[#00796d] px-5 py-2 text-xs font-bold text-white shadow-xs hover:bg-[#00685f] transition-colors cursor-pointer disabled:opacity-50"
            >
              <DownloadIcon className="w-4 h-4" />
              {isDownloading ? "Downloading..." : "Download Receipt (PDF)"}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
