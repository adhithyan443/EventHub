import { useState, useMemo, useEffect } from "react";
import { formatCurrency, formatDateTime } from "../../utils/formatters";
import notify from "../../utils/toast";
import OrganizerBookingDetailDrawer from "../../components/organizer/OrganizerBookingDetailDrawer";

// Status Badge Styling Maps
const bookingStatusBadgeStyles = {
  CONFIRMED: "bg-[#dcfce7] text-[#15803d] border border-[#bbf7d0]",
  PENDING: "bg-[#fef9c3] text-[#854d0e] border border-[#fef08a]",
  CANCELLED: "bg-[#fee2e2] text-[#b91c1c] border border-[#fecaca]",
  PARTIALLY_CANCELLED: "bg-[#ffedd5] text-[#c2410c] border border-[#fed7aa]",
};

const paymentStatusBadgeStyles = {
  SUCCESS: "bg-[#dcfce7] text-[#15803d] border border-[#bbf7d0]",
  PENDING: "bg-[#fef9c3] text-[#854d0e] border border-[#fef08a]",
  FAILED: "bg-[#fee2e2] text-[#b91c1c] border border-[#fecaca]",
  REFUND_REQUIRED: "bg-[#ffedd5] text-[#c2410c] border border-[#fed7aa]",
};

const refundStatusBadgeStyles = {
  FULL: "bg-[#fae8ff] text-[#86198f] border border-[#f5d0fe]",
  PARTIAL: "bg-[#f3e8ff] text-[#7e22ce] border border-[#e9d5ff]",
  PENDING: "bg-[#fef9c3] text-[#854d0e] border border-[#fef08a]",
  FAILED: "bg-[#fee2e2] text-[#b91c1c] border border-[#fecaca]",
};

const BOOKING_STATUS_OPTIONS = [
  { value: "ALL", label: "All Booking Statuses" },
  { value: "CONFIRMED", label: "Confirmed" },
  { value: "PENDING", label: "Pending" },
  { value: "CANCELLED", label: "Cancelled" },
  { value: "PARTIALLY_CANCELLED", label: "Partially Cancelled" },
];

const PAYMENT_STATUS_OPTIONS = [
  { value: "ALL", label: "All Payment Statuses" },
  { value: "SUCCESS", label: "Captured / Success" },
  { value: "PENDING", label: "Pending Payment" },
  { value: "FAILED", label: "Failed Payment" },
  { value: "REFUNDED", label: "Any Refund" },
  { value: "PARTIAL", label: "Partially Refunded" },
  { value: "FULL", label: "Fully Refunded" },
  { value: "PENDING_REFUND", label: "Refund Pending / Processing" },
];

const PAGE_SIZE_OPTIONS = [10, 25, 50, 100];

// ============================================================================
// TEMPORARY MOCK DATA (Phase 5 UI Demonstration - Strictly Domain-Aligned)
// ============================================================================
const MOCK_EVENTS = [
  { id: "e1111111-1111-1111-1111-111111111111", title: "Sunburn Arena EDM Night 2026", venue_name: "MMRDA Grounds, Mumbai" },
  { id: "e2222222-2222-2222-2222-222222222222", title: "Global Tech Summit & Hackathon", venue_name: "Bangalore International Exhibition Centre" },
  { id: "e3333333-3333-3333-3333-333333333333", title: "Arijit Singh Live Symphony", venue_name: "Jawaharlal Nehru Stadium, Delhi" },
];

