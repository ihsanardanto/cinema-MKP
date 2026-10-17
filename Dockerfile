FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /api .
FROM alpine:3.21
RUN adduser -D app
USER app
COPY --from=build /api /api
EXPOSE 8080
ENTRYPOINT ["/api"]
