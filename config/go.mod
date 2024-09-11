module github.com/cyralinc/supervisord/config

go 1.22

require (
	github.com/hashicorp/go-envparse v0.1.0
	github.com/ochinchina/go-ini v1.0.1
	github.com/ochinchina/supervisord/util v0.0.0-20230902082938-c2cae38b7454
	github.com/sirupsen/logrus v1.8.1
)

require golang.org/x/sys v0.0.0-20191026070338-33540a1f6037 // indirect

replace github.com/ochinchina/supervisord/util => ../util
