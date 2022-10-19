frontend-dev:
	cd frontend/ && npm run dev
	cp frontend/stubs/dist.go.stub frontend/dist/dist.go

frontend-build:
	cd frontend/ && npm run build
	cp frontend/stubs/dist.go.stub frontend/dist/dist.go

backend-dev:
	go run main.go

backend-build:
	test -d bin/ || mkdir bin/
	go build -o bin/fakeapi main.go
	ls -lh bin/

backend-update-libs:
	go get -u
	go mod tidy

build: frontend-build backend-build
