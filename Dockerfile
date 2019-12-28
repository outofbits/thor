FROM golang:alpine3.10

RUN apk add --no-cache git

ARG THOR_VERSION

LABEL maintainer="Kevin Haller <keivn.haller@outofbits.com>"
LABEL version="${THOR_VERSION}"
LABEL description="Monitoring tool for a swarm of Jormungandr nodes."

COPY . /go/src/github.com/sobitada/thor
RUN cd /go/src/github.com/sobitada/thor && go get . && go build
RUN chmod a+x /go/bin/thor

ENTRYPOINT ["/go/bin/thor"]