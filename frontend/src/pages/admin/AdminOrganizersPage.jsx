import { useState } from "react";
import SuspendModal from "../../components/admin/SuspendModal";
import OrganizerDetailDrawer from "../../components/admin/OrganizerDetailDrawer";

const initialOrganizers = [
  {
    id: "ORG001",
    name: "Rahul Kumar",
    email: "rahul@eventmasters.in",
    org: "Event Masters",
    events: 12,
    revenue: "₹3,10,000",
    status: "ACTIVE",
    joined: "10 Jun 2026",
  },
  {
    id: "ORG002",
    name: "Anjali S",
    email: "anjali@techev.in",
    org: "Tech Events India",
    events: 8,
    revenue: "₹1,85,000",
    status: "ACTIVE",
    joined: "15 Jun 2026",
  },
  {
    id: "ORG003",
    name: "Arjun P",
    email: "arjun@creativehub.in",
    org: "Creative Hub",
    events: 4,
    revenue: "₹2,10,000",
    status: "SUSPENDED",
    joined: "20 Jul 2026",
  },
  {
    id: "ORG004",
    name: "Priya M",
    email: "priya@spark.in",
    org: "Spark Conferences",
    events: 6,
    revenue: "₹95,000",
    status: "ACTIVE",
    joined: "1 Jul 2026",
  },
  {
    id: "ORG005",
    name: "Sneha R",
    email: "sneha@innovate.in",
    org: "Innovate India",
    events: 3,
    revenue: "₹42,000",
    status: "ACTIVE",
    joined: "5 Aug 2026",
  },
];

