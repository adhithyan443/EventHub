import { useState, useEffect } from "react";
import { useParams, useLocation, Link } from "react-router-dom";
import AppHeader from "../../../components/layout/AppHeader";
import Footer from "../../../components/layout/Footer";
import StepsProgress from "../components/StepsProgress";
import OrderPriceSummary from "../components/OrderPriceSummary";
import { getPublicEventById } from "../../../api/eventApi";
import { createCheckout } from "../../../api/checkoutApi";
import { createPayment, verifyPayment } from "../../../api/paymentApi";
import { loadRazorpay } from "../../../utils/loadRazorpay";
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

const formatVenueLocation = (ev) => {
  if (!ev) return "Location TBA";
  if (ev.event_type === "ONLINE") return "Online Event";
  if (ev.venue) {
    const parts = [ev.venue.city, ev.venue.state].filter(Boolean);
    if (parts.length > 0) return parts.join(", ");
    return ev.venue.name || "Venue TBA";
  }
  return "Venue TBA";
};

export default function CheckoutPage() {
  const { eventId } = useParams();
  const location = useLocation();

  const user = useAuthStore((state) => state.user);

  // Read state passed from CustomerEventDetailsPage, or fallback to client-side cache
  const cacheKey = eventId ? `eventhub_reservation_${eventId}` : null;
  const getInitialState = () => {
    if (location.state?.reservation) {
      return location.state;
    }
    if (cacheKey) {
      try {
        const cachedRaw = sessionStorage.getItem(cacheKey);
        if (cachedRaw) {
          const parsed = JSON.parse(cachedRaw);
          return parsed;
        }
      } catch {
        // ignore parse error
      }
    }
    return location.state || {};
  };

  const navState = getInitialState();
  const reservation = navState.reservation;
  const selectedTicket = navState.selectedTicket;
  const ticketPrice = navState.ticketPrice;
  const quantity = navState.quantity || 1;
  const selectedSeats = navState.selectedSeats;
  const selectedSeatObjects = navState.selectedSeatObjects;

  // Event details state
  const [event, setEvent] = useState(null);
  const [eventLoading, setEventLoading] = useState(true);

  // Reservation countdown & expiration state
  const [timeLeft, setTimeLeft] = useState(0);
  const [isExpired, setIsExpired] = useState(false);

  // Current checkout step: 2 = Checkout, 3 = Payment, 4 = Confirmation
  const [currentStep, setCurrentStep] = useState(2);
  const [agreedTerms, setAgreedTerms] = useState(false);
  const [agreedPaymentTerms, setAgreedPaymentTerms] = useState(true);

  // Checkout snapshot and payment states
  const [checkout, setCheckout] = useState(null);
  const [isCheckingOut, setIsCheckingOut] = useState(false);
  const [isPaying, setIsPaying] = useState(false);
  const [paymentError, setPaymentError] = useState(null);
  const [confirmedBooking, setConfirmedBooking] = useState(null);
  const [confirmedPayment, setConfirmedPayment] = useState(null);

  // Customer info form fields initialized from authenticated user
  const [customerInfo, setCustomerInfo] = useState({
    fullName: "",
    email: "",
    phoneNumber: "",
  });

  const effectiveFullName = customerInfo.fullName !== "" ? customerInfo.fullName : (user?.fullName || "");
  const effectiveEmail = customerInfo.email !== "" ? customerInfo.email : (user?.email || "");
  const effectivePhone = customerInfo.phoneNumber !== "" ? customerInfo.phoneNumber : (user?.phone || "");

  // Payment method UI selection state
  const [paymentMethod, setPaymentMethod] = useState("UPI");
  const [upiId, setUpiId] = useState("");
  const [cardNumber, setCardNumber] = useState("");
  const [cardExpiry, setCardExpiry] = useState("");
  const [cardCvv, setCardCvv] = useState("");
  const [selectedBank, setSelectedBank] = useState("HDFC Bank");
  const [selectedWallet, setSelectedWallet] = useState("Paytm");

  // Load live event details
  useEffect(() => {
    let cancelled = false;
    const fetchEvent = async () => {
      if (!eventId) {
        setEventLoading(false);
        return;
      }
      setEventLoading(true);
      try {
        const response = await getPublicEventById(eventId);
        if (cancelled) return;
        setEvent(response?.data);
      } catch {
        // Fallback gracefully if event details cannot be loaded
      } finally {
        if (!cancelled) setEventLoading(false);
      }
    };
    fetchEvent();
    return () => {
      cancelled = true;
    };
  }, [eventId]);

  // Server-driven countdown timer based on reservation.ExpiresAt
  useEffect(() => {
    const expiresAtRaw = reservation?.ExpiresAt || reservation?.expires_at;
    if (!expiresAtRaw) {
      return;
    }

    const expiryTime = new Date(expiresAtRaw).getTime();

    const updateTimer = () => {
      const remaining = Math.max(0, Math.floor((expiryTime - Date.now()) / 1000));
      setTimeLeft(remaining);
      if (remaining <= 0) {
        setIsExpired(true);
        if (cacheKey) {
          sessionStorage.removeItem(cacheKey);
        }
      }
    };

    updateTimer();
    const interval = setInterval(updateTimer, 1000);
    return () => clearInterval(interval);
  }, [reservation, cacheKey]);

  const formatTimer = (seconds) => {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${String(mins).padStart(2, "0")}:${String(secs).padStart(2, "0")}`;
  };

  const handleProceedToPayment = async () => {
    if (!agreedTerms || isExpired || isCheckingOut) return;
    const reservationId = reservation?.ID || reservation?.id;
    if (!reservationId) {
      setPaymentError("No reservation identifier found. Please select tickets again.");
      return;
    }

    setIsCheckingOut(true);
    setPaymentError(null);

    try {
      const response = await createCheckout(reservationId);
      const checkoutData = response?.data?.checkout;
      if (!checkoutData) {
        throw new Error("Invalid checkout response from server");
      }
      setCheckout(checkoutData);
      setCurrentStep(3);
      window.scrollTo({ top: 0, behavior: "smooth" });
    } catch (err) {
      const status = err?.response?.status;
      if (status === 409 || status === 400) {
        setIsExpired(true);
        setPaymentError(
          "Your ticket reservation has expired or is no longer valid. Please select tickets again."
        );
      } else {
        setPaymentError(
          err?.response?.data?.message ||
          "Failed to initialize checkout. Please try again."
        );
      }
    } finally {
      setIsCheckingOut(false);
    }
  };

  const handleInitiatePayment = async () => {
    if (isExpired || isPaying) return;

    const checkoutId = checkout?.ID || checkout?.id;
    if (!checkoutId) {
      setPaymentError("Checkout snapshot not found. Please return to Step 2.");
      return;
    }

    setIsPaying(true);
    setPaymentError(null);

    const sdkLoaded = await loadRazorpay();
    if (!sdkLoaded || !window.Razorpay) {
      setPaymentError(
        "Could not load the secure Razorpay Checkout SDK. Please check your network connection and try again."
      );
      setIsPaying(false);
      return;
    }

    // Map UI payment method to backend domain enum
    const methodMap = {
      "UPI": "UPI",
      "Card": "CARD",
      "Net Banking": "NET_BANKING",
      "Wallet": "WALLET",
    };
    const domainMethod = methodMap[paymentMethod] || "UPI";

    try {
      const response = await createPayment({
        checkoutId,
        method: domainMethod,
      });

      const payment = response?.data?.payment;
      const keyId = response?.data?.keyId || import.meta.env.VITE_RAZORPAY_KEY_ID;

      if (!payment || (!payment.ProviderOrderID && !payment.providerOrderId)) {
        throw new Error("Payment order could not be generated by the server.");
      }

      if (!keyId) {
        throw new Error("Razorpay Key ID is missing. Please contact platform support.");
      }

      const orderId = payment.ProviderOrderID || payment.providerOrderId;
      const amountPaise = Math.round(Number(payment.Amount || payment.amount || totalNum) * 100);

      // Map UI payment method to Razorpay checkout method code
      const rzpMethodMap = {
        "UPI": "upi",
        "Card": "card",
        "Net Banking": "netbanking",
        "Wallet": "wallet",
      };
      const rzpMethod = rzpMethodMap[paymentMethod] || "upi";

      // Configure dynamic Razorpay Checkout display blocks and sequencing
      const allRzpMethods = [
        { code: "upi", label: "UPI" },
        { code: "card", label: "Card" },
        { code: "netbanking", label: "Net Banking" },
        { code: "wallet", label: "Wallet" },
      ];

      const otherInstruments = allRzpMethods
        .filter((m) => m.code !== rzpMethod)
        .map((m) => ({ method: m.code }));

      const displayConfig = {
        blocks: {
          preferred: {
            name: `Pay via ${paymentMethod || "UPI"}`,
            instruments: [{ method: rzpMethod }],
          },
          other: {
            name: "Other Payment Methods",
            instruments: otherInstruments,
          },
        },
        sequence: ["block.preferred", "block.other"],
        preferences: {
          show_default_blocks: false,
        },
      };

      const prefill = {
        name: effectiveFullName,
        email: effectiveEmail,
        contact: effectivePhone,
        method: rzpMethod,
        ...(paymentMethod === "UPI" && upiId?.trim() ? { vpa: upiId.trim() } : {}),
      };

      const options = {
        key: keyId,
        amount: amountPaise,
        currency: "INR",
        name: "EventHub",
        description: `Tickets for ${eventTitle}`,
        image: eventThumbnail || undefined,
        order_id: orderId,
        prefill,
        config: {
          display: displayConfig,
        },
        display: displayConfig,
        notes: {
          booking_id: payment.BookingID || payment.bookingId,
          payment_id: payment.ID || payment.id,
        },
        theme: {
          color: "#00796d",
        },
        modal: {
          ondismiss: function () {
            setIsPaying(false);
          },
        },
        handler: async function (razorpayResponse) {
          try {
            const verifyRes = await verifyPayment({
              providerOrderId: orderId,
              razorpayPaymentId: razorpayResponse.razorpay_payment_id,
              razorpaySignature: razorpayResponse.razorpay_signature,
            });

            if (cacheKey) {
              sessionStorage.removeItem(cacheKey);
            }

            setConfirmedBooking(verifyRes?.data?.booking || null);
            setConfirmedPayment(verifyRes?.data?.payment || null);
            setCurrentStep(4);
            window.scrollTo({ top: 0, behavior: "smooth" });
          } catch (err) {
            setPaymentError(
              err?.response?.data?.message ||
              "Payment verification failed. If your account was debited, your booking will be confirmed via webhook shortly."
            );
          } finally {
            setIsPaying(false);
          }
        },
      };

      const rzpInstance = new window.Razorpay(options);

      rzpInstance.on("payment.failed", function (failResponse) {
        const errorDesc =
          failResponse?.error?.description ||
          "Payment authorization failed at the gateway. Please try again or choose another method.";
        setPaymentError(errorDesc);
        setIsPaying(false);
      });

      rzpInstance.open();
    } catch (err) {
      const errorMsg =
        err?.response?.data?.message ||
        err?.message ||
        "Failed to initiate payment. Please try again.";
      setPaymentError(errorMsg);
      setIsPaying(false);
    }
  };

  // Pricing calculations
  const subtotalNum = checkout
    ? Number(checkout.Subtotal || checkout.subtotal || 0)
    : selectedSeatObjects && selectedSeatObjects.length > 0
    ? selectedSeatObjects.reduce((acc, s) => acc + (Number(s.price) || 0), 0)
    : (ticketPrice != null
        ? Number(ticketPrice)
        : event?.starting_price
        ? Number(event.starting_price)
        : 0) * quantity;

  const platformFeeNum = checkout
    ? Number(checkout.PlatformFee || checkout.platformFee || 0)
    : 0;

  const taxesNum = checkout
    ? Number(checkout.TaxAmount || checkout.taxAmount || 0)
    : 0;

  const totalNum = checkout
    ? Number(checkout.TotalAmount || checkout.totalAmount || 0)
    : subtotalNum + platformFeeNum + taxesNum;

  const pricing = {
    subtotal: `₹${subtotalNum.toLocaleString("en-IN")}`,
    platformFee: `₹${platformFeeNum.toLocaleString("en-IN")}`,
    taxes: `₹${taxesNum.toLocaleString("en-IN")}`,
    total: `₹${totalNum.toLocaleString("en-IN")}`,
    rawTotal: totalNum,
  };

  if (eventLoading) {
    return (
      <div className="flex min-h-screen flex-col bg-[#f9f9ff] text-[#141b2b]">
        <AppHeader />
        <main className="mx-auto w-full max-w-[1280px] flex-1 px-4 sm:px-6 lg:px-8 py-16 flex items-center justify-center">
          <div className="flex flex-col items-center gap-4 bg-white border border-[#bcc9c6] rounded-xl p-10 shadow-xs max-w-md w-full text-center">
            <div className="size-10 border-3 border-[#00685f]/30 border-t-[#00685f] rounded-full animate-spin" />
            <h2 className="font-display font-semibold text-lg text-ink">Loading checkout...</h2>
            <p className="text-xs text-[#565e74]">Fetching event schedule and reservation details.</p>
          </div>
        </main>
        <Footer />
      </div>
    );
  }

  // If no reservation exists, direct user back to event page to select tickets
  if (!reservation) {
    return (
      <div className="flex min-h-screen flex-col bg-[#f9f9ff] text-[#141b2b]">
        <AppHeader />
        <main className="mx-auto w-full max-w-[1280px] flex-1 px-4 sm:px-6 lg:px-8 py-16 flex items-center justify-center">
          <div className="flex flex-col items-center gap-4 bg-white border border-[#bcc9c6] rounded-xl p-8 sm:p-12 shadow-xs max-w-lg w-full text-center">
            <div className="grid h-16 w-16 place-items-center rounded-full bg-[#f1f3ff] text-[#00685f] text-2xl mb-2">
              🎫
            </div>
            <h2 className="font-display font-bold text-xl sm:text-2xl text-ink">
              No Active Reservation Found
            </h2>
            <p className="text-sm text-[#565e74] leading-relaxed">
              To proceed to checkout, please select your ticket tier and quantity on the event page first.
            </p>
            <div className="mt-4">
              <Link
                to={eventId ? `/events/${eventId}` : "/events"}
                className="inline-flex rounded-lg bg-[#00796d] px-5 py-2.5 text-sm font-bold text-white shadow-xs hover:bg-[#00685f] transition-colors"
              >
                Select Tickets
              </Link>
            </div>
          </div>
        </main>
        <Footer />
      </div>
    );
  }

  const eventTitle = event?.title || "Event Tickets";
  const eventDateStr = formatDate(event?.schedule?.event_date);
  const eventLocationStr = formatVenueLocation(event);
  const eventThumbnail = event?.banner_image_url || "/assets/59c6d.png";

  return (
    <div className="flex min-h-screen flex-col bg-[#f9f9ff] text-[#141b2b]">
      <AppHeader />

      <main className="mx-auto w-full max-w-[1280px] flex-1 px-4 sm:px-6 lg:px-8 py-6 sm:py-10">
        {/* Page Title & Breadcrumb / Steps */}
        <div className="mb-6">
          <div className="flex items-center justify-between flex-wrap gap-4 mb-4">
            <h1 className="font-display text-2xl sm:text-3xl lg:text-[32px] font-bold text-ink">
              {currentStep === 4 ? "Confirmation" : currentStep === 3 ? "Payment" : "Checkout"}
            </h1>
            {currentStep === 3 && (
              <button
                type="button"
                onClick={() => {
                  setPaymentError(null);
                  setCurrentStep(2);
                }}
                className="text-xs sm:text-sm font-semibold text-[#00685f] hover:underline cursor-pointer"
              >
                ← Back to Customer Details
              </button>
            )}
          </div>

          <StepsProgress current={currentStep} />
        </div>

        {/* STEP 2: CHECKOUT SCREEN */}
        {currentStep === 2 && (
          <div>
            {/* Reservation Expire Banner */}
            {isExpired ? (
              <div className="rounded-lg bg-[#ffd8d4] p-4 text-xs sm:text-sm font-medium text-[#a70a18] flex items-center justify-between gap-4 mb-6 border border-[#fca5a5]">
                <div className="flex items-center gap-2">
                  <span>⏱</span>
                  <span>Your reservation has expired. Please select tickets again.</span>
                </div>
                <Link
                  to={`/events/${eventId}`}
                  className="rounded-md bg-[#a70a18] text-white px-3 py-1.5 text-xs font-bold hover:bg-[#850813] transition-colors whitespace-nowrap"
                >
                  Select Tickets
                </Link>
              </div>
            ) : (
              <div className="rounded-lg bg-[#ffd8d4] p-3 sm:p-4 text-xs sm:text-sm font-medium text-[#a70a18] flex items-center gap-2 mb-6">
                <span>◷</span>
                <span>
                  Your tickets are reserved for{" "}
                  <span className="font-mono font-bold">{formatTimer(timeLeft)}</span>
                </span>
              </div>
            )}

            <div className="grid gap-6 lg:grid-cols-[minmax(0,2fr)_minmax(340px,1fr)] items-start">
              {/* Left Column: Event details, Selected Tickets, Customer Info */}
              <div className="space-y-6">
                {/* Event Mini Card */}
                <section className="rounded-xl border border-[#bcc9c6] bg-white p-4 sm:p-5 shadow-xs flex items-center gap-4 sm:gap-5">
                  <img
                    src={eventThumbnail}
                    alt={eventTitle}
                    className="h-20 w-20 sm:h-24 sm:w-24 rounded-lg object-cover border border-slate-200 shrink-0"
                    onError={(e) => {
                      e.currentTarget.onerror = null;
                      e.currentTarget.src = "/assets/59c6d.png";
                    }}
                  />
                  <div className="min-w-0">
                    <h2 className="font-display text-xl sm:text-2xl font-bold text-ink leading-snug">
                      {eventTitle}
                    </h2>
                    <p className="mt-1 text-xs sm:text-sm text-[#565e74] flex items-center gap-1.5">
                      <span className="text-[#00796d]">▣</span> {eventDateStr}
                    </p>
                    <p className="text-xs sm:text-sm text-[#565e74] flex items-center gap-1.5 mt-0.5">
                      <span className="text-[#00796d]">⌖</span> {eventLocationStr}
                    </p>
                  </div>
                </section>

                {/* Selected Tickets Card */}
                <section className="rounded-xl border border-[#bcc9c6] bg-white p-5 sm:p-6 shadow-xs">
                  <div className="mb-4 flex items-center justify-between border-b border-[#bcc9c6] pb-4">
                    <h2 className="font-display text-lg sm:text-xl font-bold text-ink">
                      {selectedSeats && selectedSeats.length > 0 ? "Selected Seats" : "Selected Tickets"}
                    </h2>
                    <Link
                      to={selectedSeats && selectedSeats.length > 0 ? `/events/${eventId}/seats` : `/events/${eventId}`}
                      className="text-xs sm:text-sm font-semibold text-[#00685f] hover:underline"
                    >
                      {selectedSeats && selectedSeats.length > 0 ? "Change Seats" : "Change Tickets"}
                    </Link>
                  </div>

                  {selectedSeatObjects && selectedSeatObjects.length > 0 ? (
                    <div className="space-y-2.5">
                      {selectedSeatObjects.map((seat) => (
                        <div
                          key={seat.id || `${seat.rowName}${seat.seatNumber}`}
                          className="flex items-center justify-between rounded-lg bg-[#f1f3ff] px-4 py-3 text-xs sm:text-sm"
                        >
                          <div className="flex items-center gap-2">
                            <span className="text-[#00796d]">▱</span>
                            <div>
                              <strong className="block font-medium text-ink">
                                Row {seat.rowName}, Seat {seat.seatNumber}
                              </strong>
                              {seat.sectionName && (
                                <span className="text-xs text-[#565e74]">
                                  {seat.sectionName}
                                </span>
                              )}
                            </div>
                          </div>
                          <strong className="font-display font-bold text-ink">
                            ₹{Number(seat.price || 0).toLocaleString("en-IN")}
                          </strong>
                        </div>
                      ))}
                    </div>
                  ) : selectedSeats && selectedSeats.length > 0 ? (
                    <div className="space-y-2.5">
                      {selectedSeats.map((seat) => (
                        <div
                          key={seat}
                          className="flex items-center justify-between rounded-lg bg-[#f1f3ff] px-4 py-3 text-xs sm:text-sm"
                        >
                          <span className="font-medium text-ink flex items-center gap-2">
                            <span className="text-[#00796d]">▱</span> Seat {seat}
                          </span>
                          <strong className="font-display font-bold text-ink">
                            ₹{Number(ticketPrice || 0).toLocaleString("en-IN")}
                          </strong>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <div className="flex items-center justify-between rounded-lg bg-[#f1f3ff] px-4 py-3 text-xs sm:text-sm">
                      <div className="flex items-center gap-2">
                        <span className="text-[#00796d]">▣</span>
                        <div>
                          <strong className="block font-medium text-ink">
                            {selectedTicket || "General Admission"}
                          </strong>
                          <span className="text-xs text-[#565e74]">
                            Quantity: {quantity} {quantity === 1 ? "ticket" : "tickets"} (₹{Number(ticketPrice || 0).toLocaleString("en-IN")} each)
                          </span>
                        </div>
                      </div>
                      <strong className="font-display font-bold text-ink">
                        ₹{(Number(ticketPrice || 0) * quantity).toLocaleString("en-IN")}
                      </strong>
                    </div>
                  )}
                </section>

                {/* Customer Information Card */}
                <section className="rounded-xl border border-[#bcc9c6] bg-white p-5 sm:p-6 shadow-xs">
                  <h2 className="mb-5 font-display text-lg sm:text-xl font-bold text-ink">
                    Customer Information
                  </h2>

                  <div className="space-y-4">
                    <div>
                      <label className="block text-xs sm:text-sm font-semibold text-ink mb-1.5">
                        Full Name
                      </label>
                      <input
                        type="text"
                        value={effectiveFullName}
                        onChange={(e) =>
                          setCustomerInfo({ ...customerInfo, fullName: e.target.value })
                        }
                        className="h-12 w-full rounded-lg border border-[#bcc9c6] bg-[#f9f9ff] px-4 text-sm text-[#141b2b] outline-none transition focus:border-[#00796d] focus:ring-2 focus:ring-[#00796d]/15"
                      />
                    </div>

                    <div className="grid gap-4 sm:grid-cols-2">
                      <div>
                        <label className="block text-xs sm:text-sm font-semibold text-ink mb-1.5">
                          Email
                        </label>
                        <input
                          type="email"
                          value={effectiveEmail}
                          onChange={(e) =>
                            setCustomerInfo({ ...customerInfo, email: e.target.value })
                          }
                          className="h-12 w-full rounded-lg border border-[#bcc9c6] bg-[#f9f9ff] px-4 text-sm text-[#141b2b] outline-none transition focus:border-[#00796d] focus:ring-2 focus:ring-[#00796d]/15"
                        />
                      </div>

                      <div>
                        <label className="block text-xs sm:text-sm font-semibold text-ink mb-1.5">
                          Phone Number
                        </label>
                        <input
                          type="tel"
                          value={effectivePhone}
                          onChange={(e) =>
                            setCustomerInfo({
                              ...customerInfo,
                              phoneNumber: e.target.value,
                            })
                          }
                          className="h-12 w-full rounded-lg border border-[#bcc9c6] bg-[#f9f9ff] px-4 text-sm text-[#141b2b] outline-none transition focus:border-[#00796d] focus:ring-2 focus:ring-[#00796d]/15"
                        />
                      </div>
                    </div>
                  </div>
                </section>

                {/* Cancellation Policy Banner */}
                {event?.cancellation && (
                  <div className="rounded-xl border border-[#bcc9c6] bg-[#f1f3ff] p-4 text-xs sm:text-sm">
                    <div className="flex items-center gap-2 font-bold text-ink">
                      <span>ⓘ</span>
                      <span>Cancellation Policy</span>
                    </div>
                    <p className="ml-5 mt-1 text-[#565e74]">
                      {event.cancellation.cancellation_allowed
                        ? `Refundable up to ${event.cancellation.refund_percentage}% if cancelled at least ${event.cancellation.cancellation_deadline_hours} hours prior to event.`
                        : "Tickets are non-refundable for this event."}
                    </p>
                  </div>
                )}
              </div>

              {/* Right Sidebar: Price Summary & Proceed CTA */}
              <aside>
                <section className="sticky top-20 rounded-xl border border-[#bcc9c6] bg-white p-5 sm:p-6 shadow-xs">
                  <h2 className="mb-4 border-b border-[#bcc9c6] pb-4 font-display text-xl font-bold text-ink">
                    Price Summary
                  </h2>

                  <OrderPriceSummary
                    subtotal={pricing.subtotal}
                    platformFee={pricing.platformFee}
                    taxes={pricing.taxes}
                    total={pricing.total}
                    className="mb-6"
                  />

                  {/* Terms & Conditions Checkbox */}
                  <label className="mb-6 flex items-start gap-3 cursor-pointer text-xs sm:text-sm text-[#565e74]">
                    <input
                      type="checkbox"
                      checked={agreedTerms}
                      onChange={(e) => setAgreedTerms(e.target.checked)}
                      className="mt-1 h-4 w-4 rounded border-slate-300 text-[#00796d] focus:ring-[#00796d]"
                    />
                    <span>
                      I agree to the{" "}
                      <span className="font-semibold text-[#00685f] hover:underline">
                        EventHub Terms &amp; Conditions
                      </span>{" "}
                      and acknowledge the event policies.
                    </span>
                  </label>

                  {paymentError && (
                    <div className="mb-4 rounded-lg bg-[#ffd8d4] p-3 text-xs sm:text-sm font-medium text-[#a70a18] border border-[#fca5a5]">
                      {paymentError}
                    </div>
                  )}

                  <button
                    type="button"
                    onClick={handleProceedToPayment}
                    disabled={!agreedTerms || isExpired || isCheckingOut}
                    className="h-12 w-full rounded-lg bg-[#00796d] px-5 font-bold text-white shadow-xs transition hover:bg-[#00685f] disabled:cursor-not-allowed disabled:bg-[#9db5b1] focus:outline-none focus:ring-2 focus:ring-[#00796d]/40 cursor-pointer flex items-center justify-center gap-2"
                  >
                    {isCheckingOut ? (
                      <>
                        <div className="size-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                        <span>Creating Checkout...</span>
                      </>
                    ) : isExpired ? (
                      "Reservation Expired"
                    ) : (
                      "Proceed to Payment"
                    )}
                  </button>

                  <p className="mt-4 text-center text-xs text-[#565e74] flex items-center justify-center gap-1.5">
                    <span>♙</span> Temporary hold active
                  </p>
                </section>
              </aside>
            </div>
          </div>
        )}

        {/* STEP 3: PAYMENT SCREEN */}
        {currentStep === 3 && (
          <div>
            {/* Payment Reservation Warning */}
            {isExpired ? (
              <div className="rounded-lg bg-[#ffd8d4] p-4 text-xs sm:text-sm font-medium text-[#a70a18] flex items-center justify-between gap-4 mb-6 border border-[#fca5a5]">
                <div className="flex items-center gap-2">
                  <span>⏱</span>
                  <span>Your reservation has expired. Please select tickets again.</span>
                </div>
                <Link
                  to={`/events/${eventId}`}
                  className="rounded-md bg-[#a70a18] text-white px-3 py-1.5 text-xs font-bold hover:bg-[#850813] transition-colors whitespace-nowrap"
                >
                  Select Tickets
                </Link>
              </div>
            ) : (
              <div className="rounded-lg bg-[#ffd8d4] p-3 sm:p-4 text-xs sm:text-sm font-medium text-[#a70a18] flex items-center gap-2 mb-6">
                <span>◷</span>
                <span>
                  Complete payment before your reservation expires:{" "}
                  <span className="font-mono font-bold">{formatTimer(timeLeft)}</span>
                </span>
              </div>
            )}

            <div className="grid gap-8 lg:grid-cols-[minmax(0,2fr)_minmax(340px,1fr)] items-start">
              {/* Left Column: Payment Methods & Inputs */}
              <div>
                <h2 className="mb-5 font-display text-xl sm:text-2xl font-bold text-ink">
                  Choose Payment Method
                </h2>

                {/* Payment Method Selector Tabs */}
                <div className="mb-6 grid grid-cols-2 gap-3 sm:grid-cols-4">
                  {[
                    { id: "UPI", icon: "⌗", label: "UPI" },
                    { id: "Card", icon: "▤", label: "Card" },
                    { id: "Net Banking", icon: "♜", label: "Net Banking" },
                    { id: "Wallet", icon: "▣", label: "Wallet" },
                  ].map((method) => {
                    const isSelected = paymentMethod === method.id;
                    return (
                      <button
                        key={method.id}
                        type="button"
                        onClick={() => setPaymentMethod(method.id)}
                        className={`flex flex-col items-center justify-center h-20 rounded-xl border bg-white p-3 text-xs sm:text-sm font-medium transition-all cursor-pointer ${
                          isSelected
                            ? "border-2 border-[#00796d] text-[#00685f] shadow-xs"
                            : "border-[#bcc9c6] text-[#565e74] hover:border-slate-400"
                        }`}
                      >
                        <span className="mb-1 text-lg sm:text-xl font-bold">
                          {method.icon}
                        </span>
                        <span>{method.label}</span>
                      </button>
                    );
                  })}
                </div>

                {/* Payment Detail Form Card */}
                <section className="rounded-xl border border-[#bcc9c6] bg-white p-5 sm:p-6 shadow-xs">
                  <h3 className="mb-2 font-display text-lg sm:text-xl font-bold text-ink">
                    Pay via {paymentMethod}
                  </h3>
                  <p className="mb-5 text-xs sm:text-sm text-[#565e74]">
                    {paymentMethod === "UPI" &&
                      "Enter your Virtual Payment Address (VPA / UPI ID)."}
                    {paymentMethod === "Card" &&
                      "Enter your credit or debit card details to continue."}
                    {paymentMethod === "Net Banking" &&
                      "Select your bank to proceed with secure net banking."}
                    {paymentMethod === "Wallet" &&
                      "Choose your preferred wallet to complete payment."}
                  </p>

                  {/* UPI Inputs */}
                  {paymentMethod === "UPI" && (
                    <div className="space-y-4">
                      <div>
                        <label className="block text-xs sm:text-sm font-semibold text-ink mb-1.5">
                          Enter UPI ID
                        </label>
                        <input
                          type="text"
                          value={upiId}
                          onChange={(e) => setUpiId(e.target.value)}
                          placeholder="example@upi"
                          className="h-12 w-full rounded-lg border border-[#bcc9c6] bg-[#f9f9ff] px-4 text-sm text-[#141b2b] outline-none transition focus:border-[#00796d] focus:ring-2 focus:ring-[#00796d]/15"
                        />
                      </div>
                      <p className="text-xs text-[#565e74]">
                        Supported apps: Google Pay, PhonePe, Paytm, BHIM, CRED
                      </p>
                    </div>
                  )}

                  {/* Card Inputs */}
                  {paymentMethod === "Card" && (
                    <div className="space-y-4">
                      <div>
                        <label className="block text-xs sm:text-sm font-semibold text-ink mb-1.5">
                          Card Number
                        </label>
                        <input
                          type="text"
                          value={cardNumber}
                          onChange={(e) => setCardNumber(e.target.value)}
                          placeholder="1234 5678 9012 3456"
                          className="h-12 w-full rounded-lg border border-[#bcc9c6] bg-[#f9f9ff] px-4 text-sm text-[#141b2b] outline-none transition focus:border-[#00796d] focus:ring-2 focus:ring-[#00796d]/15"
                        />
                      </div>

                      <div className="grid grid-cols-2 gap-4">
                        <div>
                          <label className="block text-xs sm:text-sm font-semibold text-ink mb-1.5">
                            Expiry (MM/YY)
                          </label>
                          <input
                            type="text"
                            value={cardExpiry}
                            onChange={(e) => setCardExpiry(e.target.value)}
                            placeholder="MM/YY"
                            className="h-12 w-full rounded-lg border border-[#bcc9c6] bg-[#f9f9ff] px-4 text-sm text-[#141b2b] outline-none transition focus:border-[#00796d] focus:ring-2 focus:ring-[#00796d]/15"
                          />
                        </div>
                        <div>
                          <label className="block text-xs sm:text-sm font-semibold text-ink mb-1.5">
                            CVV
                          </label>
                          <input
                            type="password"
                            value={cardCvv}
                            onChange={(e) => setCardCvv(e.target.value)}
                            placeholder="123"
                            maxLength={4}
                            className="h-12 w-full rounded-lg border border-[#bcc9c6] bg-[#f9f9ff] px-4 text-sm text-[#141b2b] outline-none transition focus:border-[#00796d] focus:ring-2 focus:ring-[#00796d]/15"
                          />
                        </div>
                      </div>
                    </div>
                  )}

                  {/* Net Banking Inputs */}
                  {paymentMethod === "Net Banking" && (
                    <div className="space-y-3">
                      <label className="block text-xs sm:text-sm font-semibold text-ink mb-1.5">
                        Select Bank
                      </label>
                      <div className="grid grid-cols-2 gap-2 sm:grid-cols-3">
                        {["HDFC Bank", "SBI", "ICICI Bank", "Axis Bank", "Kotak", "Other Banks"].map(
                          (bank) => (
                            <button
                              key={bank}
                              type="button"
                              onClick={() => setSelectedBank(bank)}
                              className={`rounded-lg border p-3 text-xs font-semibold transition cursor-pointer ${
                                selectedBank === bank
                                  ? "border-[#00796d] bg-[#f8fffe] text-[#00685f]"
                                  : "border-[#bcc9c6] bg-[#f9f9ff] text-[#565e74]"
                              }`}
                            >
                              {bank}
                            </button>
                          )
                        )}
                      </div>
                    </div>
                  )}

                  {/* Wallet Inputs */}
                  {paymentMethod === "Wallet" && (
                    <div className="space-y-3">
                      <label className="block text-xs sm:text-sm font-semibold text-ink mb-1.5">
                        Select Wallet
                      </label>
                      <div className="grid grid-cols-2 gap-2 sm:grid-cols-3">
                        {["Paytm", "PhonePe", "Amazon Pay", "Mobikwik"].map((wallet) => (
                          <button
                            key={wallet}
                            type="button"
                            onClick={() => setSelectedWallet(wallet)}
                            className={`rounded-lg border p-3 text-xs font-semibold transition cursor-pointer ${
                              selectedWallet === wallet
                                  ? "border-[#00796d] bg-[#f8fffe] text-[#00685f]"
                                  : "border-[#bcc9c6] bg-[#f9f9ff] text-[#565e74]"
                            }`}
                          >
                            {wallet}
                          </button>
                        ))}
                      </div>
                    </div>
                  )}

                  {paymentError && (
                    <div className="mt-4 rounded-lg bg-[#ffd8d4] p-3 text-xs sm:text-sm font-medium text-[#a70a18] border border-[#fca5a5]">
                      {paymentError}
                    </div>
                  )}

                  <div className="mt-6 space-y-2">
                    <button
                      type="button"
                      onClick={handleInitiatePayment}
                      disabled={isExpired || isPaying || !agreedPaymentTerms}
                      className="h-12 w-full rounded-lg bg-[#00796d] px-5 font-bold text-white shadow-xs transition hover:bg-[#00685f] disabled:cursor-not-allowed disabled:bg-[#9db5b1] focus:outline-none focus:ring-2 focus:ring-[#00796d]/40 cursor-pointer flex items-center justify-center gap-2"
                    >
                      {isPaying ? (
                        <>
                          <div className="size-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                          <span>Processing Payment...</span>
                        </>
                      ) : (
                        <span>Pay {pricing.total}</span>
                      )}
                    </button>
                    <p className="text-center text-xs text-[#565e74]">
                      Secured by Razorpay. 256-bit SSL encrypted.
                    </p>
                  </div>
                </section>

                <label className="mt-4 flex items-center gap-2.5 cursor-pointer text-xs text-[#565e74]">
                  <input
                    type="checkbox"
                    checked={agreedPaymentTerms}
                    onChange={(e) => setAgreedPaymentTerms(e.target.checked)}
                    className="h-4 w-4 rounded border-slate-300 text-[#00796d] focus:ring-[#00796d]"
                  />
                  <span>
                    I agree to the EventHub booking terms and payment policy.
                  </span>
                </label>
              </div>

              {/* Right Sidebar: Order Summary Recap */}
              <aside>
                <section className="sticky top-20 rounded-xl border border-[#bcc9c6] bg-white shadow-xs overflow-hidden">
                  <div className="bg-[#f1f3ff] p-5 sm:p-6 border-b border-[#bcc9c6]/50">
                    <h2 className="font-display text-xl sm:text-2xl font-bold text-ink">
                      {eventTitle}
                    </h2>
                    <p className="mt-1 text-xs sm:text-sm text-[#565e74] flex items-center gap-1.5">
                      <span className="text-[#00796d]">▣</span> {eventDateStr}
                    </p>
                    <div className="mt-4 flex items-center gap-2 flex-wrap">
                      <span className="rounded-full bg-[#00796d] px-3 py-1 text-xs font-semibold text-white">
                        {quantity}x {selectedTicket || "General Admission"}
                      </span>
                      {selectedSeats && selectedSeats.length > 0 && (
                        <span className="text-xs sm:text-sm text-[#565e74]">
                          Seats {selectedSeats.join(", ")}
                        </span>
                      )}
                    </div>
                  </div>

                  <div className="p-5 sm:p-6">
                    <h3 className="mb-4 border-b border-[#bcc9c6] pb-3 text-sm font-semibold uppercase tracking-wider text-[#565e74]">
                      Order Summary
                    </h3>
                    <OrderPriceSummary
                      subtotal={pricing.subtotal}
                      platformFee={pricing.platformFee}
                      taxes={pricing.taxes}
                      total={pricing.total}
                    />
                  </div>
                </section>
              </aside>
            </div>
          </div>
        )}

        {/* STEP 4: CONFIRMATION SCREEN */}
        {currentStep === 4 && (
          <div className="mx-auto max-w-2xl py-6 sm:py-10">
            <div className="rounded-2xl border border-[#bcc9c6] bg-white p-6 sm:p-10 shadow-xs text-center">
              <div className="mx-auto mb-5 grid h-20 w-20 place-items-center rounded-full bg-[#e6f4f2] text-[#00796d] text-4xl">
                ✓
              </div>

              <h2 className="font-display text-2xl sm:text-3xl font-bold text-ink">
                Booking Confirmed!
              </h2>
              <p className="mt-2 text-sm text-[#565e74]">
                Thank you for your purchase. Your payment was successfully verified and your tickets have been issued.
              </p>

              {/* Booking & Transaction IDs */}
              <div className="mt-6 grid grid-cols-1 sm:grid-cols-2 gap-3 bg-[#f1f3ff] p-4 rounded-xl text-left text-xs sm:text-sm">
                <div>
                  <span className="text-[#565e74] block">Booking Reference</span>
                  <strong className="font-mono font-bold text-ink text-sm sm:text-base">
                    {confirmedBooking?.BookingNumber || confirmedBooking?.bookingNumber || "CONFIRMED"}
                  </strong>
                </div>
                <div>
                  <span className="text-[#565e74] block">Payment ID</span>
                  <strong className="font-mono font-bold text-ink text-sm sm:text-base">
                    {confirmedPayment?.TransactionID || confirmedPayment?.transactionId || "SUCCESS"}
                  </strong>
                </div>
              </div>

              {/* Event & Ticket Details */}
              <div className="mt-6 border-t border-[#bcc9c6] pt-6 text-left">
                <h3 className="font-display font-bold text-lg text-ink mb-3">
                  Event Details
                </h3>
                <div className="space-y-1.5 text-xs sm:text-sm text-[#565e74]">
                  <p className="font-semibold text-ink text-base">{eventTitle}</p>
                  <p className="flex items-center gap-2">
                    <span className="text-[#00796d]">▣</span> {eventDateStr}
                  </p>
                  <p className="flex items-center gap-2">
                    <span className="text-[#00796d]">⌖</span> {eventLocationStr}
                  </p>
                </div>

                <h3 className="font-display font-bold text-lg text-ink mt-6 mb-3">
                  Ticket Summary
                </h3>
                {selectedSeatObjects && selectedSeatObjects.length > 0 ? (
                  <div className="space-y-2">
                    {selectedSeatObjects.map((seat) => (
                      <div
                        key={seat.id || `${seat.rowName}${seat.seatNumber}`}
                        className="flex items-center justify-between rounded-lg bg-[#f9f9ff] border border-[#bcc9c6]/50 px-3.5 py-2.5 text-xs sm:text-sm"
                      >
                        <span className="text-ink font-medium">
                          Row {seat.rowName}, Seat {seat.seatNumber} {seat.sectionName ? `(${seat.sectionName})` : ""}
                        </span>
                        <strong className="font-bold text-ink">
                          ₹{Number(seat.price || 0).toLocaleString("en-IN")}
                        </strong>
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="flex items-center justify-between rounded-lg bg-[#f9f9ff] border border-[#bcc9c6]/50 px-3.5 py-2.5 text-xs sm:text-sm">
                    <span className="text-ink font-medium">
                      {quantity}x {selectedTicket || "General Admission"}
                    </span>
                    <strong className="font-bold text-ink">
                      {pricing.total}
                    </strong>
                  </div>
                )}

                <div className="mt-6 flex items-center justify-between border-t border-[#bcc9c6] pt-4 text-base font-bold text-ink">
                  <span>Total Paid</span>
                  <span className="font-display text-xl text-[#00685f]">
                    {pricing.total}
                  </span>
                </div>
              </div>

              {/* Action Buttons */}
              <div className="mt-8 flex flex-col sm:flex-row items-center justify-center gap-3">
                <Link
                  to="/events"
                  className="w-full sm:w-auto inline-flex items-center justify-center rounded-lg bg-[#00796d] px-6 py-3 text-sm font-bold text-white shadow-xs hover:bg-[#00685f] transition-colors"
                >
                  Browse More Events
                </Link>
                <button
                  type="button"
                  onClick={() => window.print()}
                  className="w-full sm:w-auto inline-flex items-center justify-center rounded-lg border border-[#bcc9c6] bg-white px-6 py-3 text-sm font-bold text-ink hover:bg-slate-50 transition-colors cursor-pointer"
                >
                  Print Receipt
                </button>
              </div>
            </div>
          </div>
        )}
      </main>

      <Footer />
    </div>
  );
}
