# MKP / APPS2PAY - Backend Development Test

**Nama Lengkap kandidat: [ISI NAMA LENGKAP ANDA]**
Tanggal: Sabtu, 17 Oktober 2026 | Email: ihsanardanto@gmail.com | Telepon: 081237595265
Repository: https://github.com/ihsanardanto/cinema-MKP

## Deliverables

- A: `jawaban/A - Topology.jpg`, `jawaban/A1 - Flowchart.jpg`, dan `jawaban/A2 - System Design.md`.
- B: `jawaban/B - Database Design.jpg` dan `db/migrations/`.
- C: API Go + PostgreSQL dan `Makefile`.
- Postman: `jawaban/C - MKP Cinema API.postman_collection.json`; token dan ID otomatis tersimpan.

## Jalankan

Prasyarat: Go 1.23+, PostgreSQL lokal, `psql`, dan `make`. PostgreSQL harus berjalan, dan user database lokal harus memiliki izin untuk membuat extension `btree_gist` dan `pgcrypto`.

```sh
make migrate
make run
```

`make migrate` menerapkan migrasi yang belum dijalankan dan mencatatnya pada tabel `schema_migrations`. Koneksi default memakai Unix socket PostgreSQL dengan user shell saat ini dan database `cinema`. Untuk mengganti koneksi, berikan `DATABASE_URL`:

```sh
make migrate DATABASE_URL='user=YOUR_USER dbname=cinema host=/var/run/postgresql sslmode=disable'
make run DATABASE_URL='user=YOUR_USER dbname=cinema host=/var/run/postgresql sslmode=disable'
```

Untuk reset data lokal (menghapus semua tabel dan data demo), hentikan API lalu jalankan:

```sh
make reset-db
```

Login demo: admin@example.com / ChangeMe123! (khusus lokal). API di http://localhost:8080. Import koleksi Postman, lalu jalankan collection berurutan. Tidak ada token statis yang perlu diisi. Tanggal contoh dihitung otomatis dari waktu sekarang. Jalankan hanya satu collection sekaligus karena studio demo tunggal.

## Endpoint

| Method | Path                             | Access        | Result                          |
| ------ | -------------------------------- | ------------- | ------------------------------- |
| POST   | /api/login                       | Public        | 200 bearer token, expiry 1 hour |
| GET    | /api/showtimes?limit=50&offset=0 | Authenticated | 200 list, limit 1..100          |
| GET    | /api/showtimes/{id}              | Authenticated | 200 or 404                      |
| POST   | /api/showtimes                   | Admin         | 201                             |
| PUT    | /api/showtimes/{id}              | Admin         | 200, full replacement           |
| DELETE | /api/showtimes/{id}              | Admin         | 204                             |

Create/update body: movie_id, auditorium_id, starts_at, ends_at, price (integer IDR). RFC3339 timestamps, future start, duration at least movie duration. Demo IDs: movie=1, auditorium=1. Unknown fields, malformed JSON and extra JSON rejected. Errors: 400 invalid input/reference; 401 missing/expired credentials; 403 wrong role; 404 missing record; 409 overlap or booked resource; 500 generic server error. All SQL parameters bound. Tokens random 256-bit, only SHA-256 digest stored; bcrypt passwords. Schedule and seat inventory change in one transaction. Booked schedules cannot be edited/deleted; cancellation requires the separate refund workflow documented in A.
