FROM golang:1.22.1-alpine3.19 as build

COPY . /snapsync
WORKDIR /snapsync

RUN go build -o snapsync main.go

FROM alpine:3.19

RUN apk add rsync

COPY --from=build /snapsync/snapsync /snapsync/snapsync
COPY --from=build /snapsync/entrypoint.sh /snapsync/entrypoint.sh
COPY --from=build /snapsync/config.yml /snapsync/config.yml

WORKDIR /snapsync

ENTRYPOINT [ "./entrypoint.sh" ] 