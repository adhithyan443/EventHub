import { useState } from "react";
import ReviewPanel from "../../components/admin/ReviewPanel";
import RejectModal from "../../components/admin/RejectModal";

const initialApplications = [
  {
    id: "APP001",
    businessName: "Event Masters Pvt Ltd",
    applicant: "Rahul Kumar",
    type: "Company",
    submitted: "10 Aug 2026",
    status: "PENDING",
    email: "rahul@eventmasters.in",
    phone: "+91 98765 43210",
    pan: "ABCDE1234F",
    gst: "29ABCDE1234F1Z5",
    website: "eventmasters.in",
    description:
      "Full-scale event production agency handling international developer conferences and concerts.",
  },
  {
    id: "APP002",
    businessName: "Tech Events India",
    applicant: "Anjali S",
    type: "Company",
    submitted: "9 Aug 2026",
    status: "PENDING",
    email: "anjali@techev.in",
    phone: "+91 98111 22334",
    pan: "TECHP8829J",
    gst: "29TECHP8829J1Z2",
    website: "techevents.in",
    description:
      "Technology meetup and hackathon organizer connecting student builders and industry leads.",
  },
  {
    id: "APP003",
    businessName: "Creative Hub",
    applicant: "Arjun P",
    type: "Individual",
    submitted: "8 Aug 2026",
    status: "APPROVED",
    email: "arjun@creativehub.in",
    phone: "+91 97222 33445",
    pan: "CRHUB9912K",
    gst: "29CRHUB9912K1Z9",
    website: "creativehub.org",
    description:
      "Art, design, and typography workshops for independent creators and students.",
  },
  {
    id: "APP004",
    businessName: "Spark Conferences",
    applicant: "Priya M",
    type: "Company",
    submitted: "7 Aug 2026",
    status: "PENDING",
    email: "priya@spark.in",
    phone: "+91 96333 44556",
    pan: "SPARK1029Q",
    gst: "29SPARK1029Q1Z4",
    website: "sparkconf.com",
    description:
      "Leadership summits and tech keynotes hosting C-level executives and entrepreneurs.",
  },
  {
    id: "APP005",
    businessName: "Mumbai Events Co.",
    applicant: "Ravi D",
    type: "Company",
    submitted: "6 Aug 2026",
    status: "REJECTED",
    email: "ravi@mumbaievents.com",
    phone: "+91 95444 55667",
    pan: "MUMEV4421X",
    gst: "27MUMEV4421X1Z1",
    website: "mumbaievents.com",
    description:
      "Local cultural celebrations and food festivals across the metropolitan area.",
  },
  {
    id: "APP006",
    businessName: "Innovate India",
    applicant: "Sneha R",
    type: "Individual",
    submitted: "5 Aug 2026",
    status: "APPROVED",
    email: "sneha@innovate.in",
    phone: "+91 94555 66778",
    pan: "INNOV7731M",
    gst: "29INNOV7731M1Z8",
    website: "innovateindia.in",
    description:
      "Incubator showcase and pitch competition series for early-stage startups.",
  },
  {
    id: "APP007",
    businessName: "Global Expo Ltd",
    applicant: "Karan P",
    type: "Company",
    submitted: "4 Aug 2026",
    status: "PENDING",
    email: "karan@globalexpo.com",
    phone: "+91 93666 77889",
    pan: "GLOBA5519L",
    gst: "29GLOBA5519L1Z6",
    website: "globalexpo.com",
    description:
      "B2B trade exhibitions and industrial vendor expos in major convention centers.",
  },
];

const statusColors = {
  PENDING: "bg-[#fef3c7] text-[#92400e]",
  APPROVED: "bg-[#dcfce7] text-[#166534]",
  REJECTED: "bg-[#ffdad6] text-[#93000a]",
};

