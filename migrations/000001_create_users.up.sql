
CREATE TABLE IF NOT EXISTS users (
    id         VARCHAR(36)              NOT NULL,
    email      VARCHAR(255)             NOT NULL,

    password   VARCHAR(255)             NOT NULL,
    name       VARCHAR(255)             NOT NULL,
  
    role       ENUM('admin','customer') NOT NULL DEFAULT 'customer',
    created_at TIMESTAMP                NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP                NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    PRIMARY KEY (id),
    UNIQUE KEY uq_users_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;



CREATE TABLE IF NOT EXISTS refresh_tokens (
    id         VARCHAR(36)  NOT NULL,
    user_id    VARCHAR(36)  NOT NULL,
    token      VARCHAR(512) NOT NULL,
    expires_at TIMESTAMP    NOT NULL,

    revoked    TINYINT(1)   NOT NULL DEFAULT 0,
    created_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (id),
    UNIQUE KEY uq_refresh_tokens_token (token),
    KEY        idx_refresh_tokens_user_id (user_id),
    CONSTRAINT fk_refresh_tokens_user
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
