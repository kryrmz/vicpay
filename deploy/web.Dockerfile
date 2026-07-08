# Build the frontend (pointing the API base at the same-origin /api proxy) and
# serve the static output with Caddy, which also reverse-proxies /api. Build
# context is the repository root (see docker-compose.prod.yml).
FROM node:22-alpine AS build
WORKDIR /app
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
ENV VITE_API_BASE=/api
RUN npm run build

FROM caddy:2-alpine
COPY deploy/Caddyfile /etc/caddy/Caddyfile
COPY --from=build /app/dist /srv
