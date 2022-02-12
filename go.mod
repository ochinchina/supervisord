module github.com/ochinchina/supervisord

go 1.16

require (
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/rpc v1.2.0
	github.com/jessevdk/go-flags v1.5.0
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0 // indirect
	github.com/kardianos/service v1.2.0
	github.com/ochinchina/go-daemon v0.1.5
	github.com/ochinchina/go-ini v1.0.1
	github.com/ochinchina/go-reaper v0.0.0-20181016012355-6b11389e79fc
	github.com/ochinchina/gorilla-xmlrpc v0.0.0-20171012055324-ecf2fe693a2c
	github.com/ochinchina/supervisord/config v0.0.0-20210503132557-74b0760cc12e
	github.com/ochinchina/supervisord/events v0.0.0-20210503132557-74b0760cc12e
	github.com/ochinchina/supervisord/faults v0.0.0-20210503132557-74b0760cc12e
	github.com/ochinchina/supervisord/logger v0.0.0-20210503132557-74b0760cc12e
	github.com/ochinchina/supervisord/process v0.0.0-20210503132557-74b0760cc12e
	github.com/ochinchina/supervisord/signals v0.0.0-20210503132557-74b0760cc12e
	github.com/ochinchina/supervisord/types v0.0.0-20210503132557-74b0760cc12e
	github.com/ochinchina/supervisord/util v0.0.0-20210503132557-74b0760cc12e
	github.com/ochinchina/supervisord/xmlrpcclient v0.0.0-20210503132557-74b0760cc12e
	github.com/prometheus/client_golang v1.10.0
	github.com/prometheus/common v0.23.0 // indirect
	github.com/sirupsen/logrus v1.8.1
	golang.org/x/sys v0.0.0-20210503080704-8803ae5d1324 // indirect
)

replace (
	github.com/ochinchina/supervisord/config => ./config
	github.com/ochinchina/supervisord/events => ./events
	github.com/ochinchina/supervisord/faults => ./faults
	github.com/ochinchina/supervisord/logger => ./logger
	github.com/ochinchina/supervisord/process => ./process
	github.com/ochinchina/supervisord/signals => ./signals
	github.com/ochinchina/supervisord/types => ./types
	github.com/ochinchina/supervisord/util => ./util
	github.com/ochinchina/supervisord/xmlrpcclient => ./xmlrpcclient
)
