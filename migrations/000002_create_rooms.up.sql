-- Tabel rooms: menyimpan data kamar/ruangan yang bisa dipesan.
-- "Room" di sini adalah abstraksi — bisa berupa kamar hotel, ruang rapat,
-- lapangan olahraga, atau resource apapun yang bisa dipesan per slot waktu.
CREATE TABLE IF NOT EXISTS rooms (
    id          VARCHAR(36)   NOT NULL,
    name        VARCHAR(255)  NOT NULL,  -- contoh: "Kamar Deluxe 101", "Meeting Room A"
    description TEXT,                   -- fasilitas, lokasi, dll
    capacity    INT           NOT NULL DEFAULT 1,  -- berapa orang yang muat
    -- price_per_night: harga per malam/hari.
    -- Nama "per_night" dipakai untuk konteks hotel, tapi bisa diinterpretasikan
    -- sebagai "per unit time" untuk konteks lain.
    price_per_night DECIMAL(15,2) NOT NULL,
    -- is_active: soft delete. Kalau false, room tidak muncul di pencarian.
    -- Lebih baik daripada hard delete karena booking lama yang reference room ini
    -- tetap bisa diquery tanpa FK violation.
    is_active   TINYINT(1)    NOT NULL DEFAULT 1,
    created_at  TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    PRIMARY KEY (id),
    -- Index pada is_active karena query list room selalu filter is_active = 1
    KEY idx_rooms_is_active (is_active),
    -- Constraint: harga dan kapasitas tidak boleh negatif atau nol
    CONSTRAINT chk_rooms_price    CHECK (price_per_night > 0),
    CONSTRAINT chk_rooms_capacity CHECK (capacity > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
