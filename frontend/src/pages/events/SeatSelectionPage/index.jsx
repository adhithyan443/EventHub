import { useState, useEffect, useRef } from "react";
import { useParams, useNavigate, useLocation, Link } from "react-router-dom";
import AppHeader from "../../../components/layout/AppHeader";
import Footer from "../../../components/layout/Footer";
import OrderPriceSummary from "../components/OrderPriceSummary";
import { getPublicEventById, getPublicEventSeatLayout } from "../../../api/eventApi";
import { createReservation } from "../../../api/reservationApi";
import useAuthStore from "../../../store/authStore";

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

export default function SeatSelectionPage() {
  const { eventId } = useParams();
  const navigate = useNavigate();
  const location = useLocation();

  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);

  const [event, setEvent] = useState(null);
  const [seatLayout, setSeatLayout] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [isNotFound, setIsNotFound] = useState(false);

  // Array of selected seat objects: { id, seatNumber, rowName, sectionName, price }
  const [selectedSeats, setSelectedSeats] = useState([]);

  const [isReserving, setIsReserving] = useState(false);
  const [reservationError, setReservationError] = useState(null);
  const reservingRef = useRef(false);

  const [retryCount, setRetryCount] = useState(0);

  useEffect(() => {
    let cancelled = false;

    const fetchSeatData = async () => {
      if (!eventId) {
        setIsNotFound(true);
        setError("Event not found");
        setLoading(false);
        return;
      }

      setLoading(true);
      setError(null);
      setIsNotFound(false);

      try {
        const [eventRes, layoutRes] = await Promise.all([
          getPublicEventById(eventId),
          getPublicEventSeatLayout(eventId),
        ]);

        if (cancelled) return;

        const eventData = eventRes?.data;
        if (!eventData) {
          throw new Error("Event details not found");
        }
        setEvent(eventData);

        const layoutData = layoutRes?.data?.seat_layout || layoutRes?.data?.layout || layoutRes?.data;
        setSeatLayout(layoutData);
      } catch (err) {
        if (cancelled) return;
        const status = err?.response?.status;
        if (status === 404 || status === 400) {
          setIsNotFound(true);
          setError("Event or seat layout not found");
        } else {
          setError(
            err?.response?.data?.message ||
              "Unable to load seat layout. Please try again."
          );
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };

    fetchSeatData();

    return () => {
      cancelled = true;
    };
  }, [eventId, retryCount]);

  const bookingLimit = event?.setting?.booking_limit_per_user || 9;

  const toggleSeat = (seat, row, section) => {
    if (seat.status !== "AVAILABLE" && !selectedSeats.some((s) => s.id === seat.id)) {
      return;
    }

    setReservationError(null);

    setSelectedSeats((prev) => {
      const exists = prev.some((s) => s.id === seat.id);
      if (exists) {
        return prev.filter((s) => s.id !== seat.id);
      }

      if (prev.length >= bookingLimit) {
        setReservationError(`You can reserve up to ${bookingLimit} seats per booking.`);
        return prev;
      }

      return [
        ...prev,
        {
          id: seat.id,
          seatNumber: seat.seat_number,
          rowName: row.row_name,
          sectionName: section.name,
          price: section.price,
        },
      ];
    });
  };

  // Dynamic pricing calculation based on selected seats and real section prices
  const subtotalNum = selectedSeats.reduce(
    (acc, seat) => acc + (Number(seat.price) || 0),
    0
  );
  const platformFeeNum = subtotalNum > 0 ? 99 : 0;
  const taxesNum = Math.round(subtotalNum * 0.06);
  const totalNum = subtotalNum + platformFeeNum + taxesNum;

  const pricing = {
    subtotal: `₹${subtotalNum.toLocaleString("en-IN")}`,
    platformFee: `₹${platformFeeNum.toLocaleString("en-IN")}`,
    taxes: `₹${taxesNum.toLocaleString("en-IN")}`,
    total: `₹${totalNum.toLocaleString("en-IN")}`,
    rawTotal: totalNum,
  };

  const handleContinueToCheckout = async () => {
    if (selectedSeats.length === 0) return;

    if (!isAuthenticated) {
      navigate("/login", {
        state: { from: `${location.pathname}${location.search}` },
      });
      return;
    }

    if (reservingRef.current) return;
    reservingRef.current = true;
    setIsReserving(true);
    setReservationError(null);

    const cacheKey = `eventhub_reservation_${eventId}`;
    const selectedSeatIds = selectedSeats.map((s) => s.id);

    try {
      // Client-side cache check as an optimization, never as proof of inventory
      const cachedRaw = sessionStorage.getItem(cacheKey);
      if (cachedRaw) {
        try {
          const cached = JSON.parse(cachedRaw);
          const expiresAtRaw =
            cached.reservation?.ExpiresAt || cached.reservation?.expires_at;
          const expiresAtTime = expiresAtRaw ? new Date(expiresAtRaw).getTime() : 0;

          // Check if cached reservation matches the exact same seats and has > 30s left
          const cachedSeatIds = (cached.selectedSeatObjects || []).map((s) => s.id);
          const seatsMatch =
            cachedSeatIds.length === selectedSeatIds.length &&
            selectedSeatIds.every((id) => cachedSeatIds.includes(id));

          if (seatsMatch && expiresAtTime - Date.now() > 30000) {
            navigate(`/events/${eventId}/checkout`, {
              state: {
                reservation: cached.reservation,
                selectedSeats: selectedSeats.map((s) => `${s.rowName}${s.seatNumber}`),
                selectedSeatObjects: selectedSeats,
                pricing,
                quantity: selectedSeats.length,
                ticketPrice: selectedSeats[0]?.price,
              },
            });
            return;
          } else {
            sessionStorage.removeItem(cacheKey);
          }
        } catch {
          sessionStorage.removeItem(cacheKey);
        }
      }

      const response = await createReservation(eventId, {
        seat_ids: selectedSeatIds,
      });

      const reservationData = response?.data?.reservation;
      if (!reservationData) {
        throw new Error("Invalid reservation response from server");
      }

      const cachePayload = {
        reservation: reservationData,
        selectedSeats: selectedSeats.map((s) => `${s.rowName}${s.seatNumber}`),
        selectedSeatObjects: selectedSeats,
        pricing,
        quantity: selectedSeats.length,
        ticketPrice: selectedSeats[0]?.price,
      };

      sessionStorage.setItem(cacheKey, JSON.stringify(cachePayload));

      navigate(`/events/${eventId}/checkout`, {
        state: cachePayload,
      });
    } catch (err) {
      const status = err?.response?.status;
      if (status === 409) {
        setReservationError(
          "One or more of your selected seats are no longer available. Please select different seats."
        );
        // Refresh layout so user sees updated seat availability
        setRetryCount((c) => c + 1);
      } else if (status === 401) {
        setReservationError("Your session has expired. Please log in again.");
      } else {
        setReservationError(
          err?.response?.data?.message ||
            "Failed to reserve seats. Please try again."
        );
      }
    } finally {
      reservingRef.current = false;
      setIsReserving(false);
    }
  };

  if (loading) {
    return (
      <div className="flex min-h-screen flex-col bg-[#f9f9ff] text-[#141b2b]">
        <AppHeader />
        <main className="mx-auto w-full max-w-[1440px] flex-1 px-4 sm:px-6 lg:px-8 py-16 flex items-center justify-center">
          <div className="flex flex-col items-center gap-4 bg-white border border-[#bcc9c6] rounded-xl p-10 shadow-xs max-w-md w-full text-center">
            <div className="size-10 border-3 border-[#00685f]/30 border-t-[#00685f] rounded-full animate-spin" />
            <h2 className="font-display font-semibold text-lg text-ink">
              Loading seat layout...
            </h2>
            <p className="text-xs text-[#565e74]">
              Fetching real-time seat availability from the venue.
            </p>
          </div>
        </main>
        <Footer />
      </div>
    );
  }

  if (error || !event || !seatLayout) {
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
              {isNotFound ? "Seat Layout Not Found" : "Unable to Load Seats"}
            </h2>
            <p className="text-sm text-[#565e74] leading-relaxed">
              {error || "The seat layout for this event is currently unavailable."}
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
                to={eventId ? `/events/${eventId}` : "/events"}
                className="rounded-lg border border-[#bcc9c6] bg-white px-5 py-2.5 text-sm font-semibold text-[#141b2b] hover:bg-slate-50 transition-colors"
              >
                Back to Event Details
              </Link>
            </div>
          </div>
        </main>
        <Footer />
      </div>
    );
  }

  const sections = seatLayout.sections || [];

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
            <div className="flex items-center gap-3 flex-wrap">
              <span className="text-[11px] font-bold uppercase tracking-wider text-[#4338ca] bg-[#e0e7ff] px-2.5 py-0.5 rounded-full">
                Reserved Seating
              </span>
              {bookingLimit && (
                <span className="text-xs text-[#565e74]">
                  Limit: up to {bookingLimit} seats per booking
                </span>
              )}
            </div>

            <h1 className="mt-2 font-display text-2xl sm:text-3xl lg:text-[32px] font-bold text-ink leading-tight">
              Select Your Seats
            </h1>
            <p className="mt-2 text-xs sm:text-sm text-[#3d4947] flex items-center gap-2 flex-wrap">
              <span className="font-semibold text-ink">{event.title}</span>
              <span className="text-[#8a8fa0]">•</span>
              <span>{formatDate(event.schedule?.event_date)}</span>
              <span className="text-[#8a8fa0]">•</span>
              <span>{formatVenueLocation(event)}</span>
            </p>

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
                <div className="mx-auto min-w-[560px] max-w-[720px]">
                  {/* STAGE Graphic */}
                  <div className="mx-auto mb-8 w-[72%] rounded-b-[38px] border border-[#bcc9c6] bg-[#dce2f7] py-4 sm:py-5 text-center text-xl sm:text-2xl font-bold tracking-[.28em] text-[#565e74] shadow-xs">
                    STAGE
                  </div>

                  {sections.length === 0 ? (
                    <div className="p-8 text-center text-xs text-[#565e74]">
                      No seating sections configured for this venue.
                    </div>
                  ) : (
                    <div className="space-y-8">
                      {sections.map((section) => (
                        <div key={section.id} className="space-y-4">
                          {/* Section Header */}
                          <div className="flex items-center justify-between border-b border-[#e2e8f0] pb-2">
                            <span className="font-display font-bold text-sm sm:text-base text-ink">
                              {section.name}
                            </span>
                            <span className="text-xs sm:text-sm font-bold text-[#00685f]">
                              ₹{Number(section.price).toLocaleString("en-IN")}
                            </span>
                          </div>

                          {/* Rows inside section */}
                          <div className="space-y-3">
                            {(section.rows || []).map((row) => (
                              <div
                                key={row.id}
                                className="flex items-center justify-center gap-2 sm:gap-2.5"
                              >
                                <span className="w-6 text-center font-bold text-xs sm:text-sm text-[#565e74]">
                                  {row.row_name}
                                </span>

                                <div className="flex items-center gap-1.5 sm:gap-2 flex-wrap justify-center">
                                  {(row.seats || []).map((seat) => {
                                    const isSelected = selectedSeats.some((s) => s.id === seat.id);
                                    const isAvailable = seat.status === "AVAILABLE";
                                    const isReserved = seat.status === "RESERVED";
                                    const isBooked = seat.status === "BOOKED";
                                    const isDisabled = seat.status === "DISABLED";

                                    let stateClass = "available";
                                    if (isSelected) stateClass = "selected";
                                    else if (isReserved) stateClass = "reserved";
                                    else if (isBooked) stateClass = "booked";
                                    else if (isDisabled) stateClass = "disabled";

                                    return (
                                      <button
                                        key={seat.id}
                                        type="button"
                                        onClick={() => toggleSeat(seat, row, section)}
                                        disabled={!isAvailable && !isSelected}
                                        aria-label={`Seat ${row.row_name}${seat.seat_number} - ₹${section.price} ${stateClass}`}
                                        aria-pressed={isSelected}
                                        title={`${section.name} - Row ${row.row_name}, Seat ${seat.seat_number} (₹${section.price})`}
                                        className={`seat ${stateClass}`}
                                      >
                                        {isDisabled ? "×" : seat.seat_number}
                                      </button>
                                    );
                                  })}
                                </div>

                                <span className="w-6 text-center font-bold text-xs sm:text-sm text-[#565e74]">
                                  {row.row_name}
                                </span>
                              </div>
                            ))}
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              </div>
            </section>
          </div>

          {/* Right Sidebar: Selection Summary */}
          <aside className="space-y-4">
            {/* Real Reservation Info Banner */}
            <div className="rounded-lg border border-[#add1cc] bg-[#dff0f2] py-3 px-4 text-xs sm:text-sm text-[#00685f] flex items-center gap-2">
              <span className="font-semibold">ⓘ</span>
              <span>Seats will be temporarily held for 10 minutes upon confirmation.</span>
            </div>

            {/* Selection Summary Card */}
            <section className="rounded-xl border border-[#bcc9c6] bg-white shadow-xs overflow-hidden">
              <h2 className="bg-[#f1f3ff] px-5 py-4 font-display text-xl sm:text-2xl font-bold text-ink border-b border-[#bcc9c6]/50">
                Your Selection
              </h2>

              <div className="p-5 sm:p-6">
                <div className="mb-5 border-b border-[#d8dfdd] pb-5">
                  <div className="flex justify-between text-xs text-[#565e74] mb-2">
                    <span className="font-semibold uppercase tracking-wider">
                      SEATS SELECTED
                    </span>
                    <span className="font-bold text-ink">
                      {selectedSeats.length} of {bookingLimit} max
                    </span>
                  </div>

                  {selectedSeats.length > 0 ? (
                    <div className="flex flex-wrap gap-1.5 mt-2">
                      {selectedSeats.map((seat) => (
                        <span
                          key={seat.id}
                          className="inline-flex items-center gap-1.5 rounded-md bg-[#e4f1f0] px-2.5 py-1 text-xs font-semibold text-[#00685f]"
                        >
                          <span>{seat.rowName}{seat.seatNumber}</span>
                          <span className="text-[10px] text-[#565e74]">({seat.sectionName})</span>
                          <button
                            type="button"
                            onClick={() =>
                              setSelectedSeats((prev) => prev.filter((s) => s.id !== seat.id))
                            }
                            className="ml-1 text-[#00685f] hover:text-red-600 font-bold"
                            aria-label={`Remove seat ${seat.rowName}${seat.seatNumber}`}
                          >
                            ×
                          </button>
                        </span>
                      ))}
                    </div>
                  ) : (
                    <p className="text-xs text-[#565e74] italic mt-1">
                      No seats selected yet. Click on available seats in the map.
                    </p>
                  )}
                </div>

                <OrderPriceSummary
                  subtotal={pricing.subtotal}
                  platformFee={pricing.platformFee}
                  taxes={pricing.taxes}
                  total={pricing.total}
                  className="mb-6"
                />

                {reservationError && (
                  <div className="mb-4 rounded-lg bg-red-50 border border-red-200 p-3 text-xs text-red-700 flex items-start gap-2">
                    <span className="font-bold">⚠</span>
                    <span>{reservationError}</span>
                  </div>
                )}

                <button
                  type="button"
                  onClick={handleContinueToCheckout}
                  disabled={selectedSeats.length === 0 || isReserving}
                  className="h-12 w-full rounded-lg bg-[#00796d] px-5 font-bold text-white shadow-xs transition hover:bg-[#00685f] disabled:cursor-not-allowed disabled:bg-[#9db5b1] focus:outline-none focus:ring-2 focus:ring-[#00796d]/40 flex items-center justify-center gap-2 cursor-pointer"
                >
                  {isReserving ? (
                    <>
                      <span className="size-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                      <span>Reserving Seats...</span>
                    </>
                  ) : (
                    "Continue to Checkout →"
                  )}
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
