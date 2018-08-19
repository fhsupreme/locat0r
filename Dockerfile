FROM golang:alpine AS build-env
RUN apk update && apk upgrade && \
    apk add --no-cache bash git openssl
RUN go get \
    github.com/influxdata/influxdb/client/v2 \
    github.com/labstack/echo \
    github.com/twpayne/go-gpx
ADD . /src
RUN cd /src && openssl req -x509 -newkey rsa:4096 -nodes -keyout key.pem -out cert.pem -days 5000 \
    -subj "/C=DE/ST=Bayern/L=Karlshuld/O=private/OU=private/CN=rohrpostix.net" \
    && cat key.pem >> cert.pem  
RUN cd /src && go build -o locat0r

FROM alpine
WORKDIR /app
COPY static /app/
COPY --from=build-env /src/key.pem /app/
COPY --from=build-env /src/cert.pem /app/
COPY --from=build-env /src/locat0r /app/
ENTRYPOINT ./locat0r
