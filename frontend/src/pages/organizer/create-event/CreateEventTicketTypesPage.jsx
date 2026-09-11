import { useState } from "react";
import { useNavigate } from "react-router-dom";
import EventCreationStepper from "../../../components/organizer/EventCreationStepper";
import EventCreationFooter from "../../../components/organizer/EventCreationFooter";
import PayoutBankAccountCard from "../../../components/organizer/PayoutBankAccountCard";
import useEventCreationStore from "../../../store/eventCreationStore";
import { ORGANIZER_ROUTES } from "../../../constants/eventConstants";

const MAX_TICKET_NAME_LENGTH = 100;
const MAX_TICKET_DESCRIPTION_LENGTH = 500;
const MAX_TICKET_QUANTITY = 1000000;

const EMPTY_TICKET = {
  name: "",
  price: "",
  capacity: "",
  description: "",
};

export default function CreateEventTicketTypesPage() {
  const navigate = useNavigate();

  const ticketTypes = useEventCreationStore((state) => state.ticketTypes);
  const addTicketType = useEventCreationStore((state) => state.addTicketType);
  const updateTicketType = useEventCreationStore(
    (state) => state.updateTicketType
  );
  const removeTicketType = useEventCreationStore(
    (state) => state.removeTicketType
  );

  const ticketSalesSettings = useEventCreationStore(
    (state) => state.ticketSalesSettings
  );
  const updateTicketSalesSettings = useEventCreationStore(
    (state) => state.updateTicketSalesSettings
  );

  const [showAddModal, setShowAddModal] = useState(false);
  const [editingTicketId, setEditingTicketId] = useState(null);

  const [newTicket, setNewTicket] = useState(EMPTY_TICKET);

  const handleBack = () => {
    navigate(ORGANIZER_ROUTES.CREATE_STEP_3);
  };

  const handleContinue = () => {
    navigate(ORGANIZER_ROUTES.CREATE_REVIEW);
  };

  const openAddTicketModal = () => {
    setEditingTicketId(null);
    setNewTicket(EMPTY_TICKET);
    setShowAddModal(true);
  };

  const handleEditTicket = (ticket) => {
    setEditingTicketId(ticket.id);

    setNewTicket({
      name: ticket.name || "",
      price: String(ticket.price ?? ""),
      capacity: String(ticket.capacity ?? ""),
      description: ticket.description || "",
    });

    setShowAddModal(true);
  };

  const closeTicketModal = () => {
    setShowAddModal(false);
    setEditingTicketId(null);
    setNewTicket(EMPTY_TICKET);
  };

  const handleCreateTicket = (e) => {
    e.preventDefault();

    const name = newTicket.name.trim();
    const description = newTicket.description.trim();
    const price = Number(newTicket.price);
    const capacity = Number(newTicket.capacity);

    // Ticket name validation
    if (!name) {
      alert("Ticket name is required.");
      return;
    }

    if (name.length < 2) {
      alert("Ticket name must be at least 2 characters.");
      return;
    }

    if (name.length > MAX_TICKET_NAME_LENGTH) {
      alert(
        `Ticket name must not exceed ${MAX_TICKET_NAME_LENGTH} characters.`
      );
      return;
    }

    // Price validation
    if (newTicket.price === "") {
      alert("Ticket price is required.");
      return;
    }

    if (!Number.isFinite(price) || price < 0) {
      alert("Ticket price must be a valid amount greater than or equal to 0.");
      return;
    }

    // Quantity validation
    if (newTicket.capacity === "") {
      alert("Ticket quantity is required.");
      return;
    }

    if (!Number.isInteger(capacity) || capacity <= 0) {
      alert("Ticket quantity must be a positive whole number.");
      return;
    }

    if (capacity > MAX_TICKET_QUANTITY) {
      alert(`Ticket quantity cannot exceed ${MAX_TICKET_QUANTITY}.`);
      return;
    }

    // Description validation
    if (description.length > MAX_TICKET_DESCRIPTION_LENGTH) {
      alert(
        `Description must not exceed ${MAX_TICKET_DESCRIPTION_LENGTH} characters.`
      );
      return;
    }

    // Duplicate ticket name validation
    const isDuplicate = ticketTypes.some(
      (ticket) =>
        ticket.id !== editingTicketId &&
        ticket.name.trim().toLowerCase() === name.toLowerCase()
    );

    if (isDuplicate) {
      alert("A ticket type with this name already exists.");
      return;
    }

    const ticketData = {
      name,
      price,
      capacity,
      description,
    };

    if (editingTicketId) {
      updateTicketType(editingTicketId, ticketData);
    } else {
      addTicketType(ticketData);
    }

    closeTicketModal();
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
              <h1 className="text-xl font-bold text-[#141b2b]">
                Ticket Types
              </h1>

              <p className="text-xs text-[#565e74] mt-1">
                Create different ticket tiers, set quantities, and manage
                pricing for general admission.
              </p>
            </div>

            <button
              type="button"
              onClick={openAddTicketModal}
              className="flex items-center gap-2 bg-[#00685f] hover:bg-[#005550] text-white text-xs font-semibold px-4 py-2.5 rounded-lg shadow-sm transition-colors cursor-pointer"
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

              <span>Add Ticket Type</span>
            </button>
          </div>

          <div className="p-6 flex flex-col gap-4">
            {ticketTypes.length === 0 ? (
              <div className="p-8 text-center text-xs text-[#565e74] border border-dashed border-[#bcc9c6] rounded-xl">
                No ticket types created yet. Click &quot;Add Ticket Type&quot;
                above to create one.
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

                      <p className="text-xs text-[#565e74] mt-0.5">
                        {ticket.description}
                      </p>
                    </div>
                  </div>

                  <div className="flex items-center gap-6 self-end md:self-center">
                    <div className="flex flex-col text-right">
                      <span className="text-base font-bold text-[#141b2b]">
                        ${Number(ticket.price).toFixed(2)}
                      </span>

                      <span className="text-[11px] text-[#565e74]">
                        {ticket.capacity} tickets total
                      </span>
                    </div>

                    <div className="flex items-center gap-2">
                      {/* Edit */}
                      <button
                        type="button"
                        onClick={() => handleEditTicket(ticket)}
                        className="p-1.5 text-gray-400 hover:text-[#00685f] rounded transition-colors cursor-pointer"
                        title="Edit ticket"
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
                            d="M16.862 3.487a2.25 2.25 0 013.182 3.182L8.25 18.464 4 19.5l1.036-4.25L16.862 3.487z"
                          />
                        </svg>
                      </button>

                      {/* Delete */}
                      <button
                        type="button"
                        onClick={() => removeTicketType(ticket.id)}
                        className="p-1.5 text-gray-400 hover:text-red-600 rounded transition-colors cursor-pointer"
                        title="Remove ticket"
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
                            d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1 1v3M4 7h16"
                          />
                        </svg>
                      </button>
                    </div>
                  </div>
                </div>
              ))
            )}
          </div>
        </div>

        {/* Card 2: Ticketing Sales Settings */}
        <div className="bg-white border border-[#bcc9c6]/50 rounded-2xl shadow-sm overflow-hidden flex flex-col">
          <div className="p-6 border-b border-[#bcc9c6]/30 bg-[#f9f9ff]">
            <h2 className="text-base font-bold text-[#141b2b]">
              Ticketing Settings
            </h2>

            <p className="text-xs text-[#565e74] mt-1">
              Sales windows, maximum tickets allowed per customer, and
              purchase rules.
            </p>
          </div>

          <div className="p-6 grid grid-cols-1 md:grid-cols-3 gap-5">
            {/* Sales Start Date */}
            <div className="flex flex-col gap-2">
              <label className="text-[13px] font-semibold text-[#141b2b]">
                Sales Start Date
              </label>

              <input
                type="date"
                value={ticketSalesSettings.salesStartDate}
                onChange={(e) =>
                  updateTicketSalesSettings({
                    salesStartDate: e.target.value,
                  })
                }
                className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg px-4 py-2.5 text-sm text-[#141b2b] focus:outline-none focus:border-[#00685f]"
              />
            </div>

            {/* Sales End Date */}
            <div className="flex flex-col gap-2">
              <label className="text-[13px] font-semibold text-[#141b2b]">
                Sales End Date
              </label>

              <input
                type="date"
                value={ticketSalesSettings.salesEndDate}
                onChange={(e) =>
                  updateTicketSalesSettings({
                    salesEndDate: e.target.value,
                  })
                }
                className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg px-4 py-2.5 text-sm text-[#141b2b] focus:outline-none focus:border-[#00685f]"
              />
            </div>

            {/* Maximum Tickets */}
            <div className="flex flex-col gap-2">
              <label className="text-[13px] font-semibold text-[#141b2b]">
                Max Tickets Per Order
              </label>

              <input
                type="number"
                value={ticketSalesSettings.maxTicketsPerBooking}
                min={1}
                max={20}
                onChange={(e) =>
                  updateTicketSalesSettings({
                    maxTicketsPerBooking: e.target.value,
                  })
                }
                className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg px-4 py-2.5 text-sm text-[#141b2b] focus:outline-none focus:border-[#00685f]"
              />
            </div>
          </div>
        </div>

        {/* Card 3: Payout Account */}
        <PayoutBankAccountCard />
      </div>

      {/* Add / Edit Ticket Modal */}
      {showAddModal && (
        <div className="fixed inset-0 z-50 bg-black/40 backdrop-blur-xs flex items-center justify-center p-4">
          <div className="bg-white rounded-2xl max-w-md w-full p-6 shadow-xl border border-[#bcc9c6] flex flex-col gap-5">
            <div className="flex items-center justify-between">
              <h3 className="font-bold text-lg text-[#141b2b]">
                {editingTicketId
                  ? "Edit Ticket Type"
                  : "Add New Ticket Type"}
              </h3>

              <button
                type="button"
                onClick={closeTicketModal}
                className="text-gray-400 hover:text-gray-700 text-lg"
                title="Close"
              >
                &times;
              </button>
            </div>

            <form
              onSubmit={handleCreateTicket}
              className="flex flex-col gap-4"
            >
              {/* Ticket Name */}
              <div className="flex flex-col gap-1.5">
                <label className="text-xs font-semibold text-[#141b2b]">
                  Ticket Name{" "}
                  <span className="text-[#ba1a1a]">*</span>
                </label>

                <input
                  type="text"
                  required
                  maxLength={MAX_TICKET_NAME_LENGTH}
                  placeholder="e.g. VIP Pass, Student Discount"
                  value={newTicket.name}
                  onChange={(e) =>
                    setNewTicket({
                      ...newTicket,
                      name: e.target.value,
                    })
                  }
                  className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-[#00685f]"
                />
              </div>

              {/* Price and Quantity */}
              <div className="grid grid-cols-2 gap-4">
                {/* Price */}
                <div className="flex flex-col gap-1.5">
                  <label className="text-xs font-semibold text-[#141b2b]">
                    Price ($){" "}
                    <span className="text-[#ba1a1a]">*</span>
                  </label>

                  <input
                    type="number"
                    required
                    min={0}
                    step="0.01"
                    placeholder="99"
                    value={newTicket.price}
                    onChange={(e) =>
                      setNewTicket({
                        ...newTicket,
                        price: e.target.value,
                      })
                    }
                    className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-[#00685f]"
                  />
                </div>

                {/* Quantity */}
                <div className="flex flex-col gap-1.5">
                  <label className="text-xs font-semibold text-[#141b2b]">
                    Quantity Available{" "}
                    <span className="text-[#ba1a1a]">*</span>
                  </label>

                  <input
                    type="number"
                    required
                    min={1}
                    max={MAX_TICKET_QUANTITY}
                    step={1}
                    placeholder="500"
                    value={newTicket.capacity}
                    onChange={(e) =>
                      setNewTicket({
                        ...newTicket,
                        capacity: e.target.value,
                      })
                    }
                    className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-[#00685f]"
                  />
                </div>
              </div>

              {/* Description */}
              <div className="flex flex-col gap-1.5">
                <label className="text-xs font-semibold text-[#141b2b]">
                  Description
                </label>

                <textarea
                  rows={2}
                  maxLength={MAX_TICKET_DESCRIPTION_LENGTH}
                  placeholder="What is included with this ticket?"
                  value={newTicket.description}
                  onChange={(e) =>
                    setNewTicket({
                      ...newTicket,
                      description: e.target.value,
                    })
                  }
                  className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg p-3 text-sm focus:outline-none focus:border-[#00685f]"
                />

                <span className="text-[10px] text-[#565e74] text-right">
                  {newTicket.description.length}/
                  {MAX_TICKET_DESCRIPTION_LENGTH}
                </span>
              </div>

              {/* Actions */}
              <div className="flex items-center justify-end gap-3 mt-2">
                <button
                  type="button"
                  onClick={closeTicketModal}
                  className="px-4 py-2 rounded-lg border border-[#bcc9c6] text-xs font-semibold text-[#565e74] hover:bg-gray-50"
                >
                  Cancel
                </button>

                <button
                  type="submit"
                  className="px-5 py-2 rounded-lg bg-[#00685f] hover:bg-[#005550] text-white text-xs font-semibold shadow-sm"
                >
                  {editingTicketId ? "Save Changes" : "Create Ticket"}
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