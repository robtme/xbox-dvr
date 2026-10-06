FROM golang:1.25-alpine3.22 AS build

RUN apk add git make \
 && apk cache clean \
 && go install golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment@latest

COPY . /go/src/github.com/robtme/xbox-dvr/

WORKDIR /go/src/github.com/robtme/xbox-dvr/

RUN make

FROM scratch

COPY --from=build /go/src/github.com/robtme/xbox-dvr/bin/xdvr /xdvr
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

ENTRYPOINT [ "/xdvr" ]