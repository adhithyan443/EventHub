import { useState, useEffect } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
import AppHeader from "../../../components/layout/AppHeader";
import Footer from "../../../components/layout/Footer";
import { getPublicEventById } from "../../../api/eventApi";

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

const formatVenueLocation = (event) => {
  if (event.event_type === "ONLINE") return "Online Event";
  if (event.venue) {
    const parts = [event.venue.city, event.venue.state].filter(Boolean);
    if (parts.length > 0) return parts.join(", ");
    return event.venue.name || "Venue TBA";
  }
  return "Venue TBA";
};

const formatFullVenue = (event) => {
  if (event.event_type === "ONLINE") {
    return event.online_url
      ? `Online Event\nAccess URL will be provided upon booking`
      : "Online Event\nStreaming link will be shared upon booking";
  }
  if (event.venue) {
    const lines = [
      event.venue.name,
      event.venue.address,
      [event.venue.city, event.venue.state, event.venue.postal_code].filter(Boolean).join(", "),
      event.venue.country,
    ].filter(Boolean);
    if (lines.length > 0) return lines.join("\n");
  }
  return "Venue details to be announced";
};

export default function CustomerEventDetailsPage() {
  const { eventId } = useParams();
  const navigate = useNavigate();

  const [event, setEvent] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [isNotFound, setIsNotFound] = useState(false);

  const [selectedTicketId, setSelectedTicketId] = useState(null);
  const [quantities, setQuantities] = useState({});
  const [bannerFallback, setBannerFallback] = useState(false);

  const [retryCount, setRetryCount] = useState(0);

  useEffect(() => {
    let cancelled = false;

    const fetchEventDetails = async () => {
      if (!eventId) {
        setIsNotFound(true);
        setError("Event not found");
        setLoading(false);
        return;
      }

      setLoading(true);
      setError(null);
      setIsNotFound(false);
      setBannerFallback(false);

      try {
        const response = await getPublicEventById(eventId);
        if (cancelled) return;

        const data = response?.data;
        if (!data) {
          throw new Error("Empty response");
        }
        setEvent(data);

        if (data.ticket_types && data.ticket_types.length > 0) {
          const availableTier =
            data.ticket_types.find((t) => t.status !== "SOLD_OUT" && t.available_quantity > 0) ||
            data.ticket_types[0];
          setSelectedTicketId(availableTier.id);
          setQuantities({ [availableTier.id]: 1 });
        }
      } catch (err) {
        if (cancelled) return;

        const status = err?.response?.status;
        if (status === 404 || status === 400) {
          setIsNotFound(true);
          setError("Event not found");
        } else {
          setError(err?.response?.data?.message || "Unable to load event details. Please try again.");
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };

    fetchEventDetails();

    return () => {
      cancelled = true;
    };
  }, [eventId, retryCount]);

  const handleQuantityChange = (tierId, delta, maxLimit) => {
    setSelectedTicketId(tierId);
    setQuantities((prev) => {
      const current = prev[tierId] || 0;
      const nextVal = Math.max(0, Math.min(maxLimit, current + delta));
      return { ...prev, [tierId]: nextVal };
    });
  };

  const isSeated = event?.setting?.seat_layout_type === "SEATED";

  const handleProceed = () => {
    if (!selectedTicketId) return;
    const qty = quantities[selectedTicketId] || 1;
    if (qty <= 0) return;

    const selectedTier = (event?.ticket_types || []).find((t) => t.id === selectedTicketId);

    if (isSeated) {
      navigate(`/events/${eventId}/seats`, {
        state: {
          selectedTicket: selectedTier?.name || "Tier",
          selectedTicketId,
          ticketPrice: selectedTier?.price ?? event?.starting_price,
          quantity: qty,
        },
      });
    } else {
      navigate(`/events/${eventId}/checkout`, {
        state: {
          selectedTicket: selectedTier?.name || "General",
          selectedTicketId,
          ticketPrice: selectedTier?.price ?? event?.starting_price,
          quantity: qty,
        },
      });
    }
  };

  if (loading) {
    return (
      <div className="flex min-h-screen flex-col bg-[#f9f9ff] text-[#141b2b]">
        <AppHeader />
        <main className="mx-auto w-full max-w-[1440px] flex-1 px-4 sm:px-6 lg:px-8 py-16 flex items-center justify-center">
          <div className="flex flex-col items-center gap-4 bg-white border border-[#bcc9c6] rounded-xl p-10 shadow-xs max-w-md w-full text-center">
            <div className="size-10 border-3 border-[#00685f]/30 border-t-[#00685f] rounded-full animate-spin" />
            <h2 className="font-display font-semibold text-lg text-ink">Loading event details...</h2>
            <p className="text-xs text-[#565e74]">Fetching event schedule, tickets, and venue information.</p>
          </div>
        </main>
        <Footer />
      </div>
    );
  }

  if (error || !event) {
    return (
      <div className="flex min-h-screen flex-col bg-[#f9f9ff] text-[#141b2b]">
        <AppHeader />
        <main className="mx-auto w-full max-w-[1440px] flex-1 px-4 sm:px-6 lg:px-8 py-16 flex items-center justify-center">
          <div className="flex flex-col items-center gap-4 bg-white border border-[#bcc9c6] rounded-xl p-8 sm:p-12 shadow-xs max-w-lg w-full text-center">
            <div className="grid h-16 w-16 place-items-center rounded-full bg-[#fde8e8] text-red-600 mb-2">
              <svg className="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth="2"
                  d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
                />
              </svg>
            </div>
            <h2 className="font-display font-bold text-xl sm:text-2xl text-ink">
              {isNotFound ? "Event Not Found" : "Unable to Load Event"}
            </h2>
            <p className="text-sm text-[#565e74] leading-relaxed">
              {isNotFound
                ? "The event you are looking for may have been removed, made private, or does not exist."
                : error || "We encountered an issue fetching this event. Please try again."}
            </p>
            <div className="mt-4 flex flex-wrap gap-3 justify-center">
              {!isNotFound && (
                <button
                  type="button"
                  onClick={() => setRetryCount((c) => c + 1)}
                  className="rounded-lg bg-[#00796d] px-5 py-2.5 text-sm font-bold text-white shadow-xs hover:bg-[#00685f] transition-colors cursor-pointer"
                >
                  Try Again
                </button>
              )}
              <Link
                to="/events"
                className="rounded-lg border border-[#bcc9c6] bg-white px-5 py-2.5 text-sm font-semibold text-[#141b2b] hover:bg-slate-50 transition-colors"
              >
                Browse Events
              </Link>
            </div>
          </div>
        </main>
        <Footer />
      </div>
    );
  }

  // Derive status badge display
  let statusBadgeText = "Tickets Available";
  const allSoldOut =
    event.ticket_types &&
    event.ticket_types.length > 0 &&
    event.ticket_types.every((t) => t.status === "SOLD_OUT" || t.available_quantity === 0);
  const anySellingFast =
    event.ticket_types &&
    event.ticket_types.some((t) => t.status === "SELLING_FAST");

  if (allSoldOut) {
    statusBadgeText = "Sold Out";
  } else if (anySellingFast) {
    statusBadgeText = "Selling Fast";
  }

  const heroImageSrc =
    !bannerFallback && event.banner_image_url
      ? event.banner_image_url
      : "/assets/2b62b.png";

  return (
    <div className="flex min-h-screen flex-col bg-[#f9f9ff] text-[#141b2b]">
      <AppHeader />

      <main className="mx-auto w-full max-w-[1440px] flex-1 px-4 sm:px-6 lg:px-8 py-6 sm:py-8">
        {/* Breadcrumb Navigation */}
        <nav aria-label="Breadcrumb" className="mb-6 sm:mb-8 text-xs sm:text-sm text-[#565e74]">
          <ol className="flex items-center flex-wrap gap-1 sm:gap-2">
            <li>
              <Link to="/events" className="hover:text-[#00685f] transition-colors">
                Events
              </Link>
            </li>
            <li className="text-[#8a8fa0]">›</li>
            <li>
              <span className="hover:text-[#00685f] transition-colors">
                {event.category_name || "Events"}
              </span>
            </li>
            <li className="text-[#8a8fa0]">›</li>
            <li className="font-medium text-[#141b2b] truncate max-w-[200px] sm:max-w-none">
              {event.title}
            </li>
          </ol>
        </nav>

        <div className="grid gap-8 lg:grid-cols-[minmax(0,2fr)_minmax(340px,1fr)] items-start">
          {/* Main Left Content */}
          <div className="space-y-6 sm:space-y-8">
            {/* Hero Image & Headline Card */}
            <section className="rounded-xl border border-[#bcc9c6] bg-white shadow-xs overflow-hidden">
              <div className="relative h-[220px] sm:h-[320px] md:h-[384px] bg-[#e9edff]">
                <img
                  src={heroImageSrc}
                  alt={`${event.title} banner`}
                  className="h-full w-full object-cover"
                  onError={() => setBannerFallback(true)}
                />
                <span className="absolute left-4 top-4 inline-flex items-center gap-1.5 rounded-full border border-[#bcc9c6] bg-white/90 backdrop-blur-xs px-3.5 py-1.5 text-xs sm:text-sm font-medium text-[#141b2b] shadow-xs">
                  <span className="h-2 w-2 rounded-full bg-[#00796d]" /> {event.category_name || "Event"}
                </span>
              </div>

              <div className="flex flex-col gap-5 p-5 sm:p-6 sm:flex-row sm:items-center sm:justify-between">
                <div>
                  <h1 className="font-display text-2xl sm:text-3xl font-bold text-ink leading-tight">
                    {event.title}
                  </h1>
                  <div className="mt-3 flex flex-wrap gap-x-5 gap-y-2 text-xs sm:text-sm text-[#565e74]">
                    <span className="inline-flex items-center gap-1.5">
                      <span className="text-[#00796d]">▣</span> {formatDate(event.schedule?.event_date)}
                    </span>
                    <span className="inline-flex items-center gap-1.5">
                      <span className="text-[#00796d]">◷</span> {formatScheduleTime(event.schedule)}
                    </span>
                    <span className="inline-flex items-center gap-1.5">
                      <span className="text-[#00796d]">⌖</span> {formatVenueLocation(event)}
                    </span>
                  </div>
                </div>
                <span
                  className={`inline-flex items-center self-start sm:self-center rounded-full border border-[#bcc9c6] px-4 py-2 text-xs sm:text-sm font-semibold whitespace-nowrap ${
                    allSoldOut
                      ? "bg-red-100 text-red-700"
                      : anySellingFast
                      ? "bg-amber-100 text-amber-800"
                      : "bg-[#e4f1f0] text-[#00685f]"
                  }`}
                >
                  {statusBadgeText}
                </span>
              </div>
            </section>

            {/* About this event Card */}
            <section className="rounded-xl border border-[#bcc9c6] bg-white p-5 sm:p-6 shadow-xs">
              <h2 className="mb-4 font-display text-lg sm:text-xl font-bold text-ink">
                About this event
              </h2>
              <p className="text-sm sm:text-base leading-relaxed text-[#565e74] whitespace-pre-line">
                {event.description}
              </p>

              {/* Highlights */}
              {event.highlights && (
                <div className="mt-6 border-t border-[#e2e8f0] pt-6">
                  <h3 className="font-semibold text-sm sm:text-base text-ink mb-2">Event Highlights</h3>
                  <p className="text-xs sm:text-sm text-[#565e74] leading-relaxed whitespace-pre-line">
                    {event.highlights}
                  </p>
                </div>
              )}

              {/* Guidelines / Rules */}
              {event.rules && (
                <div className="mt-6 border-t border-[#e2e8f0] pt-6">
                  <h3 className="font-semibold text-sm sm:text-base text-ink mb-2">Important Guidelines &amp; Rules</h3>
                  <p className="text-xs sm:text-sm text-[#565e74] leading-relaxed whitespace-pre-line">
                    {event.rules}
                  </p>
                </div>
              )}

              {/* Attendee Information */}
              {event.attendee_information && (
                <div className="mt-6 border-t border-[#e2e8f0] pt-6">
                  <h3 className="font-semibold text-sm sm:text-base text-ink mb-2">Attendee Information</h3>
                  <p className="text-xs sm:text-sm text-[#565e74] leading-relaxed whitespace-pre-line">
                    {event.attendee_information}
                  </p>
                </div>
              )}

              {/* Event Attributes Badges */}
              <div className="mt-6 flex flex-wrap gap-2 border-t border-[#e2e8f0] pt-6">
                {event.language && (
                  <span className="inline-flex items-center rounded-md bg-[#f1f5f9] px-2.5 py-1 text-xs font-medium text-[#475569]">
                    Language: {event.language}
                  </span>
                )}
                {event.age_restriction > 0 ? (
                  <span className="inline-flex items-center rounded-md bg-[#f1f5f9] px-2.5 py-1 text-xs font-medium text-[#475569]">
                    Age: {event.age_restriction}+
                  </span>
                ) : (
                  <span className="inline-flex items-center rounded-md bg-[#f1f5f9] px-2.5 py-1 text-xs font-medium text-[#475569]">
                    All Ages
                  </span>
                )}
                {event.event_type && (
                  <span className="inline-flex items-center rounded-md bg-[#f1f5f9] px-2.5 py-1 text-xs font-medium text-[#475569]">
                    Format: {event.event_type}
                  </span>
                )}
                {event.starting_price > 0 && (
                  <span className="inline-flex items-center rounded-md bg-[#e4f1f0] px-2.5 py-1 text-xs font-medium text-[#00685f]">
                    Starting from ₹{Number(event.starting_price).toLocaleString("en-IN")}
                  </span>
                )}
              </div>

              {/* Date & Time and Venue Info */}
              <div className="mt-6 sm:mt-8 grid gap-6 sm:grid-cols-2 border-t border-[#e2e8f0] pt-6">
                <div className="flex items-start gap-4">
                  <div className="grid h-10 w-10 shrink-0 place-items-center rounded-full bg-[#e4f1f0] text-[#00796d]">
                    <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <rect x="3" y="4" width="18" height="18" rx="2" strokeWidth="2" />
                      <line x1="16" y1="2" x2="16" y2="6" strokeWidth="2" strokeLinecap="round" />
                      <line x1="8" y1="2" x2="8" y2="6" strokeWidth="2" strokeLinecap="round" />
                      <line x1="3" y1="10" x2="21" y2="10" strokeWidth="2" />
                    </svg>
                  </div>
                  <div>
                    <h3 className="font-semibold text-sm sm:text-base text-ink">Date &amp; Time</h3>
                    <p className="mt-1 text-xs sm:text-sm text-[#565e74] leading-relaxed">
                      {formatFullDate(event.schedule?.event_date)}
                      <br />
                      {formatScheduleTime(event.schedule)}
                    </p>
                  </div>
                </div>

                <div className="flex items-start gap-4">
                  <div className="grid h-10 w-10 shrink-0 place-items-center rounded-full bg-[#e4f1f0] text-[#00796d]">
                    <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth="2"
                        d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z"
                      />
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth="2"
                        d="M15 11a3 3 0 11-6 0 3 3 0 016 0z"
                      />
                    </svg>
                  </div>
                  <div>
                    <h3 className="font-semibold text-sm sm:text-base text-ink">Venue</h3>
                    <p className="mt-1 text-xs sm:text-sm text-[#565e74] leading-relaxed whitespace-pre-line">
                      {formatFullVenue(event)}
                    </p>
                  </div>
                </div>
              </div>
            </section>
          </div>

          {/* Right Sidebar: Ticket Selection & Organizer */}
          <aside className="space-y-6">
            {/* Choose Your Ticket Card */}
            <section className="sticky top-20 rounded-xl border border-[#bcc9c6] bg-white p-5 sm:p-6 shadow-xs">
              <div className="mb-4 flex items-center justify-between">
                <h2 className="font-display text-lg font-bold text-ink">
                  Choose your ticket
                </h2>
                {event.starting_price > 0 && (
                  <span className="text-xs font-semibold text-[#565e74]">
                    From ₹{Number(event.starting_price).toLocaleString("en-IN")}
                  </span>
                )}
              </div>

              {event.ticket_types && event.ticket_types.length > 0 ? (
                <div className="space-y-4">
                  {event.ticket_types.map((tier) => {
                    const isSelected = selectedTicketId === tier.id;
                    const qty = quantities[tier.id] || 0;
                    const isSoldOut =
                      tier.status === "SOLD_OUT" || tier.available_quantity === 0;
                    const maxBooking = event.setting?.booking_limit_per_user || 9;
                    const maxQty = Math.min(
                      maxBooking,
                      tier.available_quantity > 0 ? tier.available_quantity : maxBooking
                    );

                    return (
                      <div
                        key={tier.id}
                        onClick={() => {
                          if (!isSoldOut) {
                            setSelectedTicketId(tier.id);
                            if (!quantities[tier.id]) {
                              setQuantities((prev) => ({ ...prev, [tier.id]: 1 }));
                            }
                          }
                        }}
                        className={`rounded-lg border p-4 transition-all ${
                          isSoldOut
                            ? "border-slate-200 bg-slate-50 opacity-60 cursor-not-allowed"
                            : isSelected
                            ? "border-[#00796d] bg-[#f8fffe] shadow-xs cursor-pointer"
                            : "border-[#bcc9c6] bg-[#f9f9ff] hover:border-slate-400 cursor-pointer"
                        }`}
                      >
                        <div className="flex items-center justify-between">
                          <span className="font-semibold text-sm tracking-wide uppercase text-ink">
                            {tier.name}
                          </span>
                          <strong className="text-base font-bold text-ink font-display">
                            ₹{Number(tier.price).toLocaleString("en-IN")}
                          </strong>
                        </div>

                        <p className="mt-1 text-xs sm:text-sm text-[#565e74]">
                          {tier.description || `${tier.name} admission`}
                        </p>

                        <div className="mt-4 flex items-center justify-between border-t border-[#d8dfdd] pt-3">
                          <span
                            className={`rounded px-2 py-0.5 text-xs font-semibold ${
                              isSoldOut
                                ? "bg-red-100 text-red-700"
                                : tier.status === "SELLING_FAST"
                                ? "bg-amber-100 text-amber-800"
                                : "bg-[#dff2ee] text-[#00796d]"
                            }`}
                          >
                            {isSoldOut
                              ? "Sold Out"
                              : tier.status === "SELLING_FAST"
                              ? "Selling Fast"
                              : "Available"}
                          </span>

                          <div
                            className="flex items-center gap-3"
                            onClick={(e) => e.stopPropagation()}
                          >
                            <button
                              type="button"
                              onClick={() => handleQuantityChange(tier.id, -1, maxQty)}
                              disabled={qty <= 0 || isSoldOut}
                              className="counter-btn disabled:opacity-30 disabled:cursor-not-allowed"
                              aria-label={`Decrease ${tier.name} ticket quantity`}
                            >
                              −
                            </button>
                            <span className="w-5 text-center font-bold text-sm text-ink">
                              {qty}
                            </span>
                            <button
                              type="button"
                              onClick={() => handleQuantityChange(tier.id, 1, maxQty)}
                              disabled={qty >= maxQty || isSoldOut}
                              className="counter-btn disabled:opacity-30 disabled:cursor-not-allowed"
                              aria-label={`Increase ${tier.name} ticket quantity`}
                            >
                              ＋
                            </button>
                          </div>
                        </div>
                      </div>
                    );
                  })}
                </div>
              ) : (
                <div className="rounded-lg border border-dashed border-[#bcc9c6] p-6 text-center text-xs text-[#565e74]">
                  No ticket tiers are currently available for this event.
                </div>
              )}

              {/* Cancellation policy note */}
              {event.cancellation && (
                <div className="my-5 rounded-lg bg-[#eef0ff] p-3 text-xs leading-relaxed text-[#565e74] flex items-start gap-2">
                  <span className="text-[#4648d4] font-bold">ⓘ</span>
                  <div>
                    {event.cancellation.cancellation_allowed ? (
                      <span>
                        <strong>Refundable:</strong> Up to{" "}
                        {event.cancellation.refund_percentage}% refund available if
                        cancelled at least{" "}
                        {event.cancellation.cancellation_deadline_hours} hours
                        prior to the event.
                        {event.cancellation.refund_policy && (
                          <span className="block mt-0.5 text-[#565e74]/90">
                            {event.cancellation.refund_policy}
                          </span>
                        )}
                      </span>
                    ) : (
                      <span>
                        <strong>Non-refundable:</strong> Cancellations are not
                        permitted for this event.
                      </span>
                    )}
                  </div>
                </div>
              )}

              <button
                type="button"
                onClick={handleProceed}
                disabled={
                  !selectedTicketId || (quantities[selectedTicketId] || 0) <= 0
                }
                className="h-12 w-full rounded-lg bg-[#00796d] px-5 font-bold text-white shadow-xs transition hover:bg-[#00685f] focus:outline-none focus:ring-2 focus:ring-[#00796d]/40 disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer"
              >
                {isSeated ? "Select Seats" : "Book Tickets"}
              </button>

              <p className="mt-3 text-center text-xs text-[#565e74]">
                {isSeated
                  ? "Select your seats to continue"
                  : "Proceed to checkout with selected tickets"}
              </p>
            </section>

            {/* Organizer Card */}
            {event.organizer && (
              <section className="rounded-xl border border-[#bcc9c6] bg-white p-5 sm:p-6 shadow-xs">
                <div className="flex items-center gap-4">
                  {event.organizer.avatar_url ? (
                    <img
                      src={event.organizer.avatar_url}
                      alt={event.organizer.name}
                      className="h-12 w-12 rounded-full object-cover border border-slate-200"
                      onError={(e) => {
                        e.currentTarget.onerror = null;
                        e.currentTarget.src = "/assets/5e545.png";
                      }}
                    />
                  ) : (
                    <div className="h-12 w-12 rounded-full bg-[#00796d]/10 text-[#00796d] font-bold flex items-center justify-center text-base border border-[#bcc9c6]">
                      {event.organizer.name
                        ? event.organizer.name.charAt(0).toUpperCase()
                        : "O"}
                    </div>
                  )}
                  <div>
                    <h3 className="font-semibold text-ink text-sm sm:text-base">
                      {event.organizer.name || "Event Organizer"}
                    </h3>
                    {event.organizer.verified && (
                      <p className="text-xs font-medium text-[#00796d] flex items-center gap-1.5 mt-0.5">
                        <span className="h-2 w-2 rounded-full bg-[#00796d]" />
                        Verified Organizer
                      </p>
                    )}
                  </div>
                </div>
              </section>
            )}

            {/* Contact Support Card if available */}
            {event.contact && (event.contact.email || event.contact.phone) && (
              <section className="rounded-xl border border-[#bcc9c6] bg-white p-5 sm:p-6 shadow-xs">
                <h3 className="font-semibold text-ink text-sm sm:text-base mb-2">
                  Event Support
                </h3>
                <div className="space-y-1 text-xs text-[#565e74]">
                  {event.contact.name && (
                    <p className="font-medium text-[#141b2b]">
                      {event.contact.name}
                    </p>
                  )}
                  {event.contact.email && <p>Email: {event.contact.email}</p>}
                  {event.contact.phone && <p>Phone: {event.contact.phone}</p>}
                </div>
              </section>
            )}
          </aside>
        </div>
      </main>

      <Footer />
    </div>
  );
}
