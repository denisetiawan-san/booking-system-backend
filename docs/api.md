# API Documentation — Booking System Backend

Base URL: `http://localhost:8090/api/v1`

---

## Format Response Standar

Semua endpoint menggunakan format response yang konsisten.

**Sukses:**

```json
{
  "success": true,
  "message": "pesan deskriptif",
  "data": {}
}
```

**Error:**

```json
{
  "success": false,
  "error": "pesan error yang jelas"
}
```

---

## Daftar Semua Endpoint

| No  | Method | Endpoint                | Auth         | Role  | Keterangan                        |
| --- | ------ | ----------------------- | ------------ | ----- | --------------------------------- |
| 1   | GET    | /health                 | —            | —     | Health check server               |
| 2   | POST   | /api/v1/auth/register   | —            | —     | Daftar user baru                  |
| 3   | POST   | /api/v1/auth/login      | —            | —     | Login user                        |
| 4   | POST   | /api/v1/auth/refresh    | —            | —     | Perbarui access token             |
| 5   | GET    | /api/v1/auth/profile    | ✓            | any   | Lihat profil sendiri              |
| 6   | GET    | /api/v1/rooms           | —            | —     | List semua kamar aktif            |
| 7   | GET    | /api/v1/rooms/available | —            | —     | Cari kamar tersedia untuk tanggal |
| 8   | GET    | /api/v1/rooms/{id}      | —            | —     | Detail satu kamar                 |
| 9   | POST   | /api/v1/rooms           | ✓            | admin | Buat kamar baru                   |
| 10  | PATCH  | /api/v1/rooms/{id}      | ✓            | admin | Update kamar                      |
| 11  | POST   | /api/v1/bookings        | ✓ + Idem-Key | any   | Buat booking baru                 |
| 12  | GET    | /api/v1/bookings        | ✓            | any   | List booking milik saya           |
| 13  | GET    | /api/v1/bookings/{id}   | ✓            | any   | Detail satu booking               |
| 14  | DELETE | /api/v1/bookings/{id}   | ✓            | any   | Cancel booking                    |

**Keterangan kolom Auth:**

- `—` = tidak perlu token
- `✓` = wajib menyertakan header `Authorization: Bearer <access_token>`
- `✓ + Idem-Key` = wajib token + header `Idempotency-Key: <uuid>`

---

## Health Check

### GET /health

Memverifikasi bahwa server sedang berjalan. Dipakai Docker healthcheck dan monitoring.

**Request:** Tidak ada body, tidak ada header khusus.

**Response 200:**

```json
{
  "status": "ok",
  "service": "booking-system"
}
```

---

## Auth

### POST /api/v1/auth/register

Mendaftarkan user baru. Mengembalikan access token dan refresh token langsung setelah register berhasil — user tidak perlu login lagi setelah mendaftar.

**Request Body:**

```json
{
  "email": "budi@example.com",
  "password": "password123",
  "name": "Budi Santoso"
}
```

| Field    | Tipe   | Wajib | Validasi           |
| -------- | ------ | ----- | ------------------ |
| email    | string | ✓     | Format email valid |
| password | string | ✓     | Minimal 8 karakter |
| name     | string | ✓     | Minimal 2 karakter |

**Response 201:**

```json
{
  "success": true,
  "message": "registrasi berhasil",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 900
  }
}
```

| Field         | Keterangan                                          |
| ------------- | --------------------------------------------------- |
| access_token  | JWT token untuk dipakai di header Authorization     |
| refresh_token | Token untuk memperbarui access token ketika expired |
| expires_in    | Durasi access token dalam detik (900 = 15 menit)    |

**Response Error:**

```json
// 400 — validasi gagal
{ "success": false, "error": "Key: 'RegisterRequest.Email' Error:Field validation for 'Email' failed on the 'email' tag" }

// 409 — email sudah terdaftar
{ "success": false, "error": "email sudah terdaftar" }
```

---

