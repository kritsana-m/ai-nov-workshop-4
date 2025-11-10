# Backend (Gin + Viper)

A small Go backend using Gin (HTTP router), Viper for configuration, and GORM (SQLite) as the persistence layer. This service exposes a simple user management API and point transfer functionality between users. It includes point ledger entries for every transfer.

## Contents

- `cmd/server/main.go` - server entrypoint
- `internal/config` - configuration helpers
- `internal/handlers` - HTTP handlers and routes
- `internal/models` - GORM models (User, Transfer, PointLedger)
- `internal/store` - DB initialization and helpers
- `config.yaml` - default configuration

## Requirements

- Go 1.20+ (or the version the project uses; check `go.mod`)
- (Optional) Docker/containers if you want to containerize the app

## Configuration

Default configuration is in `config.yaml`. Important fields:

- `server.port` — port the HTTP server listens on (default `3000`)
- `database.path` — path to the SQLite file used by GORM (defaults to `./data.db` or as used in development)

Example `config.yaml` (already included):

```yaml
server:
  port: 3000
database:
  path: ./data.db
```

## Run locally

From the `workshop-4/backend` folder:

```bash
# fetch dependencies (if needed)
go mod tidy

# run the server (uses config.yaml by default)
go run cmd/server/main.go
```

The API will be available at `http://localhost:3000` (or the configured port). You can override the configuration by editing `config.yaml` in the same folder.

### Quick API examples

Create a user:

```bash
curl -sS -X POST "http://localhost:3000/users" \
  -H 'Content-Type: application/json' \
  -d '{"member_code":"A100","name":"Alice","remaining_points":100}' | jq .
```

Create a transfer:

```bash
curl -sS -X POST "http://localhost:3000/transfers" \
  -H 'Content-Type: application/json' \
  -d '{"fromUserId":1,"toUserId":2,"amount":10}' | jq .
```

## API Endpoints

All endpoints use JSON. The project registers these routes in `internal/handlers`:

- Users
  - `GET /users` — list all users (ordered by id desc)
  - `GET /users/:id` — get user by numeric ID
  - `POST /users` — create a new user
    - body: { member_code, membership_level, name, surname, phone, email, registration_date, remaining_points }
    - `registration_date` will default to today's date if omitted
  - `PUT /users/:id` — update user (replace fields)
  - `DELETE /users/:id` — delete user

- Transfers
  - `POST /transfers` — create a transfer
    - body: { fromUserId, toUserId, amount, note }
    - Validations:
      - `fromUserId` and `toUserId` are required
      - `amount` must be > 0
      - cannot transfer to self (returns 422)
      - insufficient points returns 409 Conflict
    - The handler runs the transfer in a DB transaction and creates two `PointLedger` entries (transfer_out, transfer_in). The transfer record includes an `IdempotencyKey` header value (Idempotency-Key) in the response.
  - `GET /transfers` — list transfers (supports `userId`, `page`, `pageSize` query params)
  - `GET /transfers/:id` — get transfer by idempotency key (note: handler looks up `idempotency_key`)

## Data Models (summary)

Defined in `internal/models/models.go` (GORM tags shown in code):

- User
  - ID (uint, primary key)
  - MemberCode (string, unique, not null)
  - MembershipLevel (string)
  - Name, Surname, Phone, Email (string)
  - RegistrationDate (string)
  - RemainingPoints (int)

- Transfer
  - ID (uint, primary key)
  - FromUserID, ToUserID (uint) — foreign references to `users.id`
  - Amount (int), Status (string)
  - Note (*string)
  - IdempotencyKey (string, unique)
  - CreatedAt, UpdatedAt, CompletedAt (timestamps)
  - FailReason (*string)

- PointLedger
  - ID (uint, primary key)
  - UserID (uint) — reference to `users.id`
  - Change (int)
  - BalanceAfter (int)
  - EventType (string) e.g., `transfer_out`, `transfer_in`
  - TransferID (*uint) — optional reference to `transfers.id`
  - Reference, Metadata (*string)
  - CreatedAt (timestamp)

See `database.md` for a Mermaid ER diagram that documents the relationships.

## Testing

Unit tests live in `internal/handlers/handlers_test.go` and use Gin's test router and an in-memory SQLite database. Run unit tests with:

```bash
go test ./... -v
```

For end-to-end manual verification there is a helper script `test_api.sh` that exercises the main API flows (create user, transfer, check balances). Start the server first, then run:

```bash
./test_api.sh
```

Notes about tests:

- Tests create isolated in-memory SQLite instances per test to avoid interference.
- Tests cover user CRUD flows, successful transfers (and ledger records), insufficient-points handling, invalid user errors, self-transfer validation, and transfer listing/filtering.

## Development notes

- The project uses GORM's `AutoMigrate` to create tables at startup (see `internal/store/store.go`). For production, prefer explicit migrations.
- Idempotency: Transfers include an `IdempotencyKey` which is unique constrained by the DB. The create transfer handler generates a UUID and returns it in the `Idempotency-Key` response header.
- Error handling: handlers return appropriate HTTP status codes for common errors (400, 404, 409, 422, 500).

## Troubleshooting

- If you see `UNIQUE constraint failed: transfers.idempotency_key` during tests or development, it means a seed or previous run created a transfer with an empty or duplicated idempotency key — using unique keys or resetting the DB will resolve it.

## Contributing

If you add endpoints or change models, update `database.md` and add corresponding unit tests.

## License

This workshop repository does not include a license by default — add one if you plan to publish.

