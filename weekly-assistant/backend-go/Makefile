VERSION=v1.0.0
SERVICE_NAME=weekly-assistant
build:
	@cd cmd && go mod tidy && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o ../bin/${SERVICE_NAME}.bin

build-linux:
	@cd cmd && go mod tidy && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o ../bin/${SERVICE_NAME}.bin

docker-build:
	@docker build -t ${SERVICE_NAME}:${VERSION} -f ./docker/Dockerfile .

docker-build-linux:
	@cd cmd && go mod tidy && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o ../bin/${SERVICE_NAME}.bin
	@docker build -t ${SERVICE_NAME}:${VERSION} -f ./docker/Dockerfile-linux .

clean:
	@rm -rf ./bin
