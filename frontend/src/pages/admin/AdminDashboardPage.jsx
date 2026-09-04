import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import RevenueChart from "../../components/admin/RevenueChart";

const metricCards = [
  {
    label: "Total Users",
    value: "12,480",
    change: "+12%",
    positive: true,
    icon: (
      <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
        <path
          d="M13 10C14.66 10 15.99 8.66 15.99 7C15.99 5.34 14.66 4 13 4C11.34 4 10 5.34 10 7C10 8.66 11.34 10 13 10ZM5 10C6.66 10 7.99 8.66 7.99 7C7.99 5.34 6.66 4 5 4C3.34 4 2 5.34 2 7C2 8.66 3.34 10 5 10ZM5 12C3.34 12 0 12.84 0 14.5V16H10V14.5C10 12.84 6.66 12 5 12ZM13 12C12.79 12 12.55 12.01 12.3 12.03C13.19 12.68 13.83 13.55 13.83 14.5V16H18V14.5C18 12.84 14.66 12 13 12Z"
          fill="#5c6bc0"
        />
      </svg>
    ),
    bg: "bg-[#e8eaf6]",
  },
  {
    label: "Total Organizers",
    value: "428",
    change: "+5%",
    positive: true,
    icon: (
      <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
        <path
          d="M10 2C7.79 2 6 3.79 6 6C6 8.21 7.79 10 10 10C12.21 10 14 8.21 14 6C14 3.79 12.21 2 10 2ZM10 12C7.33 12 2 13.34 2 16V18H18V16C18 13.34 12.67 12 10 12Z"
          fill="#5c6bc0"
        />
      </svg>
    ),
    bg: "bg-[#e8eaf6]",
  },
  {
    label: "Total Events",
    value: "1,256",
    change: "+18%",
    positive: true,
    icon: (
      <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
        <path
          d="M17 2H16V0H14V2H6V0H4V2H3C1.9 2 1 2.9 1 4V18C1 19.1 1.9 20 3 20H17C18.1 20 19 19.1 19 18V4C19 2.9 18.1 2 17 2ZM17 18H3V7H17V18ZM3 5H17V7H3V5Z"
          fill="#5c6bc0"
        />
      </svg>
    ),
    bg: "bg-[#e8eaf6]",
  },
  {
    label: "Active Events",
    value: "184",
    change: "–0%",
    positive: true,
    icon: (
      <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
        <circle cx="10" cy="10" r="8" fill="#00685f" />
        <path
          d="M8 13.5L5 10.5L6.41 9.09L8 10.67L13.59 5.08L15 6.5L8 13.5Z"
          fill="white"
        />
      </svg>
    ),
    bg: "bg-[#dcfce7]",
  },
  {
    label: "Total Bookings",
    value: "8,942",
    change: "+22%",
    positive: true,
    icon: (
      <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
        <path
          d="M18 2H2C1.1 2 0 2.9 0 4V14C0 15.1 1.1 16 2 16H18C18.9 16 20 15.1 20 14V4C20 2.9 18.9 2 18 2ZM18 14H2V8H18V14ZM18 6H2V4H18V6Z"
          fill="#5c6bc0"
        />
      </svg>
    ),
    bg: "bg-[#e8eaf6]",
  },
  {
    label: "Tickets Sold",
    value: "18,650",
    change: "+15%",
    positive: true,
    icon: (
      <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
        <path
          d="M20 10C20 8.9 19.1 8 18 8V6C18 4.9 17.1 4 16 4H4C2.9 4 2 4.9 2 6V8C0.9 8 0 8.9 0 10C0 11.1 0.9 12 2 12V14C2 15.1 2.9 16 4 16H16C17.1 16 18 15.1 18 14V12C19.1 12 20 11.1 20 10Z"
          fill="#5c6bc0"
        />
      </svg>
    ),
    bg: "bg-[#e8eaf6]",
  },
  {
    label: "Total Revenue",
    value: "₹42.8L",
    change: "+8%",
    positive: true,
    icon: (
      <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
        <path
          d="M10 0C4.48 0 0 4.48 0 10C0 15.52 4.48 20 10 20C15.52 20 20 15.52 20 10C20 4.48 15.52 0 10 0ZM11.5 14.5V16H8.5V14.5C7.12 14.5 6 13.38 6 12H8C8 12.55 8.45 13 9 13H11C11.55 13 12 12.55 12 12C12 11.45 11.55 11 11 11H9C7.35 11 6 9.65 6 8C6 6.62 7.12 5.5 8.5 5.5V4H11.5V5.5C12.88 5.5 14 6.62 14 8H12C12 7.45 11.55 7 11 7H9C8.45 7 8 7.45 8 8C8 8.55 8.45 9 9 9H11C12.65 9 14 10.35 14 12C14 13.38 12.88 14.5 11.5 14.5Z"
          fill="#00685f"
        />
      </svg>
    ),
    bg: "bg-[#dcfce7]",
  },
  {
    label: "Cancelled Events",
    value: "32",
    change: "–2%",
    positive: false,
    icon: (
      <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
        <circle cx="10" cy="10" r="8" fill="#ffdad6" />
        <path
          d="M13.5 7.41L12.09 6L10 8.09L7.91 6L6.5 7.41L8.59 9.5L6.5 11.59L7.91 13L10 10.91L12.09 13L13.5 11.59L11.41 9.5L13.5 7.41Z"
          fill="#93000a"
        />
      </svg>
    ),
    bg: "bg-[#ffdad6]",
  },
];

