import { useState, useEffect } from "react";
import { useParams, Link } from "react-router-dom";
import AppHeader from "../../components/layout/AppHeader";
import Footer from "../../components/layout/Footer";
import { getBookingById, downloadBookingPdf } from "../../api/bookingApi";
import { getPublicEventById } from "../../api/eventApi";
import { getMyTickets, downloadTicketPdf } from "../../api/ticketApi";
import BookingBreadcrumbs from "./components/BookingBreadcrumbs";
import {
  CalendarIcon,
  LocationPinIcon,
  BuildingIcon,
  TicketIconOutline,
  DownloadIcon,
} from "./components/BookingIcons";

const formatDate = (dateStr) => {
  if (!dateStr) return "Date TBA";
  const d = new Date(dateStr);
  if (Number.isNaN(d.getTime())) return dateStr;
  return d.toLocaleDateString("en-IN", {
    day: "2-digit",
    month: "short",
    year: "numeric",
  });
};

const formatFullDate = (dateStr) => {
  if (!dateStr) return "Date TBA";
  const d = new Date(dateStr);
  if (Number.isNaN(d.getTime())) return dateStr;
  return d.toLocaleDateString("en-IN", {
    weekday: "long",
    day: "numeric",
    month: "long",
    year: "numeric",
  });
};

const formatTime = (timeStr) => {
  if (!timeStr) return "";
  const parts = timeStr.split(":");
  if (parts.length < 2) return timeStr;
  const d = new Date();
  d.setHours(Number(parts[0]), Number(parts[1]), 0, 0);
  return d.toLocaleTimeString("en-IN", {
    hour: "numeric",
    minute: "2-digit",
    hour12: true,
  });
};

const formatScheduleTime = (schedule) => {
  if (!schedule) return "Time TBA";
  if (schedule.is_all_day) return "All Day";
  const start = formatTime(schedule.start_time);
  const end = formatTime(schedule.end_time);
  if (start && end) return `${start} – ${end}`;
  if (start) return start;
  return "Time TBA";
};

