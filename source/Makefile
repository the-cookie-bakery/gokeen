.PHONY: generate test vet build run check assetcheck levelcheck

generate:
	go generate ./assets ./shaders

test:
	go test ./...

vet:
	go vet ./...

build:
	go build -o build/gokeen ./cmd/gokeen
	go build -o build/gokeen-assetconv ./cmd/gokeen-assetconv
	go build -o build/gokeen-assetcheck ./cmd/gokeen-assetcheck
	go build -o build/gokeen-levelcheck ./cmd/gokeen-levelcheck

run:
	go run ./cmd/gokeen

assetcheck:
	go run ./cmd/gokeen-assetcheck

levelcheck:
	go run ./cmd/gokeen-levelcheck

check: generate test vet build assetcheck levelcheck
