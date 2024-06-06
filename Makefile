.PHONY: all build

build:
	go build -o build/znet .

all: build