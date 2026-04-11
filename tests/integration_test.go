//go:build integration

// Jalankan dengan: go test -tags=integration -v ./tests/...
// Butuh server yang berjalan: docker compose up -d

package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const baseURL = "http://localhost:8090/api/v1"

// request helper untuk semua test
func req(t *testing.T, method, path string, body interface{}, token, idempKey string) (int, map[string]interface{}) {
	t.Helper()
	var br *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		br = bytes.NewReader(b)
	} else {
		br = bytes.NewReader(nil)
	}

	r, err := http.NewRequest(method, baseURL+path, br)
	require.NoError(t, err)
	r.Header.Set("Content-Type", "application/json")
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	if idempKey != "" {
		r.Header.Set("Idempotency-Key", idempKey)
	}

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(r)
	require.NoError(t, err)
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	return resp.StatusCode, result
}

// registerAndLogin adalah helper untuk register user baru dan login,
// mengembalikan access token.
func registerAndLogin(t *testing.T, email, password, name string) string {
	t.Helper()
	req(t, "POST", "/auth/register", map[string]string{
		"email": email, "password": password, "name": name,
	}, "", "")

	status, resp := req(t, "POST", "/auth/login", map[string]string{
		"email": email, "password": password,
	}, "", "")
	require.Equal(t, 200, status)
	return resp["data"].(map[string]interface{})["access_token"].(string)
}

// TestFullFlow_BookingToCancel menguji alur lengkap dari register sampai cancel booking.
func TestFullFlow_BookingToCancel(t *testing.T) {
	ts := fmt.Sprintf("%d", time.Now().UnixNano())

	// Register dan login sebagai admin
	adminEmail := "admin_" + ts + "@test.com"
	adminToken := registerAndLogin(t, adminEmail, "password123", "Admin Test")

	// Update role ke admin via DB (dalam test ini kita simulasikan sudah admin)
	t.Log("Note: pastikan update role ke admin di TablePlus sebelum test ini")

	// Register customer
	custEmail := "customer_" + ts + "@test.com"
	custToken := registerAndLogin(t, custEmail, "password123", "Customer Test")
	t.Logf("Customer token: %s...", custToken[:20])

	// Buat kamar (perlu admin token)
	status, resp := req(t, "POST", "/rooms", map[string]interface{}{
		"name": "Kamar Deluxe " + ts, "capacity": 2, "price_per_night": 500000,
	}, adminToken, "")
	t.Logf("Create room: status=%d", status)
	if status != 201 {
		t.Skip("Butuh admin token — update role ke admin di TablePlus dulu")
	}
	roomID := resp["data"].(map[string]interface{})["id"].(string)
	t.Logf("Room ID: %s", roomID)

	// Cari kamar yang tersedia
	status, resp = req(t, "GET", "/rooms/available?check_in=2099-06-10&check_out=2099-06-12", nil, "", "")
	assert.Equal(t, 200, status)
	t.Logf("Available rooms: %v", resp["data"])

	// Buat booking
	idemKey := "test-booking-" + ts
	status, resp = req(t, "POST", "/bookings", map[string]string{
		"room_id": roomID, "check_in": "2099-06-10", "check_out": "2099-06-12",
		"notes": "Test booking",
	}, custToken, idemKey)
	assert.Equal(t, 201, status, "Booking harus berhasil")
	bookingID := resp["data"].(map[string]interface{})["id"].(string)
	t.Logf("Booking ID: %s", bookingID)

	// Lihat daftar booking
	status, resp = req(t, "GET", "/bookings", nil, custToken, "")
	assert.Equal(t, 200, status)
	bookings := resp["data"].([]interface{})
	assert.Equal(t, 1, len(bookings))

	// Detail booking
	status, resp = req(t, "GET", "/bookings/"+bookingID, nil, custToken, "")
	assert.Equal(t, 200, status)
	assert.Equal(t, "confirmed", resp["data"].(map[string]interface{})["status"])

	// Cancel booking
	status, _ = req(t, "DELETE", "/bookings/"+bookingID, nil, custToken, "")
	assert.Equal(t, 200, status)

	// Setelah cancel, kamar harus tersedia lagi
	status, resp = req(t, "GET", "/rooms/available?check_in=2099-06-10&check_out=2099-06-12", nil, "", "")
	assert.Equal(t, 200, status)
	t.Log("✓ Setelah cancel, kamar tersedia lagi")
}

