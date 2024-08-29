FROM golang:1.21-alpine AS build
WORKDIR /tmp/iut
COPY . .
RUN apk add --no-cache make=4.4.1-r2 git=2.45.2-r0 && make iut

FROM alpine:3.17.3
ARG TZ
ENV TZ=$TZ

LABEL org.opencontainers.image.source=https://github.com/eiffel-community/etos-api
LABEL org.opencontainers.image.authors=etos-maintainers@googlegroups.com
LABEL org.opencontainers.image.licenses=Apache-2.0

RUN apk add --no-cache tzdata=2024a-r0
ENTRYPOINT ["/app/iut"]

COPY --from=build /tmp/iut/bin/iut /app/iut
