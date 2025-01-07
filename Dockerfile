FROM golang:alpine AS build

ARG repo=countdown-exporter
ARG cgo=0
RUN mkdir -p /go/src/github.com/tmegow/${repo}
WORKDIR /go/src/github.com/tmegow/${repo}/
COPY . /go/src/github.com/tmegow/${repo}
RUN go get \
&& ( \
    [ "${cgo}" = "0" ] && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo --ldflags '-extldflags "-static"' -v -o "/binary" \
    || CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -a --ldflags -w -v -o "/binary" \
)

FROM scratch

ENV COUNTDOWN_EXPTR_DEADLINES_FILE deadlines.yaml
ENV COUNTDOWN_EXPTR_DEADLINES_FILE_TYPE yaml
ENV COUNTDOWN_EXPTR_HTTP_PORT 9208
ENV COUNTDOWN_EXPTR_CHECK_INTERVAL_SECS 60

COPY --from=build /binary /countdown-exporter
COPY deadlines.yaml /deadlines.yaml

ENTRYPOINT ["/countdown-exporter"]
