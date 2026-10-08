import { useState, useEffect } from "react";
import { useParams, useNavigate, useLocation, Link } from "react-router-dom";
import AppHeader from "../../../components/layout/AppHeader";
import Footer from "../../../components/layout/Footer";
import OrderPriceSummary from "../components/OrderPriceSummary";
import { getCustomerEventById } from "../mockCustomerEvent";

export default function SeatSelectionPage() {
  const { eventId = "sunfield" } = useParams();
  const navigate = useNavigate();
  const location = useLocation();
  const event = getCustomerEventById(eventId);

  const [selectedSeats, setSelectedSeats] = useState(
    location.state?.selectedSeats || event.initialSelectedSeats
  );

  const reservedSeats = event.initialReservedSeats;
  const bookedSeats = event.initialBookedSeats;
  const disabledSeats = event.initialDisabledSeats;

  // Reservation countdown timer simulation (default 09:58)
  const [timeLeft, setTimeLeft] = useState(598);

  useEffect(() => {
    const timer = setInterval(() => {
      setTimeLeft((prev) => (prev > 0 ? prev - 1 : 0));
    }, 1000);
    return () => clearInterval(timer);
  }, []);

  const formatTimer = (seconds) => {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${String(mins).padStart(2, "0")}:${String(secs).padStart(2, "0")}`;
  };

  const toggleSeat = (seatId) => {
    if (
      reservedSeats.includes(seatId) ||
      bookedSeats.includes(seatId) ||
      disabledSeats.includes(seatId)
    ) {
      return;
    }

    setSelectedSeats((prev) =>
      prev.includes(seatId)
        ? prev.filter((id) => id !== seatId)
        : [...prev, seatId]
    );
  };

  // Pricing calculations
  const calculatePricing = () => {
    if (selectedSeats.length === 0) {
      return {
        subtotal: "₹0",
        platformFee: "₹0",
        taxes: "₹0",
        total: "₹0",
        rawTotal: 0,
      };
    }

    // Estimate based on rows (Row A = 1499, Row B = 999, Row C/D = 499)
    let subtotalNum = 0;
    selectedSeats.forEach((seat) => {
      const row = seat[0];
      if (row === "A") subtotalNum += 1499;
      else if (row === "B") subtotalNum += 999;
      else subtotalNum += 499;
    });

    const feeNum = 99;
    const taxesNum = Math.round(subtotalNum * 0.06);
    const totalNum = subtotalNum + feeNum + taxesNum;

    return {
      subtotal: `₹${subtotalNum.toLocaleString("en-IN")}`,
      platformFee: `₹${feeNum}`,
      taxes: `₹${taxesNum.toLocaleString("en-IN")}`,
      total: `₹${totalNum.toLocaleString("en-IN")}`,
      rawTotal: totalNum,
    };
  };

  const pricing = calculatePricing();

  const handleContinueToCheckout = () => {
    if (selectedSeats.length === 0) return;
    navigate(`/events/${eventId}/checkout`, {
      state: {
        selectedSeats,
        pricing,
      },
    });
  };

  return (
    <div className="flex min-h-screen flex-col bg-[#f9f9ff] text-[#141b2b]">
      <AppHeader />

      <main className="mx-auto w-full max-w-[1440px] flex-1 px-4 sm:px-6 lg:px-8 py-6 sm:py-8">
        {/* Navigation back link */}
        <div className="mb-4 sm:mb-6">
          <Link
            to={`/events/${eventId}`}
            className="inline-flex items-center gap-2 text-xs sm:text-sm font-semibold text-[#00685f] hover:text-[#00796d] transition-colors"
          >
            <span>←</span> Back to Event Details
          </Link>
        </div>

        <div className="grid gap-8 lg:grid-cols-[minmax(0,2fr)_minmax(330px,1fr)] items-start">
          {/* Left Column: Interactive Seat Grid & Legend */}
          <div>
            <h1 className="font-display text-2xl sm:text-3xl lg:text-[32px] font-bold text-ink leading-tight">
              Select Your Seats
            </h1>
            <p className="mt-2 text-xs sm:text-sm text-[#3d4947] flex items-center gap-1.5 flex-wrap">
              <span className="text-[#00796d]">♜</span> {event.title}, {event.date}, {event.location}
            </p>

            <div className="mt-3 inline-flex items-center gap-2 rounded border border-[#bcc9c6] bg-[#f1f3ff] px-3 py-1.5 text-xs text-[#565e74]">
              <span className="text-[#4648d4] font-semibold">ⓘ</span>
              <span>Selected seats are temporarily reserved for 10 minutes.</span>
            </div>

            {/* Status Legend Card */}
            <section className="my-6 rounded-xl border border-[#bcc9c6] bg-white p-4 shadow-xs">
              <div className="flex flex-wrap items-center justify-center gap-4 sm:gap-6 text-xs sm:text-sm font-medium text-[#565e74]">
                <span className="flex items-center gap-2">
                  <i className="h-5 w-5 rounded border-2 border-[#00796d] bg-white" />
                  Available
                </span>
                <span className="flex items-center gap-2">
                  <i className="h-5 w-5 rounded bg-[#00796d] shadow-xs" />
                  Selected
                </span>
                <span className="flex items-center gap-2">
                  <i className="h-5 w-5 rounded bg-[#565e74]" />
                  Reserved
                </span>
                <span className="flex items-center gap-2">
                  <i className="h-5 w-5 rounded border border-[#bcc9c6] bg-[#dce2f7]" />
                  Booked
                </span>
                <span className="flex items-center gap-2">
                  <i className="h-5 w-5 rounded border border-[#bcc9c6] bg-[#f3f4f6]" />
                  Disabled
                </span>
              </div>
            </section>

            {/* Responsive Seat Map Card */}
            <section className="rounded-xl border border-[#bcc9c6] bg-white p-4 sm:p-6 lg:p-8 shadow-xs overflow-hidden">
              <div className="overflow-x-auto pb-4">
                <div className="mx-auto min-w-[560px] max-w-[660px]">
                  {/* STAGE Graphic */}
                  <div className="mx-auto mb-8 w-[72%] rounded-b-[38px] border border-[#bcc9c6] bg-[#dce2f7] py-4 sm:py-5 text-center text-xl sm:text-2xl font-bold tracking-[.28em] text-[#565e74] shadow-xs">
                    STAGE
                  </div>

                  {/* Seat Grid Rows */}
                  <div className="space-y-6 sm:space-y-7">
                    {event.seatRows.map((row) => (
                      <div key={row.name} className="space-y-2">
                        {row.type && (
                          <div className="flex items-center gap-4 text-xs sm:text-sm font-bold text-[#00685f]">
                            <span>{row.type}</span>
                            <span className="h-px flex-1 bg-[#bcc9c6]" />
                          </div>
                        )}

                        <div className="flex items-center justify-center gap-2 sm:gap-2.5">
                          <span className="w-5 text-center font-bold text-xs sm:text-sm text-[#565e74]">
                            {row.name}
                          </span>

                          <div className="flex items-center gap-1.5 sm:gap-2">
                            {row.seats.map((number) => {
                              const seatId = `${row.name}${number}`;
                              const isSelected = selectedSeats.includes(seatId);
                              const isReserved = reservedSeats.includes(seatId);
                              const isBooked = bookedSeats.includes(seatId);
                              const isDisabled = disabledSeats.includes(seatId);

                              let stateClass = "available";
                              if (isSelected) stateClass = "selected";
                              else if (isReserved) stateClass = "reserved";
                              else if (isBooked) stateClass = "booked";
                              else if (isDisabled) stateClass = "disabled";

                              return (
                                <button
                                  key={seatId}
                                  type="button"
                                  onClick={() => toggleSeat(seatId)}
                                  aria-label={`Seat ${seatId} ${stateClass}`}
                                  aria-pressed={isSelected}
                                  className={`seat ${stateClass}`}
                                >
                                  {isDisabled ? "×" : number}
                                </button>
                              );
                            })}
                          </div>

                          <span className="w-5 text-center font-bold text-xs sm:text-sm text-[#565e74]">
                            {row.name}
                          </span>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            </section>
          </div>

          {/* Right Sidebar: Selection Summary */}
          <aside className="space-y-4">
            {/* Timer Pill */}
            <div className="rounded-lg border border-[#add1cc] bg-[#dff0f2] py-3 text-center text-xs sm:text-sm font-semibold text-[#00685f]">
              <span className="mr-1">◷</span> Seats reserved for{" "}
              <span className="font-mono font-bold tracking-wider ml-1">
                {formatTimer(timeLeft)}
              </span>
            </div>

            {/* Selection Summary Card */}
            <section className="rounded-xl border border-[#bcc9c6] bg-white shadow-xs overflow-hidden">
              <h2 className="bg-[#f1f3ff] px-5 py-4 font-display text-xl sm:text-2xl font-bold text-ink border-b border-[#bcc9c6]/50">
                Your Selection
              </h2>

              <div className="p-5 sm:p-6">
                <div className="mb-5 flex justify-between border-b border-[#d8dfdd] pb-5">
                  <div>
                    <span className="text-[11px] font-semibold tracking-wider uppercase text-[#565e74]">
                      SEATS SELECTED
                    </span>
                    <strong className="mt-1 block text-sm sm:text-base font-bold text-ink">
                      {selectedSeats.length > 0
                        ? selectedSeats.join(", ")
                        : "None"}
                    </strong>
                  </div>

                  <div className="text-right">
                    <span className="text-[11px] font-semibold tracking-wider uppercase text-[#565e74]">
                      QUANTITY
                    </span>
                    <strong className="mt-1 block text-sm sm:text-base font-bold text-ink">
                      {selectedSeats.length}
                    </strong>
                  </div>
                </div>

                <OrderPriceSummary
                  subtotal={pricing.subtotal}
                  platformFee={pricing.platformFee}
                  taxes={pricing.taxes}
                  total={pricing.total}
                  className="mb-6"
                />

                <button
                  type="button"
                  onClick={handleContinueToCheckout}
                  disabled={selectedSeats.length === 0}
                  className="h-12 w-full rounded-lg bg-[#00796d] px-5 font-bold text-white shadow-xs transition hover:bg-[#00685f] disabled:cursor-not-allowed disabled:bg-[#9db5b1] focus:outline-none focus:ring-2 focus:ring-[#00796d]/40"
                >
                  Continue to Checkout →
                </button>
              </div>
            </section>

            <p className="text-center text-xs text-[#565e74] flex items-center justify-center gap-1.5">
              <span>♙</span> Secure transaction
            </p>
          </aside>
        </div>
      </main>

      <Footer />
    </div>
  );
}
