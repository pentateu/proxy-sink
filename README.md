# proxy-sink
Reverse proxy with data sink on a SQLite database.

Useful to test http endpoints.


![](img/sequence.png)



Configuration:
config file (default is $HOME/.proxy-sink.yaml)
```
log: debug

services:
  proxy-sink:
    settings:
      port: 8387
      correlation-header: tp-api-correlation-id
      mode: sink
      mocks: ./mocks

  sink:
    settings:
      uri: file:memory:?mode=memory
  
  api:
    settings:
      port: "3120"

```

Mocks:
The proxy sink will return a standard response with status 200 if you don't provide a mock file.
If you provide a mock file, the status code and content from the mock file will be returned to the service calling the proxy sink.

```
# Example Mock File
{
    "statusCode": 200,
    "content": "mock content!"
}

```


Build:
```
docker build -t proxy-sink .
```

Linux build from windows (or any) machine:
```
docker run --rm -v "$PWD":/usr/src/proxy-sink -w /usr/src/proxy-sink golang:1.12 go build -v

```

Run :
```
docker run -it --rm proxy-sink proxy-sink start
```


Sequence Diagrams produced at https://www.websequencediagrams.com/ - code: 
```
title Proxy-Sink

Source Application->Source Application: Start!

Proxy-Sink->+Proxy-Sink: proxy starts...
Proxy-Sink->Proxy-Sink: load config
Proxy-Sink->-Proxy-Sink: proxy started!

Source Application->+Proxy-Sink: Request
Proxy-Sink->Redis: Save request
Redis-->Proxy-Sink : ok
Proxy-Sink->+Proxy-Sink: if sink
Proxy-Sink-->-Source Application: test mock result
Proxy-Sink->+Proxy-Sink: if forward
Proxy-Sink->+Target Application: Request

Target Application-> Target Application: Proces
note right of Target Application
    Process request
end note

Target Application-->-Proxy-Sink: Result

Proxy-Sink-->-Source Application: Result


Quality Engineer->Redis: Query test results
Redis-->Quality Engineer:  results


 
Test Automation->Redis: Query test results
Redis-->Test Automation:  results

```