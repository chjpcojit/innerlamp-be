# One-off image for the innerlamp-seed job: every cmd/seed* binary plus the
# assets they read, and a runner that applies them in dependency order.
FROM golang:1.26-alpine AS build

WORKDIR /src
ENV CGO_ENABLED=0

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN for name in seedadminunits seeddeities seedtemples seedgallery seedcalendarevents seedtempleschedules seedtempleevents seedarticles seeddharmatalks seedsutras; do \
      go build -trimpath -ldflags="-s -w" -o /out/$name ./services/appService/cmd/$name; \
    done

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=build /out /app/bin
COPY --from=build /src/assets /app/assets
COPY seeds/run /app/run
RUN chmod 0755 /app/run

ENTRYPOINT ["/app/run"]
