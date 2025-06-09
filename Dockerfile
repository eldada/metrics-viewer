# --------------------------------------------------------------
# The base image for building the binary

FROM golang:1.24.4-alpine3.22 AS build

WORKDIR /metrics-viewer
COPY ./ ./
RUN apk --no-cache add make git gcc libc-dev curl && go build .

# --------------------------------------------------------------
# Build the final Docker image

FROM alpine:3.22.0

COPY --from=build /metrics-viewer/metrics-viewer /bin/
RUN apk add --update ca-certificates \
  && apk add --update -t deps curl vim \
  && apk del --purge deps \
  && rm /var/cache/apk/*

ENTRYPOINT [ "/bin/metrics-viewer" ]
