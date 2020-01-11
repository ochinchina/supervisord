module supervisord

require (
	github.com/GeertJohan/go.rice v1.0.0
	github.com/gorilla/mux v1.7.3
	github.com/gorilla/rpc v1.2.0
	github.com/jessevdk/go-flags v1.4.0
	github.com/ochinchina/filechangemonitor v0.3.1
	github.com/ochinchina/go-daemon v0.1.5
	github.com/ochinchina/go-ini v1.0.1
	github.com/ochinchina/go-reaper v0.0.0-20181016012355-6b11389e79fc
	github.com/ochinchina/gorilla-xmlrpc v0.0.0-20171012055324-ecf2fe693a2c
	github.com/ochinchina/supervisord v0.6.4
	github.com/robfig/cron/v3 v3.0.1
	github.com/rogpeppe/go-charset v0.0.0-20190617161244-0dc95cdf6f31 // indirect
	github.com/sirupsen/logrus v1.4.2
	golang.org/x/net v0.0.0-20180921000356-2f5d2388922f
)

replace github.com/ochinchina/supervisord => ./

go 1.13
