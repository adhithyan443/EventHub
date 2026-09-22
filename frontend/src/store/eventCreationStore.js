import { create } from "zustand";

import {
  TICKET_MODES,
  LOCATION_TYPES,
  isOnlineEvent,
} from "../constants/eventConstants";

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

const initialTicketTypes = [];

const initialTicketSalesSettings = {
  salesStartDate: "",
  salesEndDate: "",
  maxTicketsPerBooking: "",
};

// const initialPayoutAccount = {
//   bankName: "",
//   accountHolder: "",
//   accountNumberMasked: "",
// };

const initialSeatingConfig = {
  layoutName: "",
  categories: [],
  sections: [],
  rows: [],
  selectedSeatIds: [],
  totalCapacity: 0,
};

const pruneSelectedSeatIds = (
  oldCategories,
  newCategories,
  selectedSeatIds
) => {
  if (!selectedSeatIds || selectedSeatIds.length === 0) {
    return [];
  }

  const oldSeatRowMap = new Map();

  for (const category of oldCategories || []) {
    for (const row of category.rows || []) {
      for (const seat of row.seats || []) {
        if (seat.id) {
          oldSeatRowMap.set(seat.id, row.id);
        }
      }
    }
  }

  const newSeatRowMap = new Map();

  for (const category of newCategories || []) {
    for (const row of category.rows || []) {
      for (const seat of row.seats || []) {
        if (seat.id) {
          newSeatRowMap.set(seat.id, row.id);
        }
      }
    }
  }

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

  // Venue
  venue: { ...initialVenue },

  // Step 1: Basic Information
  basicInformation: { ...initialBasicInfo },

  // Step 2: Date & Time + Details
  dateTime: { ...initialDateTime },
  eventDetails: { ...initialEventDetails },

  // Step 3: Ticket Mode
  ticketMode: TICKET_MODES.GENERAL,

  // Step 4A: Ticket Types
  ticketTypes: initialTicketTypes,
  ticketSalesSettings: {
    ...initialTicketSalesSettings,
  },

  // Payout account
  // payoutAccount: { ...initialPayoutAccount },

  // Step 4B: Seat Configuration
  seatingConfiguration: {
    ...initialSeatingConfig,
  },

  // --------------------------------------------------
  // Location & Event Type Actions
  // --------------------------------------------------

  setLocationType: (type) =>
    set((state) => {
      const isSwitchingToOnline = isOnlineEvent(type);

      let nextVenue = state.venue;
      let nextTicketMode = state.ticketMode;
      let nextSeatingConfig = state.seatingConfiguration;

      if (isSwitchingToOnline) {
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

  setOnlineUrl: (url) =>
    set({
      onlineUrl: url,
    }),

  setVenue: (venueData) =>
    set((state) => ({
      venue: {
        ...state.venue,
        ...venueData,
      },
    })),

  // --------------------------------------------------
  // Ticket Mode
  // --------------------------------------------------

  setTicketMode: (mode) =>
    set((state) => {
      // Online events cannot use reserved seating.
      if (
        isOnlineEvent(state.locationType || state.eventType) &&
        mode === TICKET_MODES.SEATED
      ) {
        return {
          ticketMode: TICKET_MODES.GENERAL,
        };
      }

      return {
        ticketMode: mode,
      };
    }),

  // --------------------------------------------------
  // Basic Information
  // --------------------------------------------------

  updateBasicInformation: (data) =>
    set((state) => ({
      basicInformation: {
        ...state.basicInformation,
        ...data,
      },
    })),

  // --------------------------------------------------
  // Date & Time
  // --------------------------------------------------

  updateDateTime: (data) =>
    set((state) => ({
      dateTime: {
        ...state.dateTime,
        ...data,
      },
    })),

  // --------------------------------------------------
  // Event Details
  // --------------------------------------------------

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

  // Ticket Types


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
      ticketTypes: state.ticketTypes.filter(
        (ticket) => ticket.id !== id
      ),
    })),

  updateTicketType: (id, updatedFields) =>
    set((state) => ({
      ticketTypes: state.ticketTypes.map((ticket) =>
        ticket.id === id
          ? {
            ...ticket,
            ...updatedFields,
          }
          : ticket
      ),
    })),

  updateTicketSalesSettings: (data) =>
    set((state) => ({
      ticketSalesSettings: {
        ...state.ticketSalesSettings,
        ...data,
      },
    })),


  // Seating Configuration


  addCategory: ({
    name,
    price,
    tier = "Standard",
    color = "#00685f",
    rowsCount = 1,
  }) =>
    set((state) => {
      const seatsPerRow =
        state.seatingConfiguration.seatsPerRow || 8;

      const validRowsCount = Math.max(
        1,
        Math.min(20, Number(rowsCount) || 1)
      );

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
        rows: Array.from(
          { length: validRowsCount },
          (_, rowIndex) => ({
            id: `row-temp-${rowIndex}-${Date.now()}`,
            name: "Row",
            rowLetter: "",
            seats: Array.from(
              { length: seatsPerRow },
              (_, seatIndex) => ({
                id: "",
                number: seatIndex + 1,
                status: "available",
                categoryId: "",
              })
            ),
          })
        ),
      };

      const updatedCategories = rebuildLayout(
        [
          ...state.seatingConfiguration.categories,
          newCategory,
        ],
        seatsPerRow
      );

      const allRows = updatedCategories.flatMap(
        (category) => category.rows
      );

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
          totalCapacity: allRows.reduce(
            (sum, row) => sum + row.seats.length,
            0
          ),
        },
      };
    }),

  removeCategory: (categoryId) =>
    set((state) => {
      if (
        state.seatingConfiguration.categories.length <= 1
      ) {
        return state;
      }

      const remainingCategories =
        state.seatingConfiguration.categories.filter(
          (category) => category.id !== categoryId
        );

      const seatsPerRow =
        state.seatingConfiguration.seatsPerRow || 8;

      const updatedCategories = rebuildLayout(
        remainingCategories,
        seatsPerRow
      );

      const allRows = updatedCategories.flatMap(
        (category) => category.rows
      );

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
          totalCapacity: allRows.reduce(
            (sum, row) => sum + row.seats.length,
            0
          ),
        },
      };
    }),

  updateCategory: (categoryId, updatedFields) =>
    set((state) => {
      const updatedCategories =
        state.seatingConfiguration.categories.map(
          (category) => {
            if (category.id === categoryId) {
              return {
                ...category,
                ...updatedFields,
              };
            }

            return category;
          }
        );

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
      const validCount = Math.max(
        1,
        Math.min(20, Number(newRowCount) || 1)
      );

      const seatsPerRow =
        state.seatingConfiguration.seatsPerRow || 8;

      const updated =
        state.seatingConfiguration.categories.map(
          (category) => {
            if (category.id !== categoryId) {
              return category;
            }

            const currentRows = category.rows || [];

            if (currentRows.length === validCount) {
              return category;
            }

            let nextRows;

            if (validCount > currentRows.length) {
              const needed =
                validCount - currentRows.length;

              const newRows = Array.from(
                { length: needed },
                (_, index) => ({
                  id: `row-new-${Date.now()}-${index}`,
                  name: "",
                  rowLetter: "",
                  seats: Array.from(
                    { length: seatsPerRow },
                    (_, seatIndex) => ({
                      id: "",
                      number: seatIndex + 1,
                      status: "available",
                      categoryId: category.id,
                    })
                  ),
                })
              );

              nextRows = [...currentRows, ...newRows];
            } else {
              nextRows = currentRows.slice(
                0,
                validCount
              );
            }

            return {
              ...category,
              rows: nextRows,
            };
          }
        );

      const resequenced = rebuildLayout(
        updated,
        seatsPerRow
      );

      const allRows = resequenced.flatMap(
        (category) => category.rows
      );

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
          totalCapacity: allRows.reduce(
            (sum, row) => sum + row.seats.length,
            0
          ),
        },
      };
    }),

  addRowToCategory: (categoryId) => {
    const state = get();

    const category =
      state.seatingConfiguration.categories.find(
        (item) => item.id === categoryId
      );

    if (category) {
      state.setCategoryRowCount(
        categoryId,
        (category.rows?.length || 0) + 1
      );
    }
  },

  removeRowFromCategory: (categoryId, rowId) =>
    set((state) => {
      const seatsPerRow =
        state.seatingConfiguration.seatsPerRow || 8;

      const updated =
        state.seatingConfiguration.categories.map(
          (category) => {
            if (category.id !== categoryId) {
              return category;
            }

            if (category.rows.length <= 1) {
              return category;
            }

            const nextRows = rowId
              ? category.rows.filter(
                (row) => row.id !== rowId
              )
              : category.rows.slice(0, -1);

            return {
              ...category,
              rows: nextRows,
            };
          }
        );

      const resequenced = rebuildLayout(
        updated,
        seatsPerRow
      );

      const allRows = resequenced.flatMap(
        (category) => category.rows
      );

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
          totalCapacity: allRows.reduce(
            (sum, row) => sum + row.seats.length,
            0
          ),
        },
      };
    }),

  generateLayout: (newSeatsPerRow) =>
    set((state) => {
      const seatsPerRow = Math.max(
        1,
        Math.min(
          30,
          Number(newSeatsPerRow) ||
          state.seatingConfiguration.seatsPerRow ||
          8
        )
      );

      const updatedCategories =
        state.seatingConfiguration.categories.map(
          (category) => {
            const rows = (category.rows || []).map(
              (row) => ({
                ...row,
                seats: Array.from(
                  { length: seatsPerRow },
                  (_, seatIndex) => ({
                    id: "",
                    number: seatIndex + 1,
                    status: "available",
                    categoryId: category.id,
                  })
                ),
              })
            );

            return {
              ...category,
              rows,
            };
          }
        );

      const resequenced = rebuildLayout(
        updatedCategories,
        seatsPerRow
      );

      const allRows = resequenced.flatMap(
        (category) => category.rows
      );

      return {
        seatingConfiguration: {
          ...state.seatingConfiguration,
          seatsPerRow,
          categories: resequenced,
          sections: resequenced,
          rows: allRows,
          selectedSeatIds: [],
          totalCapacity: allRows.reduce(
            (sum, row) => sum + row.seats.length,
            0
          ),
        },
      };
    }),


  // Temporary Seat Selection


  toggleSeatSelection: (seatId) =>
    set((state) => {
      const currentSelected =
        state.seatingConfiguration.selectedSeatIds || [];

      const isSelected =
        currentSelected.includes(seatId);

      const newSelected = isSelected
        ? currentSelected.filter(
          (id) => id !== seatId
        )
        : [...currentSelected, seatId];

      return {
        seatingConfiguration: {
          ...state.seatingConfiguration,
          selectedSeatIds: newSelected,
        },
      };
    }),


  // Seat Status

  setSelectedSeatsStatus: (newStatus) =>
    set((state) => {
      const selectedIds = new Set(
        state.seatingConfiguration.selectedSeatIds
      );

      const updatedCategories =
        state.seatingConfiguration.categories.map(
          (category) => ({
            ...category,
            rows: category.rows.map((row) => ({
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
          })
        );

      const allRows = updatedCategories.flatMap(
        (category) => category.rows
      );

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

  // Assign Category To Selected Seats


  assignCategoryToSelectedSeats: (categoryId) =>
    set((state) => {
      const selectedIds = new Set(
        state.seatingConfiguration.selectedSeatIds
      );

      const updatedCategories =
        state.seatingConfiguration.categories.map(
          (category) => ({
            ...category,
            rows: category.rows.map((row) => ({
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
          })
        );

      const allRows = updatedCategories.flatMap(
        (category) => category.rows
      );

      return {
        seatingConfiguration: {
          ...state.seatingConfiguration,
          categories: updatedCategories,
          sections: updatedCategories,
          rows: allRows,
        },
      };
    }),


  // Generic Step Data


  updateStepData: (stepKey, data) =>
    set((state) => ({
      [stepKey]: {
        ...state[stepKey],
        ...data,
      },
    })),


  // Payout Account

  // updatePayoutAccount: (accountData) =>
  //   set((state) => ({
  //     payoutAccount: {
  //       ...state.payoutAccount,
  //       ...accountData,
  //     },
  //   })),


  resetForm: () => {
    const currentPreview =
      get().basicInformation?.bannerPreviewUrl;

    if (currentPreview) {
      URL.revokeObjectURL(currentPreview);
    }

    set({
      locationType: LOCATION_TYPES.PHYSICAL,
      eventType: LOCATION_TYPES.PHYSICAL,
      onlineUrl: "",
      venue: { ...initialVenue },
      basicInformation: { ...initialBasicInfo },
      dateTime: { ...initialDateTime },
      eventDetails: { ...initialEventDetails },
      ticketMode: TICKET_MODES.GENERAL,
      ticketTypes: [],
      ticketSalesSettings: {
        ...initialTicketSalesSettings,
      },
      seatingConfiguration: {
        ...initialSeatingConfig,
      },
      // payoutAccount: {
      //   ...initialPayoutAccount,
      // },
    });
  },
}));

export default useEventCreationStore;