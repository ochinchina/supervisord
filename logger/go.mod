module github.com/cyralinc/supervisord/logger

go 1.22

require (
	github.com/ochinchina/supervisord/events v0.0.0-20230902082938-c2cae38b7454
	github.com/ochinchina/supervisord/faults v0.0.0-20230902082938-c2cae38b7454
)

require (
	github.com/gorilla/rpc v1.2.0 // indirect
	github.com/ochinchina/gorilla-xmlrpc v0.0.0-20171012055324-ecf2fe693a2c // indirect
	github.com/rogpeppe/go-charset v0.0.0-20190617161244-0dc95cdf6f31 // indirect
	github.com/sirupsen/logrus v1.8.1 // indirect
	golang.org/x/sys v0.0.0-20191026070338-33540a1f6037 // indirect
)

replace (
	github.com/ochinchina/supervisord/events => ../events
	github.com/ochinchina/supervisord/faults => ../faults
)
