

.PHONY: test
test: 
	go test ./...

.PHONY: example
example:
	go run _examples/up-and-down/main.go

.PHONY: acme-like
acme-like:
	go run _examples/acme-like/main.go

.PHONY: list-records
list-records:
	go run ./_examples/list-records
	
.PHONY: curl
curl:
	curl --data @testdata/request_zone_records.xml
