import { useState } from "react";
import { useParams, Link } from "react-router-dom";
import AppHeader from "../../components/layout/AppHeader";
import Footer from "../../components/layout/Footer";
import { getBookingById } from "./mockBookings";
import BookingBreadcrumbs from "./components/BookingBreadcrumbs";
import {
  CalendarIcon,
  LocationPinIcon,
  BuildingIcon,
  TicketIconOutline,
  DownloadIcon,
} from "./components/BookingIcons";

export default function BookingDetailsPage() {
  const { bookingId } = useParams();
  const booking = getBookingById(bookingId);
  const [downloadSuccess, setDownloadSuccess] = useState(false);

  const handleDownload = () => {
    setDownloadSuccess(true);
    setTimeout(() => {
      setDownloadSuccess(false);
      window.print();
    }, 400);
  };

  const breadcrumbs = [
    { label: "My Bookings", to: "/my-bookings" },
    { label: "Booking Details" },
  ];

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
              <strong className="font-semibold text-[#3d4947]">{booking.id}</strong>
              <span className="mx-2">•</span>
              Booked on {booking.bookedOnFull || booking.bookedOn}
            </p>
          </div>

          <span className="self-start sm:self-auto rounded-full bg-[#e0efed] px-3.5 py-1 text-xs font-semibold tracking-wide text-[#00685f] uppercase">
            {booking.status}
          </span>
        </div>

        {/* 2-Column Responsive Layout */}
        <div className="mt-6 sm:mt-8 grid grid-cols-1 lg:grid-cols-[2fr_1fr] gap-6 sm:gap-8 items-start">
          {/* Left Column: Event Card + Ticket Information */}
          <div className="space-y-6">
            {/* Event Summary Card */}
            <section className="overflow-hidden rounded-xl border border-[#dfe5e3] bg-white shadow-xs transition sm:flex">
              <img
                src={booking.stageImage || booking.image}
                alt={`${booking.title} stage`}
                className="h-52 w-full object-cover sm:h-auto sm:w-2/5 min-h-[220px]"
                loading="lazy"
              />
              <div className="flex-1 p-5 sm:p-6 flex flex-col justify-between">
                <div>
                  <h2 className="font-display text-xl sm:text-2xl font-semibold text-ink">
                    {booking.title}
                  </h2>

                  <div className="mt-4 space-y-2.5 text-xs sm:text-sm text-[#3d4947]">
                    <p className="flex items-center gap-2">
                      <CalendarIcon className="w-4 h-4 text-[#565e74] shrink-0" />
                      <span>{booking.dateFull || booking.date} • {booking.time}</span>
                    </p>
                    <p className="flex items-center gap-2">
                      <LocationPinIcon className="w-4 h-4 text-[#565e74] shrink-0" />
                      <span>{booking.venue || booking.location}</span>
                    </p>
                    <p className="flex items-center gap-2">
                      <BuildingIcon className="w-4 h-4 text-[#565e74] shrink-0" />
                      <span>By {booking.organizer}</span>
                    </p>
                  </div>
                </div>

                <div className="mt-6 flex justify-end">
                  <Link
                    to="/events"
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
                    {booking.ticketType || "General"}
                  </strong>
                </div>
                <div>
                  <span className="text-[#565e74] block">Quantity</span>
                  <strong className="mt-1 block font-bold text-ink text-sm sm:text-base">
                    {booking.quantity}
                  </strong>
                </div>
                <div>
                  <span className="text-[#565e74] block">Seats</span>
                  <strong className="mt-1 block font-bold text-ink text-sm sm:text-base">
                    {booking.seats || "Open Seating"}
                  </strong>
                </div>
                <div>
                  <span className="text-[#565e74] block">Price/Ticket</span>
                  <strong className="mt-1 block font-bold text-ink text-sm sm:text-base">
                    ₹{Number(booking.pricePerTicket || 0).toLocaleString("en-IN")}
                  </strong>
                </div>
              </div>
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
                  to={`/bookings/${booking.id}/ticket`}
                  className="block w-full rounded-lg bg-white py-3 text-xs sm:text-sm font-bold text-[#00685f] hover:bg-slate-50 transition-colors shadow-xs"
                >
                  View Ticket
                </Link>

                <button
                  type="button"
                  onClick={handleDownload}
                  className="w-full inline-flex items-center justify-center gap-2 rounded-lg border border-white/30 py-3 text-xs sm:text-sm font-semibold text-white hover:bg-white/10 transition-colors cursor-pointer"
                >
                  <DownloadIcon className="w-4 h-4" />
                  <span>Download Ticket</span>
                </button>
              </div>

              {downloadSuccess && (
                <p className="mt-3 text-xs text-white/90 animate-pulse">
                  Preparing your printable ticket pass...
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
                  <span>Ticket Amount (×{booking.quantity})</span>
                  <span className="font-medium text-[#3d4947]">
                    ₹{Number(booking.ticketAmount || 0).toLocaleString("en-IN")}
                  </span>
                </div>
                <div className="flex justify-between text-[#565e74]">
                  <span>Platform Fee</span>
                  <span className="font-medium text-[#3d4947]">
                    ₹{booking.platformFee}
                  </span>
                </div>
                <div className="flex justify-between text-[#565e74]">
                  <span>Taxes</span>
                  <span className="font-medium text-[#3d4947]">
                    ₹{booking.taxes}
                  </span>
                </div>
              </div>

              <div className="mt-5 flex items-center justify-between border-t border-[#dfe5e3] pt-5">
                <strong className="font-semibold text-ink text-sm sm:text-base">
                  Total Paid
                </strong>
                <strong className="font-display text-2xl font-bold text-[#00685f]">
                  ₹{Number(booking.totalAmount || 0).toLocaleString("en-IN")}
                </strong>
              </div>

              <div className="mt-5 flex items-center justify-between rounded-lg bg-[#eef1ff] p-3 text-xs sm:text-sm text-[#565e74]">
                <span className="font-medium">
                  ▣ Method: {booking.paymentMethod || "UPI"}
                </span>
                <strong className="font-bold text-[#00685f]">
                  {booking.paymentStatus || "PAID"}
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