const INITIAL_MOCK_BOOKINGS = [
  {
    id: "b1000001-0000-0000-0000-000000000001",
    booking_number: "ORD-2026-0891",
    created_at: "2026-11-25T14:32:00Z",
    customer: {
      id: "u1",
      full_name: "Aarav Sharma",
      email: "aarav.sharma@example.com",
      phone: "+91 98765 43210",
    },
    event: {
      id: "e1111111-1111-1111-1111-111111111111",
      title: "Sunburn Arena EDM Night 2026",
      event_type: "Concert",
      venue_name: "MMRDA Grounds, Mumbai",
    },
    ticket_count: 2,
    original_booking_total: "3500.00",
    currency: "INR",
    booking_status: "CONFIRMED",
    payment_status: "SUCCESS",
    refund_status: "",
    financials: {
      subtotal: "3000.00",
      platform_fee: "200.00",
      tax: "300.00",
      discount: "0.00",
      original_booking_total: "3500.00",
      successful_captured_amount: "3500.00",
      total_refunded_amount: "0.00",
      net_captured_after_refunds: "3500.00",
      currency: "INR",
    },
    items: [
      { id: "bi-1", ticket_type_name: "VIP Early Bird", quantity: 2, unit_price: "1500.00", total_price: "3000.00", seat_label: "VIP-A12" },
    ],
    tickets: [
      { id: "tck-1", ticket_code: "TCK-EDM-01", admission_number: 1, ticket_type_name: "VIP Early Bird", seat_label: "VIP-A12", price: "1500.00", status: "VALID", issued_at: "2026-11-25T14:32:00Z" },
      { id: "tck-2", ticket_code: "TCK-EDM-02", admission_number: 2, ticket_type_name: "VIP Early Bird", seat_label: "VIP-A13", price: "1500.00", status: "VALID", issued_at: "2026-11-25T14:32:00Z" },
    ],
    payments: [
      { id: "pay-1", transaction_id: "pay_EDM9918231", provider: "Razorpay", payment_method: "UPI", amount: "3500.00", status: "SUCCESS", paid_at: "2026-11-25T14:33:10Z" },
    ],
    refunds: [],
    cancellation_policy: { cancellation_allowed: true, cancellation_deadline_hours: 48, refund_policy: "Standard Tier", refund_percentage: 80 },
  },
  {
    id: "b1000002-0000-0000-0000-000000000002",
    booking_number: "ORD-2026-0892",
    created_at: "2026-11-24T18:15:00Z",
    customer: {
      id: "u2",
      full_name: "Priya Patel",
      email: "priya.patel@example.com",
      phone: "+91 91234 56789",
    },
    event: {
      id: "e2222222-2222-2222-2222-222222222222",
      title: "Global Tech Summit & Hackathon",
      event_type: "Conference",
      venue_name: "Bangalore International Exhibition Centre",
    },
    ticket_count: 1,
    original_booking_total: "1999.00",
    currency: "INR",
    booking_status: "CONFIRMED",
    payment_status: "SUCCESS",
    refund_status: "",
    financials: {
      subtotal: "1800.00",
      platform_fee: "100.00",
      tax: "99.00",
      discount: "0.00",
      original_booking_total: "1999.00",
      successful_captured_amount: "1999.00",
      total_refunded_amount: "0.00",
      net_captured_after_refunds: "1999.00",
      currency: "INR",
    },
    items: [
      { id: "bi-2", ticket_type_name: "Delegate Pass", quantity: 1, unit_price: "1800.00", total_price: "1800.00" },
    ],
    tickets: [
      { id: "tck-3", ticket_code: "TCK-GTS-44", admission_number: 1, ticket_type_name: "Delegate Pass", price: "1800.00", status: "USED", issued_at: "2026-11-24T18:15:00Z", used_at: "2026-11-25T09:00:00Z" },
    ],
    payments: [
      { id: "pay-2", transaction_id: "pay_GTS1102941", provider: "Razorpay", payment_method: "Credit Card", amount: "1999.00", status: "SUCCESS", paid_at: "2026-11-24T18:16:02Z" },
    ],
    refunds: [],
  },
  {
    id: "b1000003-0000-0000-0000-000000000003",
    booking_number: "ORD-2026-0893",
    created_at: "2026-11-23T11:05:00Z",
    customer: {
      id: "u3",
      full_name: "Rohan Verma",
      email: "rohan.v@example.com",
    },
    event: {
      id: "e3333333-3333-3333-3333-333333333333",
      title: "Arijit Singh Live Symphony",
      event_type: "Concert",
      venue_name: "Jawaharlal Nehru Stadium, Delhi",
    },
    ticket_count: 2,
    original_booking_total: "4000.00",
    currency: "INR",
    booking_status: "PARTIALLY_CANCELLED",
    payment_status: "SUCCESS",
    refund_status: "PARTIAL",
    financials: {
      subtotal: "3600.00",
      platform_fee: "200.00",
      tax: "200.00",
      discount: "0.00",
      original_booking_total: "4000.00",
      successful_captured_amount: "4000.00",
      total_refunded_amount: "2000.00",
      net_captured_after_refunds: "2000.00",
      currency: "INR",
    },
    items: [
      { id: "bi-3", ticket_type_name: "Gold Seating", quantity: 2, unit_price: "1800.00", total_price: "3600.00", seat_label: "G-10" },
    ],
    tickets: [
      { id: "tck-4", ticket_code: "TCK-ARJ-101", admission_number: 1, ticket_type_name: "Gold Seating", seat_label: "G-10", price: "1800.00", status: "VALID", issued_at: "2026-11-23T11:05:00Z" },
      { id: "tck-5", ticket_code: "TCK-ARJ-102", admission_number: 2, ticket_type_name: "Gold Seating", seat_label: "G-11", price: "1800.00", status: "REFUNDED", issued_at: "2026-11-23T11:05:00Z" },
    ],
    payments: [
      { id: "pay-3", transaction_id: "pay_ARJ7712399", provider: "Razorpay", payment_method: "Net Banking", amount: "4000.00", status: "SUCCESS", paid_at: "2026-11-23T11:06:40Z" },
    ],
    refunds: [
      { id: "ref-1", amount: "2000.00", reason: "Partial attendee schedule conflict", status: "SUCCESS", gateway_refund_id: "rfnd_ARJ881", created_at: "2026-11-24T10:00:00Z" },
    ],
  },
  {
    id: "b1000004-0000-0000-0000-000000000004",
    booking_number: "ORD-2026-0894",
    created_at: "2026-11-22T09:40:00Z",
    customer: {
      id: "u4",
      full_name: "Neha Kulkarni",
      email: "neha.k@example.com",
    },
    event: {
      id: "e1111111-1111-1111-1111-111111111111",
      title: "Sunburn Arena EDM Night 2026",
      event_type: "Concert",
      venue_name: "MMRDA Grounds, Mumbai",
    },
    ticket_count: 1,
    original_booking_total: "1750.00",
    currency: "INR",
    booking_status: "CANCELLED",
    payment_status: "SUCCESS",
    refund_status: "FULL",
    financials: {
      subtotal: "1500.00",
      platform_fee: "150.00",
      tax: "100.00",
      discount: "0.00",
      original_booking_total: "1750.00",
      successful_captured_amount: "1750.00",
      total_refunded_amount: "1750.00",
      net_captured_after_refunds: "0.00",
      currency: "INR",
    },
    items: [
      { id: "bi-4", ticket_type_name: "VIP Early Bird", quantity: 1, unit_price: "1500.00", total_price: "1500.00" },
    ],
    tickets: [
      { id: "tck-6", ticket_code: "TCK-EDM-99", admission_number: 1, ticket_type_name: "VIP Early Bird", price: "1500.00", status: "CANCELLED", issued_at: "2026-11-22T09:40:00Z" },
    ],
    payments: [
      { id: "pay-4", transaction_id: "pay_EDM551203", provider: "Razorpay", payment_method: "Debit Card", amount: "1750.00", status: "SUCCESS", paid_at: "2026-11-22T09:41:00Z" },
    ],
    refunds: [
      { id: "ref-2", amount: "1750.00", reason: "Customer requested cancellation before deadline", status: "SUCCESS", gateway_refund_id: "rfnd_EDM002", created_at: "2026-11-23T14:20:00Z" },
    ],
  },
  {
    id: "b1000005-0000-0000-0000-000000000005",
    booking_number: "ORD-2026-0895",
    created_at: "2026-11-21T16:20:00Z",
    customer: {
      id: "u5",
      full_name: "Vikram Joshi",
      email: "vikram.j@example.com",
    },
    event: {
      id: "e2222222-2222-2222-2222-222222222222",
      title: "Global Tech Summit & Hackathon",
      event_type: "Conference",
      venue_name: "Bangalore International Exhibition Centre",
    },
    ticket_count: 1,
    original_booking_total: "1999.00",
    currency: "INR",
    booking_status: "PENDING",
    payment_status: "PENDING",
    refund_status: "",
    financials: {
      subtotal: "1800.00",
      platform_fee: "100.00",
      tax: "99.00",
      discount: "0.00",
      original_booking_total: "1999.00",
      successful_captured_amount: "0.00",
      total_refunded_amount: "0.00",
      net_captured_after_refunds: "0.00",
      currency: "INR",
    },
    items: [
      { id: "bi-5", ticket_type_name: "Delegate Pass", quantity: 1, unit_price: "1800.00", total_price: "1800.00" },
    ],
    tickets: [],
    payments: [],
    refunds: [],
  },
];

