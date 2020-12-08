# --------------------------------------------------------------
# The base image for building the binary

FROM golang:1.15.6-alpine3.12 AS build

WORKDIR /metrics-viewer
COPY ./ ./
RUN apk --no-cache add make git gcc libc-dev curl && go build .

# --------------------------------------------------------------
# Build the final Docker image

FROM alpine:3.12.1

COPY --from=build /metrics-viewer/metrics-viewer /bin/
RUN apk add --update ca-certificates \
  && apk add --update -t deps curl vim \
  && apk del --purge deps \
  && rm /var/cache/apk/*

ENTRYPOINT [ "/bin/metrics-viewer" ]
