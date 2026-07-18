# Fintech API

A RESTful fintech API built in Go, featuring JWT authentication, wallet operations (deposit, withdraw, transfer), and a full transaction ledger backed by SQLite. Built and containerized as an individual DevOps project.

**Author:** Ihwe Mathias ([@maihwe](https://github.com/maihwe))
**Repository:** https://github.com/maihwe/fintech-api

## Features

- User registration with bcrypt password hashing
- JWT-based authentication (stateless, signed, 1-hour expiry)
- Wallet balance stored as integer kobo (no floating-point money bugs)
- Deposit, withdraw, and transfer between users
- Every balance-changing operation is wrapped in a database transaction, so a transfer either fully succeeds or fully fails — no partial money movement
- Full transaction history per user
- Dockerized with a multi-stage build (final image: **11.9MB**)
- Persistent storage via a Docker named volume
- Secrets and configuration supplied via environment variables, never hardcoded
- Automated CI build check via GitHub Actions

## Tech Stack

- **Language:** Go 1.22
- **Database:** SQLite via `modernc.org/sqlite` (pure Go, no CGO required)
- **Auth:** `golang-jwt/jwt` (JWT) + `golang.org/x/crypto/bcrypt` (password hashing)
- **Containerization:** Docker (multi-stage build), Docker Compose
- **CI:** GitHub Actions

## API Endpoints

| Method | Endpoint | Auth required | Description |
|---|---|---|---|
| POST | `/register` | No | Create a new user |
| POST | `/login` | No | Authenticate, returns a JWT |
| GET | `/balance` | Yes | Get current user's balance |
| POST | `/deposit` | Yes | Add funds to own balance |
| POST | `/withdraw` | Yes | Remove funds from own balance |
| POST | `/transfer` | Yes | Move funds to another user by username |
| GET | `/transactions` | Yes | List current user's transaction history |

Protected routes require an `Authorization: Bearer <token>` header, using the token returned from `/login`.

### Example requests

**Register**
```bash
curl -X POST localhost:8080/register \
  -H "Content-Type: application/json" \
  -d '{"username":"mathias","password":"testpass123"}'
```

**Login**
```bash
curl -X POST localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{"username":"mathias","password":"testpass123"}'
```

**Check balance**
```bash
curl localhost:8080/balance \
  -H "Authorization: Bearer <token>"
```

**Deposit** (amount in kobo — 500000 = ₦5,000)
```bash
curl -X POST localhost:8080/deposit \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"amount_kobo": 500000}'
```

**Transfer**
```bash
curl -X POST localhost:8080/transfer \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"to_username":"otheruser","amount_kobo":100000}'
```

## Running Locally (without Docker)

Requires Go 1.22+.

```bash
git clone https://github.com/maihwe/fintech-api.git
cd fintech-api
go mod tidy
JWT_SECRET=your-dev-secret go run .
```

Server starts on port 8080 by default. Override with the `PORT` environment variable.

## Running with Docker

**Build and run standalone:**
```bash
docker build -t fintech-api .
docker run -d --name fintech-api -p 8080:8080 \
  -e JWT_SECRET=your-dev-secret \
  -e DB_PATH=/app/data/fintech.db \
  -v fintech-data:/app/data \
  fintech-api
```

**Or with Docker Compose (recommended):**

1. Copy `.env.example` to `.env` and set a real secret:
   ```bash
   cp .env.example .env
   ```
2. Start the service:
   ```bash
   docker compose up -d --build
   ```

The `-v` volume / `docker-compose.yml` volume mount ensures the SQLite database persists across container restarts and rebuilds — verified by destroying and recreating the container and confirming user data survived.

## DevOps Decisions

This project was scoped explicitly as a DevOps exercise — the choices below were made deliberately, not by default.

- **CGO-free build (`modernc.org/sqlite`, `CGO_ENABLED=0`):** avoids needing a C toolchain in the build image, which keeps the Dockerfile simpler and enables a fully static binary.
- **Multi-stage Dockerfile:** the build stage (`golang:1.22-alpine`) compiles the binary; only the compiled binary is copied into a fresh `alpine:3.20` runtime image. This keeps the final image at **11.9MB** instead of the 300–800MB+ a naive single-stage build would produce, since none of the Go toolchain or build cache ships in the final image.
- **Config via environment variables (`PORT`, `DB_PATH`, `JWT_SECRET`):** no hardcoded values in source code, so the same image can run in different environments (dev, staging, production) with different configuration, and no secret needs to be baked into the image or committed to Git.
- **Named Docker volume for the SQLite file:** without this, the database lives in the container's writable layer and is destroyed whenever the container is removed. This was demonstrated directly during development — a container recreated without a volume lost all data; the same test with a volume attached preserved it correctly.
- **`.env` / `.env.example` pattern:** real secrets stay local and git-ignored; `.env.example` documents what configuration is required without exposing actual values.
- **GitHub Actions CI (`.github/workflows/docker-build.yml`):** every push to `main` triggers an independent build of the Docker image on GitHub's infrastructure, giving a reproducible, third-party-verifiable confirmation that the Dockerfile builds correctly — not just a claim that it works locally.

## Security Notes

- Passwords are hashed with bcrypt (default cost) before storage; plaintext passwords are never stored or logged.
- Login returns an identical error message and status code for both "user not found" and "wrong password," to prevent username enumeration.
- Money is stored as an integer (kobo), never a float, to avoid floating-point rounding errors in financial calculations.
- Every deposit, withdrawal, and transfer writes to the transaction log and updates the balance within a single database transaction, so the balance and the ledger can never drift out of sync.
- JWTs are signed with HMAC-SHA256 and expire after 1 hour, limiting the exposure window if a token is ever leaked.

## Project Structure

```
fintech-api/
├── main.go            # server setup, route registration
├── models.go           # User and Transaction structs
├── db.go                # database connection and schema
├── auth.go              # register, login, JWT generation
├── middleware.go         # JWT verification middleware
├── wallet.go              # balance, deposit, withdraw, transfer, transactions
├── Dockerfile              # multi-stage build
├── docker-compose.yml       # local orchestration with volume + env support
├── .env.example              # documents required environment variables
└── .github/workflows/
    └── docker-build.yml       # CI build check
```