export default function BookingDetailsPage() {
  const { bookingId } = useParams();

  const [booking, setBooking] = useState(null);
  const [event, setEvent] = useState(null);
  const [tickets, setTickets] = useState([]);
  const [loading, setLoading] = useState(true);
  const [isNotFound, setIsNotFound] = useState(false);
  const [error, setError] = useState(null);

  const [isDownloading, setIsDownloading] = useState(false);
  const [downloadSuccess, setDownloadSuccess] = useState(false);
  const [downloadError, setDownloadError] = useState(null);
  const [reloadKey, setReloadKey] = useState(0);

  useEffect(() => {
    let cancelled = false;

    const loadBookingData = async () => {
      if (!bookingId) {
        setIsNotFound(true);
        setLoading(false);
        return;
      }

      setLoading(true);
      setError(null);
      setIsNotFound(false);

      try {
        const bookingRes = await getBookingById(bookingId);
        if (cancelled) return;

        const bookingData = bookingRes?.data;
        if (!bookingData) {
          setIsNotFound(true);
          return;
        }

        setBooking(bookingData);

        // Fetch associated event and customer's tickets in parallel
        const eventId = bookingData.EventID || bookingData.event_id || bookingData.eventId;
        const currentBookingId = bookingData.ID || bookingData.id;

        const [eventRes, ticketsRes] = await Promise.allSettled([
          eventId ? getPublicEventById(eventId) : Promise.reject(new Error("No event ID")),
          getMyTickets(),
        ]);

        if (cancelled) return;

        if (eventRes.status === "fulfilled" && eventRes.value?.data) {
          setEvent(eventRes.value.data);
        }

        if (ticketsRes.status === "fulfilled" && Array.isArray(ticketsRes.value?.data)) {
          const matchingTickets = ticketsRes.value.data.filter(
            (t) =>
              (t.BookingID || t.booking_id || t.bookingId) === currentBookingId
          );
          setTickets(matchingTickets);
        }
      } catch (err) {
        if (cancelled) return;
        const status = err?.response?.status;
        if (status === 404) {
          setIsNotFound(true);
        } else {
          const msg =
            err?.response?.data?.error ||
            err?.response?.data?.message ||
            err?.message ||
            "Failed to load booking details.";
          setError(msg);
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };

    loadBookingData();

    return () => {
      cancelled = true;
    };
  }, [bookingId, reloadKey]);

  const handleDownload = async () => {
    const targetBookingId = booking?.ID || booking?.id || bookingId;
    if (!targetBookingId) {
      setDownloadError("No booking found for PDF download.");
      return;
    }

    setIsDownloading(true);
    setDownloadError(null);
    setDownloadSuccess(false);

    try {
      await downloadBookingPdf(targetBookingId);
      setDownloadSuccess(true);
      setTimeout(() => {
        setDownloadSuccess(false);
      }, 4000);
    } catch (err) {
      const targetTicket = tickets[0];
      const ticketId = targetTicket?.ID || targetTicket?.id;
      if (ticketId) {
        try {
          await downloadTicketPdf(ticketId);
          setDownloadSuccess(true);
          setTimeout(() => {
            setDownloadSuccess(false);
          }, 4000);
          return;
        } catch {
          // ignore secondary failure
        }
      }
      setDownloadError(err?.message || "Failed to download ticket pass. Please try again.");
    } finally {
      setIsDownloading(false);
    }
  };

  const breadcrumbs = [
    { label: "My Bookings", to: "/my-bookings" },
    { label: "Booking Details" },
  ];

  if (loading) {
    return (
      <div className="flex min-h-screen flex-col bg-[#f9faff] text-[#141b2b]">
        <AppHeader />
        <main className="mx-auto w-full max-w-7xl flex-1 px-4 sm:px-6 lg:px-8 py-16 flex items-center justify-center">
          <div className="flex flex-col items-center gap-4 bg-white border border-[#dfe5e3] rounded-xl p-10 shadow-xs max-w-md w-full text-center">
            <div className="size-10 border-3 border-[#00685f]/30 border-t-[#00685f] rounded-full animate-spin" />
            <h2 className="font-display font-semibold text-lg text-ink">
              Loading booking details...
            </h2>
            <p className="text-xs text-[#565e74]">
              Retrieving verified line items and event confirmation pass.
            </p>
          </div>
        </main>
        <Footer />
      </div>
    );
  }

  if (isNotFound || !booking) {
    return (
      <div className="flex min-h-screen flex-col bg-[#f9faff] text-[#141b2b]">
        <AppHeader />
        <main className="mx-auto w-full max-w-7xl flex-1 px-4 sm:px-6 lg:px-8 py-16 flex items-center justify-center">
          <div className="flex flex-col items-center gap-4 bg-white border border-[#dfe5e3] rounded-xl p-8 sm:p-12 shadow-xs max-w-md w-full text-center">
            <div className="grid h-16 w-16 place-items-center rounded-full bg-[#f1f3ff] text-[#00685f] text-2xl mb-2">
              🎫
            </div>
            <h2 className="font-display font-bold text-xl sm:text-2xl text-ink">
              Booking Not Found
            </h2>
            <p className="text-sm text-[#565e74] leading-relaxed">
              We couldn't locate this booking record. It may not exist or belongs to another customer account.
            </p>
            <div className="mt-4">
              <Link
                to="/my-bookings"
                className="inline-flex rounded-lg bg-[#00685f] px-5 py-2.5 text-sm font-bold text-white shadow-xs hover:bg-[#00544c] transition-colors"
              >
                Return to My Bookings
              </Link>
            </div>
          </div>
        </main>
        <Footer />
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex min-h-screen flex-col bg-[#f9faff] text-[#141b2b]">
        <AppHeader />
        <main className="mx-auto w-full max-w-7xl flex-1 px-4 sm:px-6 lg:px-8 py-16 flex items-center justify-center">
          <div className="flex flex-col items-center gap-4 bg-white border border-red-200 rounded-xl p-8 sm:p-12 shadow-xs max-w-md w-full text-center">
            <div className="grid h-16 w-16 place-items-center rounded-full bg-red-100 text-red-600 text-2xl mb-2">
              ⚠
            </div>
            <h2 className="font-display font-bold text-xl text-red-900">
              Error Loading Booking
            </h2>
            <p className="text-sm text-red-700 leading-relaxed">{error}</p>
            <div className="mt-4 flex gap-3">
              <button
                type="button"
                onClick={() => setReloadKey((k) => k + 1)}
                className="inline-flex rounded-lg bg-[#00685f] px-5 py-2.5 text-sm font-bold text-white shadow-xs hover:bg-[#00544c] transition-colors cursor-pointer"
              >
                Try Again
              </button>
              <Link
                to="/my-bookings"
                className="inline-flex rounded-lg border border-[#dfe5e3] bg-white px-5 py-2.5 text-sm font-semibold text-[#565e74] hover:bg-slate-50 transition-colors"
              >
                Back to List
              </Link>
            </div>
          </div>
        </main>
        <Footer />
      </div>
    );
  }

  // Normalization of booking and event attributes
  const currentBookingId = booking.ID || booking.id;
  const bookingNumber = booking.BookingNumber || booking.booking_number || currentBookingId;
  const bookingStatus = (booking.Status || booking.status || "CONFIRMED").toUpperCase();
  const bookedOnStr = formatDate(booking.CreatedAt || booking.created_at);
  const bookedOnFullStr = formatFullDate(booking.CreatedAt || booking.created_at);

  const eventTitle = event?.title || `Event Booking #${bookingNumber}`;
  const eventImage = event?.cover_image_url || event?.banner_image_url || "/assets/events/event-music.jpg";
  const eventDateStr = event?.schedule?.event_date
    ? formatFullDate(event.schedule.event_date)
    : bookedOnFullStr;
  const eventTimeStr = formatScheduleTime(event?.schedule);
  const venueLocationStr = event?.venue
    ? [event.venue.name, event.venue.address, event.venue.city].filter(Boolean).join(", ")
    : event?.event_type === "ONLINE"
    ? "Online Event"
    : "Venue TBA";
  const organizerName = event?.organizer_name || event?.organizer?.name || "Event Organizer";

  // Items breakdown
  const items = Array.isArray(booking.Items)
    ? booking.Items
    : Array.isArray(booking.items)
    ? booking.items
    : [];

  const totalQuantity = items.length > 0
    ? items.reduce((acc, it) => acc + (Number(it.Quantity || it.quantity) || 1), 0)
    : tickets.length || 1;

  // Resolve ticket type name from event ticket types if available
  const firstItem = items[0];
  const ticketTypeId = firstItem?.TicketTypeID || firstItem?.ticket_type_id;
  const matchedTier = event?.ticket_types?.find(
    (t) => (t.id || t.ID) === ticketTypeId
  );
  const ticketTypeName = matchedTier?.name || "General Admission";
  const unitPrice = firstItem
    ? Number(firstItem.UnitPrice || firstItem.unit_price || 0)
    : Number(booking.Subtotal || booking.subtotal || 0) / (totalQuantity || 1);

  // Financials
  const subtotalNum = Number(booking.Subtotal || booking.subtotal || 0);
  const platformFeeNum = Number(booking.PlatformFee || booking.platform_fee || 0);
  const taxesNum = Number(booking.Tax || booking.tax || 0);
  const totalNum = Number(booking.Total || booking.total || 0);

  // Resolved seats list from booking
  const assignedSeats = booking?.Seats || booking?.seats || [];

  return (
    <div className="flex min-h-screen flex-col bg-[#f9faff] text-[#141b2b]">
      <AppHeader />

      <main className="mx-auto w-full max-w-7xl flex-1 px-4 sm:px-6 lg:px-8 py-6 sm:py-9">
        {/* Breadcrumb Navigation */}
        <BookingBreadcrumbs items={breadcrumbs} />

        {/* Page Header */}
        <div className="flex flex-col sm:flex-row sm:items-end justify-between gap-4 border-b border-[#dfe5e3] pb-6">
          <div>
            <h1 className="font-display text-2xl sm:text-3xl font-bold tracking-tight text-ink">
              Booking Details
            </h1>
            <p className="mt-1 text-xs sm:text-sm text-[#565e74]">
              Booking ID:{" "}
              <strong className="font-semibold text-[#3d4947] font-mono">{bookingNumber}</strong>
              <span className="mx-2">•</span>
              Booked on {bookedOnFullStr}
            </p>
          </div>

          <span
            className={`self-start sm:self-auto rounded-full px-3.5 py-1 text-xs font-semibold tracking-wide uppercase ${
              bookingStatus === "CONFIRMED"
                ? "bg-[#e0efed] text-[#00685f]"
                : bookingStatus === "CANCELLED"
                ? "bg-red-100 text-red-700"
                : "bg-slate-100 text-slate-700"
            }`}
          >
            {bookingStatus}
          </span>
        </div>

        {/* 2-Column Responsive Layout */}
        <div className="mt-6 sm:mt-8 grid grid-cols-1 lg:grid-cols-[2fr_1fr] gap-6 sm:gap-8 items-start">
          {/* Left Column: Event Card + Ticket Information */}
          <div className="space-y-6">
            {/* Event Summary Card */}
            <section className="overflow-hidden rounded-xl border border-[#dfe5e3] bg-white shadow-xs transition sm:flex">
              <img
                src={eventImage}
                alt={eventTitle}
                className="h-52 w-full object-cover sm:h-auto sm:w-2/5 min-h-[220px] bg-slate-100"
                loading="lazy"
                onError={(e) => {
                  e.currentTarget.src = "/assets/events/event-music.jpg";
                }}
              />
              <div className="flex-1 p-5 sm:p-6 flex flex-col justify-between">
                <div>
                  <h2 className="font-display text-xl sm:text-2xl font-semibold text-ink">
                    {eventTitle}
                  </h2>

                  <div className="mt-4 space-y-2.5 text-xs sm:text-sm text-[#3d4947]">
                    <p className="flex items-center gap-2">
                      <CalendarIcon className="w-4 h-4 text-[#565e74] shrink-0" />
                      <span>{eventDateStr} • {eventTimeStr}</span>
                    </p>
                    <p className="flex items-center gap-2">
                      <LocationPinIcon className="w-4 h-4 text-[#565e74] shrink-0" />
                      <span>{venueLocationStr}</span>
                    </p>
                    <p className="flex items-center gap-2">
                      <BuildingIcon className="w-4 h-4 text-[#565e74] shrink-0" />
                      <span>By {organizerName}</span>
                    </p>
                  </div>
                </div>

                <div className="mt-6 flex justify-end">
                  <Link
                    to={event?.id ? `/events/${event.id}` : "/events"}
                    className="inline-flex items-center rounded-lg bg-[#eef2ff] px-6 py-2.5 text-xs sm:text-sm font-semibold text-[#00685f] hover:bg-[#e0e7fa] transition-colors cursor-pointer"
                  >
                    View Event
                  </Link>
                </div>
              </div>
            </section>

            {/* Ticket Information Card */}
            <section className="rounded-xl border border-[#dfe5e3] bg-white p-5 sm:p-6 shadow-xs">
              <h3 className="font-display font-semibold text-base sm:text-lg text-ink">
                Ticket Information
              </h3>
              <div className="mt-5 grid grid-cols-2 sm:grid-cols-4 gap-4 border-t border-[#dfe5e3] pt-5 text-xs sm:text-sm">
                <div>
                  <span className="text-[#565e74] block">Type</span>
                  <strong className="mt-1 block font-bold text-ink text-sm sm:text-base">
                    {ticketTypeName}
                  </strong>
                </div>
                <div>
                  <span className="text-[#565e74] block">Quantity</span>
                  <strong className="mt-1 block font-bold text-ink text-sm sm:text-base">
                    {totalQuantity}
                  </strong>
                </div>
                <div>
                  <span className="text-[#565e74] block">Passes Issued</span>
                  <strong className="mt-1 block font-bold text-ink text-sm sm:text-base">
                    {tickets.length > 0 ? `${tickets.length} Digital Passes` : "General Admission"}
                  </strong>
                </div>
                <div>
                  <span className="text-[#565e74] block">Price / Ticket</span>
                  <strong className="mt-1 block font-bold text-ink text-sm sm:text-base">
                    ₹{unitPrice.toLocaleString("en-IN")}
                  </strong>
                </div>
              </div>

              {/* Assigned Seats Breakdown */}
              {assignedSeats.length > 0 ? (
                <div className="mt-5 rounded-lg border border-[#00685f]/20 bg-[#f0f9f8] p-4">
                  <span className="text-xs font-semibold text-[#00685f] uppercase tracking-wider block mb-2.5">
                    Assigned Seats ({assignedSeats.length})
                  </span>
                  <div className="flex flex-wrap gap-2">
                    {assignedSeats.map((seat, sIdx) => (
                      <span
                        key={seat.seat_id || sIdx}
                        className="inline-flex items-center gap-1.5 rounded-md bg-white px-3 py-1.5 text-xs font-semibold text-ink border border-[#00685f]/30 shadow-xs"
                      >
                        <span className="size-1.5 rounded-full bg-[#008378]" />
                        {seat.label || `Row ${seat.row_name}, Seat ${seat.seat_number}`}
                      </span>
                    ))}
                  </div>
                </div>
              ) : (
                <div className="mt-5 rounded-lg border border-slate-200 bg-[#f9faff] p-3 text-xs text-[#565e74] flex items-center gap-2">
                  <span className="size-1.5 rounded-full bg-[#008378]" />
                  <span>Seating Arrangement: <strong className="text-slate-800">General Admission</strong> (No assigned seats)</span>
                </div>
              )}

              {/* Tickets list if present */}
              {tickets.length > 0 && (
                <div className="mt-6 border-t border-[#dfe5e3] pt-5">
                  <span className="text-xs font-semibold text-[#565e74] uppercase tracking-wider block mb-3">
                    Assigned Ticket Passes ({tickets.length})
                  </span>
                  <div className="space-y-2">
                    {tickets.map((t, idx) => {
                      const itemSeat = t.Seat || t.seat || (assignedSeats[idx] || null);
                      return (
                        <div
                          key={t.ID || t.id || idx}
                          className="flex items-center justify-between p-3 rounded-lg bg-[#f9faff] border border-[#dfe5e3] text-xs sm:text-sm"
                        >
                          <div className="flex items-center gap-2.5">
                            <span className="size-2 rounded-full bg-[#008378]" />
                            <span className="font-medium text-ink">
                              Ticket #{idx + 1} • {t.TicketCode || t.ticket_code || "EH-PASS"}
                            </span>
                            {itemSeat && (
                              <span className="rounded bg-[#e0efed] px-2 py-0.5 text-[11px] font-semibold text-[#00685f]">
                                {itemSeat.label || `Row ${itemSeat.row_name}, Seat ${itemSeat.seat_number}`}
                              </span>
                            )}
                          </div>
                          <Link
                            to={`/my-tickets/${t.ID || t.id}`}
                            className="font-semibold text-[#00685f] hover:underline"
                          >
                            View Pass →
                          </Link>
                        </div>
                      );
                    })}
                  </div>
                </div>
              )}
            </section>
          </div>

          {/* Right Column: Digital Ticket Promo + Payment Summary */}
          <div className="space-y-6">
            {/* Digital Ticket Promo Card */}
            <section className="rounded-xl bg-gradient-to-r from-[#00685f] to-[#248b80] p-6 text-center text-white shadow-xs">
              <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-white/20 text-white mb-2">
                <TicketIconOutline className="w-6 h-6" />
              </div>
              <strong className="mt-2 block font-display text-lg sm:text-xl font-bold">
                Your digital ticket is ready
              </strong>
              <p className="mt-1 text-xs sm:text-sm text-white/80">
                Keep it handy for event entry.
              </p>

              <div className="mt-6 space-y-3">
                <Link
                  to={`/bookings/${currentBookingId}/ticket`}
                  className="block w-full rounded-lg bg-white py-3 text-xs sm:text-sm font-bold text-[#00685f] hover:bg-slate-50 transition-colors shadow-xs"
                >
                  View Ticket
                </Link>

                <button
                  type="button"
                  onClick={handleDownload}
                  disabled={isDownloading}
                  className={`w-full inline-flex items-center justify-center gap-2 rounded-lg border border-white/30 py-3 text-xs sm:text-sm font-semibold text-white transition-colors cursor-pointer ${
                    isDownloading ? "opacity-75" : "hover:bg-white/10"
                  }`}
                >
                  <DownloadIcon className="w-4 h-4" />
                  <span>{isDownloading ? "Downloading PDF..." : "Download Booking PDF"}</span>
                </button>
              </div>

              {downloadSuccess && (
                <p className="mt-3 text-xs text-white font-medium animate-pulse">
                  ✓ Ticket PDF downloaded successfully!
                </p>
              )}

              {downloadError && (
                <p className="mt-3 text-xs text-rose-200 font-medium">
                  {downloadError}
                </p>
              )}
            </section>

            {/* Payment Summary Card */}
            <section className="rounded-xl border border-[#dfe5e3] bg-white p-5 sm:p-6 shadow-xs">
              <h3 className="font-display font-semibold text-base sm:text-lg text-ink">
                Payment Summary
              </h3>

              <div className="mt-4 space-y-3 border-t border-[#dfe5e3] pt-4 text-xs sm:text-sm">
                <div className="flex justify-between text-[#565e74]">
                  <span>Ticket Amount (×{totalQuantity})</span>
                  <span className="font-medium text-[#3d4947]">
                    ₹{subtotalNum.toLocaleString("en-IN")}
                  </span>
                </div>
                <div className="flex justify-between text-[#565e74]">
                  <span>Platform Fee</span>
                  <span className="font-medium text-[#3d4947]">
                    ₹{platformFeeNum.toLocaleString("en-IN")}
                  </span>
                </div>
                <div className="flex justify-between text-[#565e74]">
                  <span>Taxes</span>
                  <span className="font-medium text-[#3d4947]">
                    ₹{taxesNum.toLocaleString("en-IN")}
                  </span>
                </div>
              </div>

              <div className="mt-5 flex items-center justify-between border-t border-[#dfe5e3] pt-5">
                <strong className="font-semibold text-ink text-sm sm:text-base">
                  Total Paid
                </strong>
                <strong className="font-display text-2xl font-bold text-[#00685f]">
                  ₹{totalNum.toLocaleString("en-IN")}
                </strong>
              </div>

              <div className="mt-5 flex items-center justify-between rounded-lg bg-[#eef1ff] p-3 text-xs sm:text-sm text-[#565e74]">
                <span className="font-medium">
                  ▣ Booked: {bookedOnStr}
                </span>
                <strong className="font-bold text-[#00685f]">
                  {bookingStatus === "CONFIRMED" ? "PAID" : bookingStatus}
                </strong>
              </div>
            </section>
          </div>
        </div>
      </main>

      <Footer />
    </div>
  );
}
