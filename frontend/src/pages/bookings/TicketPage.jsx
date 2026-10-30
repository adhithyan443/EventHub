import { useState, useEffect } from "react";
import { useParams, Link } from "react-router-dom";
import AppHeader from "../../components/layout/AppHeader";
import Footer from "../../components/layout/Footer";
import { getBookingById, getBookingQrBlob, downloadBookingPdf } from "../../api/bookingApi";
import { getMyTicketById, getMyTickets, downloadTicketPdf } from "../../api/ticketApi";
import { getPublicEventById } from "../../api/eventApi";
import BookingBreadcrumbs from "./components/BookingBreadcrumbs";
import notify from "../../utils/toast";
import {
  CalendarIcon,
  ClockIcon,
  LocationPinIcon,
  DownloadIcon,
  InfoCircleIcon,
  TicketIconOutline,
  AlertTriangleIcon,
} from "./components/BookingIcons";

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

export default function TicketPage() {
  const { bookingId: routeBookingId, ticketId: routeTicketId } = useParams();

  const [booking, setBooking] = useState(null);
  const [event, setEvent] = useState(null);
  const [tickets, setTickets] = useState([]);
  const [activeTicketId, setActiveTicketId] = useState(routeTicketId || null);

  const [loading, setLoading] = useState(true);
  const [isNotFound, setIsNotFound] = useState(false);
  const [error, setError] = useState(null);

  // Authoritative Booking QR state
  const [qrCodeUrl, setQrCodeUrl] = useState(null);
  const [qrLoading, setQrLoading] = useState(false);
  const [qrError, setQrError] = useState(null);

  const [isDownloading, setIsDownloading] = useState(false);
  const [reloadKey, setReloadKey] = useState(0);

  useEffect(() => {
    let cancelled = false;

    const loadTicketAndBookingData = async () => {
      setLoading(true);
      setError(null);
      setIsNotFound(false);

      try {
        let loadedBooking = null;
        let loadedEvent = null;
        let loadedTickets = [];
        let primaryTicketId = routeTicketId;

        // Path A: Entered via /my-tickets/:ticketId
        if (routeTicketId) {
          const ticketRes = await getMyTicketById(routeTicketId);
          if (cancelled) return;

          const singleTicket = ticketRes?.data;
          if (!singleTicket) {
            setIsNotFound(true);
            return;
          }

          primaryTicketId = singleTicket.ID || singleTicket.id;
          const bId = singleTicket.BookingID || singleTicket.booking_id;
          const eId = singleTicket.EventID || singleTicket.event_id;

          const [bRes, eRes, allTicketsRes] = await Promise.allSettled([
            bId ? getBookingById(bId) : Promise.reject(new Error("No booking ID")),
            eId ? getPublicEventById(eId) : Promise.reject(new Error("No event ID")),
            getMyTickets(),
          ]);

          if (cancelled) return;

          if (bRes.status === "fulfilled" && bRes.value?.data) {
            loadedBooking = bRes.value.data;
          }
          if (eRes.status === "fulfilled" && eRes.value?.data) {
            loadedEvent = eRes.value.data;
          }
          if (allTicketsRes.status === "fulfilled" && Array.isArray(allTicketsRes.value?.data)) {
            loadedTickets = allTicketsRes.value.data.filter(
              (t) => (t.BookingID || t.booking_id) === bId
            );
          }
          if (loadedTickets.length === 0) {
            loadedTickets = [singleTicket];
          }
        }
        // Path B: Entered via /bookings/:bookingId/ticket
        else if (routeBookingId) {
          const bRes = await getBookingById(routeBookingId);
          if (cancelled) return;

          loadedBooking = bRes?.data;
          if (!loadedBooking) {
            setIsNotFound(true);
            return;
          }

          const currentBId = loadedBooking.ID || loadedBooking.id;
          const eId = loadedBooking.EventID || loadedBooking.event_id;

          const [eRes, allTicketsRes] = await Promise.allSettled([
            eId ? getPublicEventById(eId) : Promise.reject(new Error("No event ID")),
            getMyTickets(),
          ]);

          if (cancelled) return;

          if (eRes.status === "fulfilled" && eRes.value?.data) {
            loadedEvent = eRes.value.data;
          }
          if (allTicketsRes.status === "fulfilled" && Array.isArray(allTicketsRes.value?.data)) {
            loadedTickets = allTicketsRes.value.data.filter(
              (t) => (t.BookingID || t.booking_id) === currentBId
            );
          }

          if (loadedTickets.length > 0) {
            primaryTicketId = loadedTickets[0].ID || loadedTickets[0].id;
          }
        } else {
          setIsNotFound(true);
          return;
        }

        setBooking(loadedBooking);
        setEvent(loadedEvent);
        setTickets(loadedTickets);
        setActiveTicketId(primaryTicketId || (loadedTickets[0]?.ID || loadedTickets[0]?.id));
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
            "Failed to load ticket pass.";
          setError(msg);
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };

    loadTicketAndBookingData();

    return () => {
      cancelled = true;
    };
  }, [routeBookingId, routeTicketId, reloadKey]);

  const bookingId = booking?.ID || booking?.id || routeBookingId;
  const bookingNumber = booking?.BookingNumber || booking?.booking_number || bookingId || "CONFIRMED";

  // Fetch authoritative signed booking QR image
  useEffect(() => {
    if (!bookingId) return;

    let cancelled = false;
    let objectUrl = null;

    const fetchBookingQR = async () => {
      setQrLoading(true);
      setQrError(null);

      try {
        const blob = await getBookingQrBlob(bookingId);
        if (cancelled) return;
        objectUrl = URL.createObjectURL(blob);
        setQrCodeUrl(objectUrl);
      } catch (err) {
        if (cancelled) return;
        setQrError(
          err?.response?.data?.error ||
          err?.message ||
          "Failed to load booking QR code."
        );
      } finally {
        if (!cancelled) {
          setQrLoading(false);
        }
      }
    };

    fetchBookingQR();

    return () => {
      cancelled = true;
      if (objectUrl) {
        URL.revokeObjectURL(objectUrl);
      }
    };
  }, [bookingId, reloadKey]);

  // Identify active ticket object
  const activeTicket =
    tickets.find((t) => (t.ID || t.id) === activeTicketId) ||
    tickets[0] ||
    null;

  const handleDownload = async () => {
    if (!bookingId) {
      notify.error("No booking reference available for download.");
      return;
    }

    setIsDownloading(true);

    try {
      await downloadBookingPdf(bookingId);
      notify.success("Ticket pass download started.");
    } catch (err) {
      notify.error(
        err?.message || "Failed to download booking PDF pass. Please try again."
      );
    } finally {
      setIsDownloading(false);
    }
  };

  const handleDownloadSingleTicket = async (singleTicketId) => {
    if (!singleTicketId) return;
    setIsDownloading(true);

    try {
      await downloadTicketPdf(singleTicketId);
      notify.success("Individual ticket pass download started.");
    } catch (err) {
      notify.error(
        err?.message || "Failed to download individual ticket pass."
      );
    } finally {
      setIsDownloading(false);
    }
  };

  const breadcrumbs = [
    { label: "My Bookings", to: "/my-bookings" },
    ...(bookingId ? [{ label: "Booking Details", to: `/bookings/${bookingId}` }] : []),
    { label: "View Ticket" },
  ];

  if (loading) {
    return (
      <div className="flex min-h-screen flex-col bg-[#f9faff] text-[#141b2b]">
        <AppHeader />
        <main className="mx-auto w-full max-w-7xl flex-1 px-4 sm:px-6 lg:px-8 py-16 flex items-center justify-center">
          <div className="flex flex-col items-center gap-4 bg-white border border-[#dfe5e3] rounded-xl p-10 shadow-xs max-w-md w-full text-center">
            <div className="size-10 border-3 border-[#00685f]/30 border-t-[#00685f] rounded-full animate-spin" />
            <h2 className="font-display font-semibold text-lg text-ink">
              Loading ticket pass...
            </h2>
            <p className="text-xs text-[#565e74]">
              Verifying cryptographic digital entrance credential.
            </p>
          </div>
        </main>
        <Footer />
      </div>
    );
  }

  if (isNotFound || (!activeTicket && tickets.length === 0)) {
    return (
      <div className="flex min-h-screen flex-col bg-[#f9faff] text-[#141b2b]">
        <AppHeader />
        <main className="mx-auto w-full max-w-7xl flex-1 px-4 sm:px-6 lg:px-8 py-16 flex items-center justify-center">
          <div className="flex flex-col items-center gap-4 bg-white border border-[#dfe5e3] rounded-xl p-8 sm:p-12 shadow-xs max-w-md w-full text-center">
            <div className="grid h-16 w-16 place-items-center rounded-full bg-[#f1f3ff] text-[#00685f] text-2xl mb-2">
              🎫
            </div>
            <h2 className="font-display font-bold text-xl sm:text-2xl text-ink">
              Ticket Pass Not Found
            </h2>
            <p className="text-sm text-[#565e74] leading-relaxed">
              We couldn't locate this ticket pass. It may have expired or belongs to another customer account.
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
              Error Loading Ticket
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
                Back to Bookings
              </Link>
            </div>
          </div>
        </main>
        <Footer />
      </div>
    );
  }

  // Event & Ticket presentation variables
  const eventTitle = event?.title || "EventHub Ticket Pass";
  const eventImage = event?.cover_image_url || event?.banner_image_url || "/assets/events/event-music.jpg";
  const eventDateStr = event?.schedule?.event_date
    ? formatFullDate(event.schedule.event_date)
    : "Date TBA";
  const eventTimeStr = formatScheduleTime(event?.schedule);
  const venueLocationStr = event?.venue
    ? [event.venue.name, event.venue.city].filter(Boolean).join(", ")
    : event?.event_type === "ONLINE"
    ? "Online Event"
    : "Venue TBA";
  const organizerName = event?.organizer_name || event?.organizer?.name || "Event Organizer";

  // Active ticket properties
  const isCancelled =
    (
      booking?.Status ||
      booking?.status ||
      activeTicket?.Status ||
      activeTicket?.status ||
      ""
    ).toUpperCase() === "CANCELLED";

  const ticketStatus = isCancelled
    ? "CANCELLED"
    : (booking?.Status || booking?.status || activeTicket?.Status || activeTicket?.status || "CONFIRMED").toUpperCase();

  const ticketPrice = Number(activeTicket?.Price || activeTicket?.price || 0);

  // Match tier name
  const ticketTypeId = activeTicket?.TicketTypeID || activeTicket?.ticket_type_id;
  const matchedTier = event?.ticket_types?.find(
    (t) => (t.id || t.ID) === ticketTypeId
  );
  const tierName = matchedTier?.name || "General Admission";

  // Resolved seats list from booking
  const assignedSeats = booking?.Seats || booking?.seats || [];

  return (
    <div className="flex min-h-screen flex-col bg-[#f9faff] text-[#141b2b]">
      <AppHeader />

      <main className="mx-auto w-full max-w-7xl flex-1 px-4 sm:px-6 lg:px-8 py-6 sm:py-9">
        {/* Breadcrumb Navigation */}
        <BookingBreadcrumbs items={breadcrumbs} />

        {/* Page Header */}
        <div className="flex flex-col sm:flex-row sm:items-end justify-between gap-5 border-b border-[#d6dddb] pb-5">
          <div>
            <div className="flex flex-wrap items-center gap-3">
              <h1 className="font-display text-3xl sm:text-4xl md:text-5xl font-bold tracking-tight text-ink">
                Your Ticket Pass
              </h1>
              {isCancelled ? (
                <span className="inline-flex items-center gap-1.5 rounded-full bg-red-100 border border-red-200 px-3 py-1 text-xs font-bold text-red-700 tracking-wide uppercase">
                  <AlertTriangleIcon className="w-3.5 h-3.5 text-red-600" />
                  CANCELLED (VOID)
                </span>
              ) : (
                <span className="inline-flex items-center gap-1 rounded-full bg-[#008378] px-3 py-1 text-xs font-semibold text-white tracking-wide uppercase">
                  <svg
                    className="w-3.5 h-3.5"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2.5" d="M5 13l4 4L19 7" />
                  </svg>
                  {ticketStatus}
                </span>
              )}
            </div>
            <p className="mt-2 text-sm sm:text-base text-[#3d4947]">
              {isCancelled
                ? "This booking has been cancelled. This ticket pass is void and will not be accepted for event admission."
                : "Present this verified booking pass with authoritative QR credential at event entrance."}
            </p>
          </div>

          {bookingId && (
            <Link
              to={`/bookings/${bookingId}`}
              className="self-start sm:self-auto inline-flex items-center justify-center rounded-lg border border-[#bcc9c6] bg-white px-5 py-2.5 text-xs sm:text-sm font-semibold text-ink hover:bg-slate-50 transition-colors"
            >
              Back to Booking
            </Link>
          )}
        </div>

        {/* Cancelled Alert Banner */}
        {isCancelled && (
          <div className="mx-auto mt-6 max-w-[1270px] rounded-xl border border-red-200 bg-red-50 p-4 sm:p-5 text-red-900 shadow-xs flex items-start gap-3.5">
            <div className="rounded-full bg-red-100 p-2 text-red-600 shrink-0">
              <AlertTriangleIcon className="w-5 h-5" />
            </div>
            <div>
              <h2 className="font-display font-bold text-base text-red-900">
                Ticket Pass Cancelled & Inactive
              </h2>
              <p className="mt-1 text-xs sm:text-sm text-red-700 leading-relaxed">
                This ticket pass was cancelled. Any assigned seats or admissions associated with this pass have been released back to event inventory. This pass cannot be used for event admission.
              </p>
            </div>
          </div>
        )}

        {/* Main Ticket Layout */}
        <div className="mx-auto mt-8 sm:mt-12 grid max-w-[1270px] grid-cols-1 lg:grid-cols-[1fr_320px] items-start gap-6 sm:gap-8">
          {/* Digital Ticket Pass Container */}
          <section className="overflow-hidden rounded-xl border border-[#bcc9c6] bg-white shadow-sm">
            {/* Ticket Header Ribbon */}
            <div
              className={`flex flex-wrap items-center justify-between gap-2 px-5 sm:px-6 py-2.5 text-xs sm:text-sm tracking-wider font-semibold border-b uppercase ${
                isCancelled
                  ? "bg-red-50 text-red-800 border-red-200/60"
                  : "bg-[#e9edff] text-[#3d4947] border-[#bcc9c6]/40"
              }`}
            >
              <span className="flex items-center gap-2">
                <TicketIconOutline
                  className={`w-4 h-4 ${
                    isCancelled ? "text-red-600" : "text-[#00685f]"
                  }`}
                />
                {isCancelled
                  ? "EVENTHUB DIGITAL PASS — VOID / CANCELLED"
                  : "EVENTHUB DIGITAL BOOKING PASS"}
              </span>
              <span className="font-mono">Ref: {bookingNumber}</span>
            </div>

            {/* Void notification across ticket if cancelled */}
            {isCancelled && (
              <div className="bg-red-600 px-4 py-1.5 text-center text-xs font-bold tracking-widest text-white uppercase">
                ⚠ TICKET CANCELLED — INVALID FOR EVENT ADMISSION
              </div>
            )}

            {/* Ticket Body: Details (left) and Authoritative QR Stub (right) */}
            <div className="grid grid-cols-1 lg:grid-cols-[1fr_240px]">
              {/* Event & Ticket Details */}
              <div className="p-5 sm:p-7">
                {/* Event Summary */}
                <div className="flex flex-col sm:flex-row gap-4 items-start">
                  <img
                    src={eventImage}
                    alt={eventTitle}
                    className="h-28 w-32 rounded-lg object-cover shrink-0 bg-slate-100"
                    loading="lazy"
                    onError={(e) => {
                      e.currentTarget.src = "/assets/events/event-music.jpg";
                    }}
                  />
                  <div>
                    <h2 className="font-display text-xl sm:text-2xl font-semibold text-ink">
                      {eventTitle}
                    </h2>
                    <div className="mt-2.5 space-y-1.5 text-xs sm:text-sm text-[#3d4947]">
                      <p className="flex items-center gap-2">
                        <CalendarIcon className="w-4 h-4 text-[#565e74] shrink-0" />
                        <span>{eventDateStr}</span>
                      </p>
                      <p className="flex items-center gap-2">
                        <ClockIcon className="w-4 h-4 text-[#565e74] shrink-0" />
                        <span>{eventTimeStr}</span>
                      </p>
                      <p className="flex items-center gap-2">
                        <LocationPinIcon className="w-4 h-4 text-[#565e74] shrink-0" />
                        <span>{venueLocationStr}</span>
                      </p>
                    </div>
                  </div>
                </div>

                {/* 4-Column Ticket Spec Grid */}
                <div className="mt-6 grid grid-cols-2 sm:grid-cols-4 gap-4 border-t border-[#dfe5e3] pt-5 text-xs sm:text-sm">
                  <div>
                    <span className="text-slate-500 block uppercase tracking-wider text-[11px] font-medium">
                      Tier
                    </span>
                    <strong className="mt-1 block font-bold text-ink text-sm sm:text-base">
                      {tierName}
                    </strong>
                  </div>
                  <div>
                    <span className="text-slate-500 block uppercase tracking-wider text-[11px] font-medium">
                      Total Passes
                    </span>
                    <strong className="mt-1 block font-bold text-ink text-sm sm:text-base">
                      {tickets.length > 0 ? `${tickets.length} Admissions` : "1 Admission"}
                    </strong>
                  </div>
                  <div>
                    <span className="text-slate-500 block uppercase tracking-wider text-[11px] font-medium">
                      Price / Pass
                    </span>
                    <strong className="mt-1 block font-bold text-ink text-sm sm:text-base">
                      {ticketPrice > 0 ? `₹${ticketPrice.toLocaleString("en-IN")}` : "Confirmed"}
                    </strong>
                  </div>
                  <div>
                    <span className="text-slate-500 block uppercase tracking-wider text-[11px] font-medium">
                      Organizer
                    </span>
                    <strong className="mt-1 block font-bold text-ink text-sm sm:text-base truncate">
                      {organizerName}
                    </strong>
                  </div>
                </div>

                {/* Assigned Seat Numbers Section */}
                {assignedSeats.length > 0 ? (
                  <div className="mt-6 rounded-xl border border-[#00685f]/20 bg-[#f0f9f8] p-4 sm:p-5">
                    <div className="flex items-center justify-between mb-3">
                      <div className="flex items-center gap-2">
                        <span className="text-xs font-bold uppercase tracking-wider text-[#00685f]">
                          Assigned Seats ({assignedSeats.length})
                        </span>
                      </div>
                      <span className="text-[11px] text-[#565e74]">
                        Valid for all admissions in this booking
                      </span>
                    </div>

                    <div className="flex flex-wrap gap-2.5">
                      {assignedSeats.map((seat, sIdx) => (
                        <div
                          key={seat.seat_id || sIdx}
                          className="inline-flex items-center gap-2 rounded-lg bg-white px-3.5 py-2 text-xs sm:text-sm font-semibold text-ink border border-[#00685f]/30 shadow-xs"
                        >
                          <span className="size-2 rounded-full bg-[#008378]" />
                          <span>{seat.label || `Row ${seat.row_name}, Seat ${seat.seat_number}`}</span>
                        </div>
                      ))}
                    </div>
                  </div>
                ) : (
                  <div className="mt-6 rounded-xl border border-slate-200 bg-[#f9faff] p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-2 text-xs sm:text-sm">
                    <div className="flex items-center gap-2">
                      <span className="size-2 rounded-full bg-[#008378]" />
                      <span className="font-medium text-slate-700">Seating Arrangement:</span>
                      <strong className="text-ink">General Admission</strong>
                    </div>
                    <span className="text-[#565e74] text-xs">
                      Open seating / standing room entry
                    </span>
                  </div>
                )}

                {/* Individual Ticket Passes (Preserved individual records) */}
                {tickets.length > 0 && (
                  <div className="mt-6 rounded-xl bg-[#eef1ff] p-4 sm:p-5">
                    <div className="flex items-center justify-between mb-3">
                      <p className="font-semibold text-xs sm:text-sm text-[#141b2b]">
                        Individual Admission Passes ({tickets.length})
                      </p>
                      <span className="text-[11px] text-slate-500">
                        {assignedSeats.length > 0 ? "Assigned seat per admission" : "General admission passes"}
                      </span>
                    </div>

                    <div className="space-y-2">
                      {tickets.map((item, idx) => {
                        const itemId = item.ID || item.id;
                        const itemCode = item.TicketCode || item.ticket_code || itemId;
                        const admissionNum = item.AdmissionNumber || item.admission_number || idx + 1;
                        const itemSeat = item.Seat || item.seat || (assignedSeats[idx] || null);

                        return (
                          <div
                            key={itemId || idx}
                            className="w-full flex flex-col sm:flex-row sm:items-center justify-between gap-2 p-3 rounded-lg border bg-white border-slate-200/80 text-xs sm:text-sm"
                          >
                            <div className="flex items-center gap-3 font-medium text-slate-800">
                              <span className="inline-flex items-center justify-center size-6 rounded-full bg-[#e0efed] text-[#00685f] text-xs font-bold">
                                {admissionNum}
                              </span>
                              <span>Admission #{admissionNum}</span>
                              {itemSeat && (
                                <span className="rounded bg-[#e0efed] px-2 py-0.5 text-[11px] font-semibold text-[#00685f]">
                                  {itemSeat.label || `Row ${itemSeat.row_name}, Seat ${itemSeat.seat_number}`}
                                </span>
                              )}
                            </div>

                            <div className="flex items-center gap-3">
                              <code className="rounded bg-[#eef1ff] px-2.5 py-1 font-mono text-xs font-bold text-slate-800">
                                {itemCode}
                              </code>
                              {isCancelled ? (
                                <span className="rounded bg-red-100 px-2 py-0.5 text-[11px] font-bold text-red-700">
                                  VOID
                                </span>
                              ) : (
                                <button
                                  type="button"
                                  onClick={() => handleDownloadSingleTicket(itemId)}
                                  disabled={isDownloading}
                                  className="text-xs text-[#00685f] hover:underline font-semibold cursor-pointer"
                                >
                                  PDF
                                </button>
                              )}
                            </div>
                          </div>
                        );
                      })}
                    </div>
                  </div>
                )}
              </div>

              {/* Right: Authoritative Booking QR Stub */}
              <div className="border-t border-dashed border-[#bcc9c6] p-6 text-center lg:border-l lg:border-t-0 flex flex-col items-center justify-between bg-slate-50/40">
                <div className="w-full flex flex-col items-center">
                  <span className="text-xs font-bold tracking-wider text-[#565e74] uppercase">
                    Scan for Entry
                  </span>

                  {/* Authoritative Booking QR Container */}
                  <div className="relative mx-auto mt-4 rounded-xl bg-white p-3 border border-slate-200/80 shadow-xs flex items-center justify-center min-h-[170px] min-w-[170px] overflow-hidden">
                    {qrLoading && (
                      <div className="flex flex-col items-center justify-center gap-2 p-6">
                        <div className="size-8 border-2 border-[#00685f]/30 border-t-[#00685f] rounded-full animate-spin" />
                        <span className="text-[11px] text-[#565e74]">Loading QR...</span>
                      </div>
                    )}

                    {!qrLoading && qrError && (
                      <div className="flex flex-col items-center justify-center gap-2 p-4 text-center max-w-[150px]">
                        <span className="text-amber-600 text-lg">⚠</span>
                        <span className="text-[11px] text-slate-600 leading-tight">QR Unavailable</span>
                        <button
                          type="button"
                          onClick={() => setReloadKey((k) => k + 1)}
                          className="text-[11px] text-[#00685f] underline font-medium cursor-pointer"
                        >
                          Retry
                        </button>
                      </div>
                    )}

                    {!qrLoading && !qrError && qrCodeUrl && (
                      <img
                        src={qrCodeUrl}
                        alt="Authoritative Booking QR Code"
                        className={`h-36 w-36 sm:h-40 sm:w-40 object-contain ${
                          isCancelled ? "opacity-20 grayscale" : ""
                        }`}
                      />
                    )}

                    {/* Void Stamp Overlay */}
                    {isCancelled && (
                      <div className="absolute inset-0 bg-white/80 backdrop-blur-[1px] flex flex-col items-center justify-center p-2 text-center border-2 border-dashed border-red-400">
                        <div className="size-8 rounded-full bg-red-100 text-red-600 flex items-center justify-center mb-1 font-bold text-sm">
                          ✕
                        </div>
                        <span className="font-display font-bold text-xs sm:text-sm text-red-700 uppercase tracking-wider">
                          Pass Void
                        </span>
                        <span className="text-[10px] text-red-600 font-medium">
                          Cancelled Booking
                        </span>
                      </div>
                    )}
                  </div>

                  {/* Booking QR Details */}
                  <p className="mt-3 text-xs font-medium text-[#565e74]">
                    {isCancelled ? "Void Booking QR" : "Booking QR Pass"}
                  </p>
                  <code className="mt-1 inline-block rounded bg-[#eef1ff] px-3 py-1 font-mono text-xs sm:text-sm font-bold tracking-widest text-[#141b2b] border border-slate-200/60 max-w-full truncate">
                    {bookingNumber}
                  </code>
                  <span className="mt-1 text-[11px] text-[#565e74]">
                    {isCancelled
                      ? "Deactivated"
                      : assignedSeats.length > 0
                      ? `Admits ${tickets.length || assignedSeats.length} • Reserved`
                      : `Admits ${tickets.length || 1} • General`}
                  </span>
                </div>

                <div className="w-full mt-6">
                  {isCancelled ? (
                    <button
                      type="button"
                      disabled
                      className="w-full inline-flex items-center justify-center gap-2 rounded-lg bg-slate-100 border border-slate-200 py-3 px-4 text-xs sm:text-sm font-semibold text-slate-400 cursor-not-allowed"
                    >
                      <DownloadIcon className="w-4 h-4 text-slate-400" />
                      <span>Pass Void (Cancelled)</span>
                    </button>
                  ) : (
                    <button
                      type="button"
                      onClick={handleDownload}
                      disabled={isDownloading}
                      className={`w-full inline-flex items-center justify-center gap-2 rounded-lg bg-[#00685f] py-3 px-4 text-xs sm:text-sm font-semibold text-white shadow-xs transition-colors cursor-pointer ${
                        isDownloading ? "opacity-75" : "hover:bg-[#00544c]"
                      }`}
                    >
                      <DownloadIcon className="w-4 h-4" />
                      <span>{isDownloading ? "Downloading..." : "⇩ Download PDF Pass"}</span>
                    </button>
                  )}
                </div>
              </div>
            </div>
          </section>

          {/* Right Aside: Entry Information */}
          <aside className="rounded-xl border border-[#bcc9c6] bg-white p-5 sm:p-6 shadow-sm">
            <h3 className="font-display font-semibold text-base text-[#141b2b] flex items-center gap-2">
              <InfoCircleIcon className="w-4 h-4 text-[#00685f]" />
              <span>Entry Information</span>
            </h3>

            <dl className="mt-5 grid grid-cols-2 gap-y-3.5 border-t border-[#dfe5e3] pt-5 text-xs sm:text-sm">
              <dt className="text-[#3d4947]">Event Time</dt>
              <dd className="text-right font-semibold text-ink">
                {eventTimeStr}
              </dd>

              <dt className="text-[#3d4947]">Event Date</dt>
              <dd className="text-right font-semibold text-ink">
                {eventDateStr}
              </dd>

              <dt className="text-[#3d4947]">Pass Status</dt>
              <dd
                className={`text-right font-bold flex items-center justify-end gap-1 ${
                  isCancelled ? "text-red-700" : "text-[#008378]"
                }`}
              >
                <span
                  className={`w-2 h-2 rounded-full ${
                    isCancelled ? "bg-red-600" : "bg-[#008378]"
                  }`}
                />
                {ticketStatus}
              </dd>

              <dt className="text-[#3d4947]">Seating</dt>
              <dd className="text-right font-semibold text-ink">
                {isCancelled
                  ? "Released"
                  : assignedSeats.length > 0
                  ? `${assignedSeats.length} Reserved`
                  : "General"}
              </dd>
            </dl>

            <div
              className={`mt-6 rounded-lg border p-4 text-center font-body text-xs sm:text-sm ${
                isCancelled
                  ? "border-red-200 bg-red-50 text-red-800"
                  : "border-[#bcc9c6] bg-[#eef1ff] text-[#3d4947]"
              }`}
            >
              {isCancelled
                ? "This ticket pass has been cancelled and permanently deactivated. Gate scanners will reject this barcode."
                : "This booking pass contains one authoritative QR credential for all admitted party members."}
            </div>
          </aside>
        </div>
      </main>

      <Footer />
    </div>
  );
}
