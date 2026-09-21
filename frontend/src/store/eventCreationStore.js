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
  title: "",
  category: "",
  categoryId: "",
  description: "",
  banner: "",
  bannerFile: null,
  bannerPreviewUrl: "",
  language: "",
  ageRestriction: "",
};


const initialDateTime = {
  eventDate: "",
  startTime: "",
  endTime: "",
  isAllDay: false,
};

const initialEventDetails = {
  highlights: "",
  rules: "",
  contactInformation: {
    name: "",
    phone: "",
    email: "",
  },
  cancellationPolicy: {
    allowCancellation: true,
    cancellationDeadline: "48 hours before event",
    refundPolicy: "PARTIAL",
    refundPercentage: 80,
    organizerPolicyAccepted: false,
  },
  attendeeInformation: "",
  visibility: "PUBLIC",
};

const initialTicketTypes = []

const initialPayoutAccount = {
  bankName: "Silicon Valley Bank",
  accountHolder: "Sunfield Entertainment LLC",
  accountNumberMasked: "•••• •••• •••• 4892",
};

const initialSeatingConfig = {
  layoutName: "",
  categories: [],
  sections: [],
  rows: [],
  selectedSeatIds: [],
  totalCapacity: 0,
};

const pruneSelectedSeatIds = (oldCategories, newCategories, selectedSeatIds) => {
  if (!selectedSeatIds || selectedSeatIds.length === 0) {
    return [];
  }

  // Map old seat ID to its row ID
  const oldSeatRowMap = new Map();
  for (const cat of oldCategories || []) {
    for (const row of cat.rows || []) {
      for (const seat of row.seats || []) {
        if (seat.id) {
          oldSeatRowMap.set(seat.id, row.id);
        }
      }
    }
  }

  // Map new seat ID to its row ID
  const newSeatRowMap = new Map();
  for (const cat of newCategories || []) {
    for (const row of cat.rows || []) {
      for (const seat of row.seats || []) {
        if (seat.id) {
          newSeatRowMap.set(seat.id, row.id);
        }
      }
    }
  }

  // Only keep seatId if it exists in the new layout AND belongs to the exact same row
  return selectedSeatIds.filter((seatId) => {
    const oldRowId = oldSeatRowMap.get(seatId);
    const newRowId = newSeatRowMap.get(seatId);
    return oldRowId && newRowId && oldRowId === newRowId;
  });
};

const rebuildLayout = (categories, seatsPerRow = 8) => {
  let rowIndex = 0;

  return categories.map((category) => {
    const updatedRows = (category.rows || []).map((row) => {
      const rowLetter =
        rowIndex < 26
          ? String.fromCharCode(65 + rowIndex)
          : `R${rowIndex + 1}`;

      rowIndex++;

      const seats = Array.from(
        { length: seatsPerRow },
        (_, seatIndex) => {
          const number = seatIndex + 1;

          const existingSeat = row.seats?.[seatIndex];

          return {
            ...existingSeat,
            id: `${rowLetter}${number}`,
            number,
            status: existingSeat?.status || "available",
            categoryId: existingSeat?.categoryId || category.id,
          };
        }
      );

      return {
        ...row,
        id: row.id || `row-${rowLetter}-${category.id}`,
        name: `Row ${rowLetter}`,
        rowLetter,
        seats,
      };
    });

    return {
      ...category,
      rows: updatedRows,
    };
  });
};



const useEventCreationStore = create((set, get) => ({
  // Location & Event Type
  locationType: LOCATION_TYPES.PHYSICAL,
  eventType: LOCATION_TYPES.PHYSICAL,
  onlineUrl: "",

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
    salesStartDate: "",
    salesEndDate: "",
    maxTicketsPerBooking: "",
  },
  payoutAccount: { ...initialPayoutAccount },

  // Step 4B: Seat Configuration (for Reserved Seating)
  seatingConfiguration: {
    ...initialSeatingConfig,
  },

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
        nextSeatingConfig = { ...initialSeatingConfig };
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

  updateTicketSalesSettings: (data) =>
    set((state) => ({
      ticketSalesSettings: {
        ...state.ticketSalesSettings,
        ...data,
      },
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

      const updatedCategories = rebuildLayout(
        [...state.seatingConfiguration.categories, newCategory],
        seatsPerRow
      );
      const allRows = updatedCategories.flatMap((c) => c.rows);
      const prunedSelected = pruneSelectedSeatIds(
        state.seatingConfiguration.categories,
        updatedCategories,
        state.seatingConfiguration.selectedSeatIds
      );

      return {
        seatingConfiguration: {
          ...state.seatingConfiguration,
          categories: updatedCategories,
          sections: updatedCategories,
          rows: allRows,
          selectedSeatIds: prunedSelected,
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
      const updatedCategories = rebuildLayout(remainingCategories, seatsPerRow);
      const allRows = updatedCategories.flatMap((c) => c.rows);
      const prunedSelected = pruneSelectedSeatIds(
        state.seatingConfiguration.categories,
        updatedCategories,
        state.seatingConfiguration.selectedSeatIds
      );

      return {
        seatingConfiguration: {
          ...state.seatingConfiguration,
          categories: updatedCategories,
          sections: updatedCategories,
          rows: allRows,
          selectedSeatIds: prunedSelected,
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

      const resequenced = rebuildLayout(updated, seatsPerRow);
      const allRows = resequenced.flatMap((c) => c.rows);
      const prunedSelected = pruneSelectedSeatIds(
        state.seatingConfiguration.categories,
        resequenced,
        state.seatingConfiguration.selectedSeatIds
      );

      return {
        seatingConfiguration: {
          ...state.seatingConfiguration,
          categories: resequenced,
          sections: resequenced,
          rows: allRows,
          selectedSeatIds: prunedSelected,
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

      const resequenced = rebuildLayout(updated, seatsPerRow);
      const allRows = resequenced.flatMap((c) => c.rows);
      const prunedSelected = pruneSelectedSeatIds(
        state.seatingConfiguration.categories,
        resequenced,
        state.seatingConfiguration.selectedSeatIds
      );

      return {
        seatingConfiguration: {
          ...state.seatingConfiguration,
          categories: resequenced,
          sections: resequenced,
          rows: allRows,
          selectedSeatIds: prunedSelected,
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

      const resequenced = rebuildLayout(updatedCategories, seatsPerRow);
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
      const currentSelected = state.seatingConfiguration.selectedSeatIds || [];
      const isSelected = currentSelected.includes(seatId);
      const newSelected = isSelected
        ? currentSelected.filter((id) => id !== seatId)
        : [...currentSelected, seatId];

      return {
        seatingConfiguration: {
          ...state.seatingConfiguration,
          selectedSeatIds: newSelected,
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
          selectedSeatIds: [],
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
  resetForm: () => {
    const currentPreview = get().basicInformation?.bannerPreviewUrl;
    if (currentPreview) {
      URL.revokeObjectURL(currentPreview);
    }

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
    });
  },
}));

export default useEventCreationStore;
