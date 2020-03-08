# Builder Image
FROM golang:alpine3.11 AS compiler

RUN apk add --no-cache git

COPY . /go/src/github.com/sobitada/thor
RUN cd /go/src/github.com/sobitada/thor && go mod vendor && GOOS=linux go build -o thor && mv thor /thor

# Main Image
FROM alpine:3.11

ARG THOR_VERSION

LABEL maintainer="Kevin Haller <keivn.haller@outofbits.com>"
LABEL version="${THOR_VERSION}"
LABEL description="Monitoring tool for a swarm of Jormungandr nodes."

COPY --from=compiler /thor /usr/local/bin/
RUN chmod a+x /usr/local/bin/thor

ENTRYPOINT ["thor"]