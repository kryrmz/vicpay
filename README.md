# VicPay (nombre de trabajo)

Super-app financiera global de Victor Lobo: billetera multimoneda, pagos QR, envios,
ahorros y marketplace, construidos sobre un ledger de doble entrada real.

> **"VicPay" es un nombre de trabajo TEMPORAL**, no la marca definitiva. Como marca
> publica ya esta tomado por terceros (VicPay = Vic.ai / Chainnova y otros), asi que
> sirve solo como nombre interno del proyecto hasta elegir el definitivo. El nombre y
> el dominio finales estan en decision: ver [`docs/nombre-y-dominio.md`](docs/nombre-y-dominio.md).
> El front esta tokenizado para rebrandear con cambiar `frontend/src/styles/tokens.css`.

## Estructura

| Ruta | Contenido |
|---|---|
| [`backend/`](backend) | API en Go 1.26 + Postgres: ledger de doble entrada, auth, OTP, KYC, billetera. |
| [`frontend/`](frontend) | App React 19 + Vite + Tailwind v4 + Capacitor: sistema de diseno, router, onboarding. |
| [`docs/`](docs) | Brief de producto, roadmap de arranque y decision de nombre/dominio. |
| [`deploy/`](deploy) | Deploy en un VPS con Docker Compose: Caddy (TLS + BFF), rol de BD de minimo privilegio y runbook. |
| [`.github/workflows/ci.yml`](.github/workflows/ci.yml) | CI: gate de verificacion de backend y frontend. |

## Backend (`backend/`)

Monolito Go con el patron Repository -> Service -> Handler. Piezas principales:

- **Ledger de doble entrada** (`internal/ledger`, `migrations/0002_ledger.sql`): append-only,
  con trigger de balance diferido y triggers de inmutabilidad (UPDATE/DELETE prohibidos),
  idempotencia por clave, locking determinista `FOR UPDATE` y reconciliacion cache-vs-journal.
- **Auth** (`internal/auth`): Argon2id (OWASP 2024), JWT access/refresh no intercambiables,
  rotacion de refresh con deteccion de reuso, cookie httpOnly `__Host-` y CSRF montado.
- **PII cifrada en la app** (`internal/pii`): AES-256-GCM + indice ciego HMAC (subclaves via
  HKDF), independiente de GUC de sesion para ser seguro bajo PgBouncer transaccional.
- **OTP real** (`internal/otp`), **KYC progresivo** (`internal/kyc`, nivel 0 sin friccion) y
  **billetera multimoneda** de lectura (`internal/wallet`).

Comandos (desde `backend/`):

```
cp .env.example .env
docker compose up --build       # postgres + pgbouncer + api en :8080
make verify                     # build + lint + gosec + tests
make test-integration           # requiere TEST_DB_DSN (Postgres DIRECTO)
```

Dos DSN separados por diseno: `DATABASE_URL` (pool de la app, puede ir por PgBouncer) y
`DATABASE_DIRECT_URL` (directo, para migraciones y advisory locks de sesion).

## Frontend (`frontend/`)

SPA React 19 + TypeScript + Vite + Tailwind v4 (tokens en CSS) + Capacitor, con router real
(`react-router-dom`), manejo del boton atras de Android, dinero en enteros (unidades menores)
y una unica utilidad de formato, auth con token solo en memoria y cero PII en `localStorage`.

```
cd frontend
npm install
npm run dev                     # http://localhost:5173
npm run typecheck && npm run lint && npm run test && npm run build
```

Cuenta sembrada para ver la UI con datos: `+50688888888` / `VicPay#2026`.

Contra el backend real, construir/servir con `VITE_API_BASE=/api` tras el proxy BFF
(en dev, `VITE_API_BASE=/api npm run dev` usa el proxy de Vite hacia `localhost:8080`).

## Deploy

Para un VPS con Docker Compose ver [`deploy/README.md`](deploy/README.md): Caddy termina
TLS y sirve el front proxyando `/api` (cookie httpOnly first-party), Postgres y PgBouncer
quedan en la red privada, y la app corre como un rol de BD de minimo privilegio que no puede
mutar el journal. Falta para un lanzamiento real: proveedor de SMS, endpoints de movimiento
de dinero, y la via regulatoria.

## Limite regulatorio

El codigo llega hasta el limite honesto sin degradar a demo: billetera, ledger, QR, ahorros y
marketplace funcionan de verdad por el ledger. La entrada/salida de dinero real (on-ramp/payout,
rails locales, VASP, emisor de tarjetas) esta **bloqueada por licencia/partner**, no por codigo.
Mover dinero real sin via regulatoria resuelta configura captacion ilegal (Ley 7558 art. 116 CR).
Ver [`docs/brief-producto.md`](docs/brief-producto.md).

## Referencia

Proyecto previo del mismo dueno: **KiramoPay**. VicPay reusa su ledger de doble entrada y aplica
las mejoras heredadas de sus auditorias (sesion httpOnly/BFF, OTP real, PgBouncer/PII, router con
boton atras, tokens de color, KYC diferido, inmutabilidad del ledger). Ver
[`docs/roadmap-arranque.md`](docs/roadmap-arranque.md).
