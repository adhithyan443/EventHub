import { useState } from "react";
import { useNavigate } from "react-router-dom";
import EventCreationStepper from "../../../components/organizer/EventCreationStepper";
import EventCreationFooter from "../../../components/organizer/EventCreationFooter";
import PayoutBankAccountCard from "../../../components/organizer/PayoutBankAccountCard";
import useEventCreationStore from "../../../store/eventCreationStore";
import { ORGANIZER_ROUTES } from "../../../constants/eventConstants";

export default function CreateEventTicketTypesPage() {
  const navigate = useNavigate();

  const ticketTypes = useEventCreationStore((state) => state.ticketTypes);
  const addTicketType = useEventCreationStore((state) => state.addTicketType);
  const removeTicketType = useEventCreationStore((state) => state.removeTicketType);
  const ticketSalesSettings = useEventCreationStore((state) => state.ticketSalesSettings);

  const [showAddModal, setShowAddModal] = useState(false);
  const [newTicket, setNewTicket] = useState({
    name: "",
    price: "",
    capacity: "",
    description: "",
  });

  const handleBack = () => {
    navigate(ORGANIZER_ROUTES.CREATE_STEP_3);
  };

  const handleContinue = () => {
    navigate(ORGANIZER_ROUTES.CREATE_REVIEW);
  };

  const handleCreateTicket = (e) => {
    e.preventDefault();
    if (!newTicket.name || !newTicket.price || !newTicket.capacity) return;

    addTicketType({
      name: newTicket.name,
      price: Number(newTicket.price),
      capacity: Number(newTicket.capacity),
      description: newTicket.description,
    });

    setNewTicket({ name: "", price: "", capacity: "", description: "" });
    setShowAddModal(false);
  };

  return (
    <div className="min-h-full flex flex-col justify-between">
      <div className="p-8 max-w-4xl mx-auto w-full flex flex-col gap-6">
        {/* Stepper Progress Bar */}
        <EventCreationStepper currentStep={4} />

        {/* Card 1: Ticket Types */}
        <div className="bg-white border border-[#bcc9c6]/50 rounded-2xl shadow-sm overflow-hidden flex flex-col">
          <div className="p-6 border-b border-[#bcc9c6]/30 bg-[#f9f9ff] flex items-center justify-between">
            <div className="flex flex-col">
              <h1 className="text-xl font-bold text-[#141b2b]">Ticket Types</h1>
              <p className="text-xs text-[#565e74] mt-1">
                Create different ticket tiers, set quantities, and manage pricing for general admission.
              </p>
            </div>

            <button
              type="button"
              onClick={() => setShowAddModal(true)}
              className="flex items-center gap-2 bg-[#00685f] hover:bg-[#005550] text-white text-xs font-semibold px-4 py-2.5 rounded-lg shadow-sm transition-colors cursor-pointer"
            >
              <svg className="size-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2.5" d="M12 4v16m8-8H4" />
              </svg>
              <span>Add Ticket Type</span>
            </button>
          </div>

          <div className="p-6 flex flex-col gap-4">
            {ticketTypes.length === 0 ? (
              <div className="p-8 text-center text-xs text-[#565e74] border border-dashed border-[#bcc9c6] rounded-xl">
                No ticket types created yet. Click &quot;Add Ticket Type&quot; above to create one.
              </div>
            ) : (
              ticketTypes.map((ticket) => (
                <div
                  key={ticket.id}
                  className="bg-[#f9f9ff] border border-[#bcc9c6]/60 rounded-xl p-5 flex flex-col md:flex-row md:items-center justify-between gap-4"
                >
                  <div className="flex items-start gap-3.5">
                    <div className="size-10 rounded-full bg-[#00685f]/15 flex items-center justify-center text-[#00685f] shrink-0 font-bold">
                      🎟️
                    </div>
                    <div className="flex flex-col">
                      <div className="flex items-center gap-2">
                        <h3 className="font-bold text-[14px] text-[#141b2b]">
                          {ticket.name}
                        </h3>
                        <span
                          className={`text-[10px] font-bold px-2 py-0.5 rounded uppercase tracking-wider ${
                            ticket.status === "ACTIVE"
                              ? "bg-emerald-100 text-emerald-800"
                              : "bg-gray-200 text-gray-700"
                          }`}
                        >
                          {ticket.status}
                        </span>
                      </div>
                      <p className="text-xs text-[#565e74] mt-0.5">{ticket.description}</p>
                    </div>
                  </div>

                  <div className="flex items-center gap-6 self-end md:self-center">
                    <div className="flex flex-col text-right">
                      <span className="text-base font-bold text-[#141b2b]">
                        ${ticket.price}.00
                      </span>
                      <span className="text-[11px] text-[#565e74]">
                        {ticket.capacity} tickets total
                      </span>
                    </div>

                    <button
                      type="button"
                      onClick={() => removeTicketType(ticket.id)}
                      className="p-1.5 text-gray-400 hover:text-red-600 rounded transition-colors cursor-pointer"
                      title="Remove ticket"
                    >
                      <svg className="size-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                      </svg>
                    </button>
                  </div>
                </div>
              ))
            )}
          </div>
        </div>

        {/* Card 2: Ticketing Sales Settings */}
        <div className="bg-white border border-[#bcc9c6]/50 rounded-2xl shadow-sm overflow-hidden flex flex-col">
          <div className="p-6 border-b border-[#bcc9c6]/30 bg-[#f9f9ff]">
            <h2 className="text-base font-bold text-[#141b2b]">Ticketing Settings</h2>
            <p className="text-xs text-[#565e74] mt-1">
              Sales windows, maximum tickets allowed per customer, and purchase rules.
            </p>
          </div>

          <div className="p-6 grid grid-cols-1 md:grid-cols-3 gap-5">
            <div className="flex flex-col gap-2">
              <label className="text-[13px] font-semibold text-[#141b2b]">
                Sales Start Date
              </label>
              <input
                type="date"
                defaultValue={ticketSalesSettings.salesStartDate}
                className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg px-4 py-2.5 text-sm text-[#141b2b] focus:outline-none focus:border-[#00685f]"
              />
            </div>

            <div className="flex flex-col gap-2">
              <label className="text-[13px] font-semibold text-[#141b2b]">
                Sales End Date
              </label>
              <input
                type="date"
                defaultValue={ticketSalesSettings.salesEndDate}
                className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg px-4 py-2.5 text-sm text-[#141b2b] focus:outline-none focus:border-[#00685f]"
              />
            </div>

            <div className="flex flex-col gap-2">
              <label className="text-[13px] font-semibold text-[#141b2b]">
                Max Tickets Per Order
              </label>
              <input
                type="number"
                defaultValue={ticketSalesSettings.maxTicketsPerBooking}
                min={1}
                max={20}
                className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg px-4 py-2.5 text-sm text-[#141b2b] focus:outline-none focus:border-[#00685f]"
              />
            </div>
          </div>
        </div>

        {/* Card 3: Payout Account */}
        <PayoutBankAccountCard />
      </div>

      {/* Add Ticket Modal */}
      {showAddModal && (
        <div className="fixed inset-0 z-50 bg-black/40 backdrop-blur-xs flex items-center justify-center p-4">
          <div className="bg-white rounded-2xl max-w-md w-full p-6 shadow-xl border border-[#bcc9c6] flex flex-col gap-5">
            <div className="flex items-center justify-between">
              <h3 className="font-bold text-lg text-[#141b2b]">Add New Ticket Type</h3>
              <button
                type="button"
                onClick={() => setShowAddModal(false)}
                className="text-gray-400 hover:text-gray-700 text-lg"
              >
                &times;
              </button>
            </div>

            <form onSubmit={handleCreateTicket} className="flex flex-col gap-4">
              <div className="flex flex-col gap-1.5">
                <label className="text-xs font-semibold text-[#141b2b]">
                  Ticket Name <span className="text-[#ba1a1a]">*</span>
                </label>
                <input
                  type="text"
                  required
                  placeholder="e.g. VIP Pass, Student Discount"
                  value={newTicket.name}
                  onChange={(e) => setNewTicket({ ...newTicket, name: e.target.value })}
                  className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-[#00685f]"
                />
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="flex flex-col gap-1.5">
                  <label className="text-xs font-semibold text-[#141b2b]">
                    Price ($) <span className="text-[#ba1a1a]">*</span>
                  </label>
                  <input
                    type="number"
                    required
                    min={0}
                    placeholder="99"
                    value={newTicket.price}
                    onChange={(e) => setNewTicket({ ...newTicket, price: e.target.value })}
                    className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-[#00685f]"
                  />
                </div>

                <div className="flex flex-col gap-1.5">
                  <label className="text-xs font-semibold text-[#141b2b]">
                    Quantity Available <span className="text-[#ba1a1a]">*</span>
                  </label>
                  <input
                    type="number"
                    required
                    min={1}
                    placeholder="500"
                    value={newTicket.capacity}
                    onChange={(e) => setNewTicket({ ...newTicket, capacity: e.target.value })}
                    className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-[#00685f]"
                  />
                </div>
              </div>

              <div className="flex flex-col gap-1.5">
                <label className="text-xs font-semibold text-[#141b2b]">Description</label>
                <textarea
                  rows={2}
                  placeholder="What is included with this ticket?"
                  value={newTicket.description}
                  onChange={(e) => setNewTicket({ ...newTicket, description: e.target.value })}
                  className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg p-3 text-sm focus:outline-none focus:border-[#00685f]"
                />
              </div>

              <div className="flex items-center justify-end gap-3 mt-2">
                <button
                  type="button"
                  onClick={() => setShowAddModal(false)}
                  className="px-4 py-2 rounded-lg border border-[#bcc9c6] text-xs font-semibold text-[#565e74] hover:bg-gray-50"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="px-5 py-2 rounded-lg bg-[#00685f] hover:bg-[#005550] text-white text-xs font-semibold shadow-sm"
                >
                  Create Ticket
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Persistent Footer */}
      <EventCreationFooter
        onBack={handleBack}
        onContinue={handleContinue}
      />
    </div>
  );
}
