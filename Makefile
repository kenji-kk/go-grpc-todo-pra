.PHONY: client

client:
	docker-compose run --rm client bash -c "go run client/client.go"
