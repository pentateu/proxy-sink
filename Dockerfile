FROM golang:1.12

ADD . /go/src/github.com/pentateu/proxy-sink

RUN go install github.com/pentateu/proxy-sink

ENTRYPOINT /go/bin/proxy-sink

# API endpoint
EXPOSE 3100

# Proxy endpoint
EXPOSE 8387

# RUN go get -d -v ./...
# RUN go install -v ./...

# CMD ["proxy-sink"]