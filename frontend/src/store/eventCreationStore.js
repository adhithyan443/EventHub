import { create } from "zustand";
import { TICKET_MODES, LOCATION_TYPES, isOnlineEvent } from "../constants/eventConstants";

const initialVenue = {
  google_place_id: "",
  name: "",
  address: "",
  city: "",
  state: "",
  country: "",
  postal_code: "",
  latitude: null,
  longitude: null,
};

const initialBasicInfo = {
  title: "Sunfield Music Festival 2026",
  category: "Music",
  description:
    "An electrifying weekend festival featuring top international artists, live DJ sets, artisan food markets, and unforgettable stage performances.",
  banner: "",
  language: "English",
  ageRestriction: "18+",
};

const initialDateTime = {
  eventDate: "2026-10-15",
  startTime: "18:00",
  endTime: "23:00",
  isAllDay: false,
};

const initialEventDetails = {
  highlights:
    "• 3 Live Stages with 20+ Artists\n• Dedicated Food & Beverage Village\n• VIP Lounge & Priority Stage Viewing",
  rules:
    "• Government ID required at the gate\n• No outside food or beverages allowed\n• Strictly 18+ event",
  contactInformation: {
    name: "Sunfield Operations Desk",
    phone: "+1 (555) 234-5678",
    email: "support@sunfieldfest.com",
  },
  cancellationPolicy: {
    allowCancellation: true,
    cancellationDeadline: "48 hours before event",
    refundPolicy: "PARTIAL", // "FULL" | "PARTIAL" | "NO_REFUND"
    refundPercentage: 80,
    organizerPolicyAccepted: true,
  },
  attendeeInformation: "Full Name, Email, Phone Number, Emergency Contact",
  visibility: "PUBLIC", // "PUBLIC" | "PRIVATE"
};

const initialTicketTypes = [
  {
    id: "ticket-1",
    name: "VIP Early Access",
    price: 149,
    capacity: 250,
    status: "ACTIVE",
    description: "Includes priority entry, VIP lounge access, and complimentary drink voucher.",
  },
  {
    id: "ticket-2",
    name: "General Admission",
    price: 79,
    capacity: 1200,
    status: "ACTIVE",
    description: "Full access to all general admission areas and festival stages.",
  },
  {
    id: "ticket-3",
    name: "Early Bird General Admission",
    price: 59,
    capacity: 300,
    status: "SOLD_OUT",
    description: "Discounted admission for early supporters.",
  },
];

const initialPayoutAccount = {
  bankName: "Silicon Valley Bank",
  accountHolder: "Sunfield Entertainment LLC",
  accountNumberMasked: "•••• •••• •••• 4892",
};

const LETTERS = [
  "A", "B", "C", "D", "E", "F", "G", "H", "I", "J",
  "K", "L", "M", "N", "O", "P", "Q", "R", "S", "T",
  "U", "V", "W", "X", "Y", "Z",
];

const makeSeatsForRow = (rowLetter, seatsPerRow, categoryId) => {
  const seats = [];
  for (let s = 1; s <= seatsPerRow; s++) {
    let status = "available";
    let seatCatId = categoryId;

    // Demo state matching Figma reference
    if (rowLetter === "A" && (s === 4 || s === 5 || s === 6)) {
      status = "selected";
      seatCatId = "cat-vip";
    } else if (rowLetter === "B" && (s === 2 || s === 3)) {
      status = "reserved";
      seatCatId = "cat-premium";
    } else if (rowLetter === "B" && (s === 4 || s === 5)) {
      status = "blocked";
      seatCatId = "cat-premium";
    }

    seats.push({
      id: `${rowLetter}${s}`,
      number: s,
      status,
      categoryId: seatCatId,
    });
  }
  return seats;
};

export const resequenceLayout = (categories, seatsPerRow = 8) => {
  let globalRowIdx = 0;
  return categories.map((cat) => {
    const updatedRows = (cat.rows || []).map((row) => {
      const rowLetter = LETTERS[globalRowIdx] || `R${globalRowIdx + 1}`;
      globalRowIdx++;

      const seats = (row.seats && row.seats.length > 0
        ? row.seats
        : Array.from({ length: seatsPerRow }, (_, i) => ({
            id: `${rowLetter}${i + 1}`,
            number: i + 1,
            status: "available",
            categoryId: cat.id,
          }))
      ).map((seat, sIdx) => {
        const seatNum = seat.number || sIdx + 1;
        return {
          ...seat,
          id: `${rowLetter}${seatNum}`,
          number: seatNum,
          categoryId: seat.categoryId || cat.id,
        };
      });

      return {
        ...row,
        id: row.id || `row-${rowLetter}-${cat.id}`,
        name: `Row ${rowLetter}`,
        rowLetter,
        seats,
      };
    });

    return {
      ...cat,
      rows: updatedRows,
    };
  });
};

