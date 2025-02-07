.PHONY: deps generate

deps:
	npm install -g @openapitools/openapi-generator-cli 

generate:
	openapi-generator-cli generate -i openapi.json -g go -o ./client
