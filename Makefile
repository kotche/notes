.PHONY: run stop logs rebuild

WRITER_BINARY=writer
NOTIFIER_BINARY=notifier

build-writer:
	go build -o $(WRITER_BINARY) cmd/writer/main.go

build-notifier:
	go build -o $(NOTIFIER_BINARY) cmd/notifier/main.go

run-writer: build-writer
	./$(WRITER_BINARY)

run-notifier: build-notifier
	./$(NOTIFIER_BINARY)

docker:
	docker-compose up --build -d

#Запустить сразу все
run: docker build-writer build-notifier
	./$(WRITER_BINARY) &
	./$(NOTIFIER_BINARY)

stop:
	-@pkill $(WRITER_BINARY) || true
	-@pkill $(NOTIFIER_BINARY) || true
	rm -f $(WRITER_BINARY) $(NOTIFIER_BINARY)
	docker-compose down

logs:
	docker-compose logs -f

rebuild:
	docker-compose down
	docker-compose up --build -d