### POST /api/v1/auth/login

Login dengan email dan password.

**Request Body:**

```json
{
  "email": "budi@example.com",
  "password": "password123"
}
```

**Response 200:**

```json
{
  "success": true,
  "message": "login berhasil",
  "data": {
    "access_token": "eyJhbGci...",
    "refresh_token": "eyJhbGci...",
    "expires_in": 900
  }
}
```

**Response Error:**

```json
// 401 — email atau password salah
// Pesan SAMA untuk keduanya (anti user enumeration attack)
{ "success": false, "error": "email atau password salah" }
```

---

### POST /api/v1/auth/refresh

Memperbarui access token menggunakan refresh token. Refresh token lama akan langsung di-revoke (token rotation) dan tidak bisa dipakai lagi.

**Request Body:**

```json
{
  "refresh_token": "eyJhbGci..."
}
```

**Response 200:**

```json
{
  "success": true,
  "message": "token berhasil diperbarui",
  "data": {
    "access_token": "eyJhbGci...",
    "refresh_token": "eyJhbGci...",
    "expires_in": 900
  }
}
```

**Response Error:**

```json
// 401 — token tidak valid, sudah expired, atau sudah pernah dipakai
{ "success": false, "error": "token tidak valid atau sudah expired" }
```

---

### GET /api/v1/auth/profile

Mengambil data profil user yang sedang login. User ID diambil dari JWT token secara otomatis — client tidak perlu kirim user ID secara eksplisit.

**Headers:**

```
Authorization: Bearer <access_token>
```

**Response 200:**

```json
{
  "success": true,
  "message": "ok",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "budi@example.com",
    "name": "Budi Santoso",
    "role": "customer"
  }
}
```

**Response Error:**

```json
// 401 — tidak ada token atau token tidak valid
{ "success": false, "error": "unauthorized: token diperlukan" }
```

---

## Rooms

### GET /api/v1/rooms

Mengambil daftar semua kamar yang aktif dengan pagination. Endpoint ini public — siapa saja bisa mengaksesnya tanpa token.

**Query Parameters:**

| Parameter | Tipe | Default | Keterangan                                        |
| --------- | ---- | ------- | ------------------------------------------------- |
| page      | int  | 1       | Halaman ke berapa (minimal 1)                     |
| limit     | int  | 10      | Jumlah item per halaman (minimal 1, maksimal 100) |

**Contoh Request:**

```
GET /api/v1/rooms?page=1&limit=5
```

**Response 200:**

```json
{
  "success": true,
  "message": "ok",
  "data": {
    "rooms": [
      {
        "id": "room-uuid",
        "name": "Kamar Deluxe 101",
        "description": "Kamar dengan view laut, AC, WiFi",
        "capacity": 2,
        "price_per_night": 750000,
        "is_active": true,
        "created_at": "2026-04-08T10:00:00Z",
        "updated_at": "2026-04-08T10:00:00Z"
      }
    ],
    "total": 15,
    "page": 1,
    "limit": 5
  }
}
```

| Field | Keterangan                                        |
| ----- | ------------------------------------------------- |
| total | Total semua kamar aktif (bukan hanya halaman ini) |
| page  | Halaman yang sedang ditampilkan                   |
| limit | Jumlah item per halaman yang dipakai              |

---

### GET /api/v1/rooms/available

Mencari kamar yang tersedia untuk rentang tanggal tertentu. Kamar yang sudah dipesan untuk tanggal yang overlap tidak akan muncul. Response menyertakan kalkulasi total harga untuk rentang tanggal yang diminta.

**Query Parameters:**

| Parameter | Tipe   | Wajib | Keterangan                           |
| --------- | ------ | ----- | ------------------------------------ |
| check_in  | string | ✓     | Tanggal masuk, format: YYYY-MM-DD    |
| check_out | string | ✓     | Tanggal keluar, format: YYYY-MM-DD   |
| capacity  | int    | —     | Filter kapasitas minimal (default 1) |

