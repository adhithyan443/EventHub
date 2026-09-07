import musicImg from "../../../assets/events/event-music.jpg";
import techImg from "../../../assets/events/event-tech.jpg";
import workshopImg from "../../../assets/events/event-workshop.jpg";
import sportsImg from "../../../assets/events/event-sports.jpg";
import comedyImg from "../../../assets/events/event-comedy.jpg";
import conferenceImg from "../../../assets/events/event-conference.jpg";

// TODO: replace this whole file with a real GET /events API call once wired up.
export const mockEvents = [
  { id: 1, title: "Summer Soundwave Festival", category: "Music", date: "Aug 15 - Aug 17, 2024", location: "Central Park, NY", price: 85, image: musicImg, featured: true },
  { id: 2, title: "FutureTech Summit 2024", category: "Tech", date: "Sep 10, 2024 • 9:00 AM", location: "Moscone Center, SF", price: 299, image: techImg, badge: "Limited", featured: true },
  { id: 3, title: "Creative Design Masterclass", category: "Workshop", date: "Oct 05, 2024 • 10:00 AM", location: "Design Hub, Chicago", price: 45, image: workshopImg, featured: true },
  { id: 4, title: "City Finals Basketball", category: "Sports", date: "Nov 02, 2024 • 7:00 PM", location: "Riverside Arena, Austin", price: 60, image: sportsImg },
  { id: 5, title: "Standup Night: Local Legends", category: "Comedy", date: "Nov 18, 2024 • 8:00 PM", location: "The Laugh Cellar, LA", price: 25, image: comedyImg },
  { id: 6, title: "Startup Pitch Mixer", category: "Conferences", date: "Dec 03, 2024 • 6:00 PM", location: "Innovation Loft, Seattle", price: 40, image: conferenceImg },
];