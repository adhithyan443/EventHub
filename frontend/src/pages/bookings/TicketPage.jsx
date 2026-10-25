import { useState } from "react";
import { useParams, Link } from "react-router-dom";
import AppHeader from "../../components/layout/AppHeader";
import Footer from "../../components/layout/Footer";
import { getBookingById, getTicketById } from "./mockBookings";
import BookingBreadcrumbs from "./components/BookingBreadcrumbs";
import {
  CalendarIcon,
  ClockIcon,
  LocationPinIcon,
  DownloadIcon,
  InfoCircleIcon,
  TicketIconOutline,
} from "./components/BookingIcons";

export default function TicketPage() {
  const { bookingId, ticketId } = useParams();

  // If ticketId is provided directly, resolve through getTicketById
  const resolved = ticketId
    ? getTicketById(ticketId)
    : { booking: getBookingById(bookingId), ticket: null };

  const booking = resolved.booking;
  const initialTicketId =
    resolved.ticket?.id || booking.individualTickets?.[0]?.id || "EVT-000001";

  const [activeTicketId, setActiveTicketId] = useState(initialTicketId);
  const [downloadSuccess, setDownloadSuccess] = useState(false);

  const activeTicket =
    booking.individualTickets?.find((t) => t.id === activeTicketId) ||
    booking.individualTickets?.[0] || {
      id: "EVT-000001",
      type: booking.ticketType || "VIP",
      seat: booking.seats || "Seat A4",
    };

  const handleDownload = () => {
    setDownloadSuccess(true);
    setTimeout(() => {
      setDownloadSuccess(false);
      window.print();
    }, 400);
  };

  const breadcrumbs = [
    { label: "My Bookings", to: "/my-bookings" },
    { label: "Booking Details", to: `/bookings/${booking.id}` },
    { label: "View Ticket" },
  ];

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
                Your Ticket
              </h1>
              <span className="inline-flex items-center gap-1 rounded-full bg-[#008378] px-3 py-1 text-xs font-semibold text-white tracking-wide uppercase">
                <svg
                  className="w-3.5 h-3.5"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2.5" d="M5 13l4 4L19 7" />
                </svg>
                {booking.status || "CONFIRMED"}
              </span>
            </div>
            <p className="mt-2 text-sm sm:text-base text-[#3d4947]">
              Present this ticket at the event entrance.
            </p>
          </div>

          <Link
            to={`/bookings/${booking.id}`}
            className="self-start sm:self-auto inline-flex items-center justify-center rounded-lg border border-[#bcc9c6] bg-white px-5 py-2.5 text-xs sm:text-sm font-semibold text-ink hover:bg-slate-50 transition-colors"
          >
            Back to Booking
          </Link>
        </div>

        {/* Main Ticket Layout */}
        <div className="mx-auto mt-8 sm:mt-12 grid max-w-[1270px] grid-cols-1 lg:grid-cols-[1fr_320px] items-start gap-6 sm:gap-8">
          {/* Digital Ticket Pass Container */}
          <section className="overflow-hidden rounded-xl border border-[#bcc9c6] bg-white shadow-sm">
            {/* Ticket Header Ribbon */}
            <div className="flex flex-wrap items-center justify-between gap-2 bg-[#e9edff] px-5 sm:px-6 py-2.5 text-xs sm:text-sm tracking-wider font-semibold text-[#3d4947] border-b border-[#bcc9c6]/40 uppercase">
              <span className="flex items-center gap-2">
                <TicketIconOutline className="w-4 h-4 text-[#00685f]" />
                EVENTHUB DIGITAL TICKET
              </span>
              <span>Booking ID: {booking.id}</span>
            </div>

            {/* Ticket Body: Details (left) and Scan Stub (right) */}
            <div className="grid grid-cols-1 lg:grid-cols-[1fr_230px]">
              {/* Event & Ticket Details */}
              <div className="p-5 sm:p-7">
                {/* Event Summary */}
                <div className="flex flex-col sm:flex-row gap-4 items-start">
                  <img
                    src={booking.ticketImage || booking.image}
                    alt={booking.title}
                    className="h-28 w-32 rounded-lg object-cover shrink-0"
                    loading="lazy"
                  />
                  <div>
                    <h2 className="font-display text-xl sm:text-2xl font-semibold text-ink">
                      {booking.title}
                    </h2>
                    <div className="mt-2.5 space-y-1.5 text-xs sm:text-sm text-[#3d4947]">
                      <p className="flex items-center gap-2">
                        <CalendarIcon className="w-4 h-4 text-[#565e74] shrink-0" />
                        <span>{booking.dateFull || booking.date}</span>
                      </p>
                      <p className="flex items-center gap-2">
                        <ClockIcon className="w-4 h-4 text-[#565e74] shrink-0" />
                        <span>{booking.time}</span>
                      </p>
                      <p className="flex items-center gap-2">
                        <LocationPinIcon className="w-4 h-4 text-[#565e74] shrink-0" />
                        <span>{booking.venue || booking.location}</span>
                      </p>
                    </div>
                  </div>
                </div>

                {/* 4-Column Ticket Spec Grid */}
                <div className="mt-6 grid grid-cols-2 sm:grid-cols-4 gap-4 border-t border-[#dfe5e3] pt-5 text-xs sm:text-sm">
                  <div>
                    <span className="text-slate-500 block uppercase tracking-wider text-[11px] font-medium">
                      Type
                    </span>
                    <strong className="mt-1 block font-bold text-ink text-sm sm:text-base">
                      {activeTicket.type || booking.ticketType || "VIP"}
                    </strong>
                  </div>
                  <div>
                    <span className="text-slate-500 block uppercase tracking-wider text-[11px] font-medium">
                      Quantity
                    </span>
                    <strong className="mt-1 block font-bold text-ink text-sm sm:text-base">
                      {booking.quantity}
                    </strong>
                  </div>
                  <div>
                    <span className="text-slate-500 block uppercase tracking-wider text-[11px] font-medium">
                      Price / Ticket
                    </span>
                    <strong className="mt-1 block font-bold text-ink text-sm sm:text-base">
                      ₹{Number(booking.pricePerTicket || 0).toLocaleString("en-IN")}
                    </strong>
                  </div>
                  <div>
                    <span className="text-slate-500 block uppercase tracking-wider text-[11px] font-medium">
                      Organizer
                    </span>
                    <strong className="mt-1 block font-bold text-ink text-sm sm:text-base truncate">
                      {booking.organizer}
                    </strong>
                  </div>
                </div>

                {/* Individual Tickets Sub-card */}
                <div className="mt-6 rounded-xl bg-[#eef1ff] p-4 sm:p-5">
                  <div className="flex items-center justify-between mb-3">
                    <p className="font-semibold text-xs sm:text-sm text-[#141b2b]">
                      Individual Tickets ({booking.quantity})
                    </p>
                    <span className="text-[11px] text-slate-500">
                      Click to switch ticket QR
                    </span>
                  </div>

                  <div className="space-y-2">
                    {booking.individualTickets?.map((item) => {
                      const isSelected = activeTicketId === item.id;
                      return (
                        <button
                          key={item.id}
                          type="button"
                          onClick={() => setActiveTicketId(item.id)}
                          className={`w-full flex items-center justify-between p-2.5 rounded-lg border transition-all cursor-pointer text-xs sm:text-sm ${
                            isSelected
                              ? "bg-white border-[#00685f] shadow-xs"
                              : "bg-white/60 border-transparent hover:bg-white/90"
                          }`}
                        >
                          <span className="flex items-center gap-2 font-medium text-slate-800">
                            <span
                              className={`w-2 h-2 rounded-full ${
                                isSelected ? "bg-[#008378]" : "bg-slate-300"
                              }`}
                            />
                            {item.type} | {item.seat}
                          </span>
                          <code className="rounded bg-[#eef1ff] px-2 py-0.5 font-mono text-xs font-bold text-slate-800">
                            {item.id}
                          </code>
                        </button>
                      );
                    })}
                  </div>
                </div>
              </div>

              {/* Right: Tear-off QR Stub */}
              <div className="border-t border-dashed border-[#bcc9c6] p-6 text-center lg:border-l lg:border-t-0 flex flex-col items-center justify-between bg-slate-50/40">
                <div className="w-full flex flex-col items-center">
                  <span className="text-xs font-bold tracking-wider text-[#565e74] uppercase">
                    Scan for Entry
                  </span>

                  {/* QR Code Graphic */}
                  <div className="mx-auto mt-4 rounded-xl bg-white p-3 border border-slate-200/80 shadow-xs">
                    <img
                      src={booking.qrCode || "/assets/570fe.svg"}
                      alt="Ticket QR code"
                      className="h-40 w-40 sm:h-44 sm:w-44 object-contain"
                    />
                  </div>

                  {/* Ticket ID display */}
                  <p className="mt-4 text-xs text-[#565e74]">Ticket ID</p>
                  <code className="mt-1 inline-block rounded bg-[#eef1ff] px-3 py-1 font-mono text-xs sm:text-sm font-bold tracking-widest text-[#141b2b] border border-slate-200/60">
                    {activeTicket.id}
                  </code>
                </div>

                <div className="w-full mt-6">
                  <button
                    type="button"
                    onClick={handleDownload}
                    className="w-full inline-flex items-center justify-center gap-2 rounded-lg bg-[#00685f] py-3 px-4 text-xs sm:text-sm font-semibold text-white hover:bg-[#00544c] transition-colors shadow-xs cursor-pointer"
                  >
                    <DownloadIcon className="w-4 h-4" />
                    <span>⇩ Download</span>
                  </button>

                  {downloadSuccess && (
                    <p className="mt-2 text-[11px] text-emerald-700 font-medium">
                      Download initiated...
                    </p>
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
              <dt className="text-[#3d4947]">Doors Open</dt>
              <dd className="text-right font-semibold text-ink">
                {booking.doorsOpen || "5:00 PM"}
              </dd>

              <dt className="text-[#3d4947]">Event Starts</dt>
              <dd className="text-right font-semibold text-ink">
                {booking.time?.split("–")?.[0]?.trim() || "6:00 PM"}
              </dd>

              <dt className="text-[#3d4947]">Entry Type</dt>
              <dd className="text-right font-semibold text-ink">
                {booking.entryType || "VIP Gate"}
              </dd>

              <dt className="text-[#3d4947]">Status</dt>
              <dd className="text-right font-bold text-[#008378] flex items-center justify-end gap-1">
                <span className="w-2 h-2 rounded-full bg-[#008378]" />
                Valid
              </dd>
            </dl>

            <div className="mt-6 rounded-lg border border-[#bcc9c6] bg-[#eef1ff] p-4 text-center font-body italic text-xs sm:text-sm text-[#3d4947]">
              "{booking.entryNote || 'Please arrive early and keep this ticket ready for scanning to ensure a smooth entry.'}"
            </div>
          </aside>
        </div>
      </main>

      <Footer />
    </div>
  );
}