**Contoh Request:**

```
GET /api/v1/rooms/available?check_in=2099-06-10&check_out=2099-06-13&capacity=2
```

**Response 200:**

```json
{
  "success": true,
  "message": "ok",
  "data": [
    {
      "room": {
        "id": "room-uuid",
        "name": "Kamar Deluxe 101",
        "description": "View laut, AC, WiFi",
        "capacity": 2,
        "price_per_night": 750000,
        "is_active": true,
        "created_at": "2026-04-08T10:00:00Z",
        "updated_at": "2026-04-08T10:00:00Z"
      },
      "total_night": 3,
      "total_price": 2250000
    }
  ]
}
```

| Field       | Keterangan                                  |
| ----------- | ------------------------------------------- |
| total_night | Jumlah malam dari check_in sampai check_out |
| total_price | price_per_night × total_night               |

**Response Error:**

```json
// 400 — format tanggal salah
{ "success": false, "error": "format check_in tidak valid, gunakan format: 2006-01-02 (contoh: 2099-06-10)" }

// 400 — check_out sebelum atau sama dengan check_in
{ "success": false, "error": "tanggal check-out harus setelah tanggal check-in" }

// 400 — check_in di masa lalu
{ "success": false, "error": "tanggal check-in tidak boleh di masa lalu" }
```

---

### GET /api/v1/rooms/{id}

Mengambil detail satu kamar berdasarkan ID. Public endpoint.

**Path Parameter:**

- `id` — UUID kamar

**Response 200:**

```json
{
  "success": true,
  "message": "ok",
  "data": {
    "id": "room-uuid",
    "name": "Kamar Deluxe 101",
    "description": "View laut, AC, WiFi",
    "capacity": 2,
    "price_per_night": 750000,
    "is_active": true,
    "created_at": "2026-04-08T10:00:00Z",
    "updated_at": "2026-04-08T10:00:00Z"
  }
}
```

**Response Error:**

```json
// 404 — kamar tidak ditemukan
{ "success": false, "error": "data tidak ditemukan" }
```

---

### POST /api/v1/rooms

Membuat kamar baru. Hanya bisa diakses oleh user dengan role `admin`.

**Headers:**

```
Authorization: Bearer <admin_access_token>
```

**Request Body:**

```json
{
  "name": "Kamar Deluxe 101",
  "description": "Kamar dengan view laut, AC, WiFi gratis",
  "capacity": 2,
  "price_per_night": 750000
}
```

| Field           | Tipe   | Wajib | Validasi           |
| --------------- | ------ | ----- | ------------------ |
| name            | string | ✓     | Minimal 2 karakter |
| description     | string | —     | Opsional           |
| capacity        | int    | ✓     | Minimal 1          |
| price_per_night | float  | ✓     | Harus lebih dari 0 |

**Response 201:**

```json
{
  "success": true,
  "message": "kamar berhasil dibuat",
  "data": {
    "id": "room-uuid",
    "name": "Kamar Deluxe 101",
    "description": "Kamar dengan view laut, AC, WiFi gratis",
    "capacity": 2,
    "price_per_night": 750000,
    "is_active": true,
    "created_at": "2026-04-08T10:00:00Z",
    "updated_at": "2026-04-08T10:00:00Z"
  }
}
```

**Response Error:**

```json
// 401 — tidak ada token
{ "success": false, "error": "unauthorized: token diperlukan" }

// 403 — bukan admin
{ "success": false, "error": "akses ditolak: role tidak mencukupi" }

// 400 — validasi gagal
{ "success": false, "error": "..." }
```

---

### PATCH /api/v1/rooms/{id}

Mengupdate data kamar. Partial update — hanya field yang dikirim yang akan diubah. Hanya admin.

**Headers:**

```
Authorization: Bearer <admin_access_token>
```

**Request Body (semua field opsional):**

