all:
	@go install
run: all
	@go run server.go
