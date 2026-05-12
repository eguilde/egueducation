# Local Development

This project can run locally without the Forgejo build pipeline.

## Backend

The backend loads environment variables from:

- `.env`
- `backend/.env`

`backend/.env` is intentionally ignored by Git and should contain the real local runtime credentials.

Required local database settings:

```dotenv
DATABASE_HOST=db.eguilde.cloud
DATABASE_PORT=5432
DATABASE_NAME=scoalabalotesti
DATABASE_USERNAME=egueducation_scoalabalotesti_app
DATABASE_PASSWORD=<secret>
DATABASE_SSLMODE=require
DATABASE_URL=postgres://egueducation_scoalabalotesti_app:<url-encoded-secret>@db.eguilde.cloud:5432/scoalabalotesti?sslmode=require
```

Refresh the ignored local env file from the Kubernetes secrets:

```powershell
cd E:\dev\egueducation
.\scripts\update-local-env.ps1
```

Run the backend:

```powershell
cd E:\dev\egueducation
.\scripts\start-backend.ps1
```

The API listens on `http://localhost:8080`.

## Frontend

Run the Angular dev server:

```powershell
cd E:\dev\egueducation
.\scripts\start-frontend.ps1
```

The frontend listens on `http://localhost:4200` and proxies `/api` to `http://localhost:8080`.

## Docker Compose

The compose stack also reads `backend/.env`:

```powershell
cd E:\dev\egueducation\ops
docker compose up --build
```
