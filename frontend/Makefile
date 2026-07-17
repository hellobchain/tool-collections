VERSION=v1.0.0
SERVICE_NAME=weekly-assistant-web
build:
	@npm run build -- --mode production

run:
	@npm run serve

docker-build:
	@docker build -t ${SERVICE_NAME}:${VERSION} -f ./docker/Dockerfile .

clean:
	@rm -rf dist
