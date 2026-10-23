module github.com/ochinchina/supervisord

go 1.27.3

require (
	github.com/gorilla/mux v1.8.1
	github.com/gorilla/rpc v1.2.1
	github.com/hashicorp/go-envparse v0.1.0
	github.com/jessevdk/go-flags v1.6.1
	github.com/kardianos/service v1.3.0
	github.com/ochinchina/filechangemonitor v0.3.1
	github.com/ochinchina/go-daemon v0.1.5
	github.com/ochinchina/go-ini v1.0.1
	github.com/ochinchina/go-reaper v0.0.0-20181016012355-6b11389e79fc
	github.com/ochinchina/gorilla-xmlrpc v1.0.4
	github.com/ochinchina/supervisord/xmlrpcclient v0.0.0-00010101000000-000000000000
	github.com/prometheus/client_golang v1.25.0
	github.com/robfig/cron/v3 v3.0.1
	github.com/shirou/gopsutil/v4 v4.26.9
	github.com/sirupsen/logrus v1.10.2
	golang.org/x/sys v0.49.0
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/ebitengine/purego v0.11.1 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/power-devops/perfstat v0.0.0-20260805114148-88456608a4f6 // indirect
	github.com/prometheus/client_model v0.6.3 // indirect
	github.com/prometheus/common v0.72.0 // indirect
	github.com/prometheus/procfs v0.22.0 // indirect
	github.com/rogpeppe/go-charset v0.0.0-20190617161244-0dc95cdf6f31 // indirect
	github.com/tklauser/go-sysconf v0.4.0 // indirect
	github.com/tklauser/numcpus v0.12.0 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

replace github.com/ochinchina/supervisord/xmlrpcclient => ./xmlrpcclient
