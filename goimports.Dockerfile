FROM golang:1.16.5-alpine3.13

RUN go get golang.org/x/tools/cmd/goimports
