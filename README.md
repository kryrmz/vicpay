# VicPay (nombre de trabajo)

Carpeta de arranque de la nueva super-app financiera de Victor Lobo. Todo lo de esta
carpeta es punto de partida: el **desarrollo empieza en una sesion aparte**.

> **"VicPay" es un nombre de trabajo TEMPORAL**, no la marca definitiva. Como marca
> publica ya esta tomado por terceros (VicPay = Vic.ai / Chainnova y otros), asi que
> sirve solo como nombre interno del proyecto hasta elegir el definitivo. El nombre y
> el dominio finales estan en decision: ver [`docs/nombre-y-dominio.md`](docs/nombre-y-dominio.md).

## Que hay aqui

| Archivo | Contenido |
|---|---|
| [`docs/nombre-y-dominio.md`](docs/nombre-y-dominio.md) | Investigacion completa de nombre + dominio: shortlist verificada con dominio libre, metodo de verificacion, y la decision pendiente. |
| [`docs/brief-producto.md`](docs/brief-producto.md) | Que es el producto, alcance, posicionamiento, marca y limite regulatorio. |
| [`docs/roadmap-arranque.md`](docs/roadmap-arranque.md) | Por donde empezar: que se puede construir ya, que esta bloqueado, y las mejoras prioritarias heredadas de KiramoPay. |
| `.gitignore` | Listo para cuando se haga `git init` (ignora `.claude/`, secretos, build, etc.). |

## Estado actual

Fase de arranque. Aun NO se ha escrito codigo. Antes de la primera linea hay que
cerrar estas decisiones:

1. **Nombre y dominio finales** (candidatos verificados libres en `docs/nombre-y-dominio.md`).
2. **Base tecnica**: partir del back/ledger real de KiramoPay (activo probado) vs base
   nueva desde cero. Recomendacion en `docs/roadmap-arranque.md`.
3. **Stack** definitivo (propuesta heredada: Go + Postgres para el ledger, front React
   + Capacitor).

## Referencia

Proyecto previo del mismo dueno: **KiramoPay** (en `../kiramopay`). VicPay reusa sus
aprendizajes y su activo mas valioso: un ledger de doble entrada real donde el dinero
se mueve de verdad. Ver el roadmap para que heredar y que mejorar.
