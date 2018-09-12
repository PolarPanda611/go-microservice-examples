.PHONY: proto data build

proto:
	for d in srv; do \
		for f in $$d/**/proto/*.proto; do \
			protoc --proto_path=.  --micro_out=. --go_out=. $$f; \
			echo compiled: $$f; \
		done \
	done

	for d in api ; do \
		for f in $$d/**/proto/*.proto; do \
			echo ${GOPATH}/src ;\
			path=$$(pwd) ; \
			cd ${GOPATH}/src ;\
			protoc --proto_path=${GOPATH}/src   --micro_out=. --go_out=.  go-microservice-examples/$$f; \
			cd $$path ;\
			echo compiled: $$f; \
		done \
	done

lint:
	./bin/lint.sh

build:
	./bin/build.sh

data:
	go-bindata -o data/bindata.go -pkg data data/*.json

run:
	docker-compose build
	docker-compose up