const pendingApplications = [
  { name: "Event Masters Pvt Ltd", date: "Applied: 10 Aug" },
  { name: "Tech Events India", date: "Applied: 9 Aug" },
  { name: "Creative Hub", date: "Applied: 9 Aug" },
];

const eventBreakdown = [
  { label: "Published", count: "1,020", color: "bg-[#00685f]" },
  { label: "Draft", count: "156", color: "bg-[#bcc9c6]" },
  { label: "Ongoing", count: "42", color: "bg-[#5c6bc0]" },
  { label: "Completed", count: "890", color: "bg-[#00685f]/60" },
];

const recentBookings = [
  {
    id: "#BK-9284",
    customer: "Rahul Kumar",
    event: "React Dev Summit '26",
    amount: "₹1,499",
    status: "PAID",
    date: "Today, 10:42 AM",
  },
  {
    id: "#BK-9283",
    customer: "Anjali S.",
    event: "Design System Workshop",
    amount: "₹899",
    status: "PAID",
    date: "Today, 09:15 AM",
  },
  {
    id: "#BK-9282",
    customer: "Vikram R.",
    event: "Cloud Native Con",
    amount: "₹2,000",
    status: "PENDING",
    date: "Yesterday",
  },
];

export default function AdminDashboardPage() {
  const navigate = useNavigate();
  const [chartMode, setChartMode] = useState("monthly");

  return (
    <div className="flex-1 flex flex-col overflow-hidden bg-white w-full">
      {/* Page Header */}
      <div className="px-6 py-5 border-b border-[#dce2f7] shrink-0">
        <h1 className="text-[#141b2b] text-2xl font-bold leading-8 tracking-tight">
          Admin Dashboard
        </h1>
        <p className="text-[#3d4947] text-sm mt-0.5">
          Monitor and manage the EventHub platform.
        </p>
      </div>

      {/* Main Scrollable View */}
      <div className="flex-1 overflow-y-auto px-6 py-6 space-y-6">
        {/* Metric Cards: 4 x 2 Grid */}
        <section>
          <h2 className="text-[#3d4947] text-xs font-semibold uppercase tracking-wider mb-3">
            Platform Overview
          </h2>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
            {metricCards.map((card, i) => (
              <div
                key={i}
                className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-[0px_1px_1.5px_rgba(0,0,0,0.1)] hover:border-[#00685f]/40 transition-colors"
              >
                <div className="flex items-center justify-between mb-3">
                  <span className="text-[#3d4947] text-sm font-medium">
                    {card.label}
                  </span>
                  <span
                    className={`${card.bg} flex items-center justify-center rounded-lg w-8 h-8`}
                  >
                    {card.icon}
                  </span>
                </div>
                <div className="text-[#141b2b] text-2xl font-bold leading-8 tracking-tight">
                  {card.value}
                </div>
                <div
                  className={`text-xs mt-1 font-medium ${
                    card.positive ? "text-[#00685f]" : "text-[#93000a]"
                  }`}
                >
                  {card.change} from last month
                </div>
              </div>
            ))}
          </div>
        </section>

        {/* Two Columns: Pending Applications + Revenue Area Chart */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          {/* Pending Applications Card */}
          <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-5 shadow-[0px_1px_1.5px_rgba(0,0,0,0.1)] flex flex-col justify-between">
            <div>
              <div className="flex items-center justify-between mb-4">
                <div className="flex items-center gap-2">
                  <h2 className="text-[#141b2b] text-base font-semibold">
                    Pending Applications
                  </h2>
                  <span className="bg-[#fef3c7] text-[#92400e] text-xs font-bold px-2 py-0.5 rounded-full">
                    24
                  </span>
                </div>
                <Link
                  to="/admin/applications"
                  className="text-[#00685f] text-xs font-semibold hover:underline"
                >
                  View All
                </Link>
              </div>

              <div className="space-y-2.5">
                {pendingApplications.map((app, i) => (
                  <div
                    key={i}
                    onClick={() => navigate("/admin/applications")}
                    className="flex items-center justify-between bg-white border border-[#dce2f7] rounded-lg px-4 py-3 hover:bg-[#f1f3ff] transition-colors cursor-pointer"
                  >
                    <div className="flex items-center gap-3">
                      <div className="w-8 h-8 bg-[#e8eaf6] rounded-full flex items-center justify-center text-[#5c6bc0] text-xs font-bold">
                        {app.name[0]}
                      </div>
                      <div>
                        <div className="text-[#141b2b] text-sm font-medium">
                          {app.name}
                        </div>
                        <div className="text-[#6d7a77] text-xs">{app.date}</div>
                      </div>
                    </div>
                    <svg width="6" height="10" viewBox="0 0 6 10" fill="none">
                      <path
                        d="M1 1L5 5L1 9"
                        stroke="#3d4947"
                        strokeWidth="1.5"
                        strokeLinecap="round"
                        strokeLinejoin="round"
                      />
                    </svg>
                  </div>
                ))}
              </div>
            </div>

            <button
              type="button"
              onClick={() => navigate("/admin/applications")}
              className="w-full mt-4 border border-[#bcc9c6] bg-white rounded-lg py-2 text-[#141b2b] text-sm font-semibold hover:bg-[#eef0fb] transition-colors cursor-pointer"
            >
              Review Applications
            </button>
          </div>

          {/* Platform Revenue Area Chart */}
          <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-5 shadow-[0px_1px_1.5px_rgba(0,0,0,0.1)] flex flex-col justify-between">
            <div className="flex items-center justify-between mb-2">
              <div>
                <h2 className="text-[#141b2b] text-base font-semibold">
                  Platform Revenue
                </h2>
                <p className="text-[#6d7a77] text-xs">Jan – Aug 2026</p>
              </div>
              <div className="flex gap-1 bg-white p-0.5 border border-[#bcc9c6] rounded-lg">
                <button
                  type="button"
                  onClick={() => setChartMode("monthly")}
                  className={`text-xs font-semibold px-3 py-1 rounded-md transition-colors cursor-pointer ${
                    chartMode === "monthly"
                      ? "bg-[#00685f] text-white"
                      : "text-[#3d4947] hover:text-[#141b2b]"
                  }`}
                >
                  Monthly
                </button>
                <button
                  type="button"
                  onClick={() => setChartMode("quarterly")}
                  className={`text-xs font-semibold px-3 py-1 rounded-md transition-colors cursor-pointer ${
                    chartMode === "quarterly"
                      ? "bg-[#00685f] text-white"
                      : "text-[#3d4947] hover:text-[#141b2b]"
                  }`}
                >
                  Quarterly
                </button>
              </div>
            </div>

            <div className="w-full mt-2">
              <RevenueChart />
            </div>
          </div>
        </div>

        {/* Two Columns: Event Breakdown + Recent Bookings */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          {/* Event Breakdown */}
          <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-5 shadow-[0px_1px_1.5px_rgba(0,0,0,0.1)]">
            <h2 className="text-[#141b2b] text-base font-semibold mb-4">
              Event Breakdown
            </h2>
            <div className="space-y-3">
              {eventBreakdown.map((item, i) => (
                <div
                  key={i}
                  className="flex items-center justify-between py-1 border-b border-[rgba(188,201,198,0.2)] last:border-0"
                >
                  <div className="flex items-center gap-2.5">
                    <span className={`w-2.5 h-2.5 rounded-full ${item.color}`} />
                    <span className="text-[#3d4947] text-sm font-medium">
                      {item.label}
                    </span>
                  </div>
                  <span className="text-[#141b2b] text-sm font-bold">
                    {item.count}
                  </span>
                </div>
              ))}
            </div>
          </div>

          {/* Recent Bookings */}
          <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-5 shadow-[0px_1px_1.5px_rgba(0,0,0,0.1)]">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-[#141b2b] text-base font-semibold">
                Recent Bookings
              </h2>
              <button
                type="button"
                onClick={() => navigate("/admin/bookings")}
                className="text-[#00685f] text-xs font-semibold hover:underline cursor-pointer"
              >
                View All
              </button>
            </div>
            <div className="overflow-x-auto">
              <table className="w-full text-xs">
                <thead>
                  <tr className="text-[#6d7a77] uppercase tracking-wide border-b border-[rgba(188,201,198,0.4)]">
                    <th className="text-left pb-2 font-medium">Booking ID</th>
                    <th className="text-left pb-2 font-medium">Customer</th>
                    <th className="text-left pb-2 font-medium">Amount</th>
                    <th className="text-left pb-2 font-medium">Status</th>
                  </tr>
                </thead>
                <tbody>
                  {recentBookings.map((b, i) => (
                    <tr
                      key={i}
                      className="border-t border-[rgba(188,201,198,0.25)] hover:bg-white/60 transition-colors"
                    >
                      <td className="py-2.5 font-mono text-[#141b2b] font-medium">
                        {b.id}
                      </td>
                      <td className="py-2.5 text-[#3d4947] font-medium">
                        {b.customer}
                      </td>
                      <td className="py-2.5 text-[#141b2b] font-semibold">
                        {b.amount}
                      </td>
                      <td className="py-2.5">
                        <span
                          className={`px-2 py-0.5 rounded-full text-[10px] font-bold ${
                            b.status === "PAID"
                              ? "bg-[#dcfce7] text-[#166534]"
                              : "bg-[#fef3c7] text-[#92400e]"
                          }`}
                        >
                          {b.status}
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
