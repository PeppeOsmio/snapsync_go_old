FROM golang:1.22

WORKDIR /snapsync

RUN go build -o snapsync main.go

FROM alpine:3.19.1

CMD [ "./snapsync/snapsync --configs-dir /configs/snapsync" ]