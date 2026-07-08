# Deploy gestionado gratis: Neon + Render + Vercel

Alternativa al VPS, sin lotería de capacidad y $0. Tres servicios con free tier:
Postgres en **Neon**, backend en **Render**, frontend en **Vercel**. El frontend
proxya `/api` al backend (via `frontend/vercel.json`) para que la cookie httpOnly
siga siendo first-party.

## Requisito previo
El código debe estar en un repo de **GitHub** (Render y Vercel despliegan desde ahí).

## 1. Base de datos: Neon
1. Crea una cuenta en https://neon.tech y un proyecto (region cercana, ej. US West).
2. En el proyecto, abre **Connection Details** y copia DOS cadenas:
   - La **pooled** (dice "Pooled connection" / host con `-pooler`) -> sera `DATABASE_URL`.
   - La **direct** (sin `-pooler`) -> sera `DATABASE_DIRECT_URL`.
   Ambas ya vienen con `?sslmode=require` (Neon exige TLS: perfecto, sin cambios).

## 2. Backend: Render
1. Cuenta en https://render.com, conecta tu GitHub.
2. **New + -> Blueprint** -> elige el repo `vicpay`. Render lee `render.yaml` y crea
   el servicio `vicpay-api` (Docker, plan free).
3. En las variables del servicio, pon:
   - `DATABASE_URL` = la cadena **pooled** de Neon.
   - `DATABASE_DIRECT_URL` = la cadena **direct** de Neon.
   - `CORS_ORIGINS` = tu URL de Vercel (la pones tras el paso 3; ej. `https://vicpay.vercel.app`).
   - `JWT_SECRET` y `PII_ENCRYPTION_KEY` se generan solos (si el arranque se queja de
     longitud, ponles un valor de >=32 caracteres a mano).
4. Deploy. Anota la URL, ej. `https://vicpay-api.onrender.com`.

## 3. Frontend: Vercel
1. Cuenta en https://vercel.com, **Add New -> Project** -> importa el repo.
2. **Root Directory: `frontend`**. Framework: Vite (autodetectado).
3. Edita `frontend/vercel.json`: cambia el host `vicpay-api.onrender.com` por tu URL
   real de Render (del paso 2). Commit.
4. Deploy. Anota la URL, ej. `https://vicpay.vercel.app`.

## 4. Cerrar el círculo
- En Render, pon `CORS_ORIGINS` = tu URL de Vercel exacta y redeploy.
- Abre tu URL de Vercel: el front llama a `/api/*`, Vercel lo proxya a Render, y la
  cookie httpOnly funciona same-origin.

## Notas importantes (honestas)
- **OTP en el demo:** en produccion no hay proveedor de SMS gratis, asi que el codigo
  no se entrega. Para probar el registro en el demo, en Render pon temporalmente
  `ENVIRONMENT=development` y `OTP_DEV_ECHO=true`, y lee el codigo en los **logs de
  Render**. Para usuarios reales necesitas un proveedor de SMS (de pago).
- **Render free duerme** tras ~15 min de inactividad: la primera visita luego de dormir
  tarda ~30-60 s en despertar. Neon free tambien pausa la BD tras inactividad.
- **Rol de BD de minimo privilegio:** en Neon usas su rol por defecto (los triggers de
  inmutabilidad siguen protegiendo el journal). El rol `vicpay_app` separado del VPS es
  hardening extra, opcional aqui.
