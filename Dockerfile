FROM golang:latest AS compiling_stage
RUN mkdir -p /go/src/pipeservice
WORKDIR /go/src/pipeservice
ADD main.go .
ADD go.mod .
RUN go install .

FROM alpine:latest
LABEL version="1.0"
LABEL maintainer="cyber<cyber@test.ru>"
WORKDIR /root/
COPY --from=compiling_stage /go/bin/pipeservice .
ENTRYPOINT ./pipeservice