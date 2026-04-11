-- Rollback: hapus tabel dalam urutan terbalik dari pembuatannya
-- (karena refresh_tokens punya FK ke users, hapus FK dulu)
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS users;
