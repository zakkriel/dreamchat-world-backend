# The world backend as a deployable image. Used by Railway (hosted dev) and by anyone who wants to
# run the API without a Go toolchain; local development still runs `go run`/`go test` directly
# against docker-compose's Postgres, and this file changes nothing about that.
#
# The DATABASE this image talks to is NOT built here. core/db/Dockerfile stays the schema's own
# image — pinned postgres:16 with pgTAP, the one the determinism check is authoritative against —
# and migrations are applied to whatever Postgres the deployment provides, from outside the app.
# An API container that migrated on boot would race itself the moment there were two replicas.

FROM golang:1.26-alpine AS build
WORKDIR /src
# Manifests first: dependencies change far less often than source, so this layer survives most
# rebuilds and the module download is not repaid on every commit.
COPY core/api/go.mod core/api/go.sum ./
RUN go mod download
COPY core/api/ ./
# Static build: the runtime stage has no libc to link against.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/world-api .

FROM alpine:3.22
# The published JSON Schemas are read at RUNTIME by the seat validators (the structured-output leash
# for the LLM seats), so they are part of the image rather than a build-time artifact. They are
# resolved relative to the working directory, which is why the binary and `schema/` sit together.
WORKDIR /app
COPY --from=build /out/world-api /app/world-api
COPY core/api/schema /app/schema
# TLS roots: the image platform client speaks HTTPS to a hosted platform, and a scratch/alpine image
# with no ca-certificates fails every outbound call with an unhelpful x509 error.
RUN apk add --no-cache ca-certificates \
 && adduser -D -u 10001 dreamchat
USER dreamchat
EXPOSE 8080
CMD ["/app/world-api"]
