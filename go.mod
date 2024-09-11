module github.com/cyralinc/supervisord

go 1.22

require (
	github.com/gorilla/mux v1.8.1
	github.com/gorilla/rpc v1.2.1
	github.com/jessevdk/go-flags v1.6.1
	github.com/kardianos/service v1.2.2
	github.com/ochinchina/go-daemon v0.1.5
	github.com/ochinchina/go-ini v1.0.1
	github.com/ochinchina/go-reaper v0.0.0-20181016012355-6b11389e79fc
	github.com/ochinchina/gorilla-xmlrpc v0.0.0-20171012055324-ecf2fe693a2c
	github.com/ochinchina/supervisord/config v0.0.0-20230902082938-c2cae38b7454
	github.com/ochinchina/supervisord/events v0.0.0-20230902082938-c2cae38b7454
	github.com/ochinchina/supervisord/faults v0.0.0-20230902082938-c2cae38b7454
	github.com/ochinchina/supervisord/logger v0.0.0-20230902082938-c2cae38b7454
	github.com/ochinchina/supervisord/process v0.0.0-20230902082938-c2cae38b7454
	github.com/ochinchina/supervisord/signals v0.0.0-20230902082938-c2cae38b7454
	github.com/ochinchina/supervisord/types v0.0.0-20230902082938-c2cae38b7454
	github.com/ochinchina/supervisord/util v0.0.0-20230902082938-c2cae38b7454
	github.com/ochinchina/supervisord/xmlrpcclient v0.0.0-20230902082938-c2cae38b7454
	github.com/prometheus/client_golang v1.20.3
	github.com/sirupsen/logrus v1.9.3
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/hashicorp/go-envparse v0.1.0 // indirect
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/ochinchina/filechangemonitor v0.3.1 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.59.1 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/robfig/cron/v3 v3.0.1 // indirect
	github.com/rogpeppe/go-charset v0.0.0-20190617161244-0dc95cdf6f31 // indirect
	golang.org/x/sys v0.25.0 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
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
