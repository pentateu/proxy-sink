FROM golang:1.12

WORKDIR /go/src/proxy-sink
COPY . .

RUN go get -d -v ./...
RUN go install -v ./...

CMD ["proxy-sink"]