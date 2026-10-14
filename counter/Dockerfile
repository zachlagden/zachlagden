FROM golang:1.26.4-alpine AS build
WORKDIR /src
COPY counter/go.mod counter/go.sum ./
RUN go mod download
COPY counter/*.go ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /profile-views .
RUN mkdir /data

FROM scratch
COPY --from=build /profile-views /profile-views
COPY --from=build --chown=65532:65532 /data /data
USER 65532:65532
ENV DATABASE_PATH=/data/views.sqlite3
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 CMD ["/profile-views", "healthcheck"]
ENTRYPOINT ["/profile-views"]
