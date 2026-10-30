FROM node:22-alpine AS web
WORKDIR /web
COPY package.json package-lock.json ./
RUN npm ci
COPY index.html tsconfig.json vite.config.ts ./
COPY public ./public
COPY src ./src
RUN npm run build

FROM golang:1.25-alpine AS api
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /lunax ./cmd/lunax

FROM alpine:3.21
RUN adduser -D -H lunax
COPY --from=api /lunax /usr/local/bin/lunax
COPY --from=web /web/dist /srv/web
ENV LUNAX_STATIC_DIR=/srv/web
USER lunax
EXPOSE 8080
ENTRYPOINT ["lunax"]
