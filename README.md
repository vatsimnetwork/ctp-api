# ctp-api

Central Data Hub for CTP Planning — a .NET 10 Web API.

## Tech Stack

| Component | Technology |
|---|---|
| Framework | ASP.NET Core 10 |
| Database | PostgreSQL 16 |
| Containerization | Docker / Docker Compose |
| Auth | API Key Middleware (externally validated) |

## Prerequisites

- [.NET 10 SDK](https://dotnet.microsoft.com/download)
- [Docker](https://www.docker.com/) & Docker Compose

## Configuration

Copy `.env.example` to `.env` inside the `ctp-api/` folder and adjust the values:

```bash
cp ctp-api/.env.example ctp-api/.env
```

```env
AuthServiceURL=http://host.docker.internal:5000

POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres_password
POSTGRES_DB=ctp_api_db

ConnectionStrings__DefaultConnection=Host=postgres;Port=5432;Database=ctp_api_db;Username=postgres;Password=postgres_password
```

## Running

### With Docker Compose (recommended)

```bash
docker compose up --build
```

The API will be available at `http://localhost:8080`.

### Locally (without Docker)

```bash
cd ctp-api
dotnet run
```

> In development mode the OpenAPI spec is served at `/openapi/v1.json`.

## Authentication

Every request must include a valid API key in the request header:

```http
X-API-Key: your_api_key_here
```

The key is validated by the auth service configured via `AuthServiceURL`.
If the header is missing or the key is invalid, the API responds with `401 Unauthorized`.

For more details see [docs/api/Api-Auth.md](docs/api/Api-Auth.md).

## Project Structure

```
ctp-api/
├── Middleware/
│   └── ApiKeyMiddleware.cs   # API key validation
├── Program.cs                # App configuration & startup
├── appsettings.json
├── Dockerfile
docs/
└── api/
    └── Api-Auth.md           # Auth documentation
docker-compose.yml
```

## License

See [LICENSE](LICENSE).
