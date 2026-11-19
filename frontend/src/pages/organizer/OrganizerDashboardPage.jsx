import { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import {
  ORGANIZER_ROUTES,
  getEventDetailsRoute,
} from "../../constants/eventConstants";
import {
  getOrganizerPaymentOverview,
  getOrganizerPayments,
} from "../../api/organizerPaymentApi";
import { getOrganizerEvents } from "../../api/organizerApi";
import {
  formatCurrency,
  formatRelativeTime,
  formatDateTime,
} from "../../utils/formatters";
import imgBento1 from "../../assets/organizer/0b2568d2a1321299cd93ab73936efb2e8bc467fe.png";
import imgBento2 from "../../assets/organizer/19e133c323acab8e7f7eaa83c9df050818d67ee4.png";
import imgBento3 from "../../assets/organizer/4a3f88c97a88bb4cd6676d63cd40e72e066071b7.png";

const FALLBACK_IMAGES = [imgBento1, imgBento2, imgBento3];

function getStatusBadgeClass(status) {
  switch (status?.toUpperCase()) {
    case "COMPLETED":
    case "SUCCESS":
      return "bg-emerald-100 text-emerald-800";
    case "PENDING":
    case "PROCESSING":
      return "bg-amber-100 text-amber-800";
    case "FAILED":
      return "bg-red-100 text-red-800";
    default:
      return "bg-gray-100 text-gray-800";
  }
}

export default function OrganizerDashboardPage() {
  const navigate = useNavigate();

  const [overview, setOverview] = useState(null);
  const [payments, setPayments] = useState([]);
  const [events, setEvents] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    let cancelled = false;

    const timer = setTimeout(async () => {
      try {
        setLoading(true);
        setError(null);

        const [overviewRes, paymentsRes, eventsRes] = await Promise.allSettled([
          getOrganizerPaymentOverview(),
          getOrganizerPayments({ page: 1, limit: 5 }),
          getOrganizerEvents({ page: 1, limit: 3 }),
        ]);

        if (cancelled) return;

        if (overviewRes.status === "fulfilled") {
          setOverview(overviewRes.value);
        }
        if (paymentsRes.status === "fulfilled") {
          const paymentList =
            paymentsRes.value?.payments ??
            paymentsRes.value?.data?.payments ??
            [];
          setPayments(paymentList);
        }
        if (eventsRes.status === "fulfilled") {
          const eventList =
            eventsRes.value?.data?.events ??
            eventsRes.value?.events ??
            [];
          setEvents(eventList);
        }
      } catch (err) {
        if (!cancelled) {
          setError(
            err?.response?.data?.message ||
              err?.message ||
              "Failed to load dashboard data."
          );
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }, 0);

    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, []);

  const metrics = [
    {
      title: "Organizer Earnings",
      value: loading
        ? "—"
        : formatCurrency(
            overview?.organizer_earnings ??
              overview?.organizer_allocated_revenue ??
              0
          ),
      subtext: "Net proceeds after platform fee",
      icon: (
        <svg
          className="size-5 text-[#00685f]"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth="2"
            d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
          />
        </svg>
      ),
    },
    {
      title: "Gross Captured Revenue",
      value: loading
        ? "—"
        : formatCurrency(
            overview?.gross_captured_amount ??
              overview?.total_ticket_sales_amount ??
              0
          ),
      subtext: `Platform fee: ${
        loading ? "—" : formatCurrency(overview?.platform_fees ?? 0)
      }`,
      icon: (
        <svg
          className="size-5 text-[#4648d4]"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth="2"
            d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"
          />
        </svg>
      ),
    },
    {
      title: "Tickets Sold",
      value: loading
        ? "—"
        : Number(overview?.tickets_sold ?? 0).toLocaleString(),
      subtext: `${overview?.successful_payments_count ?? 0} paid bookings`,
      icon: (
        <svg
          className="size-5 text-[#00685f]"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth="2"
            d="M15 5v2m0 4v2m0 4v2M5 5a2 2 0 00-2 2v3a2 2 0 110 4v3a2 2 0 002 2h14a2 2 0 002-2v-3a2 2 0 110-4V7a2 2 0 00-2-2H5z"
          />
        </svg>
      ),
    },
    {
      title: "Total Transactions",
      value: loading
        ? "—"
        : String(
            overview?.total_payment_transactions_count ??
              overview?.successful_payments_count ??
              0
          ),
      subtext: `${overview?.active_events_count ?? 0} active events`,
      icon: (
        <svg
          className="size-5 text-[#141b2b]"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth="2"
            d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"
          />
        </svg>
      ),
    },
  ];

  return (
    <div className="p-4 sm:p-6 lg:p-8 max-w-7xl mx-auto flex flex-col gap-6 sm:gap-8">
      {/* Welcome Banner */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 bg-white border border-[#bcc9c6]/50 rounded-2xl p-4 sm:p-6 shadow-sm">
        <div className="flex flex-col gap-1">
          <h1 className="text-xl sm:text-2xl font-bold text-[#141b2b]">
            Welcome back, Event Masters 👋
          </h1>
          <p className="text-[14px] text-[#565e74]">
            Here is what is happening across your events and ticket sales today.
          </p>
        </div>
        <div className="flex items-center gap-3">
          <button
            type="button"
            onClick={() => navigate(ORGANIZER_ROUTES.PAYMENTS)}
            className="flex items-center justify-center gap-2 border border-[#00685f] text-[#00685f] hover:bg-[#00685f]/5 font-semibold text-[14px] px-4 py-2.5 rounded-xl shadow-xs transition-colors shrink-0 cursor-pointer"
          >
            <svg
              className="size-4"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth="2"
                d="M17 9V7a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2m2 4h10a2 2 0 002-2v-6a2 2 0 00-2-2H9a2 2 0 00-2 2v6a2 2 0 002 2zm7-5a2 2 0 11-4 0 2 2 0 014 0z"
              />
            </svg>
            <span>Payments Hub</span>
          </button>
          <button
            type="button"
            onClick={() => navigate(ORGANIZER_ROUTES.CREATE_STEP_1)}
            className="flex items-center justify-center gap-2 bg-[#00685f] hover:bg-[#005550] text-white font-semibold text-[14px] px-5 py-2.5 rounded-xl shadow-sm transition-colors shrink-0 cursor-pointer"
          >
            <svg
              className="size-4"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth="2.5"
                d="M12 4v16m8-8H4"
              />
            </svg>
            <span>Create New Event</span>
          </button>
        </div>
      </div>

      {/* Error Notice */}
      {error && (
        <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-xl text-sm flex items-center justify-between">
          <span>{error}</span>
          <button
            type="button"
            onClick={() => window.location.reload()}
            className="underline font-semibold hover:text-red-900"
          >
            Retry
          </button>
        </div>
      )}

      {/* 4-Column Metrics Grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-5">
        {metrics.map((m) => (
          <div
            key={m.title}
            className="bg-white border border-[#bcc9c6]/50 rounded-xl p-5 shadow-xs flex flex-col justify-between"
          >
            <div className="flex items-center justify-between mb-3">
              <span className="text-[13px] font-medium text-[#565e74]">
                {m.title}
              </span>
              <div className="p-2 bg-[#f9f9ff] rounded-lg border border-[#bcc9c6]/40">
                {m.icon}
              </div>
            </div>
            <div className="flex flex-col">
              <span className="text-2xl font-bold text-[#141b2b]">
                {m.value}
              </span>
              <div className="flex items-center gap-1.5 mt-1 text-[12px]">
                <span className="text-[#565e74]">{m.subtext}</span>
              </div>
            </div>
          </div>
        ))}
      </div>

      {/* Analytics & Performance Charts Summary */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Revenue Analytics Overview */}
        <div className="lg:col-span-2 bg-white border border-[#bcc9c6]/50 rounded-xl p-6 shadow-xs flex flex-col justify-between">
          <div className="flex items-center justify-between mb-4">
            <div className="flex flex-col">
              <h3 className="font-bold text-[16px] text-[#141b2b]">
                Revenue Performance
              </h3>
              <span className="text-xs text-[#565e74]">
                Aggregated financial performance across active events
              </span>
            </div>
            <button
              type="button"
              onClick={() => navigate(ORGANIZER_ROUTES.PAYMENTS)}
              className="text-xs font-semibold text-[#00685f] hover:underline"
            >
              Detailed Breakdown →
            </button>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 my-auto py-4">
            <div className="p-4 bg-[#f9f9ff] border border-[#bcc9c6]/30 rounded-lg">
              <span className="text-xs text-[#565e74] font-medium">
                Gross Captured
              </span>
              <p className="text-xl font-bold text-[#141b2b] mt-1">
                {loading
                  ? "—"
                  : formatCurrency(overview?.gross_captured_amount ?? 0)}
              </p>
              <span className="text-[11px] text-[#565e74] mt-0.5 block">
                Total bookings value
              </span>
            </div>
            <div className="p-4 bg-[#f9f9ff] border border-[#bcc9c6]/30 rounded-lg">
              <span className="text-xs text-[#565e74] font-medium">
                Platform Fees
              </span>
              <p className="text-xl font-bold text-[#ba1a1a] mt-1">
                {loading
                  ? "—"
                  : formatCurrency(overview?.platform_fees ?? 0)}
              </p>
              <span className="text-[11px] text-[#565e74] mt-0.5 block">
                Platform service charge
              </span>
            </div>
            <div className="p-4 bg-emerald-50/60 border border-emerald-200/50 rounded-lg">
              <span className="text-xs text-emerald-800 font-medium">
                Net Organizer Earnings
              </span>
              <p className="text-xl font-bold text-emerald-900 mt-1">
                {loading
                  ? "—"
                  : formatCurrency(
                      overview?.organizer_earnings ??
                        overview?.organizer_allocated_revenue ??
                        0
                    )}
              </p>
              <span className="text-[11px] text-emerald-700 mt-0.5 block">
                Allocated to organizer
              </span>
            </div>
          </div>

          <div className="mt-2 pt-3 border-t border-gray-100 flex items-center justify-between text-xs text-[#565e74]">
            <span>
              Real-time balance verified via EventHub Organizer Finance Ledger
            </span>
            <span className="italic text-[11px]">
              Note: Historical weekly timeseries aggregation is pending backend
              support
            </span>
          </div>
        </div>

        {/* Ticket Distribution Card */}
        <div className="bg-white border border-[#bcc9c6]/50 rounded-xl p-6 shadow-xs flex flex-col justify-between">
          <div className="flex flex-col mb-4">
            <h3 className="font-bold text-[16px] text-[#141b2b]">
              Ticket Volume
            </h3>
            <span className="text-xs text-[#565e74]">
              Total volume across all active events
            </span>
          </div>

          <div className="flex flex-col items-center justify-center my-4">
            <div className="size-36 rounded-full border-8 border-[#00685f] border-t-[#4648d4] border-r-[#e1e8fd] flex flex-col items-center justify-center">
              <span className="text-xl font-bold text-[#141b2b]">
                {loading
                  ? "—"
                  : Number(overview?.tickets_sold ?? 0).toLocaleString()}
              </span>
              <span className="text-[11px] text-[#565e74]">Tickets Sold</span>
            </div>
          </div>

          <div className="flex flex-col gap-2 text-xs">
            <div className="flex items-center justify-between">
              <span className="text-[#3d4947]">Paid Bookings</span>
              <strong className="text-[#141b2b]">
                {loading ? "—" : overview?.successful_payments_count ?? 0}
              </strong>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-[#3d4947]">Total Transactions</span>
              <strong className="text-[#141b2b]">
                {loading
                  ? "—"
                  : overview?.total_payment_transactions_count ?? 0}
              </strong>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-[#3d4947]">Active Events</span>
              <strong className="text-[#141b2b]">
                {loading ? "—" : overview?.active_events_count ?? 0}
              </strong>
            </div>
          </div>
        </div>
      </div>

      {/* Upcoming Events Section */}
      <div className="flex flex-col gap-4">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-bold text-[#141b2b]">Upcoming Events</h2>
          <button
            type="button"
            onClick={() => navigate(ORGANIZER_ROUTES.MY_EVENTS)}
            className="text-[13px] font-semibold text-[#00685f] hover:underline"
          >
            View all events →
          </button>
        </div>

        {events.length > 0 ? (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4 sm:gap-6">
            {events.slice(0, 3).map((ev, idx) => {
              const capacity = Number(ev?.TicketCapacity ?? 0);
              const sold = Number(ev?.TicketsSold ?? 0);
              const pct =
                capacity > 0 ? Math.round((sold / capacity) * 100) : 0;
              const eventId = ev?.ID ?? ev?.id;
              const image =
                ev?.CoverImageUrl ||
                FALLBACK_IMAGES[idx % FALLBACK_IMAGES.length];
              const dateStr = ev?.StartDate
                ? formatDateTime(ev.StartDate)
                : "Date TBD";
              const venueStr =
                ev?.VenueName ||
                (ev?.EventType === "ONLINE"
                  ? "Online Event"
                  : "Venue TBD");

              return (
                <div
                  key={eventId || idx}
                  onClick={() =>
                    eventId && navigate(getEventDetailsRoute(eventId))
                  }
                  className="bg-white border border-[#bcc9c6]/50 rounded-xl overflow-hidden shadow-xs hover:shadow-md transition-shadow flex flex-col cursor-pointer group"
                >
                  <div className="h-44 bg-gray-100 overflow-hidden relative">
                    <img
                      src={image}
                      alt={ev?.Title || "Event"}
                      className="size-full object-cover group-hover:scale-105 transition-transform duration-300"
                    />
                    <span
                      className={`absolute top-3 right-3 text-[11px] font-bold px-2.5 py-1 rounded-full uppercase tracking-wider ${
                        ev?.Status === "PUBLISHED"
                          ? "bg-emerald-100 text-emerald-800"
                          : "bg-amber-100 text-amber-800"
                      }`}
                    >
                      {ev?.Status || "DRAFT"}
                    </span>
                  </div>

                  <div className="p-5 flex-1 flex flex-col justify-between gap-4">
                    <div className="flex flex-col gap-1">
                      <h3 className="font-bold text-[15px] text-[#141b2b] line-clamp-1 group-hover:text-[#00685f] transition-colors">
                        {ev?.Title || "Untitled Event"}
                      </h3>
                      <p className="text-xs text-[#565e74] flex items-center gap-1.5">
                        <svg
                          className="size-3.5 shrink-0"
                          fill="none"
                          stroke="currentColor"
                          viewBox="0 0 24 24"
                        >
                          <path
                            strokeLinecap="round"
                            strokeLinejoin="round"
                            strokeWidth="2"
                            d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"
                          />
                        </svg>
                        {dateStr}
                      </p>
                      <p className="text-xs text-[#565e74] flex items-center gap-1.5 truncate">
                        <svg
                          className="size-3.5 shrink-0"
                          fill="none"
                          stroke="currentColor"
                          viewBox="0 0 24 24"
                        >
                          <path
                            strokeLinecap="round"
                            strokeLinejoin="round"
                            strokeWidth="2"
                            d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z"
                          />
                        </svg>
                        {venueStr}
                      </p>
                    </div>

                    <div className="flex flex-col gap-1.5 pt-2 border-t border-gray-100">
                      <div className="flex justify-between text-xs">
                        <span className="text-[#565e74]">
                          <strong>{sold}</strong>
                          {capacity > 0 ? ` / ${capacity}` : ""}{" "}
                          Tickets Sold
                        </span>
                        {capacity > 0 && (
                          <strong className="text-[#00685f]">{pct}%</strong>
                        )}
                      </div>
                      {capacity > 0 && (
                        <div className="w-full bg-gray-100 h-2 rounded-full overflow-hidden">
                          <div
                            className="bg-[#00685f] h-full rounded-full transition-all"
                            style={{ width: `${Math.min(pct, 100)}%` }}
                          />
                        </div>
                      )}
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        ) : (
          <div className="bg-white border border-[#bcc9c6]/50 rounded-xl p-8 text-center flex flex-col items-center justify-center gap-3">
            <p className="text-sm text-[#565e74]">
              {loading
                ? "Loading upcoming events..."
                : "No upcoming events found. Create your first event to start accepting ticket bookings."}
            </p>
            {!loading && (
              <button
                type="button"
                onClick={() => navigate(ORGANIZER_ROUTES.CREATE_STEP_1)}
                className="text-xs font-semibold text-white bg-[#00685f] hover:bg-[#005550] px-4 py-2 rounded-lg transition-colors cursor-pointer"
              >
                Create Event
              </button>
            )}
          </div>
        )}
      </div>

      {/* Recent Orders Table */}
      <div className="bg-white border border-[#bcc9c6]/50 rounded-xl shadow-xs overflow-hidden">
        <div className="p-5 border-b border-gray-100 flex items-center justify-between">
          <div className="flex flex-col">
            <h3 className="font-bold text-[16px] text-[#141b2b]">
              Recent Ticket Orders
            </h3>
            <span className="text-xs text-[#565e74]">
              Latest bookings across your active events
            </span>
          </div>
          <button
            type="button"
            onClick={() => navigate(ORGANIZER_ROUTES.PAYMENTS)}
            className="text-xs font-semibold text-[#00685f] hover:underline"
          >
            View all transactions →
          </button>
        </div>

        <div className="overflow-x-auto">
          {payments.length > 0 ? (
            <table className="w-full text-left text-xs text-[#141b2b]">
              <thead className="bg-[#f9f9ff] text-[#565e74] uppercase text-[11px] font-semibold border-b border-[#bcc9c6]/30">
                <tr>
                  <th className="px-5 py-3">Order / Reference</th>
                  <th className="px-5 py-3">Event</th>
                  <th className="px-5 py-3">Customer</th>
                  <th className="px-5 py-3">Tickets</th>
                  <th className="px-5 py-3">Amount</th>
                  <th className="px-5 py-3">Status</th>
                  <th className="px-5 py-3">Time</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {payments.map((p) => {
                  const orderRef =
                    p.booking_reference ||
                    (p.booking_id ? p.booking_id.slice(0, 8) : null) ||
                    (p.payment_id ? p.payment_id.slice(0, 8) : "—");
                  const qty = p.ticket_quantity || 1;

                  return (
                    <tr
                      key={p.payment_id || orderRef}
                      className="hover:bg-gray-50/70 transition-colors"
                    >
                      <td className="px-5 py-3.5 font-mono font-semibold text-[#00685f]">
                        {orderRef}
                      </td>
                      <td className="px-5 py-3.5 font-medium">
                        {p.event_title || "—"}
                      </td>
                      <td className="px-5 py-3.5">
                        <div className="flex flex-col">
                          <span className="font-semibold">
                            {p.customer_name || "Guest Attendee"}
                          </span>
                          <span className="text-[#565e74] text-[11px]">
                            {p.customer_email || "—"}
                          </span>
                        </div>
                      </td>
                      <td className="px-5 py-3.5 text-[#565e74]">
                        {qty}x Ticket{qty > 1 ? "s" : ""}
                      </td>
                      <td className="px-5 py-3.5 font-semibold text-[#141b2b]">
                        {formatCurrency(p.amount)}
                      </td>
                      <td className="px-5 py-3.5">
                        <span
                          className={`inline-flex items-center px-2 py-0.5 rounded text-[10px] font-bold ${getStatusBadgeClass(
                            p.payment_status
                          )}`}
                        >
                          {p.payment_status || "COMPLETED"}
                        </span>
                      </td>
                      <td className="px-5 py-3.5 text-[#565e74]">
                        {formatRelativeTime(p.created_at)}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          ) : (
            <div className="py-12 px-4 text-center flex flex-col items-center justify-center gap-2">
              <svg
                className="size-8 text-[#565e74]/40"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth="1.5"
                  d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"
                />
              </svg>
              <p className="text-sm font-medium text-[#141b2b]">
                {loading
                  ? "Loading recent orders..."
                  : "No payment transactions recorded yet"}
              </p>
              <p className="text-xs text-[#565e74] max-w-sm">
                When attendees purchase tickets for your events, transactions
                will appear here in real time.
              </p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}