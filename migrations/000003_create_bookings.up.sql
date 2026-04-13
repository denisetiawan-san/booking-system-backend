
CREATE TABLE IF NOT EXISTS bookings (
    id            VARCHAR(36)                              NOT NULL,
    user_id       VARCHAR(36)                              NOT NULL,
    room_id       VARCHAR(36)                              NOT NULL,


    check_in      DATE                                     NOT NULL,
    check_out     DATE                                     NOT NULL,


    status        ENUM('pending','confirmed','cancelled')  NOT NULL DEFAULT 'confirmed',


    total_price   DECIMAL(15,2)                           NOT NULL,


    notes         TEXT,


    idempotency_key VARCHAR(36)                           NOT NULL,

    created_at    TIMESTAMP                               NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP                               NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    PRIMARY KEY (id),


    KEY idx_bookings_room_dates (room_id, check_in, check_out),


    KEY idx_bookings_user_id (user_id),


    KEY idx_bookings_status (status),


    UNIQUE KEY uq_bookings_idempotency_key (idempotency_key),

    CONSTRAINT fk_bookings_user FOREIGN KEY (user_id) REFERENCES users(id),
    CONSTRAINT fk_bookings_room FOREIGN KEY (room_id) REFERENCES rooms(id),


    CONSTRAINT chk_bookings_dates CHECK (check_out > check_in)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
