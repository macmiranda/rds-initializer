
.PHONY: all

all: build zip

build:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o bootstrap
zip:
	zip -j event-handler.zip bootstrap

clean:
	rm -f bootstrap event-handler.zip

.DEFAULT_GOAL := all
