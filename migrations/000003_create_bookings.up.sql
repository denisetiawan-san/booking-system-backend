-- Tabel bookings: inti dari seluruh sistem ini.
-- Setiap baris merepresentasikan satu pemesanan kamar untuk rentang waktu tertentu.
CREATE TABLE IF NOT EXISTS bookings (
    id            VARCHAR(36)                              NOT NULL,
    user_id       VARCHAR(36)                              NOT NULL,
    room_id       VARCHAR(36)                              NOT NULL,

    -- check_in dan check_out mendefinisikan rentang waktu booking.
    -- Aturan bisnis: check_out HARUS lebih besar dari check_in.
    -- Format yang disimpan: DATE (tanpa jam), contoh: 2026-04-10
    -- Interpretasi: tamu datang pada check_in, pergi pada check_out.
    -- Contoh: check_in=2026-04-10, check_out=2026-04-12 = 2 malam.
    check_in      DATE                                     NOT NULL,
    check_out     DATE                                     NOT NULL,

    -- status mengikuti state machine sederhana:
    -- pending    → baru dibuat, belum dikonfirmasi
    -- confirmed  → sudah dikonfirmasi (simulasi: langsung confirmed saat dibuat)
    -- cancelled  → dibatalkan oleh user atau sistem
    -- Tidak ada transisi dari cancelled kembali ke confirmed/pending.
    status        ENUM('pending','confirmed','cancelled')  NOT NULL DEFAULT 'confirmed',

    -- total_price dihitung saat booking dibuat dan disimpan di sini.
    -- TIDAK dihitung real-time dari rooms.price_per_night karena harga bisa berubah.
    -- Sama seperti order_items.price di Project 1.
    total_price   DECIMAL(15,2)                           NOT NULL,

    -- notes: catatan tambahan dari tamu (permintaan khusus, dll)
    notes         TEXT,

    -- idempotency_key: UUID yang dikirim client di header "Idempotency-Key".
    -- UNIQUE constraint ini adalah garis pertahanan database-level melawan double submit:
    -- kalau user klik "Pesan" dua kali dengan key yang sama, INSERT kedua akan GAGAL
    -- dengan duplicate key error, yang kemudian ditangani di service layer.
    idempotency_key VARCHAR(36)                           NOT NULL,

    created_at    TIMESTAMP                               NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP                               NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    PRIMARY KEY (id),

    -- Index utama untuk cek ketersediaan kamar.
    -- Query "cari booking yang overlap dengan tanggal X-Y di room Z" akan sangat cepat
    -- karena MySQL bisa gunakan index ini untuk filter room_id dulu,
    -- lalu cek date range dalam subset yang kecil.
    KEY idx_bookings_room_dates (room_id, check_in, check_out),

    -- Index untuk query "tampilkan semua booking saya"
    KEY idx_bookings_user_id (user_id),

    -- Index untuk query filter by status (misal: tampilkan yang active)
    KEY idx_bookings_status (status),

    -- UNIQUE pada idempotency_key: satu key hanya boleh menghasilkan satu booking
    UNIQUE KEY uq_bookings_idempotency_key (idempotency_key),

    CONSTRAINT fk_bookings_user FOREIGN KEY (user_id) REFERENCES users(id),
    CONSTRAINT fk_bookings_room FOREIGN KEY (room_id) REFERENCES rooms(id),

    -- check_out harus setelah check_in (minimal 1 malam)
    CONSTRAINT chk_bookings_dates CHECK (check_out > check_in)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
