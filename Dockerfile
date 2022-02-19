FROM golang:alpine AS builder

# env stuff
ENV GO111MODULE=off \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# create working dir
WORKDIR /build
COPY uad.go .
COPY page.html .

# build
RUN go build -o main .

# move everything to /dist
WORKDIR /dist
RUN cp /build/main .

# build small image
FROM scratch
COPY --from=builder /dist/main /
EXPOSE 6969
ENTRYPOINT ["/main"]

