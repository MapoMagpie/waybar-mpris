FROM golang:latest

COPY . /opt/build

RUN cd /opt/build && go build *.go

