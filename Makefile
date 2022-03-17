SRC = uad.go # page.go
BIN = uad

all: build

build:
	CGO_ENABLED=0 GO111MODULE=on go build -o $(BIN) $(SRC)

build-linux:
	CGO_ENABLED=0 GO111MODULE=on GOOS=linux GOARCH=arm go build -v -o $(BIN)-linux-arm $(SRC)
	CGO_ENABLED=0 GO111MODULE=on GOOS=linux GOARCH=arm64 go build -v -o $(BIN)-linux-arm64 $(SRC)
	CGO_ENABLED=0 GO111MODULE=on GOOS=linux GOARCH=386 go build -v -o $(BIN)-linux-386 $(SRC)
	CGO_ENABLED=0 GO111MODULE=on GOOS=linux GOARCH=amd64 go build -v -o $(BIN)-linux-amd64 $(SRC)

build-windows:
	CGO_ENABLED=0 GO111MODULE=on GOOS=windows GOARCH=arm go build -v -o $(BIN)-windows-arm $(SRC)
	CGO_ENABLED=0 GO111MODULE=on GOOS=windows GOARCH=386 go build -v -o $(BIN)-windows-386 $(SRC)
	CGO_ENABLED=0 GO111MODULE=on GOOS=windows GOARCH=amd64 go build -v -o $(BIN)-windows-amd64 $(SRC)

build-darwin:
	CGO_ENABLED=0 GO111MODULE=on GOOS=darwin GOARCH=amd64 go build -v -o $(BIN)-darwin-amd64 $(SRC)

build-all: build-linux build-windows build-darwin

clean:
	rm -f $(BIN) $(BIN)-*
