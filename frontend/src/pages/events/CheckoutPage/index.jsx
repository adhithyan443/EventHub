import { useState, useEffect } from "react";
import { useParams, useNavigate, useLocation, Link } from "react-router-dom";
import AppHeader from "../../../components/layout/AppHeader";
import Footer from "../../../components/layout/Footer";
import StepsProgress from "../components/StepsProgress";
import OrderPriceSummary from "../components/OrderPriceSummary";
import { getCustomerEventById } from "../mockCustomerEvent";

export default function CheckoutPage() {
  const { eventId = "sunfield" } = useParams();
  const navigate = useNavigate();
  const location = useLocation();
  const event = getCustomerEventById(eventId);

  // Read state or fallback to defaults
  const initialSeats = location.state?.selectedSeats || ["A4", "A5"];
  const initialPricing = location.state?.pricing || {
    subtotal: "₹2,998",
    platformFee: "₹99",
    taxes: "₹180",
    total: "₹3,277",
  };

  // Determine current step: 2 = Checkout, 3 = Payment, 4 = Confirmation
  const [currentStep, setCurrentStep] = useState(2);
  const [agreedTerms, setAgreedTerms] = useState(false);
  const [agreedPaymentTerms, setAgreedPaymentTerms] = useState(true);

  // Customer info form fields
  const [customerInfo, setCustomerInfo] = useState({
    fullName: "John Doe",
    email: "john@example.com",
    phoneNumber: "+91 98765 43210",
  });

  // Payment method
  const [paymentMethod, setPaymentMethod] = useState("UPI");
  const [upiId, setUpiId] = useState("johndoe@okaxis");
  const [cardNumber, setCardNumber] = useState("4532 •••• •••• 8892");
  const [cardExpiry, setCardExpiry] = useState("12/28");
  const [cardCvv, setCardCvv] = useState("•••");
  const [selectedBank, setSelectedBank] = useState("HDFC Bank");
  const [selectedWallet, setSelectedWallet] = useState("Paytm");

  // Countdown timer simulation (09:42)
  const [timeLeft, setTimeLeft] = useState(582);
  const [paymentSuccessModal, setPaymentSuccessModal] = useState(false);

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

  const handleProceedToPayment = () => {
    if (!agreedTerms) return;
    setCurrentStep(3);
    window.scrollTo({ top: 0, behavior: "smooth" });
  };

  const handlePay = () => {
    setPaymentSuccessModal(true);
  };

  return (
    <div className="flex min-h-screen flex-col bg-[#f9f9ff] text-[#141b2b]">
      <AppHeader />

      <main className="mx-auto w-full max-w-[1280px] flex-1 px-4 sm:px-6 lg:px-8 py-6 sm:py-10">
        {/* Page Title & Breadcrumb / Steps */}
        <div className="mb-6">
          <div className="flex items-center justify-between flex-wrap gap-4 mb-4">
            <h1 className="font-display text-2xl sm:text-3xl lg:text-[32px] font-bold text-ink">
              {currentStep === 3 ? "Payment" : "Checkout"}
            </h1>
            {currentStep === 3 && (
              <button
                type="button"
                onClick={() => setCurrentStep(2)}
                className="text-xs sm:text-sm font-semibold text-[#00685f] hover:underline"
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
            <div className="rounded-lg bg-[#ffd8d4] p-3 sm:p-4 text-xs sm:text-sm font-medium text-[#a70a18] flex items-center gap-2 mb-6">
              <span>◷</span>
              <span>
                Your seats are reserved for{" "}
                <span className="font-mono font-bold">{formatTimer(timeLeft)}</span>
              </span>
            </div>

            <div className="grid gap-6 lg:grid-cols-[minmax(0,2fr)_minmax(340px,1fr)] items-start">
              {/* Left Column: Event details, Selected Seats, Customer Info */}
              <div className="space-y-6">
                {/* Event Mini Card */}
                <section className="rounded-xl border border-[#bcc9c6] bg-white p-4 sm:p-5 shadow-xs flex items-center gap-4 sm:gap-5">
                  <img
                    src={event.thumbnail}
                    alt={event.title}
                    className="h-20 w-20 sm:h-24 sm:w-24 rounded-lg object-cover border border-slate-200 shrink-0"
                  />
                  <div className="min-w-0">
                    <h2 className="font-display text-xl sm:text-2xl font-bold text-ink leading-snug">
                      {event.title}
                    </h2>
                    <p className="mt-1 text-xs sm:text-sm text-[#565e74] flex items-center gap-1.5">
                      <span className="text-[#00796d]">▣</span> {event.date}
                    </p>
                    <p className="text-xs sm:text-sm text-[#565e74] flex items-center gap-1.5 mt-0.5">
                      <span className="text-[#00796d]">⌖</span> {event.location}
                    </p>
                  </div>
                </section>

                {/* Selected Seats Card */}
                <section className="rounded-xl border border-[#bcc9c6] bg-white p-5 sm:p-6 shadow-xs">
                  <div className="mb-4 flex items-center justify-between border-b border-[#bcc9c6] pb-4">
                    <h2 className="font-display text-lg sm:text-xl font-bold text-ink">
                      Selected Seats
                    </h2>
                    <Link
                      to={`/events/${eventId}/seats`}
                      state={{ selectedSeats: initialSeats }}
                      className="text-xs sm:text-sm font-semibold text-[#00685f] hover:underline"
                    >
                      Change Seats
                    </Link>
                  </div>

                  <div className="space-y-2.5">
                    {initialSeats.map((seat) => (
                      <div
                        key={seat}
                        className="flex items-center justify-between rounded-lg bg-[#f1f3ff] px-4 py-3 text-xs sm:text-sm"
                      >
                        <span className="font-medium text-ink flex items-center gap-2">
                          <span className="text-[#00796d]">▱</span> Seat {seat} —{" "}
                          {seat.startsWith("A")
                            ? "VIP"
                            : seat.startsWith("B")
                            ? "Premium"
                            : "General"}
                        </span>
                        <strong className="font-display font-bold text-ink">
                          {seat.startsWith("A")
                            ? "₹1,499"
                            : seat.startsWith("B")
                            ? "₹999"
                            : "₹499"}
                        </strong>
                      </div>
                    ))}
                  </div>
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
                        value={customerInfo.fullName}
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
                          value={customerInfo.email}
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
                          value={customerInfo.phoneNumber}
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
                <div className="rounded-xl border border-[#bcc9c6] bg-[#f1f3ff] p-4 text-xs sm:text-sm">
                  <div className="flex items-center gap-2 font-bold text-ink">
                    <span>ⓘ</span>
                    <span>Cancellation Policy</span>
                  </div>
                  <p className="ml-5 mt-1 text-[#565e74]">
                    Tickets are non-refundable within 48 hours of the event.
                  </p>
                  <button
                    type="button"
                    className="ml-5 mt-2 font-semibold text-[#00685f] hover:underline"
                  >
                    View Policy
                  </button>
                </div>
              </div>

              {/* Right Sidebar: Price Summary & Proceed CTA */}
              <aside>
                <section className="sticky top-20 rounded-xl border border-[#bcc9c6] bg-white p-5 sm:p-6 shadow-xs">
                  <h2 className="mb-4 border-b border-[#bcc9c6] pb-4 font-display text-xl font-bold text-ink">
                    Price Summary
                  </h2>

                  <OrderPriceSummary
                    subtotal={initialPricing.subtotal}
                    platformFee={initialPricing.platformFee}
                    taxes={initialPricing.taxes}
                    total={initialPricing.total}
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
                      and acknowledge the cancellation policy.
                    </span>
                  </label>

                  <button
                    type="button"
                    onClick={handleProceedToPayment}
                    disabled={!agreedTerms}
                    className="h-12 w-full rounded-lg bg-[#00796d] px-5 font-bold text-white shadow-xs transition hover:bg-[#00685f] disabled:cursor-not-allowed disabled:bg-[#9db5b1] focus:outline-none focus:ring-2 focus:ring-[#00796d]/40"
                  >
                    Proceed to Payment
                  </button>

                  <p className="mt-4 text-center text-xs text-[#565e74] flex items-center justify-center gap-1.5">
                    <span>♙</span> Secure payment
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
            <div className="rounded-lg bg-[#ffd8d4] p-3 sm:p-4 text-xs sm:text-sm font-medium text-[#a70a18] flex items-center gap-2 mb-6">
              <span>◷</span>
              <span>
                Complete payment before your reservation expires:{" "}
                <span className="font-mono font-bold">{formatTimer(timeLeft)}</span>
              </span>
            </div>

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
                        className={`flex flex-col items-center justify-center h-20 rounded-xl border bg-white p-3 text-xs sm:text-sm font-medium transition-all ${
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
                              className={`rounded-lg border p-3 text-xs font-semibold transition ${
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
                            className={`rounded-lg border p-3 text-xs font-semibold transition ${
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

                  <div className="mt-6">
                    <button
                      type="button"
                      onClick={handlePay}
                      className="h-12 w-full rounded-lg bg-[#00796d] px-5 font-bold text-white shadow-xs transition hover:bg-[#00685f] focus:outline-none focus:ring-2 focus:ring-[#00796d]/40"
                    >
                      Pay {initialPricing.total}
                    </button>
                  </div>
                </section>

                {/* Secure Payment Footer Note */}
                <div className="mt-6 rounded-lg bg-[#e8ecff] p-3 text-xs sm:text-sm text-[#141b2b] flex items-center gap-2">
                  <span className="text-[#4648d4] font-bold">▣</span>
                  <span>Secure Payment: Your payment is securely processed.</span>
                </div>

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
                      {event.title}
                    </h2>
                    <p className="mt-1 text-xs sm:text-sm text-[#565e74] flex items-center gap-1.5">
                      <span className="text-[#00796d]">▣</span> {event.date}
                    </p>
                    <div className="mt-4 flex items-center gap-2 flex-wrap">
                      <span className="rounded-full bg-[#00796d] px-3 py-1 text-xs font-semibold text-white">
                        {initialSeats.length}x VIP
                      </span>
                      <span className="text-xs sm:text-sm text-[#565e74]">
                        Seats {initialSeats.join(" & ")}
                      </span>
                    </div>
                  </div>

                  <div className="p-5 sm:p-6">
                    <h3 className="mb-4 border-b border-[#bcc9c6] pb-3 text-sm font-semibold uppercase tracking-wider text-[#565e74]">
                      Order Summary
                    </h3>
                    <OrderPriceSummary
                      subtotal={initialPricing.subtotal}
                      platformFee={initialPricing.platformFee}
                      taxes={initialPricing.taxes}
                      total={initialPricing.total}
                    />
                  </div>
                </section>
              </aside>
            </div>
          </div>
        )}

        {/* Payment Success Confirmation Modal (Mock UI) */}
        {paymentSuccessModal && (
          <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4 backdrop-blur-xs">
            <div className="w-full max-w-md rounded-2xl bg-white p-6 sm:p-8 text-center shadow-xl">
              <div className="mx-auto mb-4 grid h-14 w-14 place-items-center rounded-full bg-[#e4f1f0] text-2xl text-[#00796d]">
                ✓
              </div>
              <h3 className="font-display text-2xl font-bold text-ink">
                Booking Confirmed!
              </h3>
              <p className="mt-2 text-sm text-[#565e74]">
                Your tickets for <strong className="text-ink">{event.title}</strong> have been reserved successfully.
              </p>
              <div className="mt-5 rounded-lg bg-[#f1f3ff] p-4 text-xs sm:text-sm text-left space-y-1.5">
                <div className="flex justify-between">
                  <span className="text-[#565e74]">Seats:</span>
                  <span className="font-bold text-ink">{initialSeats.join(", ")}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-[#565e74]">Amount Paid:</span>
                  <span className="font-bold text-[#00796d]">{initialPricing.total}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-[#565e74]">Method:</span>
                  <span className="font-medium text-ink">{paymentMethod}</span>
                </div>
              </div>

              <div className="mt-6 flex flex-col gap-2.5">
                <button
                  type="button"
                  onClick={() => {
                    setPaymentSuccessModal(false);
                    navigate("/events");
                  }}
                  className="h-11 w-full rounded-lg bg-[#00796d] font-bold text-white transition hover:bg-[#00685f]"
                >
                  Return to Events
                </button>
                <button
                  type="button"
                  onClick={() => setPaymentSuccessModal(false)}
                  className="h-11 w-full rounded-lg border border-[#bcc9c6] font-semibold text-[#565e74] hover:bg-slate-50"
                >
                  Close
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
