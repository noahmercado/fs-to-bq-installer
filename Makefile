all: installer docker-login image 

.PHONY: installer
installer:
	go mod tidy && go mod download
	env GOOS=linux GOARCH=amd64 go build -o ./bin/fs-to-bq-installer 

.PHONY: docker-login
docker-login:
	@echo "${DOCKER_TOKEN}" | docker login -u ${DOCKER_USER} --password-stdin

.PHONY: image
image:
	docker build -t noahmercado/fs-to-bq-installer:latest .
	docker push noahmercado/fs-to-bq-installer:latest