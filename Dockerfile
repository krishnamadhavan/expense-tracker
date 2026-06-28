# Minimal API image (PR05). SPA embed is PR13.
FROM golang:1.24-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/expense-tracker ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/expense-tracker /app/expense-tracker
COPY api/openapi/openapi.yaml /app/api/openapi/openapi.yaml
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/app/expense-tracker"]
