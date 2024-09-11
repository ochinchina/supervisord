module github.com/cyralinc/supervisord/process

go 1.22

require (
	github.com/ochinchina/filechangemonitor v0.3.1
	github.com/ochinchina/supervisord/config v0.0.0-20220721095143-c2527852d28f
	github.com/ochinchina/supervisord/events v0.0.0-20220721095143-c2527852d28f
	github.com/ochinchina/supervisord/logger v0.0.0-20220721095143-c2527852d28f
	github.com/ochinchina/supervisord/signals v0.0.0-20220721095143-c2527852d28f
	github.com/prometheus/client_golang v1.11.1
	github.com/robfig/cron/v3 v3.0.1
	github.com/sirupsen/logrus v1.8.1
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.1 // indirect
	github.com/golang/protobuf v1.4.3 // indirect
	github.com/gorilla/rpc v1.2.0 // indirect
	github.com/hashicorp/go-envparse v0.1.0 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/ochinchina/go-ini v1.0.1 // indirect
	github.com/ochinchina/gorilla-xmlrpc v0.0.0-20171012055324-ecf2fe693a2c // indirect
	github.com/ochinchina/supervisord/faults v0.0.0-20220721095143-c2527852d28f // indirect
	github.com/ochinchina/supervisord/util v0.0.0-20220721095143-c2527852d28f // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.26.0 // indirect
	github.com/prometheus/procfs v0.6.0 // indirect
	github.com/rogpeppe/go-charset v0.0.0-20190617161244-0dc95cdf6f31 // indirect
	golang.org/x/sys v0.0.0-20210603081109-ebe580a85c40 // indirect
	google.golang.org/protobuf v1.26.0-rc.1 // indirect
)

replace (
	github.com/ochinchina/supervisord/config => ../config
	github.com/ochinchina/supervisord/events => ../events
	github.com/ochinchina/supervisord/faults => ../faults
	github.com/ochinchina/supervisord/logger => ../logger
	github.com/ochinchina/supervisord/signals => ../signals
	github.com/ochinchina/supervisord/util => ../util
)
