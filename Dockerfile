FROM golang:alpine3.10

RUN apk add --no-cache git

COPY . /go/src/github.com/sobitada/thor
RUN cd /go/src/github.com/sobitada/thor && go get . && go build
RUN chmod a+x /go/bin/thor

ENTRYPOINT ["/go/bin/thor"]