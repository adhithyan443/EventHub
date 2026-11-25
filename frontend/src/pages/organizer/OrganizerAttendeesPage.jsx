import { useState, useMemo, useEffect } from "react";
import { formatDateTime } from "../../utils/formatters";
import notify from "../../utils/toast";

// Ticket / Check-in Status Styling Maps
const attendeeStatusBadgeStyles = {
  USED: "bg-[#dcfce7] text-[#15803d] border border-[#bbf7d0]",
  VALID: "bg-[#fef9c3] text-[#854d0e] border border-[#fef08a]",
  CANCELLED: "bg-[#fee2e2] text-[#b91c1c] border border-[#fecaca]",
  REFUNDED: "bg-[#f3e8ff] text-[#7e22ce] border border-[#e9d5ff]",
};

const attendeeStatusLabels = {
  USED: "Checked In",
  VALID: "Pending Check-In",
  CANCELLED: "Cancelled",
  REFUNDED: "Refunded",
};

const CHECKIN_STATUS_OPTIONS = [
  { value: "ALL", label: "All Check-In Statuses" },
  { value: "VALID", label: "Pending Check-In (Valid)" },
  { value: "USED", label: "Checked In (Used)" },
  { value: "CANCELLED", label: "Cancelled" },
  { value: "REFUNDED", label: "Refunded" },
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

const INITIAL_MOCK_ATTENDEES = [
  {
    ticket_id: "tck-001",
    ticket_code: "TCK-EDM-01",
    admission_number: 1,
    booking_id: "b1000001-0000-0000-0000-000000000001",
    booking_number: "ORD-2026-0891",
    booking_reference: "ORD-2026-0891",
    customer: {
      id: "u1",
      full_name: "Aarav Sharma",
      email: "aarav.sharma@example.com",
    },
    event: {
      id: "e1111111-1111-1111-1111-111111111111",
      title: "Sunburn Arena EDM Night 2026",
    },
    event_id: "e1111111-1111-1111-1111-111111111111",
    event_title: "Sunburn Arena EDM Night 2026",
    ticket_type_name: "VIP Early Bird",
    ticket_tier_id: "tt-1",
    ticket_tier_name: "VIP Early Bird",
    seat_id: "s-101",
    seat_label: "VIP-A12",
    attendee_name: "Aarav Sharma",
    attendee_email: "aarav.sharma@example.com",
    status: "VALID",
    price: "1500.00",
    issued_at: "2026-11-25T14:32:00Z",
    created_at: "2026-11-25T14:32:00Z",
    used_at: null,
  },
  {
    ticket_id: "tck-002",
    ticket_code: "TCK-EDM-02",
    admission_number: 2,
    booking_id: "b1000001-0000-0000-0000-000000000001",
    booking_number: "ORD-2026-0891",
    booking_reference: "ORD-2026-0891",
    customer: {
      id: "u1",
      full_name: "Aarav Sharma",
      email: "aarav.sharma@example.com",
    },
    event: {
      id: "e1111111-1111-1111-1111-111111111111",
      title: "Sunburn Arena EDM Night 2026",
    },
    event_id: "e1111111-1111-1111-1111-111111111111",
    event_title: "Sunburn Arena EDM Night 2026",
    ticket_type_name: "VIP Early Bird",
    ticket_tier_id: "tt-1",
    ticket_tier_name: "VIP Early Bird",
    seat_id: "s-102",
    seat_label: "VIP-A13",
    attendee_name: "Aarav Sharma",
    attendee_email: "aarav.sharma@example.com",
    status: "USED",
    price: "1500.00",
    issued_at: "2026-11-25T14:32:00Z",
    created_at: "2026-11-25T14:32:00Z",
    used_at: "2026-11-25T17:15:30Z",
  },
  {
    ticket_id: "tck-003",
    ticket_code: "TCK-GTS-01",
    admission_number: 1,
    booking_id: "b1000002-0000-0000-0000-000000000002",
    booking_number: "ORD-2026-0892",
    booking_reference: "ORD-2026-0892",
    customer: {
      id: "u2",
      full_name: "Priya Nair",
      email: "priya.nair@techcorp.io",
    },
    event: {
      id: "e2222222-2222-2222-2222-222222222222",
      title: "Global Tech Summit & Hackathon",
    },
    event_id: "e2222222-2222-2222-2222-222222222222",
    event_title: "Global Tech Summit & Hackathon",
    ticket_type_name: "Developer All-Access",
    ticket_tier_id: "tt-2",
    ticket_tier_name: "Developer All-Access",
    seat_id: null,
    seat_label: null,
    attendee_name: "Priya Nair",
    attendee_email: "priya.nair@techcorp.io",
    status: "USED",
    price: "4999.00",
    issued_at: "2026-11-24T10:15:00Z",
    created_at: "2026-11-24T10:15:00Z",
    used_at: "2026-11-25T08:45:10Z",
  },
  {
    ticket_id: "tck-004",
    ticket_code: "TCK-GTS-02",
    admission_number: 2,
    booking_id: "b1000002-0000-0000-0000-000000000002",
    booking_number: "ORD-2026-0892",
    booking_reference: "ORD-2026-0892",
    customer: {
      id: "u2",
      full_name: "Rahul Verma",
      email: "rahul.verma@techcorp.io",
    },
    event: {
      id: "e2222222-2222-2222-2222-222222222222",
      title: "Global Tech Summit & Hackathon",
    },
    event_id: "e2222222-2222-2222-2222-222222222222",
    event_title: "Global Tech Summit & Hackathon",
    ticket_type_name: "Developer All-Access",
    ticket_tier_id: "tt-2",
    ticket_tier_name: "Developer All-Access",
    seat_id: null,
    seat_label: null,
    attendee_name: "Rahul Verma",
    attendee_email: "rahul.verma@techcorp.io",
    status: "VALID",
    price: "4999.00",
    issued_at: "2026-11-24T10:15:00Z",
    created_at: "2026-11-24T10:15:00Z",
    used_at: null,
  },
  {
    ticket_id: "tck-005",
    ticket_code: "TCK-GTS-03",
    admission_number: 3,
    booking_id: "b1000002-0000-0000-0000-000000000002",
    booking_number: "ORD-2026-0892",
    booking_reference: "ORD-2026-0892",
    customer: {
      id: "u2",
      full_name: "Sneha Patel",
      email: "sneha.patel@techcorp.io",
    },
    event: {
      id: "e2222222-2222-2222-2222-222222222222",
      title: "Global Tech Summit & Hackathon",
    },
    event_id: "e2222222-2222-2222-2222-222222222222",
    event_title: "Global Tech Summit & Hackathon",
    ticket_type_name: "Developer All-Access",
    ticket_tier_id: "tt-2",
    ticket_tier_name: "Developer All-Access",
    seat_id: null,
    seat_label: null,
    attendee_name: "Sneha Patel",
    attendee_email: "sneha.patel@techcorp.io",
    status: "USED",
    price: "4999.00",
    issued_at: "2026-11-24T10:15:00Z",
    created_at: "2026-11-24T10:15:00Z",
    used_at: "2026-11-25T09:02:44Z",
  },
  {
    ticket_id: "tck-006",
    ticket_code: "TCK-ARJ-01",
    admission_number: 1,
    booking_id: "b1000003-0000-0000-0000-000000000003",
    booking_number: "ORD-2026-0893",
    booking_reference: "ORD-2026-0893",
    customer: {
      id: "u3",
      full_name: "Rohan Kapoor",
      email: "rohan.kapoor@example.com",
    },
    event: {
      id: "e3333333-3333-3333-3333-333333333333",
      title: "Arijit Singh Live Symphony",
    },
    event_id: "e3333333-3333-3333-3333-333333333333",
    event_title: "Arijit Singh Live Symphony",
    ticket_type_name: "Balcony Gold",
    ticket_tier_id: "tt-3",
    ticket_tier_name: "Balcony Gold",
    seat_id: "s-201",
    seat_label: "BALC-B04",
    attendee_name: "Rohan Kapoor",
    attendee_email: "rohan.kapoor@example.com",
    status: "CANCELLED",
    price: "2499.00",
    issued_at: "2026-11-23T18:20:00Z",
    created_at: "2026-11-23T18:20:00Z",
    used_at: null,
  },
  {
    ticket_id: "tck-007",
    ticket_code: "TCK-ARJ-02",
    admission_number: 2,
    booking_id: "b1000004-0000-0000-0000-000000000004",
    booking_number: "ORD-2026-0894",
    booking_reference: "ORD-2026-0894",
    customer: {
      id: "u4",
      full_name: "Meera Krishnan",
      email: "meera.k@example.com",
    },
    event: {
      id: "e3333333-3333-3333-3333-333333333333",
      title: "Arijit Singh Live Symphony",
    },
    event_id: "e3333333-3333-3333-3333-333333333333",
    event_title: "Arijit Singh Live Symphony",
    ticket_type_name: "Balcony Gold",
    ticket_tier_id: "tt-3",
    ticket_tier_name: "Balcony Gold",
    seat_id: "s-202",
    seat_label: "BALC-B05",
    attendee_name: "Meera Krishnan",
    attendee_email: "meera.k@example.com",
    status: "REFUNDED",
    price: "2499.00",
    issued_at: "2026-11-22T09:40:00Z",
    created_at: "2026-11-22T09:40:00Z",
    used_at: null,
  },
  {
    ticket_id: "tck-008",
    ticket_code: "TCK-EDM-03",
    admission_number: 1,
    booking_id: "b1000005-0000-0000-0000-000000000005",
    booking_number: "ORD-2026-0895",
    booking_reference: "ORD-2026-0895",
    customer: {
      id: "u5",
      full_name: "Devendra Rathore",
      email: "devendra.r@example.com",
    },
    event: {
      id: "e1111111-1111-1111-1111-111111111111",
      title: "Sunburn Arena EDM Night 2026",
    },
    event_id: "e1111111-1111-1111-1111-111111111111",
    event_title: "Sunburn Arena EDM Night 2026",
    ticket_type_name: "General Admission Pass",
    ticket_tier_id: "tt-4",
    ticket_tier_name: "General Admission Pass",
    seat_id: null,
    seat_label: null,
    attendee_name: "Devendra Rathore",
    attendee_email: "devendra.r@example.com",
    status: "USED",
    price: "999.00",
    issued_at: "2026-11-21T11:10:00Z",
    created_at: "2026-11-21T11:10:00Z",
    used_at: "2026-11-25T16:50:12Z",
  },
  {
    ticket_id: "tck-009",
    ticket_code: "TCK-EDM-04",
    admission_number: 2,
    booking_id: "b1000005-0000-0000-0000-000000000005",
    booking_number: "ORD-2026-0895",
    booking_reference: "ORD-2026-0895",
    customer: {
      id: "u5",
      full_name: "Ananya Deshmukh",
      email: "ananya.d@example.com",
    },
    event: {
      id: "e1111111-1111-1111-1111-111111111111",
      title: "Sunburn Arena EDM Night 2026",
    },
    event_id: "e1111111-1111-1111-1111-111111111111",
    event_title: "Sunburn Arena EDM Night 2026",
    ticket_type_name: "General Admission Pass",
    ticket_tier_id: "tt-4",
    ticket_tier_name: "General Admission Pass",
    seat_id: null,
    seat_label: null,
    attendee_name: "Ananya Deshmukh",
    attendee_email: "ananya.d@example.com",
    status: "VALID",
    price: "999.00",
    issued_at: "2026-11-21T11:10:00Z",
    created_at: "2026-11-21T11:10:00Z",
    used_at: null,
  },
  {
    ticket_id: "tck-010",
    ticket_code: "TCK-ARJ-03",
    admission_number: 1,
    booking_id: "b1000006-0000-0000-0000-000000000006",
    booking_number: "ORD-2026-0896",
    booking_reference: "ORD-2026-0896",
    customer: {
      id: "u6",
      full_name: "Vikramaditya Bose",
      email: "v.bose@enterprise.com",
    },
    event: {
      id: "e3333333-3333-3333-3333-333333333333",
      title: "Arijit Singh Live Symphony",
    },
    event_id: "e3333333-3333-3333-3333-333333333333",
    event_title: "Arijit Singh Live Symphony",
    ticket_type_name: "Front Row VIP",
    ticket_tier_id: "tt-5",
    ticket_tier_name: "Front Row VIP",
    seat_id: "s-301",
    seat_label: "ROW-A-01",
    attendee_name: "Vikramaditya Bose",
    attendee_email: "v.bose@enterprise.com",
    status: "USED",
    price: "7500.00",
    issued_at: "2026-11-20T15:00:00Z",
    created_at: "2026-11-20T15:00:00Z",
    used_at: "2026-11-24T19:10:05Z",
  },
];

export default function OrganizerAttendeesPage() {
  // Filters State
  const [searchTerm, setSearchTerm] = useState("");
  const [selectedEventId, setSelectedEventId] = useState("ALL");
  const [selectedStatus, setSelectedStatus] = useState("ALL");

  // Pagination State
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);

  // Async Simulator State
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  // Copy Ticket Code State
  const [copiedCode, setCopiedCode] = useState(null);

  const handleCopyCode = (code) => {
    if (navigator.clipboard) {
      navigator.clipboard.writeText(code).then(() => {
        setCopiedCode(code);
        notify.success(`Ticket code ${code} copied!`);
        setTimeout(() => setCopiedCode(null), 2000);
      });
    } else {
      notify.info(`Ticket Code: ${code}`);
    }
  };

  const handleResetFilters = () => {
    setSearchTerm("");
    setSelectedEventId("ALL");
    setSelectedStatus("ALL");
    setCurrentPage(1);
  };

  const hasActiveFilters = Boolean(
    searchTerm.trim() || selectedEventId !== "ALL" || selectedStatus !== "ALL"
  );

  // Filtered Attendees
  const filteredAttendees = useMemo(() => {
    return INITIAL_MOCK_ATTENDEES.filter((item) => {
      // Event filter
      const itemEventId = item.event?.id || item.event_id;
      if (selectedEventId !== "ALL" && itemEventId !== selectedEventId) {
        return false;
      }

      // Check-in / Ticket status filter
      if (selectedStatus !== "ALL" && item.status !== selectedStatus) {
        return false;
      }

      // Search keyword
      if (searchTerm.trim()) {
        const query = searchTerm.toLowerCase();
        const aName = (item.customer?.full_name || item.attendee_name || "").toLowerCase();
        const aEmail = (item.customer?.email || item.attendee_email || "").toLowerCase();
        const tCode = (item.ticket_code || "").toLowerCase();
        const bRef = (item.booking_number || item.booking_reference || "").toLowerCase();
        const tTier = (item.ticket_type_name || item.ticket_tier_name || "").toLowerCase();
        if (
          !aName.includes(query) &&
          !aEmail.includes(query) &&
          !tCode.includes(query) &&
          !bRef.includes(query) &&
          !tTier.includes(query)
        ) {
          return false;
        }
      }

      return true;
    });
  }, [searchTerm, selectedEventId, selectedStatus]);

  // Overall Statistics calculated across all issued tickets (scoped to selected event if filtered)
  const stats = useMemo(() => {
    const scopedList =
      selectedEventId === "ALL"
        ? INITIAL_MOCK_ATTENDEES
        : INITIAL_MOCK_ATTENDEES.filter(
            (a) => (a.event?.id || a.event_id) === selectedEventId
          );

    const totalIssued = scopedList.length;
    const activeEligible = scopedList.filter(
      (a) => a.status === "VALID" || a.status === "USED"
    ).length;
    const checkedIn = scopedList.filter((a) => a.status === "USED").length;
    const pendingCheckin = scopedList.filter((a) => a.status === "VALID").length;
    const ratePercent = activeEligible > 0 ? ((checkedIn / activeEligible) * 100).toFixed(1) : "0.0";

    return {
      totalIssued,
      activeEligible,
      checkedIn,
      pendingCheckin,
      ratePercent,
    };
  }, [selectedEventId]);

  // Paginated Slices
  const totalItems = filteredAttendees.length;
  const totalPages = Math.ceil(totalItems / pageSize) || 1;
  const paginatedAttendees = useMemo(() => {
    const startIndex = (currentPage - 1) * pageSize;
    return filteredAttendees.slice(startIndex, startIndex + pageSize);
  }, [filteredAttendees, currentPage, pageSize]);

  // Auto-correct currentPage when filter causes totalPages to shrink
  useEffect(() => {
    if (currentPage > totalPages) {
      setCurrentPage(1);
    }
  }, [totalPages, currentPage]);

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <div className="flex items-center gap-2 mb-1">
            <h1 className="text-2xl font-bold text-[#141B2B] tracking-tight font-space">
              Attendee Management
            </h1>
            <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-semibold bg-[#ccfbf1] text-[#0f766e] border border-[#99f6e4]">
              Gate Admissions
            </span>
          </div>
          <p className="text-sm text-slate-500 font-sans">
            Monitor individual admission passes, check-in status, gate progress, and attendance rates across your events.
          </p>
        </div>

        {/* Global summary badge */}
        <div className="flex items-center gap-3">
          <div className="px-3.5 py-2 rounded-xl bg-slate-50 border border-slate-200 text-xs font-medium text-slate-600 flex items-center gap-2 shadow-sm">
            <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse"></span>
            <span>Gate Scanner Status: <strong>Live & Synchronized</strong></span>
          </div>
        </div>
      </div>

      {/* Summary KPI Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-4">
        {/* Card 1: Total Issued Tickets */}
        <div className="bg-white rounded-2xl p-5 border border-slate-200 shadow-sm transition-all hover:border-slate-300">
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs font-semibold uppercase tracking-wider text-slate-400">Total Issued</span>
            <div className="w-8 h-8 rounded-lg bg-blue-50 text-blue-600 flex items-center justify-center">
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 5v2m0 4v2m0 4v2M5 5a2 2 0 00-2 2v3a2 2 0 110 4v3a2 2 0 002 2h14a2 2 0 002-2v-3a2 2 0 110-4V7a2 2 0 00-2-2H5z" />
              </svg>
            </div>
          </div>
          <div className="text-2xl font-bold text-[#141B2B] font-space">{stats.totalIssued}</div>
          <p className="text-xs text-slate-500 mt-1">All tickets created</p>
        </div>

        {/* Card 2: Active Eligible Admissions */}
        <div className="bg-white rounded-2xl p-5 border border-slate-200 shadow-sm transition-all hover:border-slate-300">
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs font-semibold uppercase tracking-wider text-slate-400">Active Admissions</span>
            <div className="w-8 h-8 rounded-lg bg-[#ccfbf1] text-[#00685F] flex items-center justify-center">
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
              </svg>
            </div>
          </div>
          <div className="text-2xl font-bold text-[#00685F] font-space">{stats.activeEligible}</div>
          <p className="text-xs text-slate-500 mt-1">Valid + Checked-in passes</p>
        </div>

        {/* Card 3: Checked-In Attendees */}
        <div className="bg-white rounded-2xl p-5 border border-slate-200 shadow-sm transition-all hover:border-slate-300">
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs font-semibold uppercase tracking-wider text-slate-400">Checked In</span>
            <div className="w-8 h-8 rounded-lg bg-emerald-50 text-emerald-600 flex items-center justify-center">
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
              </svg>
            </div>
          </div>
          <div className="text-2xl font-bold text-emerald-600 font-space">{stats.checkedIn}</div>
          <p className="text-xs text-slate-500 mt-1">Admitted at gate</p>
        </div>

        {/* Card 4: Pending Check-Ins */}
        <div className="bg-white rounded-2xl p-5 border border-slate-200 shadow-sm transition-all hover:border-slate-300">
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs font-semibold uppercase tracking-wider text-slate-400">Pending Gate</span>
            <div className="w-8 h-8 rounded-lg bg-amber-50 text-amber-600 flex items-center justify-center">
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
            </div>
          </div>
          <div className="text-2xl font-bold text-amber-600 font-space">{stats.pendingCheckin}</div>
          <p className="text-xs text-slate-500 mt-1">Awaiting arrival</p>
        </div>

        {/* Card 5: Check-In Rate (%) with Progress Bar */}
        <div className="bg-white rounded-2xl p-5 border border-slate-200 shadow-sm transition-all hover:border-slate-300 flex flex-col justify-between">
          <div>
            <div className="flex items-center justify-between mb-2">
              <span className="text-xs font-semibold uppercase tracking-wider text-slate-400">Gate Progress</span>
              <span className="text-xs font-bold text-[#00685F]">{stats.ratePercent}%</span>
            </div>
            <div className="text-2xl font-bold text-[#141B2B] font-space">{stats.ratePercent}%</div>
          </div>
          <div className="mt-2">
            <div className="w-full bg-slate-100 rounded-full h-2 overflow-hidden">
              <div
                className="bg-[#00685F] h-2 rounded-full transition-all duration-500"
                style={{ width: `${Math.min(parseFloat(stats.ratePercent) || 0, 100)}%` }}
              ></div>
            </div>
            <p className="text-[11px] text-slate-400 mt-1.5 flex justify-between">
              <span>{stats.checkedIn} admitted</span>
              <span>{stats.activeEligible} total active</span>
            </p>
          </div>
        </div>
      </div>

      {/* Filter and Search Bar */}
      <div className="bg-white rounded-2xl p-4 sm:p-5 border border-slate-200 shadow-sm space-y-4">
        <div className="grid grid-cols-1 md:grid-cols-12 gap-3 items-center">
          {/* Search Input */}
          <div className="md:col-span-5 relative">
            <div className="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none text-slate-400">
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
              </svg>
            </div>
            <input
              type="text"
              value={searchTerm}
              onChange={(e) => {
                setSearchTerm(e.target.value);
                setCurrentPage(1);
              }}
              placeholder="Search attendee, email, ticket code, booking #..."
              className="w-full pl-10 pr-4 py-2.5 text-sm bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-[#00685F] focus:border-transparent transition-all"
            />
            {searchTerm && (
              <button
                onClick={() => setSearchTerm("")}
                className="absolute inset-y-0 right-0 pr-3 flex items-center text-slate-400 hover:text-slate-600"
              >
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            )}
          </div>

          {/* Event Filter */}
          <div className="md:col-span-4">
            <select
              value={selectedEventId}
              onChange={(e) => {
                setSelectedEventId(e.target.value);
                setCurrentPage(1);
              }}
              className="w-full px-3.5 py-2.5 text-sm bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-[#00685F] focus:border-transparent transition-all text-slate-700 font-medium"
            >
              <option value="ALL">All Events</option>
              {MOCK_EVENTS.map((ev) => (
                <option key={ev.id} value={ev.id}>
                  {ev.title}
                </option>
              ))}
            </select>
          </div>

          {/* Check-In Status Filter */}
          <div className="md:col-span-3">
            <select
              value={selectedStatus}
              onChange={(e) => {
                setSelectedStatus(e.target.value);
                setCurrentPage(1);
              }}
              className="w-full px-3.5 py-2.5 text-sm bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-[#00685F] focus:border-transparent transition-all text-slate-700 font-medium"
            >
              {CHECKIN_STATUS_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </select>
          </div>
        </div>

        {/* Filter Summary & Reset Action */}
        {hasActiveFilters && (
          <div className="pt-2 border-t border-slate-100 flex flex-wrap items-center justify-between gap-2">
            <div className="text-xs text-slate-500 flex items-center gap-2">
              <span>Showing filtered results:</span>
              <span className="font-semibold text-slate-700">{filteredAttendees.length} matches found</span>
            </div>
            <button
              onClick={handleResetFilters}
              className="inline-flex items-center gap-1.5 text-xs font-semibold text-[#00685F] hover:text-[#00524b] transition-colors"
            >
              <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
              </svg>
              Clear All Filters
            </button>
          </div>
        )}
      </div>

      {/* Attendees Table / States */}
      <div className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
        {/* Loading State Skeleton */}
        {loading && (
          <div className="p-8 space-y-4">
            {[1, 2, 3, 4, 5].map((idx) => (
              <div key={idx} className="h-14 bg-slate-100 animate-pulse rounded-xl"></div>
            ))}
          </div>
        )}

        {/* Error State */}
        {!loading && error && (
          <div className="p-12 text-center">
            <div className="w-12 h-12 rounded-full bg-red-100 text-red-600 flex items-center justify-center mx-auto mb-3">
              <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
              </svg>
            </div>
            <h3 className="text-base font-semibold text-slate-800 mb-1">Failed to Load Attendees</h3>
            <p className="text-sm text-slate-500 mb-4">{error}</p>
            <button
              onClick={() => {
                setError(null);
                setLoading(false);
              }}
              className="px-4 py-2 bg-[#00685F] text-white rounded-xl text-sm font-semibold hover:bg-[#00524b] transition-all"
            >
              Retry
            </button>
          </div>
        )}

        {/* Empty State */}
        {!loading && !error && filteredAttendees.length === 0 && (
          <div className="p-12 text-center">
            <div className="w-12 h-12 rounded-full bg-slate-100 text-slate-400 flex items-center justify-center mx-auto mb-3">
              <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
              </svg>
            </div>
            <h3 className="text-base font-semibold text-slate-800 mb-1">No Attendees Found</h3>
            <p className="text-sm text-slate-500 mb-4 max-w-sm mx-auto">
              {hasActiveFilters
                ? "No attendee passes match your selected filter criteria. Try adjusting your search or filters."
                : "No ticket admissions have been issued for your events yet."}
            </p>
            {hasActiveFilters && (
              <button
                onClick={handleResetFilters}
                className="px-4 py-2 bg-[#00685F] text-white rounded-xl text-sm font-semibold hover:bg-[#00524b] transition-all"
              >
                Reset Filters
              </button>
            )}
          </div>
        )}

        {/* Table View */}
        {!loading && !error && filteredAttendees.length > 0 && (
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse">
              <thead>
                <tr className="bg-slate-50 border-b border-slate-200 text-[11px] font-bold text-slate-500 uppercase tracking-wider font-space">
                  <th className="py-3.5 px-4">Attendee</th>
                  <th className="py-3.5 px-4">Ticket Code</th>
                  <th className="py-3.5 px-4">Event</th>
                  <th className="py-3.5 px-4">Ticket Type</th>
                  <th className="py-3.5 px-4">Seat</th>
                  <th className="py-3.5 px-4">Booking Ref</th>
                  <th className="py-3.5 px-4">Check-In Status</th>
                  <th className="py-3.5 px-4">Checked-In At</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 text-sm">
                {paginatedAttendees.map((attendee) => {
                  const statusStyle =
                    attendeeStatusBadgeStyles[attendee.status] ||
                    "bg-slate-100 text-slate-600 border border-slate-200";
                  const statusLabel = attendeeStatusLabels[attendee.status] || attendee.status;
                  const attendeeName = attendee.customer?.full_name || attendee.attendee_name || "—";
                  const attendeeEmail = attendee.customer?.email || attendee.attendee_email || "—";
                  const eventTitle = attendee.event?.title || attendee.event_title || "—";
                  const ticketTierName = attendee.ticket_type_name || attendee.ticket_tier_name || "Standard Pass";
                  const bookingRef = attendee.booking_number || attendee.booking_reference || "—";

                  return (
                    <tr
                      key={attendee.ticket_id}
                      className="hover:bg-slate-50/70 transition-colors"
                    >
                      {/* Attendee Name & Email */}
                      <td className="py-3.5 px-4">
                        <div className="font-semibold text-slate-900 font-sans">
                          {attendeeName}
                        </div>
                        <div className="text-xs text-slate-500 font-sans">
                          {attendeeEmail}
                        </div>
                      </td>

                      {/* Ticket Code with Copy Button */}
                      <td className="py-3.5 px-4">
                        <div className="flex items-center gap-1.5">
                          <span className="font-mono text-xs font-semibold text-slate-700 bg-slate-100 px-2 py-1 rounded-md border border-slate-200">
                            {attendee.ticket_code}
                          </span>
                          <button
                            onClick={() => handleCopyCode(attendee.ticket_code)}
                            title="Copy ticket code"
                            className="p-1 rounded hover:bg-slate-200 text-slate-400 hover:text-slate-600 transition-colors"
                          >
                            {copiedCode === attendee.ticket_code ? (
                              <svg className="w-3.5 h-3.5 text-emerald-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                              </svg>
                            ) : (
                              <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                              </svg>
                            )}
                          </button>
                        </div>
                      </td>

                      {/* Event Title */}
                      <td className="py-3.5 px-4 max-w-[220px]">
                        <div className="font-medium text-slate-900 truncate" title={eventTitle}>
                          {eventTitle}
                        </div>
                      </td>

                      {/* Ticket Tier */}
                      <td className="py-3.5 px-4">
                        <span className="inline-flex items-center px-2 py-0.5 rounded-md text-xs font-medium bg-slate-100 text-slate-700 border border-slate-200">
                          {ticketTierName}
                        </span>
                      </td>

                      {/* Seat */}
                      <td className="py-3.5 px-4 font-mono text-xs">
                        {attendee.seat_label ? (
                          <span className="font-semibold text-slate-800 bg-amber-50 text-amber-800 px-2 py-0.5 rounded border border-amber-200">
                            {attendee.seat_label}
                          </span>
                        ) : (
                          <span className="text-slate-400 font-sans italic">General Admission</span>
                        )}
                      </td>

                      {/* Booking Reference */}
                      <td className="py-3.5 px-4 font-mono text-xs font-medium text-slate-700">
                        {bookingRef}
                      </td>

                      {/* Check-In Status Badge */}
                      <td className="py-3.5 px-4">
                        <span
                          className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold ${statusStyle}`}
                        >
                          <span
                            className={`w-1.5 h-1.5 rounded-full ${
                              attendee.status === "USED"
                                ? "bg-emerald-600"
                                : attendee.status === "VALID"
                                ? "bg-amber-600"
                                : attendee.status === "CANCELLED"
                                ? "bg-red-600"
                                : "bg-purple-600"
                            }`}
                          ></span>
                          {statusLabel}
                        </span>
                      </td>

                      {/* Checked-In Timestamp */}
                      <td className="py-3.5 px-4 text-xs font-sans text-slate-600">
                        {attendee.used_at ? (
                          <div className="flex flex-col">
                            <span className="font-medium text-slate-900">
                              {formatDateTime(attendee.used_at)}
                            </span>
                          </div>
                        ) : (
                          <span className="text-slate-400 italic">Not Checked In</span>
                        )}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}

        {/* Pagination Bar */}
        {!loading && !error && filteredAttendees.length > 0 && (
          <div className="py-4 px-5 border-t border-slate-200 bg-white flex flex-col sm:flex-row items-center justify-between gap-4">
            {/* Range and Total */}
            <div className="text-xs text-slate-500">
              Showing <span className="font-semibold text-slate-700">{(currentPage - 1) * pageSize + 1}</span> to{" "}
              <span className="font-semibold text-slate-700">
                {Math.min(currentPage * pageSize, totalItems)}
              </span>{" "}
              of <span className="font-semibold text-slate-700">{totalItems}</span> attendees
            </div>

            {/* Page Size & Navigation */}
            <div className="flex items-center gap-4">
              {/* Page size dropdown */}
              <div className="flex items-center gap-2">
                <span className="text-xs text-slate-500">Rows per page:</span>
                <select
                  value={pageSize}
                  onChange={(e) => {
                    setPageSize(Number(e.target.value));
                    setCurrentPage(1);
                  }}
                  className="text-xs border border-slate-200 rounded-lg px-2.5 py-1.5 bg-slate-50 font-medium text-slate-700 focus:outline-none focus:ring-2 focus:ring-[#00685F]"
                >
                  {PAGE_SIZE_OPTIONS.map((size) => (
                    <option key={size} value={size}>
                      {size}
                    </option>
                  ))}
                </select>
              </div>

              {/* Prev / Next buttons */}
              <div className="flex items-center gap-1.5">
                <button
                  onClick={() => setCurrentPage((p) => Math.max(1, p - 1))}
                  disabled={currentPage <= 1}
                  className="px-3 py-1.5 rounded-lg border border-slate-200 text-xs font-semibold text-slate-600 hover:bg-slate-50 disabled:opacity-40 disabled:cursor-not-allowed transition-all"
                >
                  Previous
                </button>
                <div className="text-xs font-semibold text-slate-700 px-2 font-mono">
                  {currentPage} / {totalPages}
                </div>
                <button
                  onClick={() => setCurrentPage((p) => Math.min(totalPages, p + 1))}
                  disabled={currentPage >= totalPages}
                  className="px-3 py-1.5 rounded-lg border border-slate-200 text-xs font-semibold text-slate-600 hover:bg-slate-50 disabled:opacity-40 disabled:cursor-not-allowed transition-all"
                >
                  Next
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