const createInitialCategories = (seatsPerRow = 8) => {
  const initial = [
    {
      id: "cat-vip",
      name: "VIP",
      tier: "VIP",
      price: "₹1,499",
      color: "#00685f",
      rows: [
        {
          id: "row-A",
          name: "Row A",
          rowLetter: "A",
          seats: makeSeatsForRow("A", seatsPerRow, "cat-vip"),
        },
      ],
    },
    {
      id: "cat-premium",
      name: "Premium",
      tier: "Premium",
      price: "₹2,499",
      color: "#4648d4",
      rows: [
        {
          id: "row-B",
          name: "Row B",
          rowLetter: "B",
          seats: makeSeatsForRow("B", seatsPerRow, "cat-premium"),
        },
      ],
    },
    {
      id: "cat-regular",
      name: "Regular",
      tier: "Regular",
      price: "₹799",
      color: "#6d7a77",
      rows: [
        {
          id: "row-C",
          name: "Row C",
          rowLetter: "C",
          seats: makeSeatsForRow("C", seatsPerRow, "cat-regular"),
        },
        {
          id: "row-D",
          name: "Row D",
          rowLetter: "D",
          seats: makeSeatsForRow("D", seatsPerRow, "cat-regular"),
        },
      ],
    },
  ];

  return resequenceLayout(initial, seatsPerRow);
};

const initialCategories = createInitialCategories(8);
const initialAllRows = initialCategories.flatMap((c) => c.rows);

const initialSeatingConfig = {
  stageName: "STAGE",
  seatsPerRow: 8,
  selectedSeatIds: ["A4", "A5", "A6"],
  categories: initialCategories,
  sections: initialCategories, // Alias for backwards compatibility
  rows: initialAllRows,
  totalCapacity: initialAllRows.reduce((sum, r) => sum + r.seats.length, 0),
};

