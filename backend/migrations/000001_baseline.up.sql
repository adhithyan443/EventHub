--
-- PostgreSQL database dump
--


-- Dumped from database version 14.24 (Ubuntu 14.24-0ubuntu0.22.04.1)
-- Dumped by pg_dump version 14.24 (Ubuntu 14.24-0ubuntu0.22.04.1)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: categories; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.categories (
    id uuid NOT NULL,
    name character varying(255) NOT NULL,
    description text,
    status character varying(20) NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


--
-- Name: checkouts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.checkouts (
    id uuid NOT NULL,
    reservation_id uuid NOT NULL,
    user_id uuid NOT NULL,
    subtotal numeric(10,2) NOT NULL,
    platform_fee numeric(10,2) NOT NULL,
    tax_amount numeric(10,2) NOT NULL,
    discount_amount numeric(10,2) DEFAULT '0'::numeric NOT NULL,
    total_amount numeric(10,2) NOT NULL,
    status character varying(20) NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


--
-- Name: event_cancellations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.event_cancellations (
    id uuid NOT NULL,
    event_id uuid NOT NULL,
    cancellation_allowed boolean DEFAULT false NOT NULL,
    cancellation_deadline_hours bigint DEFAULT 0 NOT NULL,
    refund_policy character varying(50) NOT NULL,
    refund_percentage bigint DEFAULT 0 NOT NULL,
    cancelled_at timestamp without time zone,
    cancellation_reason text,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


--
-- Name: event_contacts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.event_contacts (
    id uuid NOT NULL,
    event_id uuid NOT NULL,
    name character varying(100) NOT NULL,
    phone character varying(20) NOT NULL,
    email character varying(255) NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


--
-- Name: event_schedules; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.event_schedules (
    id uuid NOT NULL,
    event_id uuid NOT NULL,
    event_date date NOT NULL,
    start_time time without time zone,
    end_time time without time zone,
    is_all_day boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


--
-- Name: event_settings; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.event_settings (
    id uuid NOT NULL,
    event_id uuid NOT NULL,
    seat_layout_type character varying(50) NOT NULL,
    booking_limit_per_user bigint NOT NULL,
    sales_start_date date,
    sales_end_date date,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


--
-- Name: events; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.events (
    id uuid NOT NULL,
    organizer_id uuid NOT NULL,
    category_id uuid NOT NULL,
    venue_id uuid,
    event_type character varying(20) NOT NULL,
    online_url character varying(500),
    title character varying(255) NOT NULL,
    description text NOT NULL,
    banner_url character varying(500),
    language character varying(100),
    age_restriction bigint DEFAULT 0 NOT NULL,
    visibility character varying(20),
    highlights text,
    rules text,
    attendee_information text,
    status character varying(30) NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


--
-- Name: organizer_addresses; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.organizer_addresses (
    id uuid NOT NULL,
    organizer_id uuid NOT NULL,
    address_line text NOT NULL,
    city character varying(100) NOT NULL,
    state character varying(100) NOT NULL,
    country character varying(100) NOT NULL,
    postal_code character varying(20) NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


--
-- Name: organizer_applications; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.organizer_applications (
    id uuid NOT NULL,
    user_id uuid NOT NULL,
    business_name character varying(255) NOT NULL,
    business_type character varying(100) NOT NULL,
    description text NOT NULL,
    phone character varying(30) NOT NULL,
    website character varying(500),
    gst_number character varying(50),
    pan_number character varying(20) NOT NULL,
    bank_name character varying(255) NOT NULL,
    account_holder_name character varying(255) NOT NULL,
    account_number_encrypted text NOT NULL,
    account_number_last4 character varying(4) NOT NULL,
    ifsc_code character varying(20) NOT NULL,
    address_line text NOT NULL,
    city character varying(100) NOT NULL,
    state character varying(100) NOT NULL,
    country character varying(100) NOT NULL,
    postal_code character varying(20) NOT NULL,
    logo_url character varying(1000),
    verification_document_url character varying(1000) NOT NULL,
    status character varying(30) NOT NULL,
    rejection_reason text,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


--
-- Name: organizer_bank_accounts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.organizer_bank_accounts (
    id uuid NOT NULL,
    organizer_id uuid NOT NULL,
    bank_name character varying(255) NOT NULL,
    account_holder_name character varying(255) NOT NULL,
    account_number_encrypted text NOT NULL,
    account_number_last4 character varying(4) NOT NULL,
    ifsc_code character varying(20) NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


--
-- Name: organizer_profiles; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.organizer_profiles (
    id uuid NOT NULL,
    organizer_id uuid NOT NULL,
    business_name character varying(255) NOT NULL,
    business_type character varying(100) NOT NULL,
    description text,
    phone character varying(20) NOT NULL,
    email character varying(255) NOT NULL,
    website character varying(255),
    logo_url text,
    gst_number character varying(50),
    pan_number character varying(50) NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


--
-- Name: organizers; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.organizers (
    id uuid NOT NULL,
    user_id uuid NOT NULL,
    status character varying(30) NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


--
-- Name: password_reset_tokens; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.password_reset_tokens (
    id uuid NOT NULL,
    user_id uuid NOT NULL,
    token_hash character varying(255) NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    used_at timestamp with time zone,
    created_at timestamp with time zone NOT NULL
);


--
-- Name: pending_registrations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.pending_registrations (
    id uuid NOT NULL,
    full_name character varying(100) NOT NULL,
    email character varying(255) NOT NULL,
    phone character varying(20) NOT NULL,
    password_hash character varying(255) NOT NULL,
    otp_hash character varying(64) NOT NULL,
    otp_expires_at timestamp with time zone NOT NULL,
    otp_attempts bigint DEFAULT 0 NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


--
-- Name: refresh_tokens; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.refresh_tokens (
    id uuid NOT NULL,
    user_id uuid NOT NULL,
    token_hash character varying(255) NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    revoked_at timestamp with time zone,
    created_at timestamp with time zone NOT NULL
);


--
-- Name: seat_layouts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.seat_layouts (
    id uuid NOT NULL,
    event_id uuid NOT NULL,
    layout_name character varying(255) NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


--
-- Name: seat_reservation_items; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.seat_reservation_items (
    id uuid NOT NULL,
    reservation_id uuid NOT NULL,
    seat_id uuid,
    ticket_type_id uuid,
    quantity bigint DEFAULT 1 NOT NULL,
    price numeric(10,2) NOT NULL,
    created_at timestamp with time zone NOT NULL
);


--
-- Name: seat_reservations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.seat_reservations (
    id uuid NOT NULL,
    user_id uuid NOT NULL,
    event_id uuid NOT NULL,
    status character varying(20) NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


--
-- Name: seat_rows; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.seat_rows (
    id uuid NOT NULL,
    section_id uuid NOT NULL,
    row_name character varying(50) NOT NULL,
    created_at timestamp with time zone NOT NULL
);


--
-- Name: seat_sections; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.seat_sections (
    id uuid NOT NULL,
    seat_layout_id uuid NOT NULL,
    name character varying(100) NOT NULL,
    price numeric(10,2) NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


--
-- Name: seats; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.seats (
    id uuid NOT NULL,
    row_id uuid NOT NULL,
    seat_number bigint NOT NULL,
    status character varying(20) NOT NULL,
    created_at timestamp with time zone NOT NULL
);


--
-- Name: ticket_types; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.ticket_types (
    id uuid NOT NULL,
    event_id uuid NOT NULL,
    name character varying(100) NOT NULL,
    price numeric(10,2) NOT NULL,
    total_quantity bigint NOT NULL,
    available_quantity bigint NOT NULL,
    description text,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);


--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    id uuid NOT NULL,
    full_name character varying(100) NOT NULL,
    email character varying(255) NOT NULL,
    phone character varying(20),
    password_hash character varying(255) NOT NULL,
    role character varying(20) NOT NULL,
    status character varying(20) NOT NULL,
    profile_image text,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


--
-- Name: venues; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.venues (
    id uuid NOT NULL,
    google_place_id character varying(255) NOT NULL,
    name character varying(255) NOT NULL,
    address text NOT NULL,
    city character varying(100) NOT NULL,
    state character varying(100) NOT NULL,
    country character varying(100) NOT NULL,
    postal_code character varying(20),
    latitude numeric NOT NULL,
    longitude numeric NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


--
-- Name: categories categories_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.categories
    ADD CONSTRAINT categories_pkey PRIMARY KEY (id);


--
-- Name: checkouts checkouts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.checkouts
    ADD CONSTRAINT checkouts_pkey PRIMARY KEY (id);


--
-- Name: event_cancellations event_cancellations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.event_cancellations
    ADD CONSTRAINT event_cancellations_pkey PRIMARY KEY (id);


--
-- Name: event_contacts event_contacts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.event_contacts
    ADD CONSTRAINT event_contacts_pkey PRIMARY KEY (id);


--
-- Name: event_schedules event_schedules_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.event_schedules
    ADD CONSTRAINT event_schedules_pkey PRIMARY KEY (id);


--
-- Name: event_settings event_settings_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.event_settings
    ADD CONSTRAINT event_settings_pkey PRIMARY KEY (id);


--
-- Name: events events_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.events
    ADD CONSTRAINT events_pkey PRIMARY KEY (id);


--
-- Name: organizer_addresses organizer_addresses_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organizer_addresses
    ADD CONSTRAINT organizer_addresses_pkey PRIMARY KEY (id);


--
-- Name: organizer_applications organizer_applications_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organizer_applications
    ADD CONSTRAINT organizer_applications_pkey PRIMARY KEY (id);


--
-- Name: organizer_bank_accounts organizer_bank_accounts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organizer_bank_accounts
    ADD CONSTRAINT organizer_bank_accounts_pkey PRIMARY KEY (id);


--
-- Name: organizer_profiles organizer_profiles_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organizer_profiles
    ADD CONSTRAINT organizer_profiles_pkey PRIMARY KEY (id);


--
-- Name: organizers organizers_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organizers
    ADD CONSTRAINT organizers_pkey PRIMARY KEY (id);


--
-- Name: password_reset_tokens password_reset_tokens_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.password_reset_tokens
    ADD CONSTRAINT password_reset_tokens_pkey PRIMARY KEY (id);


--
-- Name: pending_registrations pending_registrations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pending_registrations
    ADD CONSTRAINT pending_registrations_pkey PRIMARY KEY (id);


--
-- Name: refresh_tokens refresh_tokens_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.refresh_tokens
    ADD CONSTRAINT refresh_tokens_pkey PRIMARY KEY (id);


--
-- Name: seat_layouts seat_layouts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.seat_layouts
    ADD CONSTRAINT seat_layouts_pkey PRIMARY KEY (id);


--
-- Name: seat_reservation_items seat_reservation_items_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.seat_reservation_items
    ADD CONSTRAINT seat_reservation_items_pkey PRIMARY KEY (id);


--
-- Name: seat_reservations seat_reservations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.seat_reservations
    ADD CONSTRAINT seat_reservations_pkey PRIMARY KEY (id);


--
-- Name: seat_rows seat_rows_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.seat_rows
    ADD CONSTRAINT seat_rows_pkey PRIMARY KEY (id);


--
-- Name: seat_sections seat_sections_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.seat_sections
    ADD CONSTRAINT seat_sections_pkey PRIMARY KEY (id);


--
-- Name: seats seats_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.seats
    ADD CONSTRAINT seats_pkey PRIMARY KEY (id);


--
-- Name: ticket_types ticket_types_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ticket_types
    ADD CONSTRAINT ticket_types_pkey PRIMARY KEY (id);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: venues venues_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.venues
    ADD CONSTRAINT venues_pkey PRIMARY KEY (id);


--
-- Name: idx_categories_name; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_categories_name ON public.categories USING btree (name);


--
-- Name: idx_categories_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_categories_status ON public.categories USING btree (status);


--
-- Name: idx_checkouts_expires_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_checkouts_expires_at ON public.checkouts USING btree (expires_at);


--
-- Name: idx_checkouts_reservation_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_checkouts_reservation_id ON public.checkouts USING btree (reservation_id);


--
-- Name: idx_checkouts_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_checkouts_status ON public.checkouts USING btree (status);


--
-- Name: idx_checkouts_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_checkouts_user_id ON public.checkouts USING btree (user_id);


--
-- Name: idx_event_cancellations_event_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_event_cancellations_event_id ON public.event_cancellations USING btree (event_id);


--
-- Name: idx_event_contacts_event_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_event_contacts_event_id ON public.event_contacts USING btree (event_id);


--
-- Name: idx_event_schedules_event_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_event_schedules_event_id ON public.event_schedules USING btree (event_id);


--
-- Name: idx_event_settings_event_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_event_settings_event_id ON public.event_settings USING btree (event_id);


--
-- Name: idx_events_category_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_events_category_id ON public.events USING btree (category_id);


--
-- Name: idx_events_event_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_events_event_type ON public.events USING btree (event_type);


--
-- Name: idx_events_organizer_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_events_organizer_id ON public.events USING btree (organizer_id);


--
-- Name: idx_events_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_events_status ON public.events USING btree (status);


--
-- Name: idx_events_venue_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_events_venue_id ON public.events USING btree (venue_id);


--
-- Name: idx_organizer_addresses_organizer_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_organizer_addresses_organizer_id ON public.organizer_addresses USING btree (organizer_id);


--
-- Name: idx_organizer_applications_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_organizer_applications_status ON public.organizer_applications USING btree (status);


--
-- Name: idx_organizer_applications_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_organizer_applications_user_id ON public.organizer_applications USING btree (user_id);


--
-- Name: idx_organizer_bank_accounts_organizer_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_organizer_bank_accounts_organizer_id ON public.organizer_bank_accounts USING btree (organizer_id);


--
-- Name: idx_organizer_profiles_organizer_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_organizer_profiles_organizer_id ON public.organizer_profiles USING btree (organizer_id);


--
-- Name: idx_organizers_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_organizers_status ON public.organizers USING btree (status);


--
-- Name: idx_organizers_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_organizers_user_id ON public.organizers USING btree (user_id);


--
-- Name: idx_password_reset_tokens_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_password_reset_tokens_user_id ON public.password_reset_tokens USING btree (user_id);


--
-- Name: idx_pending_registrations_email; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_pending_registrations_email ON public.pending_registrations USING btree (email);


--
-- Name: idx_refresh_tokens_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_refresh_tokens_user_id ON public.refresh_tokens USING btree (user_id);


--
-- Name: idx_row_seat; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_row_seat ON public.seats USING btree (row_id, seat_number);


--
-- Name: idx_seat_layouts_event_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_seat_layouts_event_id ON public.seat_layouts USING btree (event_id);


--
-- Name: idx_seat_reservation_items_reservation_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_seat_reservation_items_reservation_id ON public.seat_reservation_items USING btree (reservation_id);


--
-- Name: idx_seat_reservation_items_seat_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_seat_reservation_items_seat_id ON public.seat_reservation_items USING btree (seat_id);


--
-- Name: idx_seat_reservation_items_ticket_type_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_seat_reservation_items_ticket_type_id ON public.seat_reservation_items USING btree (ticket_type_id);


--
-- Name: idx_seat_reservations_event_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_seat_reservations_event_id ON public.seat_reservations USING btree (event_id);


--
-- Name: idx_seat_reservations_expires_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_seat_reservations_expires_at ON public.seat_reservations USING btree (expires_at);


--
-- Name: idx_seat_reservations_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_seat_reservations_status ON public.seat_reservations USING btree (status);


--
-- Name: idx_seat_reservations_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_seat_reservations_user_id ON public.seat_reservations USING btree (user_id);


--
-- Name: idx_seat_rows_section_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_seat_rows_section_id ON public.seat_rows USING btree (section_id);


--
-- Name: idx_seat_sections_seat_layout_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_seat_sections_seat_layout_id ON public.seat_sections USING btree (seat_layout_id);


--
-- Name: idx_seats_row_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_seats_row_id ON public.seats USING btree (row_id);


--
-- Name: idx_seats_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_seats_status ON public.seats USING btree (status);


--
-- Name: idx_ticket_types_event_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ticket_types_event_id ON public.ticket_types USING btree (event_id);


--
-- Name: idx_users_email; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_users_email ON public.users USING btree (email);


--
-- Name: idx_users_phone; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_users_phone ON public.users USING btree (phone);


--
-- Name: idx_venues_google_place_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_venues_google_place_id ON public.venues USING btree (google_place_id);


--
-- PostgreSQL database dump complete
--


