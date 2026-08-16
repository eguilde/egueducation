FROM node:24@sha256:934240a162082fd8b8a2f90cd5114446443f1eba1c5378f6687167ca405e6584 AS builder

WORKDIR /app

ARG SOURCE_REVISION=development

COPY package.json package-lock.json ./
RUN npm ci

COPY . .
RUN --mount=type=secret,id=primeui_license,required=true \
    VITE_PRIMEUI_LICENSE="$(cat /run/secrets/primeui_license)" npm run build
RUN printf '{"revision":"%s"}\n' "$SOURCE_REVISION" > dist/revision.json

FROM nginx:1.29-alpine@sha256:5616878291a2eed594aee8db4dade5878cf7edcb475e59193904b198d9b830de

COPY ops/nginx/default.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 8080

CMD ["nginx", "-g", "daemon off;"]
