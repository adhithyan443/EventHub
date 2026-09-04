# EventHub

EventHub is an online event management and ticket booking platform designed to manage the complete event lifecycle — from authentication and event discovery to seat reservation, booking, payment, digital ticket generation, attendee validation, refunds, organizer management, and administration.

The project is designed as an MVP-oriented event management system with three primary roles:

- **Customer** — discovers events, reserves seats, books tickets, manages bookings, and accesses digital tickets.
- **Organizer** — applies to become an organizer, manages organizer information, creates and manages events, handles seating and ticketing, and validates attendees.
- **Admin** — manages users, organizers, events, categories, reports, and platform-level operations.

The project follows the EventHub API, database, architecture, security, logging, and development guidelines defined for the project.

---

## Features

### Authentication

- Customer registration
- Email/password login
- Google OAuth login
- Logout
- JWT access tokens
- Refresh token flow
- Forgot password
- Password reset
- User profile retrieval and update
- Role-based access control

### Public Event Discovery

- Browse published events
- Event details
- Seat layout viewing
- Event categories
- Event search
- Event filtering
- Pagination

### Customer

- Reserve seats temporarily
- Checkout
- Payment
- Booking history
- Booking details
- Booking cancellation
- Refund status
- My Tickets
- Ticket details
- Digital ticket download
- QR-based event entry

### Organizer

- Apply as an organizer
- View organizer application status
- Update organizer application
- Organizer profile management
- Create events
- Manage events
- Create and manage seat layouts
- Publish events
- Cancel events
- Event dashboard
- Event bookings
- Attendee list
- Ticket validation

### Admin

- Admin authentication
- Dashboard
- User management
- Block/unblock users
- Organizer application management
- Approve/reject organizer applications
- Organizer management
- Event management
- Event category management
- Dashboard statistics
- Revenue reports
- Booking reports
- Refund reports
- Event performance reports

---

## Booking Flow

The customer booking lifecycle is designed around temporary seat reservations:

```text
Browse Events
     ↓
View Event Details
     ↓
View Seat Layout
     ↓
Select Seats
     ↓
Reserve Seats
     ↓
Checkout
     ↓
Payment
     ↓
Booking Confirmed
     ↓
Digital Ticket + QR Code
     ↓
Event Entry / Ticket Validation
```

Reserved seats remain temporarily locked. If checkout/payment is not completed before the reservation expires, the seats become available again.

---

## Technology Stack

| Area | Technology |
|---|---|
| Backend Language | Go (Golang) |
| Backend Framework | Gin |
| Architecture | Clean Architecture |
| ORM | GORM |
| Database | PostgreSQL |
| Authentication | JWT |
| Social Authentication | Google OAuth |
| Password Hashing | bcrypt |
| Frontend | React |
| Frontend Build Tool | Vite |
| State Management | Zustand |
| HTTP Client | Axios |
| API Style | REST + JSON |
| API Documentation / Testing | Postman |
| Logging | Go `slog` |
| Configuration | Environment Variables / `.env` |
| Version Control | Git + GitHub |

---

## Architecture

The backend follows **Clean Architecture** to keep business logic independent from frameworks and infrastructure.

```text
                    ┌──────────────────────┐
                    │      Frontend        │
                    │   React + Vite       │
                    │      Zustand         │
                    │       Axios          │
                    └──────────┬───────────┘
                               │
                         REST + JSON
                               │
                    ┌──────────▼───────────┐
                    │      Delivery        │
                    │   HTTP / Gin         │
                    └──────────┬───────────┘
                               │
                    ┌──────────▼───────────┐
                    │       Use Case       │
                    │   Business Logic     │
                    └──────────┬───────────┘
                               │
                    ┌──────────▼───────────┐
                    │     Repository       │
                    │    Interfaces        │
                    └──────────┬───────────┘
                               │
                    ┌──────────▼───────────┐
                    │       GORM           │
                    │     PostgreSQL        │
                    └──────────────────────┘
```

The use-case layer does not depend directly on Gin or GORM.

---

## Project Structure

The project is organized around the Clean Architecture layers and application responsibilities.

```text
EventHub/
├── backend/
│   ├── cmd/
│   │   └── ...
│   ├── internal/
│   │   ├── domain/
│   │   ├── usecase/
│   │   ├── repository/
│   │   └── delivery/
│   │       └── http/
│   ├── ...
│   └── .env
│
├── frontend/
│   ├── src/
│   │   ├── components/
│   │   ├── pages/
│   │   ├── services/
│   │   ├── store/
│   │   └── ...
│   ├── ...
│   └── .env
│
└── README.md
```

