FROM golang AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /crashloopbackon
FROM alpine AS alpine
RUN echo user:x:1000:1000:unprivileged:/:/bin/false > /etc/passwd
FROM scratch
COPY --from=alpine /etc/ssl /etc/ssl
COPY --from=alpine /etc/passwd /etc/passwd
COPY --from=build /crashloopbackon /crashloopbackon
USER user
ENTRYPOINT ["/crashloopbackon"]