export default function AdminOrganizersPage() {
  const [orgs, setOrgs] = useState(initialOrganizers);
  const [search, setSearch] = useState("");
  const [activeDrawerOrg, setActiveDrawerOrg] = useState(null);
  const [suspendTarget, setSuspendTarget] = useState(null);
  const [currentPage, setCurrentPage] = useState(1);

  const filtered = orgs.filter(
    (o) =>
      search === "" ||
      o.name.toLowerCase().includes(search.toLowerCase()) ||
      o.org.toLowerCase().includes(search.toLowerCase()) ||
      o.email.toLowerCase().includes(search.toLowerCase())
  );

  const handleConfirmSuspend = (id, reason) => {
    setOrgs((prev) =>
      prev.map((o) => (o.id === id ? { ...o, status: "SUSPENDED" } : o))
    );
    if (activeDrawerOrg?.id === id) {
      setActiveDrawerOrg((prev) => (prev ? { ...prev, status: "SUSPENDED" } : null));
    }
    setSuspendTarget(null);
  };

  return (
    <div className="flex-1 flex flex-col overflow-hidden bg-white w-full">
      {/* Header */}
      <div className="px-6 py-4 border-b border-[#dce2f7] flex items-center justify-between shrink-0">
        <div>
          <h1 className="text-[#141b2b] text-xl font-bold tracking-tight">
            Organizers
          </h1>
          <p className="text-[#3d4947] text-sm">
            Manage approved organizers and monitor their event activity.
          </p>
        </div>
        <button
          type="button"
          onClick={() => alert("Exporting organizers list...")}
          className="bg-[#f9f9ff] border border-[#bcc9c6] text-[#141b2b] text-sm font-semibold px-4 py-2 rounded-lg flex items-center gap-2 hover:bg-[#eef0fb] transition-colors cursor-pointer"
        >
          <svg width="12" height="12" viewBox="0 0 12 12" fill="none">
            <path
              d="M6 0L10.5 4.5H7.5V9H4.5V4.5H1.5L6 0ZM0 10.5H12V12H0V10.5Z"
              fill="currentColor"
            />
          </svg>
          Export Organizers
        </button>
      </div>

      {/* Main Content Area */}
      <div className="flex-1 overflow-y-auto p-6 space-y-5">
        {/* Metric Cards (1 x 4) */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          {[
            {
              label: "Total Organizers",
              value: "428",
              sub: "↑ 12% from last month",
              subColor: "text-[#00685f]",
            },
            {
              label: "Active",
              value: "402",
              sub: "94% health score",
              subColor: "text-[#00685f]",
            },
            {
              label: "Suspended",
              value: "18",
              sub: "4 pending appeals",
              subColor: "text-[#92400e]",
            },
            {
              label: "New",
              value: "24",
              sub: "Since last week",
              subColor: "text-[#00685f]",
            },
          ].map((card, i) => (
            <div
              key={i}
              className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-[0px_1px_1.5px_rgba(0,0,0,0.1)]"
            >
              <div className="text-[#3d4947] text-sm font-medium mb-2">
                {card.label}
              </div>
              <div className="text-[#141b2b] text-2xl font-bold tracking-tight">
                {card.value}
              </div>
              <div className={`text-xs mt-1 font-medium ${card.subColor}`}>
                {card.sub}
              </div>
            </div>
          ))}
        </div>

        {/* Organizers Table Card */}
        <div className="bg-white border border-[#bcc9c6] rounded-xl overflow-hidden shadow-[0px_1px_1.5px_rgba(0,0,0,0.1)]">
          {/* Table Search & Filter Bar */}
          <div className="px-4 py-3 border-b border-[#dce2f7] flex items-center justify-between gap-3 bg-[#f1f3ff]">
            <div className="relative flex-1 max-w-xs">
              <svg
                className="absolute left-3 top-1/2 -translate-y-1/2"
                width="13"
                height="13"
                viewBox="0 0 13 13"
                fill="none"
              >
                <path
                  d="M8.5 7.5H7.71L7.43 7.23C8.41 6.09 9 4.61 9 3C9 1.34 7.66 0 6 0C4.34 0 3 1.34 3 3C3 4.66 4.34 6 6 6C6.73 6 7.4 5.75 7.93 5.32L8.2 5.6V6.35L11.25 9.39L12.24 8.4L9.2 5.35H8.5V7.5ZM6 5.5C4.62 5.5 3.5 4.38 3.5 3C3.5 1.62 4.62 0.5 6 0.5C7.38 0.5 8.5 1.62 8.5 3C8.5 4.38 7.38 5.5 6 5.5Z"
                  fill="#6D7A77"
                />
              </svg>
              <input
                type="text"
                placeholder="Search by organizer name..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="bg-white border border-[#bcc9c6] rounded-lg pl-9 pr-3 py-2 text-sm text-[#141b2b] placeholder:text-[#6d7a77] w-full focus:outline-none focus:ring-1 focus:ring-[#00685f]/40"
              />
            </div>
            <button
              type="button"
              className="bg-[#f9f9ff] border border-[#bcc9c6] text-[#141b2b] text-sm font-semibold px-3 py-2 rounded-lg flex items-center gap-2 hover:bg-[#eef0fb] cursor-pointer"
            >
              <svg width="13" height="9" viewBox="0 0 13 9" fill="none">
                <path
                  d="M0 0H13V1.5H0V0ZM2 3.75H11V5.25H2V3.75ZM4.5 7.5H8.5V9H4.5V7.5Z"
                  fill="currentColor"
                />
              </svg>
              More Filters
            </button>
          </div>

          {/* Table */}
          <div className="overflow-x-auto">
            <table className="w-full text-left">
              <thead>
                <tr className="bg-white border-b border-[rgba(188,201,198,0.5)]">
                  <th className="px-4 py-3 text-[#3d4947] text-xs font-medium uppercase tracking-wider">
                    Organizer
                  </th>
                  <th className="px-4 py-3 text-[#3d4947] text-xs font-medium uppercase tracking-wider">
                    Organization
                  </th>
                  <th className="px-4 py-3 text-[#3d4947] text-xs font-medium uppercase tracking-wider">
                    Events
                  </th>
                  <th className="px-4 py-3 text-[#3d4947] text-xs font-medium uppercase tracking-wider">
                    Revenue
                  </th>
                  <th className="px-4 py-3 text-[#3d4947] text-xs font-medium uppercase tracking-wider">
                    Status
                  </th>
                  <th className="px-4 py-3 text-[#3d4947] text-xs font-medium uppercase tracking-wider">
                    Joined
                  </th>
                  <th className="px-4 py-3 text-right text-[#3d4947] text-xs font-medium uppercase tracking-wider">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody>
                {filtered.map((org, i) => (
                  <tr
                    key={org.id}
                    className={`border-t border-[rgba(188,201,198,0.3)] ${
                      i % 2 === 0 ? "bg-white" : "bg-[#fafbff]"
                    } hover:bg-[#f1f3ff] transition-colors`}
                  >
                    <td className="px-4 py-4">
                      <div className="flex items-center gap-3">
                        <div className="w-9 h-9 rounded-full bg-[#e8eaf6] flex items-center justify-center text-sm font-bold text-[#5c6bc0]">
                          {org.name[0]}
                        </div>
                        <div>
                          <div className="text-[#141b2b] text-sm font-medium">
                            {org.name}
                          </div>
                          <div className="text-[#6d7a77] text-xs">{org.email}</div>
                        </div>
                      </div>
                    </td>
                    <td className="px-4 py-4 text-[#3d4947] text-sm font-medium">
                      {org.org}
                    </td>
                    <td className="px-4 py-4 text-[#141b2b] text-sm font-semibold">
                      {org.events}
                    </td>
                    <td className="px-4 py-4 text-[#141b2b] text-sm font-semibold">
                      {org.revenue}
                    </td>
                    <td className="px-4 py-4">
                      <span
                        className={`px-2.5 py-1 rounded-full text-[10px] font-bold ${
                          org.status === "ACTIVE"
                            ? "bg-[#dcfce7] text-[#166534]"
                            : "bg-[#ffdad6] text-[#93000a]"
                        }`}
                      >
                        {org.status}
                      </span>
                    </td>
                    <td className="px-4 py-4 text-[#3d4947] text-sm">
                      {org.joined}
                    </td>
                    <td className="px-4 py-4 text-right">
                      <button
                        type="button"
                        onClick={() => setActiveDrawerOrg(org)}
                        className="text-sm text-[#00685f] font-semibold hover:underline cursor-pointer"
                      >
                        View
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {/* Pagination Footer */}
          <div className="flex items-center justify-between px-4 py-3 border-t border-[rgba(188,201,198,0.3)] bg-white">
            <span className="text-[#3d4947] text-sm">
              Showing 1 to {filtered.length} of 428 organizers
            </span>
            <div className="flex items-center gap-1">
              {[1, 2, 3].map((p) => (
                <button
                  key={p}
                  type="button"
                  onClick={() => setCurrentPage(p)}
                  className={`w-8 h-8 rounded text-sm font-medium transition-colors cursor-pointer ${
                    currentPage === p
                      ? "bg-[#00685f] text-white"
                      : "bg-[#f9f9ff] border border-[#bcc9c6] text-[#3d4947] hover:bg-[#eef0fb]"
                  }`}
                >
                  {p}
                </button>
              ))}
              <button
                type="button"
                className="w-8 h-8 rounded bg-[#f9f9ff] border border-[#bcc9c6] text-[#3d4947] hover:bg-[#eef0fb] cursor-pointer"
              >
                ›
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* Organizer Profile Detail Drawer */}
      {activeDrawerOrg && (
        <OrganizerDetailDrawer
          organizer={activeDrawerOrg}
          onClose={() => setActiveDrawerOrg(null)}
          onOpenSuspend={(org) => setSuspendTarget(org)}
        />
      )}

      {/* Suspend Confirmation Modal */}
      {suspendTarget && (
        <SuspendModal
          organizer={suspendTarget}
          onConfirm={handleConfirmSuspend}
          onCancel={() => setSuspendTarget(null)}
        />
      )}
    </div>
  );
}
