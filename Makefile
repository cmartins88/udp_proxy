APP_NAME=udp_proxy
SRC=.
BIN=$(APP_NAME).exe
MANIFEST=app.manifest
RSRC=rsrc

all: build_gui

rsrc:
	@echo ">> Generating rsrc.syso from manifest..."
	$(RSRC) -manifest $(MANIFEST) -o rsrc.syso

build: rsrc
	@echo ">> Initializing Go module..."
	go mod init $(APP_NAME) || true
	@echo ">> Fetching dependencies..."
	go mod tidy
	@echo ">> Building application (console mode)..."
	go build -o $(BIN) $(SRC)

build_gui: rsrc
	@echo ">> Initializing Go module..."
	go mod init $(APP_NAME) || true
	@echo ">> Fetching dependencies..."
	go mod tidy
	@echo ">> Building application (no console)..."
	go build -ldflags="-H=windowsgui" -o $(BIN) $(SRC)

clean:
	@echo ">> Cleaning up..."
	del /q $(BIN) rsrc.syso go.mod go.sum 2> nul || exit 0

run:
	@echo ">> Running app..."
	$(BIN)
