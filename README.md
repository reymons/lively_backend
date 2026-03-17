# Lively

A backend for a live-streaming platform based on RTMP and WebSocket<br>
This project uses my self-written [RTMP library](https://github.com/reymons/rtmp-go) to showcase its capabilities as an example<br>
For the detailed usage, see `transport/rtmp` directory<br>
You can also check out the [frontend side](https://github.com/reymons/lively_frontend) of the platform<br>
The platform is available at https://reymons.net

## Implementation
The platform consists of several transports:
- HTTP, for REST API
- Main WebSocket, for handling client notifications
- Media transport which is a simple protocol on top of WebSocket, for forwarding stream media to a client
- RTMP transport, for stream ingest
- Media channel, for connecting Media and RTMP transports
<br>
All the transports run within a single process which I'm going to change in the future<br>
A simple in-memory event bus is used for communication between transports<br>
For a persistent storage, PostgreSQL and standard SQL library are used<br><br>

![](https://github.com/reymons/lively_backend/raw/master/doc/arch.png)

## Deploy
- The platform is deployed on AWS using CloudFormation<br>You can set up the infrastructure using `./aws/create-stack.sh` or delete it with `./aws/delete-stack.sh`
- For containerization, Docker Compose is used
