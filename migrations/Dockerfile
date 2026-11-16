# Migration image for the innerlamp-migrate Cloud Run job.
# Build from the repository root so the build context includes migrations/:
#   docker build -f migrations/Dockerfile -t <repo>/migrate:<tag> .
FROM postgres:16-alpine

COPY migrations/*.sql /migrations/
COPY migrations/apply /usr/local/bin/apply
RUN chmod 0755 /usr/local/bin/apply

ENTRYPOINT ["/usr/local/bin/apply"]
