import { useNavigate } from "react-router-dom";
import { ORGANIZER_ROUTES } from "../../constants/eventConstants";
import imgBento1 from "../../assets/organizer/0b2568d2a1321299cd93ab73936efb2e8bc467fe.png";
import imgBento2 from "../../assets/organizer/19e133c323acab8e7f7eaa83c9df050818d67ee4.png";
import imgBento3 from "../../assets/organizer/4a3f88c97a88bb4cd6676d63cd40e72e066071b7.png";

export default function OrganizerDashboardPage() {
  const navigate = useNavigate();

  const metrics = [
    {
      title: "Total Revenue",
      value: "$42,850.00",
      change: "+12.5%",
      isPositive: true,
      subtext: "vs last month",
      icon: (
        <svg className="size-5 text-[#00685f]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
      ),
    },
    {
      title: "Tickets Sold",
      value: "1,420",
      change: "+8.2%",
      isPositive: true,
      subtext: "vs last month",
      icon: (
        <svg className="size-5 text-[#4648d4]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M15 5v2m0 4v2m0 4v2M5 5a2 2 0 00-2 2v3a2 2 0 110 4v3a2 2 0 002 2h14a2 2 0 002-2v-3a2 2 0 110-4V7a2 2 0 00-2-2H5z" />
        </svg>
      ),
    },
    {
      title: "Page Views",
      value: "28,450",
      change: "+24.1%",
      isPositive: true,
      subtext: "across all events",
      icon: (
        <svg className="size-5 text-[#141b2b]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
        </svg>
      ),
    },
    {
      title: "Active Events",
      value: "4",
      change: "2 Drafts",
      isPositive: null,
      subtext: "currently listed",
      icon: (
        <svg className="size-5 text-[#00685f]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
        </svg>
      ),
    },
  ];

  const upcomingEvents = [
    {
      id: "ev-1",
      title: "Sunfield Music Festival 2026",
      date: "Oct 15 - 17, 2026",
      venue: "Grand Central Arena, Austin, TX",
      sold: 1200,
      capacity: 1500,
      image: imgBento1,
      status: "PUBLISHED",
    },
    {
      id: "ev-2",
      title: "Global Tech Innovators Summit",
      date: "Nov 02, 2026",
      venue: "Convention Center Hall A, San Francisco, CA",
      sold: 840,
      capacity: 1000,
      image: imgBento2,
      status: "PUBLISHED",
    },
    {
      id: "ev-3",
      title: "Artisan Culinary & Wine Expo",
      date: "Dec 10, 2026",
      venue: "Metropolitan Pavilion, New York, NY",
      sold: 310,
      capacity: 500,
      image: imgBento3,
      status: "DRAFT",
    },
  ];

  const recentOrders = [
    {
      id: "ORD-94821",
      event: "Sunfield Music Festival",
      customer: "Sarah Jenkins",
      email: "sarah.j@example.com",
      tickets: "2x VIP",
      total: "$298.00",
      status: "PAID",
      date: "12 mins ago",
    },
    {
      id: "ORD-94820",
      event: "Sunfield Music Festival",
      customer: "Michael Chen",
      email: "m.chen@example.com",
      tickets: "4x General Admission",
      total: "$316.00",
      status: "PAID",
      date: "35 mins ago",
    },
    {
      id: "ORD-94819",
      event: "Global Tech Summit",
      customer: "Elena Rostova",
      email: "elena@techcorp.io",
      tickets: "1x Pass",
      total: "$450.00",
      status: "PAID",
      date: "2 hours ago",
    },
    {
      id: "ORD-94818",
      event: "Artisan Culinary Expo",
      customer: "David Kim",
      email: "dkim@gourmet.net",
      tickets: "2x General Admission",
      total: "$120.00",
      status: "PAID",
      date: "4 hours ago",
    },
  ];

  return (
    <div className="p-8 max-w-7xl mx-auto flex flex-col gap-8">
      {/* Welcome Banner */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 bg-white border border-[#bcc9c6]/50 rounded-2xl p-6 shadow-sm">
        <div className="flex flex-col gap-1">
          <h1 className="text-2xl font-bold text-[#141b2b]">
            Welcome back, Event Masters 👋
          </h1>
          <p className="text-[14px] text-[#565e74]">
            Here is what is happening across your events and ticket sales today.
          </p>
        </div>
        <button
          type="button"
          onClick={() => navigate(ORGANIZER_ROUTES.CREATE_STEP_1)}
          className="flex items-center justify-center gap-2 bg-[#00685f] hover:bg-[#005550] text-white font-semibold text-[14px] px-5 py-2.5 rounded-xl shadow-sm transition-colors shrink-0 cursor-pointer"
        >
          <svg className="size-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2.5" d="M12 4v16m8-8H4" />
          </svg>
          <span>Create New Event</span>
        </button>
      </div>

      {/* 4-Column Metrics Grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-5">
        {metrics.map((m) => (
          <div
            key={m.title}
            className="bg-white border border-[#bcc9c6]/50 rounded-xl p-5 shadow-xs flex flex-col justify-between"
          >
            <div className="flex items-center justify-between mb-3">
              <span className="text-[13px] font-medium text-[#565e74]">{m.title}</span>
              <div className="p-2 bg-[#f9f9ff] rounded-lg border border-[#bcc9c6]/40">
                {m.icon}
              </div>
            </div>
            <div className="flex flex-col">
              <span className="text-2xl font-bold text-[#141b2b]">{m.value}</span>
              <div className="flex items-center gap-1.5 mt-1 text-[12px]">
                {m.isPositive !== null && (
                  <span
                    className={`font-semibold ${
                      m.isPositive ? "text-emerald-700" : "text-red-600"
                    }`}
                  >
                    {m.change}
                  </span>
                )}
                {m.isPositive === null && (
                  <span className="font-semibold text-blue-700">{m.change}</span>
                )}
                <span className="text-[#565e74]">{m.subtext}</span>
              </div>
            </div>
          </div>
        ))}
      </div>

      {/* Analytics & Performance Charts Mock */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Revenue Chart */}
        <div className="lg:col-span-2 bg-white border border-[#bcc9c6]/50 rounded-xl p-6 shadow-xs flex flex-col justify-between">
          <div className="flex items-center justify-between mb-4">
            <div className="flex flex-col">
              <h3 className="font-bold text-[16px] text-[#141b2b]">Revenue Analytics</h3>
              <span className="text-xs text-[#565e74]">Weekly breakdown of ticket proceeds</span>
            </div>
            <div className="flex items-center gap-1 bg-[#f9f9ff] border border-[#bcc9c6]/50 rounded-lg p-1 text-xs">
              {["7D", "30D", "90D", "1Y"].map((period, i) => (
                <button
                  key={period}
                  type="button"
                  className={`px-2.5 py-1 rounded font-medium ${
                    i === 1 ? "bg-[#00685f] text-white" : "text-[#565e74] hover:bg-gray-200/50"
                  }`}
                >
                  {period}
                </button>
              ))}
            </div>
          </div>

          {/* Graphical Bar Preview */}
          <div className="h-56 flex items-end justify-between gap-3 pt-6 pb-2 px-2">
            {[45, 62, 58, 85, 92, 78, 95, 68, 82, 105, 90, 115].map((val, idx) => (
              <div key={idx} className="flex-1 flex flex-col items-center gap-2 group">
                <div
                  className="w-full bg-[#00685f]/80 group-hover:bg-[#00685f] rounded-t transition-all"
                  style={{ height: `${val * 1.5}px` }}
                />
                <span className="text-[10px] text-[#565e74]">W{idx + 1}</span>
              </div>
            ))}
          </div>
        </div>

        {/* Ticket Breakdown Card */}
        <div className="bg-white border border-[#bcc9c6]/50 rounded-xl p-6 shadow-xs flex flex-col justify-between">
          <div className="flex flex-col mb-4">
            <h3 className="font-bold text-[16px] text-[#141b2b]">Ticket Distribution</h3>
            <span className="text-xs text-[#565e74]">Share by event category</span>
          </div>

          <div className="flex flex-col items-center justify-center my-4">
            <div className="size-36 rounded-full border-8 border-[#00685f] border-t-[#4648d4] border-r-[#e1e8fd] flex flex-col items-center justify-center">
              <span className="text-xl font-bold text-[#141b2b]">1,420</span>
              <span className="text-[11px] text-[#565e74]">Tickets</span>
            </div>
          </div>

          <div className="flex flex-col gap-2 text-xs">
            <div className="flex items-center justify-between">
              <span className="flex items-center gap-2 text-[#3d4947]">
                <span className="size-2.5 rounded-full bg-[#00685f]" /> Music Festivals
              </span>
              <strong className="text-[#141b2b]">58%</strong>
            </div>
            <div className="flex items-center justify-between">
              <span className="flex items-center gap-2 text-[#3d4947]">
                <span className="size-2.5 rounded-full bg-[#4648d4]" /> Tech Conferences
              </span>
              <strong className="text-[#141b2b]">26%</strong>
            </div>
            <div className="flex items-center justify-between">
              <span className="flex items-center gap-2 text-[#3d4947]">
                <span className="size-2.5 rounded-full bg-[#e1e8fd]" /> Culinary & Arts
              </span>
              <strong className="text-[#141b2b]">16%</strong>
            </div>
          </div>
        </div>
      </div>

      {/* Upcoming Events Bento Section */}
      <div className="flex flex-col gap-4">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-bold text-[#141b2b]">Upcoming Events</h2>
          <button
            type="button"
            onClick={() => navigate(ORGANIZER_ROUTES.CREATE_STEP_1)}
            className="text-[13px] font-semibold text-[#00685f] hover:underline"
          >
            View all events →
          </button>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          {upcomingEvents.map((ev) => {
            const pct = Math.round((ev.sold / ev.capacity) * 100);
            return (
              <div
                key={ev.id}
                className="bg-white border border-[#bcc9c6]/50 rounded-xl overflow-hidden shadow-xs hover:shadow-md transition-shadow flex flex-col"
              >
                <div className="h-44 bg-gray-100 overflow-hidden relative">
                  <img
                    src={ev.image}
                    alt={ev.title}
                    className="size-full object-cover"
                  />
                  <span
                    className={`absolute top-3 right-3 text-[11px] font-bold px-2.5 py-1 rounded-full uppercase tracking-wider ${
                      ev.status === "PUBLISHED"
                        ? "bg-emerald-100 text-emerald-800"
                        : "bg-amber-100 text-amber-800"
                    }`}
                  >
                    {ev.status}
                  </span>
                </div>

                <div className="p-5 flex-1 flex flex-col justify-between gap-4">
                  <div className="flex flex-col gap-1">
                    <h3 className="font-bold text-[15px] text-[#141b2b] line-clamp-1">
                      {ev.title}
                    </h3>
                    <p className="text-xs text-[#565e74] flex items-center gap-1.5">
                      <svg className="size-3.5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
                      </svg>
                      {ev.date}
                    </p>
                    <p className="text-xs text-[#565e74] flex items-center gap-1.5 truncate">
                      <svg className="size-3.5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" />
                      </svg>
                      {ev.venue}
                    </p>
                  </div>

                  <div className="flex flex-col gap-1.5 pt-2 border-t border-gray-100">
                    <div className="flex justify-between text-xs">
                      <span className="text-[#565e74]">
                        <strong>{ev.sold}</strong> / {ev.capacity} Tickets Sold
                      </span>
                      <strong className="text-[#00685f]">{pct}%</strong>
                    </div>
                    <div className="w-full bg-gray-100 h-2 rounded-full overflow-hidden">
                      <div
                        className="bg-[#00685f] h-full rounded-full transition-all"
                        style={{ width: `${pct}%` }}
                      />
                    </div>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      </div>

      {/* Recent Orders Table */}
      <div className="bg-white border border-[#bcc9c6]/50 rounded-xl shadow-xs overflow-hidden">
        <div className="p-5 border-b border-gray-100 flex items-center justify-between">
          <div className="flex flex-col">
            <h3 className="font-bold text-[16px] text-[#141b2b]">Recent Ticket Orders</h3>
            <span className="text-xs text-[#565e74]">Latest bookings across your active events</span>
          </div>
          <span className="text-xs text-[#565e74]">Live order feed</span>
        </div>

        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs text-[#141b2b]">
            <thead className="bg-[#f9f9ff] text-[#565e74] uppercase text-[11px] font-semibold border-b border-[#bcc9c6]/30">
              <tr>
                <th className="px-5 py-3">Order ID</th>
                <th className="px-5 py-3">Event</th>
                <th className="px-5 py-3">Customer</th>
                <th className="px-5 py-3">Tickets</th>
                <th className="px-5 py-3">Amount</th>
                <th className="px-5 py-3">Status</th>
                <th className="px-5 py-3">Time</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {recentOrders.map((order) => (
                <tr key={order.id} className="hover:bg-gray-50/70 transition-colors">
                  <td className="px-5 py-3.5 font-mono font-semibold text-[#00685f]">
                    {order.id}
                  </td>
                  <td className="px-5 py-3.5 font-medium">{order.event}</td>
                  <td className="px-5 py-3.5">
                    <div className="flex flex-col">
                      <span className="font-semibold">{order.customer}</span>
                      <span className="text-[#565e74] text-[11px]">{order.email}</span>
                    </div>
                  </td>
                  <td className="px-5 py-3.5 text-[#565e74]">{order.tickets}</td>
                  <td className="px-5 py-3.5 font-semibold text-[#141b2b]">{order.total}</td>
                  <td className="px-5 py-3.5">
                    <span className="inline-flex items-center px-2 py-0.5 rounded text-[10px] font-bold bg-emerald-100 text-emerald-800">
                      {order.status}
                    </span>
                  </td>
                  <td className="px-5 py-3.5 text-[#565e74]">{order.date}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
