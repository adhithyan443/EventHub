import { useState, useMemo } from "react";
import { Link } from "react-router-dom";
import AppHeader from "../../components/layout/AppHeader";
import Footer from "../../components/layout/Footer";
import { MOCK_BOOKINGS } from "./mockBookings";
import {
  CalendarIcon,
  LocationPinIcon,
  TagReceiptIcon,
  RefundUndoIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
} from "./components/BookingIcons";
import RefundDetailsModal from "./components/RefundDetailsModal";

export default function MyBookingsPage() {
  const [tab, setTab] = useState("All");
  const [searchQuery, setSearchQuery] = useState("");
  const [sortBy, setSortBy] = useState("newest");
  const [currentPage, setCurrentPage] = useState(1);
  const [selectedRefundBooking, setSelectedRefundBooking] = useState(null);

  const tabs = ["All", "Confirmed", "Cancelled", "Refunded"];

  // Filter bookings based on tab and search
  const filteredBookings = useMemo(() => {
    return MOCK_BOOKINGS.filter((booking) => {
      // Tab filter
      const matchesTab =
        tab === "All" ||
        booking.status.toLowerCase() === tab.toLowerCase();

      // Search filter
      const query = searchQuery.trim().toLowerCase();
      const matchesSearch =
        !query ||
        booking.title.toLowerCase().includes(query) ||
        booking.id.toLowerCase().includes(query) ||
        booking.location.toLowerCase().includes(query);

      return matchesTab && matchesSearch;
    }).sort((a, b) => {
      if (sortBy === "oldest") {
        return a.id.localeCompare(b.id);
      }
      return b.id.localeCompare(a.id);
    });
  }, [tab, searchQuery, sortBy]);

  const getStatusBadgeClass = (status) => {
    switch (status) {
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
            View and manage all your event bookings.
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
                  onClick={() => setTab(item)}
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
                onChange={(e) => setSearchQuery(e.target.value)}
                placeholder="Search by event name or booking ID"
                className="h-11 sm:h-12 w-full rounded-lg border border-[#e3e6e5] bg-white pl-9 pr-4 text-xs sm:text-sm text-[#141b2b] placeholder:text-[#565e74] outline-none transition focus:border-[#008378] focus:ring-1 focus:ring-[#008378]"
              />
            </div>

            <select
              value={sortBy}
              onChange={(e) => setSortBy(e.target.value)}
              className="h-11 sm:h-12 rounded-lg border border-[#e3e6e5] bg-white px-3 sm:px-4 text-xs sm:text-sm text-[#565e74] outline-none transition focus:border-[#008378] cursor-pointer"
            >
              <option value="newest">Newest First</option>
              <option value="oldest">Oldest First</option>
            </select>
          </div>
        </div>

        {/* Bookings List */}
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
                  : `You have no ${tab.toLowerCase()} bookings at this time.`}
              </p>
              {(searchQuery || tab !== "All") && (
                <button
                  type="button"
                  onClick={() => {
                    setTab("All");
                    setSearchQuery("");
                  }}
                  className="mt-4 inline-flex items-center rounded-lg bg-[#00685f] px-4 py-2 text-xs sm:text-sm font-semibold text-white hover:bg-[#00544c] transition-colors cursor-pointer"
                >
                  Reset Filters
                </button>
              )}
            </div>
          ) : (
            filteredBookings.map((booking) => (
              <article
                key={booking.id}
                className="relative flex flex-col gap-5 rounded-xl border border-[#dfe5e3] bg-white p-5 sm:p-6 shadow-xs transition hover:shadow-sm sm:flex-row sm:items-start"
              >
                {/* Event Thumbnail */}
                <img
                  src={booking.image}
                  alt={booking.title}
                  className="h-44 w-full rounded-lg object-cover sm:h-36 sm:w-48 shrink-0"
                  loading="lazy"
                />

                {/* Event Booking Details */}
                <div className="min-w-0 flex-1">
                  <div className="flex items-start justify-between gap-4">
                    <h2 className="font-display text-xl sm:text-2xl font-semibold text-ink line-clamp-1 pr-24 sm:pr-28">
                      {booking.title}
                    </h2>

                    {/* Status Badge */}
                    <span
                      className={`absolute right-5 top-5 sm:right-6 sm:top-6 rounded-full px-3 py-1 text-[11px] sm:text-xs font-semibold tracking-wide uppercase ${getStatusBadgeClass(
                        booking.status
                      )}`}
                    >
                      {booking.status}
                    </span>
                  </div>

                  {/* Meta items: Date, Location, Booking ID */}
                  <div className="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1.5 text-xs sm:text-sm text-[#565e74]">
                    <span className="inline-flex items-center gap-1.5">
                      <CalendarIcon className="w-3.5 h-3.5 text-[#565e74]" />
                      {booking.date}
                    </span>
                    <span className="hidden sm:inline text-slate-300">•</span>
                    <span className="inline-flex items-center gap-1.5">
                      <LocationPinIcon className="w-3.5 h-3.5 text-[#565e74]" />
                      {booking.location}
                    </span>
                    <span className="hidden sm:inline text-slate-300">•</span>
                    <span className="inline-flex items-center gap-1.5">
                      <TagReceiptIcon className="w-3.5 h-3.5 text-[#565e74]" />
                      ID: {booking.id}
                    </span>
                  </div>

                  {/* Booking sub-details */}
                  {booking.status === "CONFIRMED" && (
                    <p className="mt-3 text-xs sm:text-sm text-[#3d4947]">
                      Booked on {booking.bookedOn} • {booking.quantity} Tickets (Seats {booking.seats}) • Total: ₹
                      {Number(booking.totalAmount).toLocaleString("en-IN")}
                    </p>
                  )}

                  {booking.status === "COMPLETED" && (
                    <p className="mt-3 text-xs sm:text-sm text-[#3d4947]">
                      Booked on {booking.bookedOn} • {booking.quantity} Ticket • Total: ₹
                      {Number(booking.totalAmount).toLocaleString("en-IN")}
                    </p>
                  )}

                  {booking.status === "CANCELLED" && (
                    <div className="mt-3 space-y-1">
                      <p className="text-xs sm:text-sm text-[#ba1a1a] font-medium">
                        {booking.detail || `Cancelled on ${booking.bookedOn}`}
                      </p>
                      <p className="text-xs sm:text-sm text-[#565e74] inline-flex items-center gap-1.5">
                        <RefundUndoIcon className="w-3.5 h-3.5 text-[#565e74]" />
                        Refund Status: {booking.refundStatus || "PROCESSING"}
                      </p>
                    </div>
                  )}

                  {booking.status === "REFUNDED" && (
                    <p className="mt-3 text-xs sm:text-sm text-[#3d4947]">
                      {booking.detail || `Refund Amount: ₹${booking.refundAmount}`}
                    </p>
                  )}

                  {/* Actions Bar */}
                  <div className="mt-4 flex flex-wrap items-center gap-3">
                    {booking.actions?.map((action) => {
                      if (action === "View Booking") {
                        return (
                          <Link
                            key={action}
                            to={`/bookings/${booking.id}`}
                            className="inline-flex items-center justify-center rounded-lg border border-[#dfe5e3] bg-white px-4 py-2 text-xs sm:text-sm font-semibold text-[#565e74] hover:bg-slate-50 hover:text-ink transition-colors cursor-pointer"
                          >
                            {action}
                          </Link>
                        );
                      }

                      if (action === "View Ticket") {
                        return (
                          <Link
                            key={action}
                            to={`/bookings/${booking.id}/ticket`}
                            className={`inline-flex items-center justify-center rounded-lg px-4 py-2 text-xs sm:text-sm font-semibold transition-colors cursor-pointer ${
                              booking.status === "CONFIRMED"
                                ? "bg-[#00685f] text-white hover:bg-[#00544c] shadow-xs"
                                : "border border-[#dfe5e3] bg-white text-[#565e74] hover:bg-slate-50 hover:text-ink"
                            }`}
                          >
                            {action}
                          </Link>
                        );
                      }

                      if (action === "View Refund") {
                        return (
                          <button
                            key={action}
                            type="button"
                            onClick={() => setSelectedRefundBooking(booking)}
                            className="inline-flex items-center justify-center rounded-lg border border-[#dfe5e3] bg-white px-4 py-2 text-xs sm:text-sm font-semibold text-[#565e74] hover:bg-slate-50 hover:text-ink transition-colors cursor-pointer"
                          >
                            {action}
                          </button>
                        );
                      }

                      return null;
                    })}
                  </div>
                </div>
              </article>
            ))
          )}
        </div>

        {/* Pagination */}
        <div className="mt-8 sm:mt-10 flex items-center justify-center gap-2">
          <button
            type="button"
            onClick={() => setCurrentPage((p) => Math.max(1, p - 1))}
            className="flex h-10 w-10 items-center justify-center rounded-lg border border-[#dfe5e3] bg-white text-[#565e74] hover:bg-slate-50 transition-colors cursor-pointer"
            aria-label="Previous page"
          >
            <ChevronLeftIcon className="w-3.5 h-3.5" />
          </button>

          {[1, 2, 3].map((pageNumber) => (
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
            onClick={() => setCurrentPage((p) => Math.min(3, p + 1))}
            className="flex h-10 w-10 items-center justify-center rounded-lg border border-[#dfe5e3] bg-white text-[#565e74] hover:bg-slate-50 transition-colors cursor-pointer"
            aria-label="Next page"
          >
            <ChevronRightIcon className="w-3.5 h-3.5" />
          </button>
        </div>
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
