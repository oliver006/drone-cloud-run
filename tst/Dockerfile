FROM golang:1 as builder

WORKDIR /go/src/github.com/oliver006/drone-cloud-run/
COPY . .

RUN cd tst && CGO_ENABLED=0 GOOS=linux go build -v -o app



FROM scratch

COPY --from=builder /go/src/github.com/oliver006/drone-cloud-run/tst/app    /app
COPY --from=builder /go/src/github.com/oliver006/drone-cloud-run/tst/static /static

CMD ["/app"]
