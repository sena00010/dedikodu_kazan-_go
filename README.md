# Dedikodu Kazani Backend

Go API, MySQL, Redis Pub/Sub, Firebase Auth, RevenueCat webhook ve AI provider katmanlarini icerir.

## Local kurulum

1. MySQL/XAMPP calistir.
2. Migration uygula:

```bash
/Applications/XAMPP/xamppfiles/bin/mysql -h 127.0.0.1 -P 3307 -u root -e "source /Users/mac/Desktop/pratik-gumruk/dedikodu-kazani/backend/migrations/001_init.sql"
```

3. Ortam degiskenlerini ayarla:

```bash
cp backend/.env.example backend/.env
```

4. API'yi calistir:

```bash
cd backend
go mod tidy
go run ./cmd/api
```

## Ana endpointler

- `POST /api/auth/login`: Firebase ID token ile Go JWT alir.
- `POST /api/auth/register`: Email/sifre ile MySQL kullanicisi olusturur.
- `POST /api/auth/email-login`: Email/sifre girisi.
- `GET /api/user/me`: Profil, VIP ve jeton.
- `GET /api/threads`: Ana kazan listesi.
- `POST /api/threads`: Yeni giybet konusu.
- `POST /api/threads/:id/comments`: Mesaj gonderir ve AI cevabini tetikler.
- `POST /api/rooms/:id/invite-bot`: 50 jeton duser, oda AI botunu 1 saat aktif eder.
- `GET /api/rooms/:id/comments`: Ozel oda mesaj gecmisi.
- `POST /api/rooms/:id/comments`: Ozel odaya mesaj gonderir; AI aktifse bot cevabini tetikler.
- `POST /api/webhooks/revenuecat`: RevenueCat eventleri ile credits/VIP gunceller.
- `GET /api/ws?rooms=thread:<id>,room:<id>`: JWT ile WebSocket baglantisi.