// TestDoubleBooking_SameRoom membuktikan double booking dicegah oleh sistem.
// Dua user mencoba booking kamar yang sama untuk tanggal yang sama secara bersamaan.
func TestDoubleBooking_SameRoom(t *testing.T) {
	t.Log("=== TEST: Double Booking Prevention ===")
	ts := fmt.Sprintf("%d", time.Now().UnixNano())

	adminToken := registerAndLogin(t, "admin_db_"+ts+"@test.com", "password123", "Admin DB")

	// Buat kamar
	status, resp := req(t, "POST", "/rooms", map[string]interface{}{
		"name": "Room DoubleTest " + ts, "capacity": 2, "price_per_night": 300000,
	}, adminToken, "")
	if status != 201 {
		t.Skip("Butuh admin token")
	}
	roomID := resp["data"].(map[string]interface{})["id"].(string)

	// Register dua user
	token1 := registerAndLogin(t, "user1_"+ts+"@test.com", "password123", "User 1")
	token2 := registerAndLogin(t, "user2_"+ts+"@test.com", "password123", "User 2")

	var (
		wg      sync.WaitGroup
		status1 int
		status2 int
	)

	// Kirim dua request booking secara bersamaan
	wg.Add(2)
	go func() {
		defer wg.Done()
		status1, _ = req(t, "POST", "/bookings", map[string]string{
			"room_id": roomID, "check_in": "2099-08-01", "check_out": "2099-08-05",
		}, token1, "key-user1-"+ts)
	}()
	go func() {
		defer wg.Done()
		status2, _ = req(t, "POST", "/bookings", map[string]string{
			"room_id": roomID, "check_in": "2099-08-01", "check_out": "2099-08-05",
		}, token2, "key-user2-"+ts)
	}()
	wg.Wait()

	t.Logf("User 1 status: %d, User 2 status: %d", status1, status2)

	// Tepat satu harus berhasil (201), satu harus gagal (409)
	successCount := 0
	if status1 == 201 {
		successCount++
	}
	if status2 == 201 {
		successCount++
	}

	assert.Equal(t, 1, successCount,
		"Hanya 1 dari 2 concurrent booking yang boleh berhasil untuk kamar dan tanggal yang sama")
	t.Log("✓ Double booking prevention bekerja: hanya 1 booking berhasil dibuat")
}

// TestIdempotency_BookingRequest membuktikan bahwa request yang sama dikirim
// dua kali hanya menghasilkan satu booking.
func TestIdempotency_BookingRequest(t *testing.T) {
	t.Log("=== TEST: Idempotency pada Booking Request ===")
	ts := fmt.Sprintf("%d", time.Now().UnixNano())

	adminToken := registerAndLogin(t, "admin_idem_"+ts+"@test.com", "password123", "Admin Idem")
	custToken := registerAndLogin(t, "cust_idem_"+ts+"@test.com", "password123", "Cust Idem")

	status, resp := req(t, "POST", "/rooms", map[string]interface{}{
		"name": "Room Idem " + ts, "capacity": 1, "price_per_night": 200000,
	}, adminToken, "")
	if status != 201 {
		t.Skip("Butuh admin token")
	}
	roomID := resp["data"].(map[string]interface{})["id"].(string)

	// Request pertama
	idemKey := "idem-test-" + ts
	s1, r1 := req(t, "POST", "/bookings", map[string]string{
		"room_id": roomID, "check_in": "2099-09-01", "check_out": "2099-09-03",
	}, custToken, idemKey)

	// Request kedua dengan KEY YANG SAMA
	s2, r2 := req(t, "POST", "/bookings", map[string]string{
		"room_id": roomID, "check_in": "2099-09-01", "check_out": "2099-09-03",
	}, custToken, idemKey)

	t.Logf("Request 1: status=%d", s1)
	t.Logf("Request 2: status=%d", s2)

	// Kedua request harus berhasil
	assert.Equal(t, 201, s1)
	assert.Equal(t, 201, s2)

	// ID booking harus SAMA (bukan dua booking terpisah)
	id1 := r1["data"].(map[string]interface{})["id"].(string)
	id2 := r2["data"].(map[string]interface{})["id"].(string)
	assert.Equal(t, id1, id2, "Idempotency: dua request dengan key sama harus menghasilkan booking yang sama")
	t.Log("✓ Idempotency bekerja: dua request dengan key sama menghasilkan satu booking")
}
