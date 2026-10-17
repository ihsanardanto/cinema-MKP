# A. Rancangan sistem pembelian tiket bioskop nasional

## Alur untuk pelanggan

Pilih kota, bioskop, film dan jam -> lihat kursi -> pilih kursi -> sistem menahan kursi 5 menit -> bayar -> tiket terbit. Jika waktu habis sebelum pembayaran dikonfirmasi, kursi dilepas. Jika bioskop membatalkan pertunjukan, pelanggan diberi tahu dan uang dikembalikan; jadwal tersebut tidak dijual kembali.

## Arsitektur dan skala

Client -> CDN/WAF/API gateway (TLS, rate limiting) -> stateless API replicas -> PostgreSQL primary. Read replicas dan cache melayani katalog; keputusan kepemilikan kursi selalu di primary. Worker membaca transactional outbox untuk notifikasi, payment reconciliation, refund dan invalidasi cache. Payment gateway berada di luar transaksi database. Komponen eksternal ini adalah rancangan A, bukan layanan yang diimplementasikan pada C.

Partition/shard berdasarkan showtime atau bioskop bila pengukuran beban membutuhkannya; seluruh kursi satu booking tetap satu shard. Cache boleh menampilkan data tertunda, tetapi booking harus memeriksa ulang database. Tidak mengandalkan Redis lock untuk menjamin kepemilikan kursi. Batasi pool koneksi, gunakan pagination, indeks, timeout, observability dan backpressure.

## Jaminan tidak ada kursi ganda

Materialisasikan satu showtime_seats per kursi per pertunjukan. PK (showtime_id,seat_id) memberikan satu sumber kebenaran. Setiap transaksi booking, callback pembayaran, expiry dan cancellation mengunci showtime terlebih dahulu, kemudian booking dan kursi dengan urutan seat_id yang konsisten. Jalur booking menggunakan lock showtime FOR SHARE; cancellation dan perubahan jadwal menggunakan FOR UPDATE. Tidak ada panggilan jaringan saat lock ditahan.

Booking: BEGIN; periksa showtime masih scheduled dan belum mulai; lock semua kursi yang diminta FOR UPDATE menurut seat_id; pastikan semuanya available (atau lakukan expiry pemilik lama secara atomik); buat booking held dan item harga; ubah semua kursi menjadi held dengan booking_id dan deadline yang sama; COMMIT. Jika satu kursi gagal, ROLLBACK seluruh booking. Batasi jumlah kursi per permintaan. Gunakan request idempotency key unik per user pada implementasi booking produksi.

Payment callback: verifikasi signature, merchant, currency, jumlah dan provider reference; deduplikasi event. Lock dengan urutan yang sama. Hanya konversi held -> sold jika pemilik, deadline dan status jadwal masih valid; update booking paid, payment dan outbox dalam transaksi sama. Callback duplikat mengembalikan hasil lama. Pembayaran terlambat tetap dicatat, tetapi tidak mengambil kursi dari pemilik baru: buat refund_pending dan outbox refund. Callback bukan bukti bahwa tiket harus diterbitkan.

## Pencatatan dan restok

Kapasitas dihitung per pertunjukan, bukan mengurangi kapasitas fisik studio. available = jumlah baris available pada showtime_seats. Booking_items dan payments menyimpan sejarah; harga disalin saat booking. Inventori bukan counter cache yang dapat drift. Worker expiry memakai batch, FOR UPDATE SKIP LOCKED pada baris showtime terlebih dahulu, lalu booking/kursi; hanya melepas kursi held milik booking expired, memperbarui booking expired secara atomik. Periksa deadline saat pembelian juga sehingga keterlambatan worker tidak merusak konsistensi.

Pembatalan pelanggan tunduk kebijakan: sebelum cutoff, batalkan tiket dan release sold/held miliknya dalam transaksi, lalu refund asinkron. Kursi hanya dapat dijual lagi jika showtime scheduled dan belum mulai. Kegagalan refund dicatat untuk retry/penanganan petugas; status refund tidak boleh ditampilkan succeeded sebelum konfirmasi provider.

Pembatalan bioskop: lock showtime FOR UPDATE, ubah cancelled sehingga seluruh penjualan berhenti; tandai booking held cancelled dan paid refund_pending, blokir inventori dan tulis outbox pada transaksi yang sama. Untuk volume besar gunakan state cancelling sebagai gerbang penjualan dan proses booking dalam batch idempoten (perlu perluasan enum schema). Kirim refund penuh ke pembayaran asal dengan idempotency key refund ID. Retry dengan backoff, rekonsiliasi settlement dan eskalasi kegagalan; beri notifikasi progres. Jadwal cancelled tidak direstok. Jadwal pengganti adalah showtime baru.

## Keandalan dan keamanan

Outbox memastikan commit database dan pengiriman event tidak terpisah; pengiriman setidaknya sekali menuntut consumer idempoten. Terapkan PITR, backup teruji, failover primary dengan fencing, audit trail operator serta metrik lock contention, expiry, payment mismatch dan refund backlog. TLS dan pengelolaan secret wajib untuk produksi; jangan log password atau bearer token. Rate limit login di gateway dengan limiter bersama lintas replica, bukan memory lokal.
