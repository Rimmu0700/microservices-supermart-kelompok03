# ---------- STAGE 1: Node builder ----------
FROM node:22-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci --prefer-offline --no-audit
COPY . .
RUN npm run build -- --configuration production

# ---------- STAGE 2: Nginx ----------
FROM nginx:1.25-alpine
RUN rm -rf /etc/nginx/conf.d/*
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/dist/frontend/browser /usr/share/nginx/html

# Hardening: jalankan sebagai non-root
RUN touch /var/run/nginx.pid && \
    chown -R nginx:nginx /var/run/nginx.pid /usr/share/nginx/html /var/cache/nginx /var/log/nginx
USER nginx
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]