```json
{
  "name": "Kamar Deluxe 101 Renovasi",
  "price_per_night": 850000,
  "is_active": false
}
```

| Field           | Tipe   | Validasi           |
| --------------- | ------ | ------------------ |
| name            | string | Minimal 2 karakter |
| description     | string | —                  |
| capacity        | int    | Minimal 1          |
| price_per_night | float  | Lebih dari 0       |
| is_active       | bool   | true atau false    |

**Response 200:**

```json
{
  "success": true,
  "message": "kamar berhasil diupdate",
  "data": {
    "id": "room-uuid",
    "name": "Kamar Deluxe 101 Renovasi",
    "description": "Kamar dengan view laut, AC, WiFi gratis",
    "capacity": 2,
    "price_per_night": 850000,
    "is_active": false,
    "created_at": "2026-04-08T10:00:00Z",
    "updated_at": "2026-04-09T08:00:00Z"
  }
}
```

**Response Error:**

```json
// 404 — kamar tidak ditemukan
{ "success": false, "error": "data tidak ditemukan" }

// 403 — bukan admin
{ "success": false, "error": "akses ditolak: role tidak mencukupi" }
```

---

## Bookings

### POST /api/v1/bookings

Membuat booking baru. Endpoint ini memerlukan **dua header sekaligus**: token JWT dan Idempotency-Key.

Proses ini sepenuhnya atomic — kalau ada error di tengah jalan (misalnya kamar sudah dipesan orang lain), semua perubahan akan dibatalkan dan tidak ada data yang setengah tersimpan.

**Headers:**

```
Authorization: Bearer <access_token>
Idempotency-Key: 550e8400-e29b-41d4-a716-446655440001
```

> **Penting tentang Idempotency-Key:**
>
> - Harus UUID unik yang di-generate di sisi client sebelum request dikirim
> - Kalau request dikirim dua kali dengan key yang sama, response identik dan booking hanya terbuat sekali
> - Gunakan UUID v4 random yang berbeda untuk setiap booking yang berbeda
> - Maksimal 64 karakter

**Request Body:**

```json
{
  "room_id": "room-uuid",
  "check_in": "2099-06-10",
  "check_out": "2099-06-13",
  "notes": "Mohon siapkan extra pillow"
}
```

| Field     | Tipe   | Wajib | Validasi                                    |
| --------- | ------ | ----- | ------------------------------------------- |
| room_id   | string | ✓     | UUID kamar yang valid                       |
| check_in  | string | ✓     | Format YYYY-MM-DD, tidak boleh di masa lalu |
| check_out | string | ✓     | Format YYYY-MM-DD, harus setelah check_in   |
| notes     | string | —     | Catatan tambahan opsional                   |

**Response 201:**

```json
{
  "success": true,
  "message": "booking berhasil dibuat",
  "data": {
    "id": "booking-uuid",
    "room_id": "room-uuid",
    "room_name": "Kamar Deluxe 101",
    "check_in": "2099-06-10T00:00:00Z",
    "check_out": "2099-06-13T00:00:00Z",
    "status": "confirmed",
    "total_price": 2250000,
    "total_night": 3,
    "notes": "Mohon siapkan extra pillow",
    "created_at": "2026-04-09T10:00:00Z"
  }
}
```

| Field       | Keterangan                                          |
| ----------- | --------------------------------------------------- |
| status      | Selalu 'confirmed' untuk booking baru yang berhasil |
| total_price | Harga yang dikunci saat booking ini dibuat          |
| total_night | Jumlah malam (check_out - check_in)                 |
| room_name   | Nama kamar dari JOIN dengan tabel rooms             |

**Response Error:**

