# appService image for Cloud Run.
#
# Dependencies are downloaded at build time (vendor/ is a local, git-ignored
# cache and is not part of the build context). The runtime is distroless static: the
# binary is CGO-free, needs CA roots for Cloud SQL/GCS/Secret Manager, and
# nothing else.

FROM golang:1.26-alpine AS build

WORKDIR /src
ENV CGO_ENABLED=0

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -trimpath -ldflags="-s -w" -o /out/innerlamp ./services/appService/cmd

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=build /out/innerlamp /app/innerlamp

USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/app/innerlamp"]
