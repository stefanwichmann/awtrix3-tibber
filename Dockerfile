FROM alpine:latest

WORKDIR /opt/awtrix3-tibber

RUN apk --no-cache add ca-certificates tzdata && update-ca-certificates
COPY awtrix3-tibber /opt/awtrix3-tibber/

ENTRYPOINT /opt/awtrix3-tibber/awtrix3-tibber