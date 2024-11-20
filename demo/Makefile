SHELL := /bin/bash
VERSION := 0.1.1.alpha4

.PHONY: docker-build
docker-build:
	docker build -t jingkaihe/o11y-demo:$(VERSION) .

.PHONY: docker-push
docker-push:
	docker push jingkaihe/o11y-demo:$(VERSION)

.PHONY: deploy
deploy:
	kubectl apply -f deploy.yaml
