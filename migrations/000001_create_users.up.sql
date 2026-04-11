-- Tabel users: menyimpan akun pengguna sistem booking.
-- Sama persis strukturnya dengan Project 1 karena masalah auth tidak berubah.
CREATE TABLE IF NOT EXISTS users (
    id         VARCHAR(36)              NOT NULL,
    email      VARCHAR(255)             NOT NULL,
    -- password disimpan sebagai bcrypt hash, BUKAN plain text.
    -- bcrypt menghasilkan string ~60 karakter.
    password   VARCHAR(255)             NOT NULL,
    name       VARCHAR(255)             NOT NULL,
    -- role menentukan apa yang bisa dilakukan user:
    -- 'admin'    → bisa buat/edit/hapus room
    -- 'customer' → bisa booking, lihat, dan cancel
    role       ENUM('admin','customer') NOT NULL DEFAULT 'customer',
    created_at TIMESTAMP                NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP                NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    PRIMARY KEY (id),
    -- Email harus unik di seluruh sistem
    UNIQUE KEY uq_users_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;


-- Tabel refresh_tokens: menyimpan refresh token yang aktif.
-- Dipakai untuk mendapatkan access token baru tanpa harus login ulang.
-- Satu user bisa punya banyak refresh token (dari berbagai device/browser).
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id         VARCHAR(36)  NOT NULL,
    user_id    VARCHAR(36)  NOT NULL,
    token      VARCHAR(512) NOT NULL,
    expires_at TIMESTAMP    NOT NULL,
    -- revoked = 1 artinya token sudah dipakai atau dicabut, tidak bisa dipakai lagi.
    -- Ini adalah implementasi refresh token rotation:
    -- setiap kali token dipakai, ia di-revoke dan token baru dibuat.
    revoked    TINYINT(1)   NOT NULL DEFAULT 0,
    created_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (id),
    UNIQUE KEY uq_refresh_tokens_token (token),
    KEY        idx_refresh_tokens_user_id (user_id),
    CONSTRAINT fk_refresh_tokens_user
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
