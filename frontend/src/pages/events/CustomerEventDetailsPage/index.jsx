import { useState } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
import AppHeader from "../../../components/layout/AppHeader";
import Footer from "../../../components/layout/Footer";
import { getCustomerEventById } from "../mockCustomerEvent";

export default function CustomerEventDetailsPage() {
  const { eventId = "sunfield" } = useParams();
  const navigate = useNavigate();
  const event = getCustomerEventById(eventId);

  const [selectedTicket, setSelectedTicket] = useState("VIP");
  const [quantities, setQuantities] = useState({ VIP: 1, General: 0 });

  const handleQuantityChange = (tierName, delta) => {
    setSelectedTicket(tierName);
    setQuantities((prev) => {
      const current = prev[tierName] || 0;
      const nextVal = Math.max(0, Math.min(9, current + delta));
      return { ...prev, [tierName]: nextVal };
    });
  };

  const handleSelectSeats = () => {
    navigate(`/events/${eventId}/seats`, {
      state: {
        selectedTicket,
        quantity: quantities[selectedTicket] || 1,
      },
    });
  };

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
              <span className="hover:text-[#00685f] transition-colors cursor-pointer">
                {event.category}
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
                  src={event.heroImage}
                  alt={`${event.title} stage and crowd`}
                  className="h-full w-full object-cover"
                />
                <span className="absolute left-4 top-4 inline-flex items-center gap-1.5 rounded-full border border-[#bcc9c6] bg-white/90 backdrop-blur-xs px-3.5 py-1.5 text-xs sm:text-sm font-medium text-[#141b2b] shadow-xs">
                  <span>{event.categoryIcon}</span> {event.category}
                </span>
              </div>

              <div className="flex flex-col gap-5 p-5 sm:p-6 sm:flex-row sm:items-center sm:justify-between">
                <div>
                  <h1 className="font-display text-2xl sm:text-3xl font-bold text-ink leading-tight">
                    {event.title}
                  </h1>
                  <div className="mt-3 flex flex-wrap gap-x-5 gap-y-2 text-xs sm:text-sm text-[#565e74]">
                    <span className="inline-flex items-center gap-1.5">
                      <span className="text-[#00796d]">▣</span> {event.date}
                    </span>
                    <span className="inline-flex items-center gap-1.5">
                      <span className="text-[#00796d]">◷</span> {event.time}
                    </span>
                    <span className="inline-flex items-center gap-1.5">
                      <span className="text-[#00796d]">⌖</span> {event.location}
                    </span>
                  </div>
                </div>
                <span className="inline-flex items-center self-start sm:self-center rounded-full border border-[#bcc9c6] bg-[#e4f1f0] px-4 py-2 text-xs sm:text-sm font-semibold text-[#00685f] whitespace-nowrap">
                  {event.status}
                </span>
              </div>
            </section>

            {/* About this event Card */}
            <section className="rounded-xl border border-[#bcc9c6] bg-white p-5 sm:p-6 shadow-xs">
              <h2 className="mb-4 font-display text-lg sm:text-xl font-bold text-ink">
                About this event
              </h2>
              <p className="text-sm sm:text-base leading-relaxed text-[#565e74]">
                {event.description}
              </p>

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
                      {event.fullDate}
                      <br />
                      {event.timeWithTimezone}
                    </p>
                  </div>
                </div>

                <div className="flex items-start gap-4">
                  <div className="grid h-10 w-10 shrink-0 place-items-center rounded-full bg-[#e4f1f0] text-[#00796d]">
                    <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" />
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z" />
                    </svg>
                  </div>
                  <div>
                    <h3 className="font-semibold text-sm sm:text-base text-ink">Venue</h3>
                    <p className="mt-1 text-xs sm:text-sm text-[#565e74] leading-relaxed whitespace-pre-line">
                      {event.venueAddress}
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
              <h2 className="mb-4 font-display text-lg font-bold text-ink">
                Choose your ticket
              </h2>

              <div className="space-y-4">
                {event.ticketTiers.map((tier) => {
                  const isSelected = selectedTicket === tier.name;
                  const qty = quantities[tier.name] || 0;

                  return (
                    <div
                      key={tier.id}
                      onClick={() => setSelectedTicket(tier.name)}
                      className={`cursor-pointer rounded-lg border p-4 transition-all ${
                        isSelected
                          ? "border-[#00796d] bg-[#f8fffe] shadow-xs"
                          : "border-[#bcc9c6] bg-[#f9f9ff] hover:border-slate-400"
                      }`}
                    >
                      <div className="flex items-center justify-between">
                        <span className="font-semibold text-sm tracking-wide uppercase text-ink">
                          {tier.name}
                        </span>
                        <strong className="text-base font-bold text-ink font-display">
                          {tier.formattedPrice}
                        </strong>
                      </div>

                      <p className="mt-1 text-xs sm:text-sm text-[#565e74]">
                        {tier.description}
                      </p>

                      <div className="mt-4 flex items-center justify-between border-t border-[#d8dfdd] pt-3">
                        <span className="rounded bg-[#dff2ee] px-2 py-0.5 text-xs font-semibold text-[#00796d]">
                          {tier.status}
                        </span>

                        <div
                          className="flex items-center gap-3"
                          onClick={(e) => e.stopPropagation()}
                        >
                          <button
                            type="button"
                            onClick={() => handleQuantityChange(tier.name, -1)}
                            disabled={qty <= 0}
                            className="counter-btn"
                            aria-label={`Decrease ${tier.name} ticket quantity`}
                          >
                            −
                          </button>
                          <span className="w-5 text-center font-bold text-sm text-ink">
                            {qty}
                          </span>
                          <button
                            type="button"
                            onClick={() => handleQuantityChange(tier.name, 1)}
                            disabled={qty >= 9}
                            className="counter-btn"
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

              <div className="my-5 rounded-lg bg-[#eef0ff] p-3 text-xs leading-relaxed text-[#565e74] flex items-start gap-2">
                <span className="text-[#4648d4] font-bold">ⓘ</span>
                <span>
                  Cancellation eligibility depends on the event&apos;s cancellation policy.
                </span>
              </div>

              <button
                type="button"
                onClick={handleSelectSeats}
                className="h-12 w-full rounded-lg bg-[#00796d] px-5 font-bold text-white shadow-xs transition hover:bg-[#00685f] focus:outline-none focus:ring-2 focus:ring-[#00796d]/40"
              >
                Select Seats
              </button>

              <p className="mt-3 text-center text-xs text-[#565e74]">
                Select your seats to continue
              </p>
            </section>

            {/* Organizer Card */}
            <section className="rounded-xl border border-[#bcc9c6] bg-white p-5 sm:p-6 shadow-xs">
              <div className="flex items-center gap-4">
                <img
                  src={event.organizer.avatar}
                  alt={event.organizer.name}
                  className="h-12 w-12 rounded-full object-cover border border-slate-200"
                />
                <div>
                  <h3 className="font-semibold text-ink text-sm sm:text-base">
                    {event.organizer.name}
                  </h3>
                  <p className="text-xs font-medium text-[#00796d] flex items-center gap-1.5 mt-0.5">
                    <span className="h-2 w-2 rounded-full bg-[#00796d]" />
                    Verified Organizer
                  </p>
                </div>
              </div>
              <button
                type="button"
                className="mt-4 text-xs sm:text-sm font-semibold text-[#00685f] hover:underline"
              >
                View Organizer
              </button>
            </section>
          </aside>
        </div>
      </main>

      <Footer />
    </div>
  );
}
