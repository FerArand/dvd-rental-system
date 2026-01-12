
# DVD Rental API (Go)

- REST API in Go against the **dvdrental** PostgreSQL sample.
- Endpoints:
  - `POST /api/auth/login` (email + role: "staff" or "customer")
  - `POST /api/rentals` (create rental)
  - `POST /api/returns/{rental_id}` (return a rental)
  - `POST /api/rentals/{rental_id}/cancel` (cancel an open rental by deleting it)
  - `GET /api/inventory/available?film_id=ID`
  - Reports:
    - `GET /api/reports/customer/{customer_id}/rentals`
    - `GET /api/reports/not-returned`
    - `GET /api/reports/top-rented?limit=N`
    - `GET /api/reports/revenue-by-staff`
- Run locally with Docker Compose (place `db-init/dvdrental.tar` first).

## Quickstart

```bash
cd backend
# copy dvdrental.tar into backend/db-init/
docker compose up --build
# in another shell:
./scripts/test.sh
```
