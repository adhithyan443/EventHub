-- WARNING: This migration drops all EventHub baseline tables and their data.
-- Do not run against a database whose data must be preserved.

DROP TABLE IF EXISTS public.venues;
DROP TABLE IF EXISTS public.users;
DROP TABLE IF EXISTS public.ticket_types;
DROP TABLE IF EXISTS public.seats;
DROP TABLE IF EXISTS public.seat_sections;
DROP TABLE IF EXISTS public.seat_rows;
DROP TABLE IF EXISTS public.seat_reservations;
DROP TABLE IF EXISTS public.seat_reservation_items;
DROP TABLE IF EXISTS public.seat_layouts;
DROP TABLE IF EXISTS public.refresh_tokens;
DROP TABLE IF EXISTS public.pending_registrations;
DROP TABLE IF EXISTS public.password_reset_tokens;
DROP TABLE IF EXISTS public.organizers;
DROP TABLE IF EXISTS public.organizer_profiles;
DROP TABLE IF EXISTS public.organizer_bank_accounts;
DROP TABLE IF EXISTS public.organizer_applications;
DROP TABLE IF EXISTS public.organizer_addresses;
DROP TABLE IF EXISTS public.events;
DROP TABLE IF EXISTS public.event_settings;
DROP TABLE IF EXISTS public.event_schedules;
DROP TABLE IF EXISTS public.event_contacts;
DROP TABLE IF EXISTS public.event_cancellations;
DROP TABLE IF EXISTS public.checkouts;
DROP TABLE IF EXISTS public.categories;