export default function AdminApplicationsPage() {
  const [apps, setApps] = useState(initialApplications);
  const [activeTab, setActiveTab] = useState("all");
  const [search, setSearch] = useState("");
  const [reviewApp, setReviewApp] = useState(null);
  const [rejectTarget, setRejectTarget] = useState(null);
  const [page, setPage] = useState(1);
  const perPage = 10;

  const counts = {
    all: apps.length,
    pending: apps.filter((a) => a.status === "PENDING").length,
    approved: apps.filter((a) => a.status === "APPROVED").length,
    rejected: apps.filter((a) => a.status === "REJECTED").length,
  };

  const filtered = apps
    .filter((a) => activeTab === "all" || a.status === activeTab.toUpperCase())
    .filter(
      (a) =>
        search === "" ||
        a.id.toLowerCase().includes(search.toLowerCase()) ||
        a.businessName.toLowerCase().includes(search.toLowerCase()) ||
        a.applicant.toLowerCase().includes(search.toLowerCase())
    );

  const paginated = filtered.slice((page - 1) * perPage, page * perPage);
  const totalPages = Math.max(1, Math.ceil(filtered.length / perPage));

  const handleApprove = (id) => {
    setApps((prev) =>
      prev.map((a) => (a.id === id ? { ...a, status: "APPROVED" } : a))
    );
    setReviewApp((prev) =>
      prev?.id === id ? { ...prev, status: "APPROVED" } : prev
    );
  };

  const handleReject = (id) => {
    setApps((prev) =>
      prev.map((a) => (a.id === id ? { ...a, status: "REJECTED" } : a))
    );
    setReviewApp((prev) =>
      prev?.id === id ? { ...prev, status: "REJECTED" } : prev
    );
    setRejectTarget(null);
  };

  const tabs = [
    { key: "all", label: "All", count: counts.all },
    { key: "pending", label: "Pending", count: counts.pending },
    { key: "approved", label: "Approved", count: counts.approved },
    { key: "rejected", label: "Rejected", count: counts.rejected },
  ];

  return (
    <div className="flex-1 flex flex-col overflow-hidden bg-white w-full">
      {/* Page Header */}
      <div className="px-6 py-4 border-b border-[#dce2f7] flex items-center justify-between shrink-0">
        <div>
          <h1 className="text-[#141b2b] text-xl font-bold tracking-tight">
            Organizer Applications
          </h1>
          <p className="text-[#6d7a77] text-xs mt-0.5">
            Review and take action on incoming organizer onboarding requests.
          </p>
        </div>
        <button
          type="button"
          onClick={() => alert("Exporting applications data...")}
          className="bg-[#f9f9ff] border border-[#bcc9c6] text-[#141b2b] text-sm font-semibold px-4 py-2 rounded-lg flex items-center gap-2 hover:bg-[#eef0fb] transition-colors cursor-pointer"
        >
          <svg width="12" height="12" viewBox="0 0 12 12" fill="none">
            <path
              d="M6 0L10.5 4.5H7.5V9H4.5V4.5H1.5L6 0ZM0 10.5H12V12H0V10.5Z"
              fill="currentColor"
            />
          </svg>
          Export Applications
        </button>
      </div>

      {/* Main Content Area */}
      <div className="flex-1 overflow-y-auto p-6 space-y-5">
        {/* Metric Cards (1 x 4) */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          {[
            {
              label: "Pending",
              count: counts.pending,
              color: "bg-[#fef3c7]",
              iconColor: "#92400e",
              icon: (
                <svg width="14" height="16" viewBox="0 0 14 16" fill="none">
                  <path
                    d="M13 5.5H9.5V1.5C9.5 0.675 8.825 0 8 0H6C5.175 0 4.5 0.675 4.5 1.5V5.5H1C0.175 5.5 0 6.175 0 7L0 14.5C0 15.325 0.675 16 1.5 16H12.5C13.325 16 14 15.325 14 14.5V7C14 6.175 13.325 5.5 13 5.5Z"
                    fill="#92400e"
                  />
                </svg>
              ),
            },
            {
              label: "Approved",
              count: counts.approved,
              color: "bg-[#dcfce7]",
              iconColor: "#166534",
              icon: (
                <svg width="15" height="15" viewBox="0 0 15 15" fill="none">
                  <path
                    d="M5.5 10.5L1.5 6.5L2.91 5.09L5.5 7.67L12.09 1.08L13.5 2.5L5.5 10.5Z"
                    fill="#166534"
                  />
                </svg>
              ),
            },
            {
              label: "Rejected",
              count: counts.rejected,
              color: "bg-[#ffdad6]",
              iconColor: "#93000a",
              icon: (
                <svg width="15" height="15" viewBox="0 0 15 15" fill="none">
                  <path
                    d="M13.5 2.91L12.09 1.5L7.5 6.09L2.91 1.5L1.5 2.91L6.09 7.5L1.5 12.09L2.91 13.5L7.5 8.91L12.09 13.5L13.5 12.09L8.91 7.5L13.5 2.91Z"
                    fill="#93000a"
                  />
                </svg>
              ),
            },
            {
              label: "Total Applications",
              count: counts.all,
              color: "bg-[#e1e8fd]",
              iconColor: "#141b2b",
              icon: (
                <svg width="13" height="15" viewBox="0 0 13 15" fill="none">
                  <path
                    d="M11.5 0H1.5C0.675 0 0 0.675 0 1.5V13.5C0 14.325 0.675 15 1.5 15H11.5C12.325 15 13 14.325 13 13.5V1.5C13 0.675 12.325 0 11.5 0ZM5 11.25L2 8.25L3.06 7.19L5 9.12L9.44 4.69L10.5 5.75L5 11.25Z"
                    fill="#141b2b"
                  />
                </svg>
              ),
            },
          ].map((card, i) => (
            <div
              key={i}
              className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-[0px_1px_1.5px_rgba(0,0,0,0.1)]"
            >
              <div className="flex items-center justify-between mb-2">
                <span className="text-[#3d4947] text-sm font-semibold">
                  {card.label}
                </span>
                <span
                  className={`${card.color} w-8 h-8 rounded-lg flex items-center justify-center`}
                >
                  {card.icon}
                </span>
              </div>
              <div
                className={`text-2xl font-bold ${
                  i === 0 ? "text-[#00685f]" : "text-[#141b2b]"
                }`}
              >
                {card.count}
              </div>
            </div>
          ))}
        </div>

        {/* Applications Table Card */}
        <div className="bg-white border border-[#bcc9c6] rounded-xl overflow-hidden shadow-[0px_1px_1.5px_rgba(0,0,0,0.1)]">
          {/* Table Controls */}
          <div className="bg-[#f1f3ff] border-b border-[#bcc9c6] px-4 py-3 flex flex-wrap items-center justify-between gap-3">
            {/* Status Tabs */}
            <div className="flex items-center">
              {tabs.map(({ key, label, count }) => (
                <button
                  key={key}
                  type="button"
                  onClick={() => {
                    setActiveTab(key);
                    setPage(1);
                  }}
                  className={`flex items-center gap-2 px-4 py-2 text-sm font-semibold transition-colors relative cursor-pointer ${
                    activeTab === key
                      ? "text-[#00685f] after:absolute after:bottom-0 after:left-0 after:right-0 after:h-0.5 after:bg-[#00685f]"
                      : "text-[#3d4947] hover:text-[#141b2b]"
                  }`}
                >
                  {label}
                  {count > 0 && (
                    <span
                      className={`text-[10px] font-bold px-1.5 py-0.5 rounded-full ${
                        activeTab === key && key === "pending"
                          ? "bg-[#fef3c7] text-[#92400e]"
                          : "bg-[#dce2f7] text-[#141b2b]"
                      }`}
                    >
                      {count}
                    </span>
                  )}
                </button>
              ))}
            </div>

            {/* Search & Filters */}
            <div className="flex items-center gap-2">
              <div className="relative">
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
                  placeholder="Search applications..."
                  value={search}
                  onChange={(e) => {
                    setSearch(e.target.value);
                    setPage(1);
                  }}
                  className="bg-white border border-[#bcc9c6] rounded-lg pl-9 pr-3 py-2 text-sm text-[#141b2b] placeholder:text-[#6d7a77] w-52 focus:outline-none focus:ring-1 focus:ring-[#00685f]/40"
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
                Filters
              </button>
            </div>
          </div>

          {/* Table */}
          <div className="overflow-x-auto">
            <table className="w-full text-left">
              <thead>
                <tr className="bg-white border-b border-[rgba(188,201,198,0.5)]">
                  <th className="px-4 py-3 text-[#3d4947] text-xs font-medium uppercase tracking-wider">
                    Application ID
                  </th>
                  <th className="px-4 py-3 text-[#3d4947] text-xs font-medium uppercase tracking-wider">
                    Business / Applicant
                  </th>
                  <th className="px-4 py-3 text-[#3d4947] text-xs font-medium uppercase tracking-wider">
                    Type
                  </th>
                  <th className="px-4 py-3 text-[#3d4947] text-xs font-medium uppercase tracking-wider">
                    Submitted
                  </th>
                  <th className="px-4 py-3 text-[#3d4947] text-xs font-medium uppercase tracking-wider">
                    Status
                  </th>
                  <th className="px-4 py-3 text-right text-[#3d4947] text-xs font-medium uppercase tracking-wider">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody>
                {paginated.length === 0 ? (
                  <tr>
                    <td
                      colSpan={6}
                      className="px-4 py-10 text-center text-[#6d7a77] text-sm"
                    >
                      No applications found matching your criteria.
                    </td>
                  </tr>
                ) : (
                  paginated.map((app, i) => (
                    <tr
                      key={app.id}
                      className={`border-t border-[rgba(188,201,198,0.3)] ${
                        i % 2 === 0 ? "bg-white" : "bg-[#fafbff]"
                      } hover:bg-[#f1f3ff] transition-colors`}
                    >
                      <td className="px-4 py-4 font-mono text-[#141b2b] text-xs font-semibold">
                        {app.id}
                      </td>
                      <td className="px-4 py-4">
                        <div className="text-[#141b2b] text-sm font-medium">
                          {app.businessName}
                        </div>
                        <div className="text-[#6d7a77] text-xs">{app.applicant}</div>
                      </td>
                      <td className="px-4 py-4 text-[#3d4947] text-sm">
                        {app.type}
                      </td>
                      <td className="px-4 py-4 text-[#3d4947] text-sm">
                        {app.submitted}
                      </td>
                      <td className="px-4 py-4">
                        <span
                          className={`px-2.5 py-1 rounded-full text-[10px] font-bold ${
                            statusColors[app.status]
                          }`}
                        >
                          {app.status}
                        </span>
                      </td>
                      <td className="px-4 py-4 text-right">
                        <button
                          type="button"
                          onClick={() => setReviewApp(app)}
                          className={`text-sm cursor-pointer transition-all ${
                            app.status === "PENDING"
                              ? "text-[#00685f] font-semibold hover:underline"
                              : "text-[#3d4947] font-medium hover:text-[#141b2b]"
                          }`}
                        >
                          {app.status === "PENDING" ? "Review" : "View"}
                        </button>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>

          {/* Pagination Footer */}
          <div className="flex items-center justify-between px-4 py-3 border-t border-[rgba(188,201,198,0.3)] bg-white">
            <span className="text-[#3d4947] text-sm">
              Showing {filtered.length === 0 ? 0 : (page - 1) * perPage + 1} to{" "}
              {Math.min(page * perPage, filtered.length)} of {filtered.length}{" "}
              {activeTab === "pending" ? "pending " : ""}applications
            </span>
            <div className="flex gap-2">
              <button
                type="button"
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page === 1}
                className="bg-[#f9f9ff] border border-[#bcc9c6] text-[#3d4947] text-sm px-3 py-1 rounded disabled:opacity-40 hover:bg-[#eef0fb] transition-colors cursor-pointer disabled:cursor-not-allowed"
              >
                Previous
              </button>
              <button
                type="button"
                onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                disabled={page === totalPages}
                className="bg-[#f9f9ff] border border-[#bcc9c6] text-[#3d4947] text-sm px-3 py-1 rounded disabled:opacity-40 hover:bg-[#eef0fb] transition-colors cursor-pointer disabled:cursor-not-allowed"
              >
                Next
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* Review Drawer Panel */}
      {reviewApp && (
        <ReviewPanel
          app={reviewApp}
          onClose={() => setReviewApp(null)}
          onReject={() => setRejectTarget(reviewApp)}
          onApprove={() => handleApprove(reviewApp.id)}
        />
      )}

      {/* Reject Confirmation Modal */}
      {rejectTarget && (
        <RejectModal
          appId={rejectTarget.id}
          businessName={rejectTarget.businessName}
          onConfirm={() => handleReject(rejectTarget.id)}
          onCancel={() => setRejectTarget(null)}
        />
      )}
    </div>
  );
}
