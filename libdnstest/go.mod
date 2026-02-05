module github.com/libdns/arvancloud/libdnstest

go 1.23.0

require (
	github.com/libdns/arvancloud v0.1.1
	github.com/libdns/libdns v1.1.1
)

replace (
	github.com/libdns/arvancloud => ../
	github.com/libdns/libdns => github.com/libdns/libdns v1.2.0-alpha.1.0.20250913035451-da352cac42d0
)