> The exact directory structure may evolve as implementation progresses. Related routes, handlers, use cases, repositories, and models should remain logically organized.

---

## Database

EventHub uses **PostgreSQL** with a normalized relational schema.

The current database design contains 28 entities:

```text
User
Refresh Token
Password Reset Token

Organizer Application
Organizer
Organizer Profile
Organizer Address
Organizer Bank Account

Category
Venue
Event
Event Schedule
Event Settings
Event Cancellation
Ticket Type

Seat Layout
Seat Section
Seat Row
Seat

Seat Reservation
Seat Reservation Item
Booking
Booking Item

Payment
Refund

Ticket
Ticket Scan

Notification
```

### Database Design Principles

- Normalized relational schema
- UUID primary keys
- Foreign key constraints
- Unique constraints
- Indexes for frequently queried fields
- Data integrity constraints
- Efficient joins
- N+1 query prevention
- Sensitive data protection
- Encrypted organizer bank account numbers

The database design separates organizer identity, profile, address, and bank information, and also separates event identity, schedule, settings, and cancellation information.

---

## API

EventHub exposes a RESTful JSON API.

### Base URL

For local development:

```text
http://localhost:8080/api/v1
```

### API Modules

The API is organized into five major modules:

```text
Authentication APIs
Public APIs
Customer APIs
Organizer APIs
Admin APIs
```

The current API documentation contains **62 documented APIs** covering authentication, event discovery, booking, payments, ticketing, organizer management, event management, attendee validation, administration, reporting, and category management.

### Example Endpoints

#### Authentication

```text
POST   /api/v1/auth/register
POST   /api/v1/auth/login
POST   /api/v1/auth/logout
GET    /api/v1/auth/google
GET    /api/v1/auth/google/callback
POST   /api/v1/auth/refresh-token
POST   /api/v1/auth/forgot-password
POST   /api/v1/auth/reset-password
GET    /api/v1/auth/profile
PUT    /api/v1/auth/profile
```

#### Public Events

```text
GET /api/v1/events
GET /api/v1/events/{eventId}
GET /api/v1/events/{eventId}/seats
GET /api/v1/categories
GET /api/v1/events/search
```

#### Customer Booking

```text
POST  /api/v1/bookings/reserve
POST  /api/v1/bookings/checkout
POST  /api/v1/payments
GET   /api/v1/bookings
GET   /api/v1/bookings/{bookingId}
PATCH /api/v1/bookings/{bookingId}/cancel
GET   /api/v1/bookings/{bookingId}/refund
```

#### Customer Tickets

```text
GET /api/v1/tickets
GET /api/v1/tickets/{ticketId}
GET /api/v1/tickets/{ticketId}/download
```

Protected endpoints use JWT Bearer authentication:

```http
Authorization: Bearer <access_token>
```

---

## Authentication & Authorization

EventHub uses JWT-based authentication for protected APIs.

Supported authentication methods:

```text
Email + Password
       +
Google OAuth
```

Role-based access control separates permissions for:

```text
Customer
Organizer
Admin
```

Security requirements include:

- Password hashing using bcrypt
- JWT access-token expiration
- Refresh-token support
- Token invalidation during logout
- Input validation
- Secure error responses
- Protected user and booking data
- Secure payment processing
- No hardcoded credentials or secrets
- Environment-based configuration

---

## Seat Management

The seat layout follows a hierarchy:

```text
Seat Layout
    │
    └── Seat Section
            │
            └── Seat Row
                    │
                    └── Seat
```

Supported seat states include:

```text
AVAILABLE
RESERVED
BOOKED
DISABLED
```

A seat can only be selected when it is available.

---

## Payments & Tickets

After a successful checkout and payment:

```text
Payment
   ↓
Booking Confirmation
   ↓
Seats Permanently Booked
   ↓
Digital Ticket Generated
   ↓
QR Code Generated
   ↓
Confirmation Notification
```

Tickets can be accessed through the customer's ticket list and downloaded as PDF files.

Cancelled or refunded tickets cannot be used for event entry.

---

## Refunds

Refund handling is connected to booking cancellation and event cancellation policies.

The customer can track refund progress through:

```text
GET /api/v1/bookings/{booking_id}/refund
```

Refund amounts may depend on the applicable cancellation policy.

---

## Logging

The backend uses Go's structured logging package:

