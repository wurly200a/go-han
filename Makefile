.PHONY: deploy

DEPLOY_HOST ?= $(shell git config deploy.host)

deploy:
	ssh ubuntu@$(DEPLOY_HOST) "bash ~/go-han/deploy.sh"
