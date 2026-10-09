export const DEFAULT_CUSTOMER_EVENT = {
  id: "sunfield",
  title: "Sunfield Music Festival",
  category: "Music",
  categoryIcon: "♪",
  date: "Sept 15, 2026",
  time: "6:00 PM – 11:00 PM",
  timeWithTimezone: "6:00 PM – 11:00 PM IST",
  fullDate: "September 15, 2026",
  location: "Bengaluru, Karnataka",
  venue: "Bangalore International Exhibition Centre",
  venueAddress: "Bangalore International Exhibition Centre\nBengaluru, Karnataka",
  heroImage: "/assets/2b62b.png",
  thumbnail: "/assets/59c6d.png",
  status: "Tickets Available",
  description:
    "Get ready for an unforgettable night of music, art, and community at the Sunfield Music Festival. Featuring a diverse lineup of international headliners and emerging local talent across multiple stages, Sunfield promises a sonically immersive experience. Enjoy curated food vendors, interactive art installations, and a vibrant atmosphere designed to celebrate the end of summer. Don't miss out on the biggest musical event of the year!",
  organizer: {
    name: "Event Masters",
    verified: true,
    avatar: "/assets/5e545.png",
  },
  ticketTiers: [
    {
      id: "vip",
      name: "VIP",
      price: 1499,
      formattedPrice: "₹1,499",
      description: "Priority entry, exclusive lounge",
      status: "Available",
    },
    {
      id: "general",
      name: "General",
      price: 499,
      formattedPrice: "₹499",
      description: "Standard entry pass",
      status: "Available",
    },
  ],
  seatRows: [
    { name: "A", type: "VIP - ₹1,499", seats: [1, 2, 3, 4, 5, 6, 7, 8] },
    { name: "B", type: "Premium - ₹999", seats: [1, 2, 3, 4, 5, 6, 7, 8, 9] },
    { name: "C", type: "", seats: [1, 2, 3, 4, 5, 6, 7, 8, 9] },
    { name: "D", type: "General - ₹499", seats: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10] },
  ],
  initialReservedSeats: ["A6", "C3", "C4"],
  initialBookedSeats: ["A3", "B6", "C7", "D7", "D8"],
  initialDisabledSeats: ["B1", "C9"],
  initialSelectedSeats: ["A4", "A5"],
};

export function getCustomerEventById(id) {
  // Return the default event mock data for any ID, adapted if needed
  return {
    ...DEFAULT_CUSTOMER_EVENT,
    id: id || DEFAULT_CUSTOMER_EVENT.id,
  };
}