```text
log/slog
```

Logging should focus on meaningful application events and errors.

Sensitive information such as:

- Passwords
- Authentication tokens
- Payment credentials
- Bank account numbers

must never be logged.

---

## Environment Variables

Environment-specific configuration should be stored in `.env` files and must not be committed to Git.

Example configuration categories include:

```env
DATABASE_URL=
JWT_SECRET=
GOOGLE_CLIENT_ID=
GOOGLE_CLIENT_SECRET=
GOOGLE_REDIRECT_URL=
```

> Use the actual environment variables required by the current implementation. Never commit real credentials, API keys, passwords, or secrets.

---

## Getting Started

### Prerequisites

Make sure the following are installed:

- Go
- Node.js
- npm
- PostgreSQL
- Git

### Clone the Repository

```bash
git clone <repository-url>
cd EventHub
```

### Backend Setup

Navigate to the backend:

```bash
cd backend
```

Create/configure the environment file:

```bash
cp .env.example .env
```

Configure the required database, authentication, OAuth, and application settings.

Run the backend:

```bash
go run ./...
```

The API is intended to run locally at:

```text
http://localhost:8080
```

with the API prefix:

```text
http://localhost:8080/api/v1
```

### Frontend Setup

Navigate to the frontend:

```bash
cd frontend
```

Install dependencies:

```bash
npm install
```

Start the development server:

```bash
npm run dev
```

---

## API Documentation

The API is documented and tested using Postman.

The API documentation includes:

- API description
- HTTP method
- Endpoint
- Authentication requirements
- Headers
- Path parameters
- Query parameters
- Request body
- Success responses
- Error responses
- HTTP status codes
- Example requests and responses

---

## API Response & HTTP Standards

The API follows REST conventions:

| Method | Purpose |
|---|---|
| GET | Retrieve resources |
| POST | Create resources or perform actions |
| PUT | Update existing resources |
| PATCH | Partially update resources or change status |
| DELETE | Delete resources |

Common HTTP status codes:

```text
200 OK
201 Created
400 Bad Request
401 Unauthorized
403 Forbidden
404 Not Found
409 Conflict
500 Internal Server Error
```

Collection endpoints support pagination where applicable using parameters such as:

```text
page
limit
```

Filtering, searching, and sorting are provided where required.

---

## Security Guidelines

The project follows secure development practices including:

- Never commit `.env` files
- Never hardcode credentials or secrets
- Hash passwords securely
- Validate user input
- Protect authenticated endpoints
- Apply role-based authorization
- Avoid leaking sensitive database information
- Encrypt sensitive financial information at rest
- Avoid exposing sensitive fields in API responses
- Use secure and informative error messages
- Prevent SQL injection, XSS, and CSRF-related risks
- Follow OWASP Top 10 principles

---

## Git Workflow

Feature branches are used for new functionality.

```text
main
  │
  └── develop
        │
        ├── feature/...
        ├── fix/...
        └── chore/...
```

Do not commit directly to `main`.

Commit messages should use clear type prefixes:

```text
feat: add organizer application flow
fix: handle expired seat reservations
chore: update project configuration
refactor: simplify booking repository
docs: update API documentation
```

---

## Development Guidelines

The project follows the Brocamp implementation guidelines, including:

- Consistent naming conventions
- User-friendly validation messages
- Confirmation before destructive actions
- Secure authentication
- Proper error handling
- Secure file uploads
- Responsive and consistent UI
- Reusable frontend components
- Pagination, searching, filtering, and sorting for admin data
- Clean project structure
- Modular and reusable code
- Environment-based configuration
- Secure API design
- Normalized database design
- Structured logging
- Feature-branch Git workflow
- Clear project documentation

---

## Project Documentation

The repository/project documentation includes:

- EventHub API documentation
- PostgreSQL database design
- Project planning and modules
- Weekly development plan
- Implementation guidelines

These documents should remain aligned with the Figma designs, database schema, and API documentation.

---

## Current Project Scope

EventHub is being developed as an MVP-oriented event management and ticket booking platform.

The primary scope covers:

```text
Authentication
      ↓
Event Discovery
      ↓
Organizer Management
      ↓
Event Creation
      ↓
Seat Management
      ↓
Booking
      ↓
Payment
      ↓
Ticket Generation
      ↓
Ticket Validation
      ↓
Refunds
      ↓
Administration & Reports
```

---

## License

This project is developed as part of the Brocamp first project and is intended for educational and project-development purposes.