```json
// 400 — header Idempotency-Key tidak ada
{ "success": false, "error": "header Idempotency-Key wajib diisi..." }

// 400 — format tanggal salah
{ "success": false, "error": "format check_in tidak valid, gunakan format: 2006-01-02 (contoh: 2099-06-10)" }

// 400 — check_out sebelum check_in
{ "success": false, "error": "tanggal check-out harus setelah tanggal check-in" }

// 400 — check_in di masa lalu
{ "success": false, "error": "tanggal check-in tidak boleh di masa lalu" }

// 404 — room tidak ditemukan
{ "success": false, "error": "data tidak ditemukan" }

// 409 — kamar sudah dipesan untuk tanggal yang overlap
{ "success": false, "error": "kamar tidak tersedia untuk tanggal yang dipilih, sudah ada booking yang overlap" }

// 401 — tidak ada token
{ "success": false, "error": "unauthorized: token diperlukan" }
```

---

### GET /api/v1/bookings

Mengambil semua booking milik user yang sedang login, diurutkan dari yang terbaru.

**Headers:**

```
Authorization: Bearer <access_token>
```

**Response 200:**

```json
{
  "success": true,
  "message": "ok",
  "data": [
    {
      "id": "booking-uuid",
      "room_id": "room-uuid",
      "room_name": "Kamar Deluxe 101",
      "check_in": "2099-06-10T00:00:00Z",
      "check_out": "2099-06-13T00:00:00Z",
      "status": "confirmed",
      "total_price": 2250000,
      "total_night": 3,
      "notes": "Extra pillow",
      "created_at": "2026-04-09T10:00:00Z"
    },
    {
      "id": "booking-uuid-2",
      "room_id": "room-uuid-2",
      "room_name": "Suite Premium",
      "check_in": "2099-08-01T00:00:00Z",
      "check_out": "2099-08-05T00:00:00Z",
      "status": "cancelled",
      "total_price": 5000000,
      "total_night": 4,
      "notes": "",
      "created_at": "2026-04-08T08:00:00Z"
    }
  ]
}
```

> Kalau tidak ada booking, `data` adalah array kosong `[]`, bukan `null`.

---

### GET /api/v1/bookings/{id}

Mengambil detail satu booking. User hanya bisa mengakses booking miliknya sendiri.

**Headers:**

```
Authorization: Bearer <access_token>
```

**Path Parameter:**

- `id` — UUID booking

**Response 200:**

```json
{
  "success": true,
  "message": "ok",
  "data": {
    "id": "booking-uuid",
    "room_id": "room-uuid",
    "room_name": "Kamar Deluxe 101",
    "check_in": "2099-06-10T00:00:00Z",
    "check_out": "2099-06-13T00:00:00Z",
    "status": "confirmed",
    "total_price": 2250000,
    "total_night": 3,
    "notes": "Extra pillow",
    "created_at": "2026-04-09T10:00:00Z"
  }
}
```

**Response Error:**

```json
// 404 — booking tidak ditemukan
{ "success": false, "error": "data tidak ditemukan" }

// 403 — booking milik user lain
{ "success": false, "error": "kamu tidak memiliki akses ke booking ini" }
```

---

### DELETE /api/v1/bookings/{id}

Membatalkan booking. Perubahan status booking dari `confirmed` menjadi `cancelled`.

Ada 4 aturan bisnis yang divalidasi sebelum pembatalan diproses:

1. Hanya pemilik booking yang bisa membatalkan
2. Booking yang sudah `cancelled` tidak bisa dibatalkan lagi
3. Booking yang tanggal `check_in`-nya sudah lewat tidak bisa dibatalkan (tamu dianggap sudah check in)

Setelah dibatalkan, kamar otomatis tersedia kembali untuk tanggal tersebut.

**Headers:**

```
Authorization: Bearer <access_token>
```

**Path Parameter:**

- `id` — UUID booking

**Response 200:**

```json
{
  "success": true,
  "message": "booking berhasil dibatalkan",
  "data": null
}
```

**Response Error:**