export default function OrganizerBookingsPage() {
  // State for Filters & Search
  const [search, setSearch] = useState("");
  const [selectedEventId, setSelectedEventId] = useState("");
  const [bookingStatus, setBookingStatus] = useState("ALL");
  const [paymentStatus, setPaymentStatus] = useState("ALL");
  const [fromDate, setFromDate] = useState("");
  const [toDate, setToDate] = useState("");

  // Pagination State
  const [page, setPage] = useState(1);
  const [limit, setLimit] = useState(10);

  // UI Interactive States
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [activeBookingDetail, setActiveBookingDetail] = useState(null);
  const [copiedCode, setCopiedCode] = useState(null);

  // Filter Bookings locally (Phase 5 UI demonstration; Phase 6 swaps with API)
  const filteredBookings = useMemo(() => {
    return INITIAL_MOCK_BOOKINGS.filter((item) => {
      // Event filter
      if (selectedEventId && item.event?.id !== selectedEventId) {
        return false;
      }
      // Booking Status filter
      if (bookingStatus !== "ALL" && item.booking_status !== bookingStatus) {
        return false;
      }
      // Payment Status filter
      if (paymentStatus !== "ALL") {
        if (paymentStatus === "REFUNDED" && !["FULL", "PARTIAL", "REFUNDED"].includes(item.refund_status)) return false;
        if (paymentStatus === "PARTIAL" && item.refund_status !== "PARTIAL") return false;
        if (paymentStatus === "FULL" && item.refund_status !== "FULL") return false;
        if (paymentStatus === "PENDING_REFUND" && item.refund_status !== "PENDING") return false;
        if (["SUCCESS", "PENDING", "FAILED"].includes(paymentStatus) && item.payment_status !== paymentStatus) {
          return false;
        }
      }
      // Search
      if (search.trim()) {
        const query = search.toLowerCase();
        const refMatch = item.booking_number?.toLowerCase().includes(query);
        const nameMatch = item.customer?.full_name?.toLowerCase().includes(query);
        const emailMatch = item.customer?.email?.toLowerCase().includes(query);
        if (!refMatch && !nameMatch && !emailMatch) return false;
      }
      // Date Range
      if (fromDate) {
        const bookingDate = new Date(item.created_at).toISOString().split("T")[0];
        if (bookingDate < fromDate) return false;
      }
      if (toDate) {
        const bookingDate = new Date(item.created_at).toISOString().split("T")[0];
        if (bookingDate > toDate) return false;
      }
      return true;
    });
  }, [selectedEventId, bookingStatus, paymentStatus, search, fromDate, toDate]);

  // Aggregate Stats (Calculated strictly based on domain definitions)
  const stats = useMemo(() => {
    let totalBookings = filteredBookings.length;
    let confirmedBookings = 0;
    let cancelledBookings = 0;
    let totalTicketsSold = 0;
    let grossCaptured = 0;
    let totalRefunded = 0;

    filteredBookings.forEach((b) => {
      if (b.booking_status === "CONFIRMED") confirmedBookings++;
      if (b.booking_status === "CANCELLED") cancelledBookings++;

      // Tickets sold (active eligible admissions)
      const activeTickets = (b.tickets || []).filter(
        (t) => t.status === "VALID" || t.status === "USED"
      ).length;
      totalTicketsSold += activeTickets > 0 ? activeTickets : b.booking_status === "CONFIRMED" ? b.ticket_count : 0;

      // Captured & Refunded
      const captured = parseFloat(b.financials?.successful_captured_amount || "0");
      const refunded = parseFloat(b.financials?.total_refunded_amount || "0");
      grossCaptured += captured;
      totalRefunded += refunded;
    });

    const netCaptured = grossCaptured - totalRefunded;

    return {
      totalBookings,
      confirmedBookings,
      cancelledBookings,
      totalTicketsSold,
      grossCaptured,
      totalRefunded,
      netCaptured,
    };
  }, [filteredBookings]);

  // Pagination calculation
  const totalRecords = filteredBookings.length;
  const totalPages = Math.max(1, Math.ceil(totalRecords / limit));
  const paginatedBookings = useMemo(() => {
    const start = (page - 1) * limit;
    return filteredBookings.slice(start, start + limit);
  }, [filteredBookings, page, limit]);

  // Auto-correct page when filter changes cause totalPages to shrink
  useEffect(() => {
    if (page > totalPages) {
      setPage(1);
    }
  }, [totalPages, page]);

  const copyToClipboard = async (text, label) => {
    if (!text) return;
    try {
      await navigator.clipboard.writeText(text);
      setCopiedCode(text);
      notify.info(`Copied ${label} to clipboard`);
      setTimeout(() => setCopiedCode(null), 2000);
    } catch {
      notify.error("Failed to copy");
    }
  };

  const handleResetFilters = () => {
    setSearch("");
    setSelectedEventId("");
    setBookingStatus("ALL");
    setPaymentStatus("ALL");
    setFromDate("");
    setToDate("");
    setPage(1);
    notify.info("Filters cleared");
  };

  return (
    <div className="flex-1 flex flex-col overflow-hidden bg-white w-full">
      {/* Page Header */}
      <div className="px-4 sm:px-6 py-4 border-b border-[#dce2f7] flex flex-col md:flex-row md:items-center justify-between gap-4 shrink-0 bg-white">
        <div>
          <div className="flex items-center gap-3">
            <h1 className="text-[#141b2b] text-xl font-bold tracking-tight">
              Bookings Management
            </h1>
            <span className="bg-[#e6f4f2] text-[#00685f] text-xs font-semibold px-2.5 py-0.5 rounded-full border border-[#00685f]/20">
              Orders & Admissions
            </span>
          </div>
          <p className="text-[#3d4947] text-xs sm:text-sm mt-0.5">
            View and manage attendee orders, ticket allocations, payment captures, and cancellations.
          </p>
        </div>

        <button
          type="button"
          onClick={() => {
            setLoading(true);
            setTimeout(() => {
              setLoading(false);
              notify.info("Bookings refreshed");
            }, 400);
          }}
          disabled={loading}
          className="bg-[#f9f9ff] border border-[#bcc9c6] text-[#141b2b] text-xs sm:text-sm font-semibold px-3 py-1.5 rounded-lg flex items-center gap-1.5 hover:bg-[#eef0fb] transition-colors cursor-pointer self-start md:self-auto shrink-0"
        >
          <svg className={`w-3.5 h-3.5 ${loading ? "animate-spin text-[#00685f]" : ""}`} viewBox="0 0 20 20" fill="currentColor">
            <path fillRule="evenodd" d="M4 2a1 1 0 011 1v2.101a7.002 7.002 0 0111.601 2.566 1 1 0 11-1.885.666A5.002 5.002 0 005.999 7H9a1 1 0 010 2H4a1 1 0 01-1-1V3a1 1 0 011-1zm.008 9.057a1 1 0 011.276.61A5.002 5.002 0 0014.001 13H11a1 1 0 110-2h5a1 1 0 011 1v5a1 1 0 11-2 0v-2.101a7.002 7.002 0 01-11.601-2.566 1 1 0 01.61-1.276z" clipRule="evenodd" />
          </svg>
          <span>Refresh</span>
        </button>
      </div>

      {/* Main Content Area */}
      <div className="flex-1 overflow-y-auto p-4 sm:p-6 space-y-5">
        {/* =================================================================== */}
        {/* SUMMARY METRICS CARDS                                               */}
        {/* =================================================================== */}
        <div className="grid grid-cols-2 sm:grid-cols-3 xl:grid-cols-6 gap-3.5">
          {/* Card 1: Total Bookings */}
          <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-xs flex flex-col justify-between">
            <span className="text-[#3d4947] text-[11px] font-semibold uppercase tracking-wider mb-1">
              Total Orders
            </span>
            <span className="text-[#141b2b] text-xl lg:text-2xl font-bold tracking-tight">
              {stats.totalBookings}
            </span>
            <span className="text-[11px] mt-1 text-[#6d7a77] font-medium">All booking orders</span>
          </div>

          {/* Card 2: Confirmed Bookings */}
          <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-xs flex flex-col justify-between">
            <span className="text-[#3d4947] text-[11px] font-semibold uppercase tracking-wider mb-1">
              Confirmed
            </span>
            <span className="text-[#15803d] text-xl lg:text-2xl font-bold tracking-tight">
              {stats.confirmedBookings}
            </span>
            <span className="text-[11px] mt-1 text-[#15803d] font-medium">Active reservations</span>
          </div>

          {/* Card 3: Cancelled Bookings */}
          <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-xs flex flex-col justify-between">
            <span className="text-[#3d4947] text-[11px] font-semibold uppercase tracking-wider mb-1">
              Cancelled
            </span>
            <span className="text-[#b91c1c] text-xl lg:text-2xl font-bold tracking-tight">
              {stats.cancelledBookings}
            </span>
            <span className="text-[11px] mt-1 text-[#b91c1c] font-medium">Cancelled orders</span>
          </div>

          {/* Card 4: Tickets Sold */}
          <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-xs flex flex-col justify-between">
            <span className="text-[#3d4947] text-[11px] font-semibold uppercase tracking-wider mb-1">
              Tickets Sold
            </span>
            <span className="text-[#141b2b] text-xl lg:text-2xl font-bold tracking-tight">
              {stats.totalTicketsSold}
            </span>
            <span className="text-[11px] mt-1 text-[#6d7a77] font-medium">Valid admissions</span>
          </div>

          {/* Card 5: Gross Captured */}
          <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-xs flex flex-col justify-between">
            <span className="text-[#3d4947] text-[11px] font-semibold uppercase tracking-wider mb-1">
              Gross Captured
            </span>
            <span className="text-[#141b2b] text-xl lg:text-2xl font-bold tracking-tight">
              {formatCurrency(stats.grossCaptured)}
            </span>
            <span className="text-[11px] mt-1 text-[#6d7a77] font-medium">Total gateway captures</span>
          </div>

          {/* Card 6: Net Captured */}
          <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-xs flex flex-col justify-between">
            <span className="text-[#3d4947] text-[11px] font-semibold uppercase tracking-wider mb-1">
              Net Captured
            </span>
            <span className="text-[#00685f] text-xl lg:text-2xl font-bold tracking-tight">
              {formatCurrency(stats.netCaptured)}
            </span>
            <span className="text-[11px] mt-1 text-[#00685f] font-medium">Captured after refunds</span>
          </div>
        </div>

        {/* =================================================================== */}
        {/* FILTER BAR                                                          */}
        {/* =================================================================== */}
        <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-3.5 space-y-3">
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-12 gap-3">
            {/* Search Input (4 columns on lg) */}
            <div className="lg:col-span-4 relative">
              <svg className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-[#6d7a77]" viewBox="0 0 20 20" fill="currentColor">
                <path fillRule="evenodd" d="M8 4a4 4 0 100 8 4 4 0 000-8zM2 8a6 6 0 1110.89 3.476l4.817 4.817a1 1 0 01-1.414 1.414l-4.816-4.816A6 6 0 012 8z" clipRule="evenodd" />
              </svg>
              <input
                type="text"
                placeholder="Search booking #, customer, email..."
                value={search}
                onChange={(e) => {
                  setSearch(e.target.value);
                  setPage(1);
                }}
                className="w-full bg-white border border-[#bcc9c6] rounded-lg pl-9 pr-8 py-2 text-xs sm:text-sm text-[#141b2b] placeholder:text-[#6d7a77] focus:outline-none focus:ring-1 focus:ring-[#00685f]/40"
              />
              {search && (
                <button
                  type="button"
                  onClick={() => setSearch("")}
                  className="absolute right-2.5 top-1/2 -translate-y-1/2 text-[#6d7a77] hover:text-[#141b2b] cursor-pointer text-xs"
                >
                  ✕
                </button>
              )}
            </div>

            {/* Event Dropdown (3 columns on lg) */}
            <div className="lg:col-span-3">
              <select
                value={selectedEventId}
                onChange={(e) => {
                  setSelectedEventId(e.target.value);
                  setPage(1);
                }}
                className="w-full bg-white border border-[#bcc9c6] rounded-lg px-3 py-2 text-xs sm:text-sm text-[#141b2b] focus:outline-none focus:ring-1 focus:ring-[#00685f]/40 cursor-pointer"
              >
                <option value="">All Events</option>
                {MOCK_EVENTS.map((evt) => (
                  <option key={evt.id} value={evt.id}>
                    {evt.title}
                  </option>
                ))}
              </select>
            </div>

            {/* Booking Status Dropdown (2 columns on lg) */}
            <div className="lg:col-span-2">
              <select
                value={bookingStatus}
                onChange={(e) => {
                  setBookingStatus(e.target.value);
                  setPage(1);
                }}
                className="w-full bg-white border border-[#bcc9c6] rounded-lg px-3 py-2 text-xs sm:text-sm text-[#141b2b] focus:outline-none focus:ring-1 focus:ring-[#00685f]/40 cursor-pointer"
              >
                {BOOKING_STATUS_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {opt.label}
                  </option>
                ))}
              </select>
            </div>

            {/* Payment Status Dropdown (3 columns on lg) */}
            <div className="lg:col-span-3">
              <select
                value={paymentStatus}
                onChange={(e) => {
                  setPaymentStatus(e.target.value);
                  setPage(1);
                }}
                className="w-full bg-white border border-[#bcc9c6] rounded-lg px-3 py-2 text-xs sm:text-sm text-[#141b2b] focus:outline-none focus:ring-1 focus:ring-[#00685f]/40 cursor-pointer"
              >
                {PAYMENT_STATUS_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {opt.label}
                  </option>
                ))}
              </select>
            </div>
          </div>

          {/* Date Filters & Clear Button */}
          <div className="flex flex-wrap items-center justify-between gap-3 pt-2 border-t border-[#bcc9c6]/30">
            <div className="flex flex-wrap items-center gap-3">
              <div className="flex items-center gap-1.5">
                <span className="text-[11px] font-semibold text-[#6d7a77] uppercase">From:</span>
                <input
                  type="date"
                  value={fromDate}
                  onChange={(e) => {
                    setFromDate(e.target.value);
                    setPage(1);
                  }}
                  className="bg-white border border-[#bcc9c6] rounded-lg px-2.5 py-1 text-xs text-[#141b2b] focus:outline-none focus:ring-1 focus:ring-[#00685f]/40"
                />
              </div>

              <div className="flex items-center gap-1.5">
                <span className="text-[11px] font-semibold text-[#6d7a77] uppercase">To:</span>
                <input
                  type="date"
                  value={toDate}
                  onChange={(e) => {
                    setToDate(e.target.value);
                    setPage(1);
                  }}
                  className="bg-white border border-[#bcc9c6] rounded-lg px-2.5 py-1 text-xs text-[#141b2b] focus:outline-none focus:ring-1 focus:ring-[#00685f]/40"
                />
              </div>
            </div>

            {(search || selectedEventId || bookingStatus !== "ALL" || paymentStatus !== "ALL" || fromDate || toDate) && (
              <button
                type="button"
                onClick={handleResetFilters}
                className="text-xs text-[#b91c1c] hover:text-[#991b1b] font-semibold flex items-center gap-1 cursor-pointer transition-colors"
              >
                <svg className="w-3.5 h-3.5" viewBox="0 0 20 20" fill="currentColor">
                  <path fillRule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clipRule="evenodd" />
                </svg>
                <span>Clear All Filters</span>
              </button>
            )}
          </div>
        </div>

        {/* =================================================================== */}
        {/* BOOKINGS TABLE SECTION                                              */}
        {/* =================================================================== */}
        <div className="bg-white border border-[#bcc9c6] rounded-xl shadow-xs overflow-hidden">
          {error ? (
            <div className="p-8 text-center text-xs text-[#b91c1c]">
              <p className="font-bold">Failed to load bookings</p>
              <p className="mt-1 text-[#6d7a77]">{error}</p>
              <button
                type="button"
                onClick={() => setError(null)}
                className="mt-3 px-3 py-1.5 bg-[#fef2f2] border border-[#fecaca] rounded-lg text-xs font-semibold cursor-pointer"
              >
                Retry
              </button>
            </div>
          ) : loading ? (
            <div className="p-6 space-y-3">
              {[1, 2, 3, 4, 5].map((i) => (
                <div key={i} className="h-12 bg-gray-100 rounded-lg animate-pulse" />
              ))}
            </div>
          ) : paginatedBookings.length === 0 ? (
            <div className="py-12 px-4 text-center">
              <svg className="w-12 h-12 mx-auto text-[#bcc9c6] mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
              </svg>
              <h3 className="text-sm font-bold text-[#141b2b]">No bookings found</h3>
              <p className="text-xs text-[#6d7a77] mt-1 max-w-sm mx-auto">
                No orders match your active filter criteria. Try adjusting your search query, status filters, or date range.
              </p>
              <button
                type="button"
                onClick={handleResetFilters}
                className="mt-4 px-4 py-2 bg-[#00685f] text-white rounded-lg text-xs font-bold hover:bg-[#005149] transition-colors cursor-pointer"
              >
                Reset Filters
              </button>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-left text-xs">
                <thead className="bg-[#f9f9ff] border-b border-[#bcc9c6] text-[#6d7a77] text-[11px] uppercase font-semibold">
                  <tr>
                    <th className="py-3 px-4">Booking Ref</th>
                    <th className="py-3 px-4">Customer</th>
                    <th className="py-3 px-4">Event</th>
                    <th className="py-3 px-3 text-center">Tickets</th>
                    <th className="py-3 px-4">Date</th>
                    <th className="py-3 px-4 text-right">Amount</th>
                    <th className="py-3 px-4 text-center">Payment / Refund</th>
                    <th className="py-3 px-3 text-center">Status</th>
                    <th className="py-3 px-4 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-[#bcc9c6]/30">
                  {paginatedBookings.map((b) => {
                    const bStatus = b.booking_status?.toUpperCase() || "CONFIRMED";
                    const bBadge = bookingStatusBadgeStyles[bStatus] || "bg-[#f3f4f6] text-[#4b5563]";

                    const pStatus = b.payment_status?.toUpperCase() || "SUCCESS";
                    const pBadge = paymentStatusBadgeStyles[pStatus] || "bg-[#f3f4f6] text-[#4b5563]";

                    const rStatus = b.refund_status?.toUpperCase();
                    const rBadge = rStatus ? refundStatusBadgeStyles[rStatus] : null;

                    return (
                      <tr key={b.id} className="hover:bg-[#f9f9ff]/70 transition-colors">
                        {/* Reference */}
                        <td className="py-3 px-4">
                          <div className="flex items-center gap-1.5 font-mono font-bold text-[#00685f]">
                            <span>{b.booking_number}</span>
                            <button
                              type="button"
                              onClick={() => copyToClipboard(b.booking_number, "Reference")}
                              className="text-[#6d7a77] hover:text-[#00685f] p-0.5 cursor-pointer"
                              title="Copy"
                            >
                              <svg className="w-3.5 h-3.5" viewBox="0 0 20 20" fill="currentColor">
                                <path d="M8 3a1 1 0 011-1h2a1 1 0 110 2H9a1 1 0 01-1-1z" />
                                <path d="M6 3a2 2 0 00-2 2v11a2 2 0 002 2h8a2 2 0 002-2V5a2 2 0 00-2-2 3 3 0 01-3 3H9a3 3 0 01-3-3z" />
                              </svg>
                            </button>
                            {copiedCode === b.booking_number && (
                              <span className="text-[10px] text-[#00685f] font-sans font-semibold">✓</span>
                            )}
                          </div>
                        </td>

                        {/* Customer */}
                        <td className="py-3 px-4">
                          <div className="font-bold text-[#141b2b]">{b.customer?.full_name}</div>
                          <div className="text-[11px] text-[#6d7a77]">{b.customer?.email}</div>
                        </td>

                        {/* Event */}
                        <td className="py-3 px-4 max-w-[200px]">
                          <div className="font-semibold text-[#141b2b] truncate" title={b.event?.title}>
                            {b.event?.title}
                          </div>
                          <div className="text-[11px] text-[#6d7a77] truncate">
                            {b.event?.venue_name}
                          </div>
                        </td>

                        {/* Tickets */}
                        <td className="py-3 px-3 text-center">
                          <span className="inline-block bg-[#eef0fb] text-[#141b2b] px-2 py-0.5 rounded font-bold text-[11px]">
                            {b.ticket_count}
                          </span>
                        </td>

                        {/* Date */}
                        <td className="py-3 px-4 text-[#3d4947] whitespace-nowrap">
                          {formatDateTime(b.created_at)}
                        </td>

                        {/* Amount */}
                        <td className="py-3 px-4 text-right font-bold text-[#141b2b] whitespace-nowrap">
                          {formatCurrency(b.original_booking_total, b.currency)}
                        </td>

                        {/* Payment & Refund Status Badges */}
                        <td className="py-3 px-4 text-center">
                          <div className="flex flex-col items-center gap-1">
                            <span className={`text-[10px] font-bold px-2 py-0.5 rounded-full ${pBadge}`}>
                              {pStatus}
                            </span>
                            {rBadge && (
                              <span className={`text-[9px] font-bold px-1.5 py-0.2 rounded-full ${rBadge}`}>
                                REFUND: {rStatus}
                              </span>
                            )}
                          </div>
                        </td>

                        {/* Booking Status */}
                        <td className="py-3 px-3 text-center">
                          <span className={`text-[10px] font-bold px-2 py-0.5 rounded-full ${bBadge}`}>
                            {bStatus}
                          </span>
                        </td>

                        {/* Action Drawer Button */}
                        <td className="py-3 px-4 text-right">
                          <button
                            type="button"
                            onClick={() => setActiveBookingDetail(b)}
                            className="inline-flex items-center gap-1 px-2.5 py-1.5 text-xs font-bold text-[#00685f] hover:text-white bg-[#e6f4f2] hover:bg-[#00685f] rounded-lg transition-colors cursor-pointer"
                          >
                            <svg className="w-3.5 h-3.5" viewBox="0 0 20 20" fill="currentColor">
                              <path d="M10 12a2 2 0 100-4 2 2 0 000 4z" />
                              <path fillRule="evenodd" d="M.458 10C1.732 5.943 5.522 3 10 3s8.268 2.943 9.542 7c-1.274 4.057-5.064 7-9.542 7S1.732 14.057.458 10zM14 10a4 4 0 11-8 0 4 4 0 018 0z" clipRule="evenodd" />
                            </svg>
                            <span>Details</span>
                          </button>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}

          {/* Table Footer with Pagination */}
          {!loading && filteredBookings.length > 0 && (
            <div className="px-4 py-3 border-t border-[#bcc9c6] bg-[#f9f9ff] flex flex-col sm:flex-row items-center justify-between gap-3 text-xs">
              <div className="text-[#6d7a77]">
                Showing <span className="font-bold text-[#141b2b]">{(page - 1) * limit + 1}</span> to{" "}
                <span className="font-bold text-[#141b2b]">
                  {Math.min(page * limit, totalRecords)}
                </span>{" "}
                of <span className="font-bold text-[#141b2b]">{totalRecords}</span> bookings
              </div>

              <div className="flex items-center gap-2">
                <div className="flex items-center gap-1.5 mr-2">
                  <span className="text-[#6d7a77]">Rows:</span>
                  <select
                    value={limit}
                    onChange={(e) => {
                      setLimit(Number(e.target.value));
                      setPage(1);
                    }}
                    className="bg-white border border-[#bcc9c6] rounded px-2 py-1 text-xs text-[#141b2b] cursor-pointer"
                  >
                    {PAGE_SIZE_OPTIONS.map((size) => (
                      <option key={size} value={size}>
                        {size}
                      </option>
                    ))}
                  </select>
                </div>

                <button
                  type="button"
                  disabled={page <= 1}
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                  className="px-2.5 py-1 border border-[#bcc9c6] rounded bg-white font-medium text-[#141b2b] disabled:opacity-40 disabled:cursor-not-allowed hover:bg-gray-50 cursor-pointer"
                >
                  Prev
                </button>
                <span className="px-2 text-[#3d4947] font-semibold">
                  {page} / {totalPages}
                </span>
                <button
                  type="button"
                  disabled={page >= totalPages}
                  onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                  className="px-2.5 py-1 border border-[#bcc9c6] rounded bg-white font-medium text-[#141b2b] disabled:opacity-40 disabled:cursor-not-allowed hover:bg-gray-50 cursor-pointer"
                >
                  Next
                </button>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Booking Detail Slide-in Drawer */}
      <OrganizerBookingDetailDrawer
        booking={activeBookingDetail}
        onClose={() => setActiveBookingDetail(null)}
      />
    </div>
  );
}
