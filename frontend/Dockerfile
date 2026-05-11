FROM node:24 AS builder

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

COPY . .
RUN npm run build

FROM nginx:1.29-alpine

COPY ops/nginx/default.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/dist/frontend/browser /usr/share/nginx/html

EXPOSE 8080

CMD ["nginx", "-g", "daemon off;"]
