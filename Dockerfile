FROM golang:1.16-alpine AS builder

# env stuff
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# create working dir
WORKDIR /build
COPY uad.go .
COPY go.mod .
COPY page.html .

# build
RUN go mod tidy
RUN go build -o uad uad.go

# move everything to /dist
WORKDIR /dist
RUN cp /build/uad .

# build small image
FROM scratch
COPY --from=builder /dist/uad /
EXPOSE 6969
CMD ["/uad", "/files"]

