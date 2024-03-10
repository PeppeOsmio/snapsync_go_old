FROM golang:1.22 as build

COPY . /snapsync
WORKDIR /snapsync

RUN go build -o snapsync main.go

FROM alpine:3.19.1

COPY --from=build /snapsync/snapsync /snapsync
COPY --from=build /snapsync/entrypoint.sh /entrypoint.sh

WORKDIR /snapsync

ENTRYPOINT [ "./entrypoint.sh" ] 