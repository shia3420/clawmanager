FROM --platform=$BUILDPLATFORM node:20-alpine AS frontend-builder

WORKDIR /app/frontend

COPY frontend/package*.json ./
RUN npm ci

# Optional Vite base path so the app can be served under a sub-path (e.g.
# https://api.chase-science.cn/agent/). Defaults to root ("/").
ARG VITE_BASE_URL=/agent/
COPY frontend/ ./
RUN npm run build -- --base=${VITE_BASE_URL}

FROM --platform=$BUILDPLATFORM golang:1.26.1-alpine AS backend-builder

WORKDIR /app/backend

ARG TARGETOS
ARG TARGETARCH

ARG GOPROXY
ARG GOSUMDB
ARG GOFLAGS
ENV GOFLAGS=${GOFLAGS}

# Optional Alpine mirror override (default preserves upstream behaviour). Speeds
# up apk steps on networks where dl-cdn.alpinelinux.org is slow/unreachable.
ARG ALPINE_REPO_MIRROR=https://dl-cdn.alpinelinux.org/alpine
RUN if [ "$ALPINE_REPO_MIRROR" != "https://dl-cdn.alpinelinux.org/alpine" ]; then sed -i "s#https://dl-cdn.alpinelinux.org/alpine#$ALPINE_REPO_MIRROR#g" /etc/apk/repositories; fi && apk add --no-cache git

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o /out/clawreef-server ./cmd/server

FROM nginx:1.27-alpine

# nginx-module-njs provides ngx_http_js_module.so, loaded by nginx.conf to run
# the desktop access-token verification (deployments/nginx/njs/desktop_auth.js).
ARG ALPINE_REPO_MIRROR=https://dl-cdn.alpinelinux.org/alpine
RUN if [ "$ALPINE_REPO_MIRROR" != "https://dl-cdn.alpinelinux.org/alpine" ]; then sed -i "s#https://dl-cdn.alpinelinux.org/alpine#$ALPINE_REPO_MIRROR#g" /etc/apk/repositories; fi && apk add --no-cache dumb-init openssl nginx-module-njs

WORKDIR /app

COPY --from=backend-builder /out/clawreef-server /usr/local/bin/clawreef-server
COPY --from=frontend-builder /app/frontend/dist /usr/share/nginx/html
COPY deployments/nginx/nginx.conf /etc/nginx/nginx.conf
COPY deployments/nginx/njs/desktop_auth.js /etc/nginx/njs/desktop_auth.js
COPY deployments/container/start.sh /app/start.sh

RUN chmod +x /app/start.sh \
    && mkdir -p /etc/nginx/tls /var/log/clawreef

EXPOSE 8443

ENTRYPOINT ["dumb-init", "--"]
CMD ["/app/start.sh"]
