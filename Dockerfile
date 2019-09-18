FROM golang:1.12

WORKDIR /go/bin

COPY ./proxy-sink .
COPY ./moleculer-config.yaml .

CMD ["proxy-sink start"]


# API endpoint
EXPOSE 3100

# Proxy endpoint
EXPOSE 8387
