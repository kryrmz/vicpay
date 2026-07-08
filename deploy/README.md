# Deploy en un VPS (Docker Compose)

Stack de produccion en un solo host: Caddy (TLS + BFF) expuesto a internet;
Postgres, PgBouncer y la API en la red privada de Docker (sin puertos publicados).
La API corre migraciones como dueno y sirve trafico como el rol de minimo
privilegio `vicpay_app`, que no puede mutar el journal.

## Prueba local gratis (sin dominio, sin cuenta cloud)

Para verlo funcionando en tu maquina, sin pagar ni esperar a ningun proveedor:

```sh
docker compose -f deploy/docker-compose.local.yml up --build
```

Abre http://localhost. Durante el registro, el codigo OTP aparece en los logs:

```sh
docker compose -f deploy/docker-compose.local.yml logs api | grep "otp dev echo"
```

Corre en modo desarrollo (HTTP, un solo rol de BD, OTP a log); es solo para tu
maquina. Para exponerlo a internet usa el stack de produccion de abajo.

## Opcion 100% gratis para publicarlo: Oracle Always Free + DuckDNS

Un servidor publico sin costo:

1. Crea una VM ARM "Always Free" (Ampere A1) en Oracle Cloud, con Ubuntu. La cuenta
   tarda un rato en aprovisionarse tras el registro; espera el correo de "listo".
2. En Oracle, abre los puertos 80 y 443 en la Security List (ingress 0.0.0.0/0) y en
   el firewall del SO (`sudo ufw allow 80/tcp && sudo ufw allow 443/tcp`).
3. Instala Docker en la VM: `curl -fsSL https://get.docker.com | sh`.
4. Consigue un subdominio gratis en https://www.duckdns.org (ej. `vicpay.duckdns.org`)
   y apuntalo a la IP publica de la VM.
5. `cd deploy && cp .env.production.example .env`, y pon
   `SITE_ADDRESS=vicpay.duckdns.org`, `SITE_ORIGIN=https://vicpay.duckdns.org` y los
   secretos. Sigue el bootstrap y el arranque de abajo: Caddy saca el certificado TLS
   gratis automaticamente para ese subdominio.

Costo: $0. Un dominio propio de marca (mas serio que duckdns) cuesta ~$1-13/ano y
solo cambia `SITE_ADDRESS`/`SITE_ORIGIN`.

## Requisitos

- Un VPS con Docker y Docker Compose v2.
- Un dominio apuntando al VPS (registro A) para el HTTPS automatico de Caddy.
- Puertos 80 y 443 abiertos.

## Configuracion

```sh
cd deploy
cp .env.production.example .env
# Edita .env. Genera secretos fuertes:
#   openssl rand -base64 36   # JWT_SECRET, PII_ENCRYPTION_KEY (>= 32 bytes)
#   openssl rand -base64 24   # OWNER_DB_PASSWORD, APP_DB_PASSWORD
```

## Bootstrap (una sola vez)

```sh
set -a; . ./.env; set +a
COMPOSE="docker compose -f docker-compose.prod.yml"

# 1. Levanta solo la base de datos.
$COMPOSE up -d postgres

# 2. Aplica el esquema como dueno (migrate-only sale al terminar).
$COMPOSE run --rm -e MIGRATE_ONLY=true api

# 3. Crea el rol de minimo privilegio y sus grants (ya existen las tablas).
$COMPOSE exec -T postgres \
  psql -U vicpay -d vicpay -v app_password="'$APP_DB_PASSWORD'" < roles.sql
```

## Arranque

```sh
$COMPOSE up -d --build
curl -sf https://$SITE_ADDRESS/api/healthz    # -> {"data":{"status":"ok"}}
```

Abre `https://$SITE_ADDRESS` y prueba el flujo de registro.

## Actualizaciones

```sh
git pull
docker compose -f docker-compose.prod.yml up -d --build
```

La API corre las migraciones pendientes al arrancar (como dueno). Vuelve a
ejecutar el paso 3 (`roles.sql`) solo si una migracion nueva agrego tablas.

## Respaldos

```sh
docker compose -f docker-compose.prod.yml exec -T postgres \
  pg_dump -U vicpay -Fc vicpay > backup-$(date +%F).dump
```

Guarda los respaldos cifrados y fuera del host. Prueba la restauracion.

## Notas de seguridad

- `SITE_ORIGIN` debe coincidir exactamente con el origen que ve el navegador; es
  la allowlist de CORS/CSRF del backend. Sin coincidencia, `/api/auth/refresh`
  y `/api/auth/logout` responden 403.
- Los secretos van en `deploy/.env` (ignorado por git). En un entorno serio,
  usa un gestor de secretos en vez de un archivo plano.
- El rol `vicpay_app` no tiene privilegio de DDL ni de UPDATE/DELETE sobre el
  journal: es un muro que complementa los triggers de inmutabilidad.

## Lo que aun falta para un lanzamiento real (no cubierto por este deploy)

- Proveedor de SMS real para el OTP (hoy solo el emisor de log en desarrollo).
- Endpoints de movimiento de dinero (transferencias/QR) sobre el ledger con
  aplicacion de limites KYC.
- Redis para rate-limit y revocacion de sesion si se escala a varias instancias.
- Observabilidad (metricas/trazas) y una cadencia de restore-test.
- La via regulatoria (EDE/sponsor bank) antes de custodiar dinero real.
