CREATE TABLE public.email_change_requests (
    id uuid NOT NULL,
    user_id uuid NOT NULL,
    new_email character varying(255) NOT NULL,
    otp_hash character varying(64) NOT NULL,
    otp_expires_at timestamp with time zone NOT NULL,
    otp_attempts bigint DEFAULT 0 NOT NULL,
    resend_available_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    CONSTRAINT email_change_requests_pkey PRIMARY KEY (id),
    CONSTRAINT fk_email_change_requests_user_id FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX idx_email_change_requests_user_id ON public.email_change_requests USING btree (user_id);
CREATE UNIQUE INDEX idx_email_change_requests_new_email ON public.email_change_requests USING btree (new_email);
