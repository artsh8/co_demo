FROM golang:1.23.1-alpine AS builder

WORKDIR /build
COPY ./endpoints .
RUN go mod download
RUN CGO_ENABLED=0 go build -o ./co-api

FROM alpine AS final
WORKDIR /endpoints
COPY --from=builder /build/co-api ./co-api
EXPOSE 8081
CMD ["./co-api"]