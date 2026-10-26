import { useState, useEffect, useMemo } from "react";
import { Link } from "react-router-dom";
import AppHeader from "../../components/layout/AppHeader";
import Footer from "../../components/layout/Footer";
import { getMyBookings } from "../../api/bookingApi";
import { getPublicEventById } from "../../api/eventApi";
import {
  CalendarIcon,
  LocationPinIcon,
  TagReceiptIcon,
  RefundUndoIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
} from "./components/BookingIcons";
import RefundDetailsModal from "./components/RefundDetailsModal";

const ITEMS_PER_PAGE = 5;

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

const formatVenueLocation = (event) => {
  if (!event) return "Location TBA";
  if (event.event_type === "ONLINE") return "Online Event";
  if (event.venue) {
    const parts = [event.venue.city, event.venue.state].filter(Boolean);
    if (parts.length > 0) return parts.join(", ");
    return event.venue.name || "Venue TBA";
  }
  return "Venue TBA";
};

export default function MyBookingsPage() {
  const [bookings, setBookings] = useState([]);
  const [eventsMap, setEventsMap] = useState({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  const [tab, setTab] = useState("All");
  const [searchQuery, setSearchQuery] = useState("");
  const [sortBy, setSortBy] = useState("newest");
  const [currentPage, setCurrentPage] = useState(1);
  const [selectedRefundBooking, setSelectedRefundBooking] = useState(null);

  const [reloadKey, setReloadKey] = useState(0);

  const tabs = ["All", "Confirmed", "Cancelled", "Refunded"];

  const handleTabChange = (newTab) => {
    setTab(newTab);
    setCurrentPage(1);
  };

  const handleSearchChange = (e) => {
    setSearchQuery(e.target.value);
    setCurrentPage(1);
  };

  const handleSortChange = (e) => {
    setSortBy(e.target.value);
    setCurrentPage(1);
  };

  const handleRetry = () => {
    setReloadKey((prev) => prev + 1);
  };

  // Fetch live customer bookings and resolve associated event data
  useEffect(() => {
    let cancelled = false;

    const fetchCustomerBookings = async () => {
      setLoading(true);
      setError(null);

      try {
        const response = await getMyBookings();
        if (cancelled) return;

        const rawBookings = Array.isArray(response?.data) ? response.data : [];
        setBookings(rawBookings);

        const eventIds = [
          ...new Set(
            rawBookings
              .map((b) => b.EventID || b.event_id || b.eventId)
              .filter(Boolean)
          ),
        ];

        if (eventIds.length > 0) {
          const eventResults = await Promise.allSettled(
            eventIds.map((id) => getPublicEventById(id))
          );
          if (cancelled) return;

          const newEventsMap = {};
          eventResults.forEach((res, index) => {
            const id = eventIds[index];
            if (res.status === "fulfilled" && res.value?.data) {
              newEventsMap[id] = res.value.data;
            }
          });
          setEventsMap(newEventsMap);
        }
      } catch (err) {
        if (cancelled) return;
        const msg =
          err?.response?.data?.error ||
          err?.response?.data?.message ||
          err?.message ||
          "Failed to load your bookings. Please try again.";
        setError(msg);
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };

    fetchCustomerBookings();

    return () => {
      cancelled = true;
    };
  }, [reloadKey]);

  // Filter bookings based on tab and search
  const filteredBookings = useMemo(() => {
    return bookings
      .filter((booking) => {
        const rawStatus = (booking.Status || booking.status || "CONFIRMED").toUpperCase();

        // Tab filter
        let matchesTab = true;
        if (tab === "Confirmed") {
          matchesTab = rawStatus === "CONFIRMED" || rawStatus === "COMPLETED";
        } else if (tab === "Cancelled") {
          matchesTab = rawStatus === "CANCELLED";
        } else if (tab === "Refunded") {
          matchesTab = rawStatus === "REFUNDED";
        }

        if (!matchesTab) return false;

        // Search filter
        const query = searchQuery.trim().toLowerCase();
        if (!query) return true;

        const eventId = booking.EventID || booking.event_id || booking.eventId;
        const event = eventsMap[eventId];
        const eventTitle = (event?.title || "").toLowerCase();
        const bookingNum = (booking.BookingNumber || booking.booking_number || booking.ID || "").toLowerCase();
        const locationStr = formatVenueLocation(event).toLowerCase();

        return (
          eventTitle.includes(query) ||
          bookingNum.includes(query) ||
          locationStr.includes(query)
        );
      })
      .sort((a, b) => {
        const dateA = new Date(a.CreatedAt || a.created_at || 0).getTime();
        const dateB = new Date(b.CreatedAt || b.created_at || 0).getTime();
        if (sortBy === "oldest") {
          return dateA - dateB;
        }
        return dateB - dateA;
      });
  }, [bookings, eventsMap, tab, searchQuery, sortBy]);

  // Pagination calculation
  const totalPages = Math.max(1, Math.ceil(filteredBookings.length / ITEMS_PER_PAGE));
  const paginatedBookings = useMemo(() => {
    const start = (currentPage - 1) * ITEMS_PER_PAGE;
    return filteredBookings.slice(start, start + ITEMS_PER_PAGE);
  }, [filteredBookings, currentPage]);

  const getStatusBadgeClass = (status) => {
    const s = (status || "").toUpperCase();
    switch (s) {
      case "CONFIRMED":
        return "bg-[#008378] text-white";
      case "COMPLETED":
        return "bg-[#f2f3f6] text-[#565e74]";
      case "CANCELLED":
        return "bg-[#ffdad6] text-[#93000a]";
      case "REFUNDED":
        return "bg-[#e4e9f8] text-[#565e74]";
      default:
        return "bg-slate-100 text-slate-700";
    }
  };

  return (
    <div className="flex min-h-screen flex-col bg-[#f9faff] text-[#141b2b]">
      <AppHeader />

      <main className="mx-auto w-full max-w-7xl flex-1 px-4 sm:px-6 lg:px-8 py-8 sm:py-12">
        {/* Page Title & Subtitle */}
        <div>
          <h1 className="font-display text-2xl sm:text-3xl lg:text-[32px] font-bold tracking-tight text-ink">
            My Bookings
          </h1>
          <p className="mt-1 text-sm sm:text-base text-[#565e74]">
            View and manage all your event bookings and ticket passes.
          </p>
        </div>

        {/* Filter Tabs and Search Bar */}
        <div className="mt-6 sm:mt-8 flex flex-col justify-between gap-4 lg:flex-row lg:items-end">
          {/* Status Tabs */}
          <div className="flex overflow-x-auto border-b border-[#d9dfdd] no-scrollbar">
            {tabs.map((item) => {
              const isActive = tab === item;
              return (
                <button
                  key={item}
                  type="button"
                  onClick={() => handleTabChange(item)}
                  className={`min-w-20 sm:min-w-24 px-3 sm:px-4 py-3 text-xs sm:text-sm font-semibold transition-colors shrink-0 cursor-pointer ${
                    isActive
                      ? "border-b-2 border-[#008378] text-[#00685f]"
                      : "text-[#565e74] hover:text-ink"
                  }`}
                >
                  {item}
                </button>
              );
            })}
          </div>

          {/* Search & Sort Controls */}
          <div className="flex flex-col sm:flex-row gap-3 w-full lg:w-auto">
            <div className="relative flex-1 sm:w-72">
              <span className="absolute inset-y-0 left-3 flex items-center pointer-events-none text-[#565e74]">
                <svg
                  className="w-4 h-4"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <circle cx="11" cy="11" r="8" strokeWidth="2" />
                  <path strokeLinecap="round" strokeWidth="2" d="M21 21l-4.35-4.35" />
                </svg>
              </span>
              <input
                type="text"
                value={searchQuery}
                onChange={handleSearchChange}
                placeholder="Search by event or booking ID"
                className="h-11 sm:h-12 w-full rounded-lg border border-[#e3e6e5] bg-white pl-9 pr-4 text-xs sm:text-sm text-[#141b2b] placeholder:text-[#565e74] outline-none transition focus:border-[#008378] focus:ring-1 focus:ring-[#008378]"
              />
            </div>

            <select
              value={sortBy}
              onChange={handleSortChange}
              className="h-11 sm:h-12 rounded-lg border border-[#e3e6e5] bg-white px-3 sm:px-4 text-xs sm:text-sm text-[#565e74] outline-none transition focus:border-[#008378] cursor-pointer"
            >
              <option value="newest">Newest First</option>
              <option value="oldest">Oldest First</option>
            </select>
          </div>
        </div>

        {/* Loading State */}
        {loading && (
          <div className="mt-8 rounded-xl border border-[#dfe5e3] bg-white p-12 text-center shadow-xs">
            <div className="mx-auto size-10 border-3 border-[#00685f]/30 border-t-[#00685f] rounded-full animate-spin" />
            <h3 className="mt-4 font-display font-semibold text-base text-ink">
              Loading your bookings...
            </h3>
            <p className="mt-1 text-xs text-[#565e74]">
              Retrieving your active reservations and confirmed ticket passes.
            </p>
          </div>
        )}

        {/* Error State */}
        {!loading && error && (
          <div className="mt-8 rounded-xl border border-red-200 bg-red-50/50 p-8 text-center">
            <div className="mx-auto flex size-12 items-center justify-center rounded-full bg-red-100 text-red-600 mb-3">
              <svg className="size-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
              </svg>
            </div>
            <h3 className="font-display font-semibold text-base text-red-900">
              Unable to load bookings
            </h3>
            <p className="mt-1 text-xs sm:text-sm text-red-700 max-w-md mx-auto">
              {error}
            </p>
            <button
              type="button"
              onClick={handleRetry}
              className="mt-4 inline-flex items-center rounded-lg bg-[#00685f] px-4 py-2 text-xs sm:text-sm font-semibold text-white hover:bg-[#00544c] transition-colors cursor-pointer"
            >
              Try Again
            </button>
          </div>
        )}

        {/* Bookings List */}
        {!loading && !error && (
          <div className="mt-6 sm:mt-8 space-y-4">
            {filteredBookings.length === 0 ? (
              <div className="rounded-xl border border-dashed border-[#dfe5e3] bg-white p-8 sm:p-12 text-center">
                <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-slate-100 text-[#565e74] mb-3">
                  <TagReceiptIcon className="w-6 h-6" />
                </div>
                <h3 className="font-display font-semibold text-base text-ink">
                  No bookings found
                </h3>
                <p className="mt-1 text-xs sm:text-sm text-[#565e74]">
                  {searchQuery
                    ? `No bookings matched "${searchQuery}".`
                    : tab !== "All"
                    ? `You have no ${tab.toLowerCase()} bookings at this time.`
                    : "You haven't made any event bookings yet."}
                </p>
                {searchQuery || tab !== "All" ? (
                  <button
                    type="button"
                    onClick={() => {
                      setTab("All");
                      setSearchQuery("");
                      setCurrentPage(1);
                    }}
                    className="mt-4 inline-flex items-center rounded-lg bg-[#00685f] px-4 py-2 text-xs sm:text-sm font-semibold text-white hover:bg-[#00544c] transition-colors cursor-pointer"
                  >
                    Reset Filters
                  </button>
                ) : (
                  <Link
                    to="/events"
                    className="mt-4 inline-flex items-center rounded-lg bg-[#00685f] px-5 py-2.5 text-xs sm:text-sm font-bold text-white shadow-xs hover:bg-[#00544c] transition-colors"
                  >
                    Browse Events
                  </Link>
                )}
              </div>
            ) : (
              paginatedBookings.map((booking) => {
                const bookingId = booking.ID || booking.id;
                const bookingNumber = booking.BookingNumber || booking.booking_number || bookingId;
                const eventId = booking.EventID || booking.event_id || booking.eventId;
                const event = eventsMap[eventId];
                const status = (booking.Status || booking.status || "CONFIRMED").toUpperCase();

                const eventTitle = event?.title || `Event Booking (${bookingNumber})`;
                const eventImage = event?.cover_image_url || event?.banner_image_url || "/assets/events/event-music.jpg";
                const eventDate = event?.schedule?.event_date
                  ? formatDate(event.schedule.event_date)
                  : formatDate(booking.CreatedAt || booking.created_at);
                const eventLocation = formatVenueLocation(event);
                const bookedOnDate = formatDate(booking.CreatedAt || booking.created_at);
                const totalAmount = Number(booking.Total || booking.total || 0);
                const assignedSeats = booking.Seats || booking.seats || [];

                return (
                  <article
                    key={bookingId}
                    className="relative flex flex-col gap-5 rounded-xl border border-[#dfe5e3] bg-white p-5 sm:p-6 shadow-xs transition hover:shadow-sm sm:flex-row sm:items-start"
                  >
                    {/* Event Thumbnail */}
                    <img
                      src={eventImage}
                      alt={eventTitle}
                      className="h-44 w-full rounded-lg object-cover sm:h-36 sm:w-48 shrink-0 bg-slate-100"
                      loading="lazy"
                      onError={(e) => {
                        e.currentTarget.src = "/assets/events/event-music.jpg";
                      }}
                    />

                    {/* Event Booking Details */}
                    <div className="min-w-0 flex-1">
                      <div className="flex items-start justify-between gap-4">
                        <h2 className="font-display text-xl sm:text-2xl font-semibold text-ink line-clamp-1 pr-24 sm:pr-28">
                          {eventTitle}
                        </h2>

                        {/* Status Badge */}
                        <span
                          className={`absolute right-5 top-5 sm:right-6 sm:top-6 rounded-full px-3 py-1 text-[11px] sm:text-xs font-semibold tracking-wide uppercase ${getStatusBadgeClass(
                            status
                          )}`}
                        >
                          {status}
                        </span>
                      </div>

                      {/* Meta items: Date, Location, Booking ID */}
                      <div className="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1.5 text-xs sm:text-sm text-[#565e74]">
                        <span className="inline-flex items-center gap-1.5">
                          <CalendarIcon className="w-3.5 h-3.5 text-[#565e74]" />
                          {eventDate}
                        </span>
                        <span className="hidden sm:inline text-slate-300">•</span>
                        <span className="inline-flex items-center gap-1.5">
                          <LocationPinIcon className="w-3.5 h-3.5 text-[#565e74]" />
                          {eventLocation}
                        </span>
                        <span className="hidden sm:inline text-slate-300">•</span>
                        <span className="inline-flex items-center gap-1.5 font-mono">
                          <TagReceiptIcon className="w-3.5 h-3.5 text-[#565e74]" />
                          {bookingNumber}
                        </span>
                      </div>

                      {/* Assigned Seat Badges or General Admission indicator */}
                      {assignedSeats.length > 0 ? (
                        <div className="mt-2.5 flex flex-wrap items-center gap-1.5">
                          <span className="text-[11px] font-semibold uppercase tracking-wider text-[#00685f]">
                            Seats:
                          </span>
                          {assignedSeats.map((seat, sIdx) => (
                            <span
                              key={seat.seat_id || sIdx}
                              className="inline-flex items-center gap-1 rounded bg-[#e0efed] px-2 py-0.5 text-xs font-semibold text-[#00685f]"
                            >
                              <span className="size-1 rounded-full bg-[#008378]" />
                              {seat.label || `Row ${seat.row_name}, Seat ${seat.seat_number}`}
                            </span>
                          ))}
                        </div>
                      ) : (
                        <div className="mt-2.5 flex items-center gap-1.5 text-xs text-[#565e74]">
                          <span className="size-1.5 rounded-full bg-slate-300" />
                          <span>General Admission</span>
                        </div>
                      )}

                      {/* Booking sub-details */}
                      {(status === "CONFIRMED" || status === "COMPLETED") && (
                        <p className="mt-3 text-xs sm:text-sm text-[#3d4947]">
                          Booked on {bookedOnDate} • Total: ₹
                          {totalAmount.toLocaleString("en-IN")}
                        </p>
                      )}

                      {status === "CANCELLED" && (
                        <div className="mt-3 space-y-1">
                          <p className="text-xs sm:text-sm text-[#ba1a1a] font-medium">
                            Cancelled on {bookedOnDate}
                          </p>
                          <p className="text-xs sm:text-sm text-[#565e74] inline-flex items-center gap-1.5">
                            <RefundUndoIcon className="w-3.5 h-3.5 text-[#565e74]" />
                            Refund Status: PROCESSING
                          </p>
                        </div>
                      )}

                      {status === "REFUNDED" && (
                        <p className="mt-3 text-xs sm:text-sm text-[#3d4947]">
                          Refund Amount: ₹{totalAmount.toLocaleString("en-IN")}
                        </p>
                      )}

                      {/* Actions Bar */}
                      <div className="mt-4 flex flex-wrap items-center gap-3">
                        <Link
                          to={`/bookings/${bookingId}`}
                          className="inline-flex items-center justify-center rounded-lg border border-[#dfe5e3] bg-white px-4 py-2 text-xs sm:text-sm font-semibold text-[#565e74] hover:bg-slate-50 hover:text-ink transition-colors cursor-pointer"
                        >
                          View Booking
                        </Link>

                        {(status === "CONFIRMED" || status === "COMPLETED") && (
                          <Link
                            to={`/bookings/${bookingId}/ticket`}
                            className="inline-flex items-center justify-center rounded-lg bg-[#00685f] px-4 py-2 text-xs sm:text-sm font-semibold text-white hover:bg-[#00544c] shadow-xs transition-colors cursor-pointer"
                          >
                            View Ticket
                          </Link>
                        )}

                        {(status === "CANCELLED" || status === "REFUNDED") && (
                          <button
                            type="button"
                            onClick={() =>
                              setSelectedRefundBooking({
                                id: bookingNumber,
                                title: eventTitle,
                                bookedOn: bookedOnDate,
                                totalAmount,
                                refundAmount: totalAmount,
                                refundStatus: status === "REFUNDED" ? "REFUNDED" : "PROCESSING",
                                refundDate: bookedOnDate,
                                refundMethod: "Original Payment Method",
                                refundNote:
                                  "Your refund has been initiated and will reflect in your account within 3–5 business days.",
                              })
                            }
                            className="inline-flex items-center justify-center rounded-lg border border-[#dfe5e3] bg-white px-4 py-2 text-xs sm:text-sm font-semibold text-[#565e74] hover:bg-slate-50 hover:text-ink transition-colors cursor-pointer"
                          >
                            View Refund
                          </button>
                        )}
                      </div>
                    </div>
                  </article>
                );
              })
            )}
          </div>
        )}

        {/* Pagination */}
        {!loading && !error && filteredBookings.length > ITEMS_PER_PAGE && (
          <div className="mt-8 sm:mt-10 flex items-center justify-center gap-2">
            <button
              type="button"
              disabled={currentPage <= 1}
              onClick={() => setCurrentPage((p) => Math.max(1, p - 1))}
              className={`flex h-10 w-10 items-center justify-center rounded-lg border border-[#dfe5e3] bg-white text-[#565e74] transition-colors ${
                currentPage <= 1 ? "opacity-40 cursor-not-allowed" : "hover:bg-slate-50 cursor-pointer"
              }`}
              aria-label="Previous page"
            >
              <ChevronLeftIcon className="w-3.5 h-3.5" />
            </button>

            {Array.from({ length: totalPages }, (_, i) => i + 1).map((pageNumber) => (
              <button
                key={pageNumber}
                type="button"
                onClick={() => setCurrentPage(pageNumber)}
                className={`flex h-10 w-10 items-center justify-center rounded-lg border text-xs sm:text-sm font-semibold transition-colors cursor-pointer ${
                  currentPage === pageNumber
                    ? "border-[#00685f] bg-[#00685f] text-white shadow-xs"
                    : "border-[#dfe5e3] bg-white text-[#565e74] hover:bg-slate-50"
                }`}
              >
                {pageNumber}
              </button>
            ))}

            <button
              type="button"
              disabled={currentPage >= totalPages}
              onClick={() => setCurrentPage((p) => Math.min(totalPages, p + 1))}
              className={`flex h-10 w-10 items-center justify-center rounded-lg border border-[#dfe5e3] bg-white text-[#565e74] transition-colors ${
                currentPage >= totalPages ? "opacity-40 cursor-not-allowed" : "hover:bg-slate-50 cursor-pointer"
              }`}
              aria-label="Next page"
            >
              <ChevronRightIcon className="w-3.5 h-3.5" />
            </button>
          </div>
        )}
      </main>

      {/* Refund Details Modal */}
      {selectedRefundBooking && (
        <RefundDetailsModal
          booking={selectedRefundBooking}
          onClose={() => setSelectedRefundBooking(null)}
        />
      )}

      <Footer />
    </div>
  );
}
