# ---- build stage ----
FROM --platform=linux/amd64 golang:1.26-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/api ./cmd

# ---- final stage ----
FROM --platform=linux/amd64 gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=build /out/api /api
COPY --from=build /src/openapi.yaml /openapi.yaml

EXPOSE 8080
ENTRYPOINT ["/api"]