const useEventCreationStore = create((set, get) => ({
  // Location & Event Type
  locationType: LOCATION_TYPES.PHYSICAL,
  eventType: LOCATION_TYPES.PHYSICAL,
  onlineUrl: "https://zoom.us/j/sunfield-live-stream",

  // Venue (Google Places payload)
  venue: { ...initialVenue },

  // Step 1: Basic Information
  basicInformation: { ...initialBasicInfo },

  // Step 2: Date & Time + Details
  dateTime: { ...initialDateTime },
  eventDetails: { ...initialEventDetails },

  // Step 3: Ticket Mode
  ticketMode: TICKET_MODES.GENERAL,

  // Step 4A: Ticket Types (for General Admission)
  ticketTypes: initialTicketTypes,
  ticketSalesSettings: {
    salesStartDate: "2026-09-15",
    salesEndDate: "2026-10-14",
    maxTicketsPerBooking: 6,
  },
  payoutAccount: { ...initialPayoutAccount },

  // Step 4B: Seat Configuration (for Reserved Seating)
  seatingConfiguration: initialSeatingConfig,

  // UI state actions
  setLocationType: (type) =>
    set((state) => {
      const isSwitchingToOnline = isOnlineEvent(type);

      let nextVenue = state.venue;
      let nextTicketMode = state.ticketMode;
      let nextSeatingConfig = state.seatingConfiguration;

      if (isSwitchingToOnline) {
        // Clear physical-only state
        nextVenue = { ...initialVenue };
        nextTicketMode = TICKET_MODES.GENERAL;
        const freshCategories = createInitialCategories(8);
        const freshRows = freshCategories.flatMap((c) => c.rows);
        nextSeatingConfig = {
          stageName: "STAGE",
          seatsPerRow: 8,
          selectedSeatIds: [],
          categories: freshCategories,
          sections: freshCategories,
          rows: freshRows,
          totalCapacity: freshRows.reduce((sum, r) => sum + r.seats.length, 0),
        };
      }

      return {
        locationType: type,
        eventType: type,
        venue: nextVenue,
        ticketMode: nextTicketMode,
        seatingConfiguration: nextSeatingConfig,
      };
    }),

  setEventType: (type) => get().setLocationType(type),
  setOnlineUrl: (url) => set({ onlineUrl: url }),

  setVenue: (venueData) =>
    set((state) => ({
      venue: {
        ...state.venue,
        ...venueData,
      },
    })),

  setTicketMode: (mode) =>
    set((state) => {
      // ONLINE events cannot use SEATED/RESERVED mode
      if (isOnlineEvent(state.locationType || state.eventType) && mode === TICKET_MODES.SEATED) {
        return { ticketMode: TICKET_MODES.GENERAL };
      }
      return { ticketMode: mode };
    }),

  updateBasicInformation: (data) =>
    set((state) => ({
      basicInformation: {
        ...state.basicInformation,
        ...data,
      },
    })),

  updateDateTime: (data) =>
    set((state) => ({
      dateTime: {
        ...state.dateTime,
        ...data,
      },
    })),

  updateEventDetails: (data) =>
    set((state) => ({
      eventDetails: {
        ...state.eventDetails,
        ...data,
      },
    })),

  updateContactInformation: (data) =>
    set((state) => ({
      eventDetails: {
        ...state.eventDetails,
        contactInformation: {
          ...state.eventDetails.contactInformation,
          ...data,
        },
      },
    })),

  updateCancellationPolicy: (policyUpdates) =>
    set((state) => ({
      eventDetails: {
        ...state.eventDetails,
        cancellationPolicy: {
          ...state.eventDetails.cancellationPolicy,
          ...policyUpdates,
        },
      },
    })),

  updateVisibility: (visibility) =>
    set((state) => ({
      eventDetails: {
        ...state.eventDetails,
        visibility,
      },
    })),

  // Ticket types actions
  addTicketType: (newTicket) =>
    set((state) => ({
      ticketTypes: [
        ...state.ticketTypes,
        {
          id: `ticket-${Date.now()}`,
          status: "ACTIVE",
          ...newTicket,
        },
      ],
    })),

  removeTicketType: (id) =>
    set((state) => ({
      ticketTypes: state.ticketTypes.filter((t) => t.id !== id),
    })),

  updateTicketType: (id, updatedFields) =>
    set((state) => ({
      ticketTypes: state.ticketTypes.map((t) =>
        t.id === id ? { ...t, ...updatedFields } : t
      ),
    })),

  // Seating configuration actions
  addCategory: ({ name, price, tier = "Standard", color = "#00685f", rowsCount = 1 }) =>
    set((state) => {
      const seatsPerRow = state.seatingConfiguration.seatsPerRow || 8;
      const validRowsCount = Math.max(1, Math.min(20, Number(rowsCount) || 1));
      const formattedPrice =
        typeof price === "string" && price.startsWith("₹")
          ? price
          : `₹${price || 0}`;

      const newCategory = {
        id: `cat-${Date.now()}`,
        name: name.trim() || "New Tier",
        tier: tier || "Standard",
        price: formattedPrice,
        color: color || "#00685f",
        rows: Array.from({ length: validRowsCount }, (_, rIdx) => ({
          id: `row-temp-${rIdx}-${Date.now()}`,
          name: `Row`,
          rowLetter: "",
          seats: Array.from({ length: seatsPerRow }, (_, sIdx) => ({
            id: "",
            number: sIdx + 1,
            status: "available",
            categoryId: "",
          })),
        })),
      };

      const updatedCategories = resequenceLayout(
        [...state.seatingConfiguration.categories, newCategory],
        seatsPerRow
      );
      const allRows = updatedCategories.flatMap((c) => c.rows);

      return {
        seatingConfiguration: {
          ...state.seatingConfiguration,
          categories: updatedCategories,
          sections: updatedCategories,
          rows: allRows,
          totalCapacity: allRows.reduce((sum, r) => sum + r.seats.length, 0),
        },
      };
    }),

  removeCategory: (categoryId) =>
    set((state) => {
      if (state.seatingConfiguration.categories.length <= 1) {
        return state; // Prevent removing the last category
      }

      const remainingCategories = state.seatingConfiguration.categories.filter(
        (c) => c.id !== categoryId
      );
      const seatsPerRow = state.seatingConfiguration.seatsPerRow || 8;
      const updatedCategories = resequenceLayout(remainingCategories, seatsPerRow);
      const allRows = updatedCategories.flatMap((c) => c.rows);
      const validSeatIds = new Set(allRows.flatMap((r) => r.seats.map((s) => s.id)));

      return {
        seatingConfiguration: {
          ...state.seatingConfiguration,
          categories: updatedCategories,
          sections: updatedCategories,
          rows: allRows,
          selectedSeatIds: state.seatingConfiguration.selectedSeatIds.filter((id) =>
            validSeatIds.has(id)
          ),
          totalCapacity: allRows.reduce((sum, r) => sum + r.seats.length, 0),
        },
      };
    }),

  updateCategory: (categoryId, updatedFields) =>
    set((state) => {
      const updatedCategories = state.seatingConfiguration.categories.map((cat) => {
        if (cat.id === categoryId) {
          return {
            ...cat,
            ...updatedFields,
          };
        }
        return cat;
      });

      return {
        seatingConfiguration: {
          ...state.seatingConfiguration,
          categories: updatedCategories,
          sections: updatedCategories,
        },
      };
    }),

  setCategoryRowCount: (categoryId, newRowCount) =>
    set((state) => {
      const validCount = Math.max(1, Math.min(20, Number(newRowCount) || 1));
      const seatsPerRow = state.seatingConfiguration.seatsPerRow || 8;

      const updated = state.seatingConfiguration.categories.map((cat) => {
        if (cat.id !== categoryId) return cat;

        const currentRows = cat.rows || [];
        if (currentRows.length === validCount) return cat;

        let nextRows;
        if (validCount > currentRows.length) {
          const needed = validCount - currentRows.length;
          const newRows = Array.from({ length: needed }, (_, i) => ({
            id: `row-new-${Date.now()}-${i}`,
            name: "",
            rowLetter: "",
            seats: Array.from({ length: seatsPerRow }, (_, sIdx) => ({
              id: "",
              number: sIdx + 1,
              status: "available",
              categoryId: cat.id,
            })),
          }));
          nextRows = [...currentRows, ...newRows];
        } else {
          nextRows = currentRows.slice(0, validCount);
        }

        return {
          ...cat,
          rows: nextRows,
        };
      });

      const resequenced = resequenceLayout(updated, seatsPerRow);
      const allRows = resequenced.flatMap((c) => c.rows);

      return {
        seatingConfiguration: {
          ...state.seatingConfiguration,
          categories: resequenced,
          sections: resequenced,
          rows: allRows,
          totalCapacity: allRows.reduce((sum, r) => sum + r.seats.length, 0),
        },
      };
    }),

  addRowToCategory: (categoryId) => {
    const state = get();
    const cat = state.seatingConfiguration.categories.find((c) => c.id === categoryId);
    if (cat) {
      state.setCategoryRowCount(categoryId, (cat.rows?.length || 0) + 1);
    }
  },

  removeRowFromCategory: (categoryId, rowId) =>
    set((state) => {
      const seatsPerRow = state.seatingConfiguration.seatsPerRow || 8;
      const updated = state.seatingConfiguration.categories.map((cat) => {
        if (cat.id !== categoryId) return cat;
        if (cat.rows.length <= 1) return cat; // Keep at least 1 row

        const nextRows = rowId
          ? cat.rows.filter((r) => r.id !== rowId)
          : cat.rows.slice(0, -1);

        return {
          ...cat,
          rows: nextRows,
        };
      });

      const resequenced = resequenceLayout(updated, seatsPerRow);
      const allRows = resequenced.flatMap((c) => c.rows);
      const validSeatIds = new Set(allRows.flatMap((r) => r.seats.map((s) => s.id)));

      return {
        seatingConfiguration: {
          ...state.seatingConfiguration,
          categories: resequenced,
          sections: resequenced,
          rows: allRows,
          selectedSeatIds: state.seatingConfiguration.selectedSeatIds.filter((id) =>
            validSeatIds.has(id)
          ),
          totalCapacity: allRows.reduce((sum, r) => sum + r.seats.length, 0),
        },
      };
    }),

  generateLayout: (newSeatsPerRow) =>
    set((state) => {
      const seatsPerRow = Math.max(1, Math.min(30, Number(newSeatsPerRow) || state.seatingConfiguration.seatsPerRow || 8));

      // Re-generate seats for every row across all existing categories preserving each category's row count
      const updatedCategories = state.seatingConfiguration.categories.map((cat) => {
        const rows = (cat.rows || []).map((row) => ({
          ...row,
          seats: Array.from({ length: seatsPerRow }, (_, sIdx) => ({
            id: "",
            number: sIdx + 1,
            status: "available",
            categoryId: cat.id,
          })),
        }));

        return {
          ...cat,
          rows,
        };
      });

      const resequenced = resequenceLayout(updatedCategories, seatsPerRow);
      const allRows = resequenced.flatMap((c) => c.rows);

      return {
        seatingConfiguration: {
          ...state.seatingConfiguration,
          seatsPerRow,
          categories: resequenced,
          sections: resequenced,
          rows: allRows,
          selectedSeatIds: [],
          totalCapacity: allRows.reduce((sum, r) => sum + r.seats.length, 0),
        },
      };
    }),

  toggleSeatSelection: (seatId) =>
    set((state) => {
      const isSelected = state.seatingConfiguration.selectedSeatIds.includes(seatId);
      const newSelected = isSelected
        ? state.seatingConfiguration.selectedSeatIds.filter((id) => id !== seatId)
        : [...state.seatingConfiguration.selectedSeatIds, seatId];

      const updatedCategories = state.seatingConfiguration.categories.map((cat) => ({
        ...cat,
        rows: cat.rows.map((row) => ({
          ...row,
          seats: row.seats.map((seat) => {
            if (seat.id === seatId) {
              return {
                ...seat,
                status: isSelected ? "available" : "selected",
              };
            }
            return seat;
          }),
        })),
      }));

      const allRows = updatedCategories.flatMap((c) => c.rows);

      return {
        seatingConfiguration: {
          ...state.seatingConfiguration,
          selectedSeatIds: newSelected,
          categories: updatedCategories,
          sections: updatedCategories,
          rows: allRows,
        },
      };
    }),

  setSelectedSeatsStatus: (newStatus) =>
    set((state) => {
      const selectedIds = new Set(state.seatingConfiguration.selectedSeatIds);
      const updatedCategories = state.seatingConfiguration.categories.map((cat) => ({
        ...cat,
        rows: cat.rows.map((row) => ({
          ...row,
          seats: row.seats.map((seat) => {
            if (selectedIds.has(seat.id)) {
              return {
                ...seat,
                status: newStatus,
              };
            }
            return seat;
          }),
        })),
      }));

      const allRows = updatedCategories.flatMap((c) => c.rows);

      return {
        seatingConfiguration: {
          ...state.seatingConfiguration,
          categories: updatedCategories,
          sections: updatedCategories,
          rows: allRows,
          selectedSeatIds:
            newStatus === "available" ? [] : state.seatingConfiguration.selectedSeatIds,
        },
      };
    }),

  assignCategoryToSelectedSeats: (categoryId) =>
    set((state) => {
      const selectedIds = new Set(state.seatingConfiguration.selectedSeatIds);

      const updatedCategories = state.seatingConfiguration.categories.map((cat) => ({
        ...cat,
        rows: cat.rows.map((row) => ({
          ...row,
          seats: row.seats.map((seat) => {
            if (selectedIds.has(seat.id)) {
              return {
                ...seat,
                categoryId,
              };
            }
            return seat;
          }),
        })),
      }));

      const allRows = updatedCategories.flatMap((c) => c.rows);

      return {
        seatingConfiguration: {
          ...state.seatingConfiguration,
          categories: updatedCategories,
          sections: updatedCategories,
          rows: allRows,
        },
      };
    }),

  updateStepData: (stepKey, data) =>
    set((state) => ({
      [stepKey]: {
        ...state[stepKey],
        ...data,
      },
    })),

  updatePayoutAccount: (accountData) =>
    set((state) => ({
      payoutAccount: {
        ...state.payoutAccount,
        ...accountData,
      },
    })),

  // Reset form to default
  resetForm: () =>
    set({
      locationType: LOCATION_TYPES.PHYSICAL,
      eventType: LOCATION_TYPES.PHYSICAL,
      venue: { ...initialVenue },
      basicInformation: { ...initialBasicInfo },
      dateTime: { ...initialDateTime },
      eventDetails: { ...initialEventDetails },
      ticketMode: TICKET_MODES.GENERAL,
      ticketTypes: initialTicketTypes,
      seatingConfiguration: initialSeatingConfig,
      payoutAccount: { ...initialPayoutAccount },
    }),
}));

export default useEventCreationStore;