```json
// 404 — booking tidak ditemukan
{ "success": false, "error": "data tidak ditemukan" }

// 403 — bukan pemilik booking
{ "success": false, "error": "kamu tidak memiliki akses ke booking ini" }

// 409 — booking sudah dibatalkan sebelumnya
{ "success": false, "error": "booking ini sudah dibatalkan sebelumnya" }

// 409 — tanggal check_in sudah lewat
{ "success": false, "error": "tidak bisa membatalkan booking yang tanggal check-in sudah lewat" }
```

---

## Tabel Error Code Lengkap

| HTTP Status | Kondisi                    | Contoh Pesan                                       |
| ----------- | -------------------------- | -------------------------------------------------- |
| 400         | Body JSON tidak valid      | "format JSON tidak valid"                          |
| 400         | Validasi field gagal       | "Key: '...' Error:..."                             |
| 400         | Format tanggal salah       | "format check_in tidak valid..."                   |
| 400         | check_out sebelum check_in | "tanggal check-out harus setelah..."               |
| 400         | check_in di masa lalu      | "tanggal check-in tidak boleh..."                  |
| 400         | Idempotency-Key tidak ada  | "header Idempotency-Key wajib..."                  |
| 401         | Token tidak ada            | "unauthorized: token diperlukan"                   |
| 401         | Token tidak valid/expired  | "token tidak valid atau sudah expired"             |
| 403         | Role tidak cukup           | "akses ditolak: role tidak mencukupi"              |
| 403         | Bukan pemilik resource     | "kamu tidak memiliki akses ke booking ini"         |
| 404         | Data tidak ditemukan       | "data tidak ditemukan"                             |
| 409         | Email sudah terdaftar      | "email sudah terdaftar"                            |
| 409         | Kamar tidak tersedia       | "kamar tidak tersedia untuk tanggal..."            |
| 409         | Booking sudah cancelled    | "booking ini sudah dibatalkan sebelumnya"          |
| 409         | Check-in sudah lewat       | "tidak bisa membatalkan booking yang..."           |
| 500         | Error server               | "terjadi kesalahan pada server, silakan coba lagi" |

---

## Contoh Flow Lengkap di Postman

Urutan request untuk demo end-to-end:

```
1.  POST /auth/register          → dapat access_token
2.  [TablePlus: ubah role ke admin]
3.  POST /auth/login             → dapat access_token admin
4.  POST /rooms                  → buat kamar, dapat room_id
5.  GET  /rooms                  → lihat list kamar (public)
6.  GET  /rooms/available        → cari kamar tersedia + total_price
7.  GET  /rooms/{id}             → detail kamar
8.  POST /bookings               → booking kamar (Idempotency-Key wajib!)
9.  GET  /bookings               → lihat semua booking saya
10. GET  /bookings/{id}          → detail booking
11. GET  /rooms/available        → kamar yang baru dipesan tidak muncul
12. DELETE /bookings/{id}        → cancel booking
13. GET  /rooms/available        → kamar muncul kembali setelah cancel
14. POST /auth/refresh           → perbarui access token
15. GET  /auth/profile           → lihat profil
```

---

## Catatan Penting untuk Postman

### Setup Environment Variables

Buat Postman Environment dengan variable berikut:

```
base_url        = http://localhost:8090/api/v1
admin_token     = (isi setelah login admin)
customer_token  = (isi setelah login customer)
room_id         = (isi setelah buat kamar)
booking_id      = (isi setelah buat booking)
```

### Cara Generate Idempotency-Key di Postman

Di tab Headers, untuk field `Idempotency-Key`, gunakan nilai:

```
{{$guid}}
```

Postman akan otomatis generate UUID baru setiap kali request dikirim. Kalau ingin test idempotency, ubah ke string tetap seperti `test-idem-key-001`.

### Format Tanggal

Semua tanggal menggunakan format ISO 8601: `YYYY-MM-DD`

Contoh valid: `2099-06-10`
Contoh tidak valid: `10-06-2099`, `10/06/2099`, `June 10 2099`

Gunakan tahun jauh ke depan (2099, 2100, dll) untuk testing agar tidak error "check_in di masa lalu".
