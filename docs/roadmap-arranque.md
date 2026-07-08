# Roadmap de arranque

## Decision previa: base tecnica

**Recomendacion**: partir del back/ledger real de **KiramoPay** (en `../kiramopay`),
que ya tiene un ledger de doble entrada probado, y construir encima un front y una
marca nuevos + las mejoras de abajo. Alternativa (mas cara): base nueva desde cero.
Decidir esto antes de escribir codigo.

Stack propuesto (heredado, a confirmar): **Go + Postgres** (ledger), front **React** +
**Capacitor** (Android/iOS).

## Mejoras prioritarias (heredadas de auditorias de KiramoPay, aplicar desde el inicio)

1. **Seguridad de sesion**: cookie httpOnly / patron BFF; sacar PII de `localStorage`;
   biometria web honesta; passkeys / WebAuthn.
2. **OTP de telefono REAL** + UI de recuperacion de cuenta (no un `setTimeout` que no
   valida).
3. **Topologia de base de datos**: cuidar PgBouncer + GUC de cifrado de PII para evitar
   fuga de datos entre usuarios si se enruta por pooler transaccional.
4. **UX movil**: router con historial + manejo del boton "atras" de Capacitor (que el
   back de Android no cierre la app).
5. **Sistema de color con tokens** (naranja funcional real), no colores crudos sueltos.
6. **Dinero**: abstraccion de on-ramp como espejo del payout; sub-cuentas / vaults;
   links de cobro.
7. **Onboarding progresivo**: KYC diferido, nivel 0 disponible sin friccion.
8. **Integridad contable**: triggers anti-mutacion en asientos + test de invariante
   "todo se mueve por el ledger".

## Que se puede construir YA (no bloqueado por licencia)

Billetera, ledger doble entrada, pagos QR, ahorros, marketplace con gasto real por el
ledger, contabilidad multimoneda, feed de precios cripto real, andamiaje de rails.

## Bloqueado por licencia / partner (no es codigo)

SINPE real, on-ramp / payout real, VASP cripto, emisor de tarjetas.

## Verificacion estandar a heredar

- Backend: build + vet + `golangci-lint` en 0 + `gosec` en 0 + tests.
- Frontend: typecheck + eslint en 0 + test + build.

## Primeros pasos sugeridos para la sesion de desarrollo

1. Cerrar nombre + dominio (`nombre-y-dominio.md`) y registrar el dominio.
2. Decidir base tecnica (reuse de KiramoPay vs nueva) y stack.
3. `git init` (el `.gitignore` ya esta listo) e inicializar el esqueleto del proyecto.
4. Traer el ledger + auth con las mejoras de seguridad de la lista de arriba.
