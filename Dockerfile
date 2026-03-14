FROM golang:1.20-alpine AS builder
WORKDIR /src
COPY . .
RUN apk add --no-cache git
RUN go build -o /manager main.go

FROM alpine:3.18
RUN apk add --no-cache ca-certificates
COPY --from=builder /manager /manager
ENTRYPOINT ["/manager"]
