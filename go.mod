module supervisord

require (
	github.com/gorilla/mux v1.7.2
	github.com/gorilla/rpc v1.2.0
	github.com/jessevdk/go-flags v1.4.0
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0 // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/ochinchina/go-ini v1.0.1
	github.com/ochinchina/go-reaper v0.0.0-20181016012355-6b11389e79fc
	github.com/ochinchina/gorilla-xmlrpc v0.0.0-20171012055324-ecf2fe693a2c
	github.com/ochinchina/supervisord v0.6.3
	github.com/rogpeppe/go-charset v0.0.0-20180617210344-2471d30d28b4 // indirect
	github.com/sevlyar/go-daemon v0.1.5
	github.com/sirupsen/logrus v1.4.2
	github.com/stretchr/testify v1.3.0 // indirect
	golang.org/x/net v0.0.0-20190522155817-f3200d17e092
	golang.org/x/sys v0.0.0-20190529164535-6a60838ec259 // indirect
)

replace (
	golang.org/x/crypto v0.0.0-20190513172903-22d7a77e9e5f => github.com/golang/crypto v0.0.0-20190513172903-22d7a77e9e5f
	golang.org/x/net v0.0.0-20190522155817-f3200d17e092 => github.com/golang/net v0.0.0-20190522155817-f3200d17e092
	golang.org/x/sys v0.0.0-20190529164535-6a60838ec259 => github.com/golang/sys v0.0.0-20190529164535-6a60838ec259
	golang.org/x/text v0.3.2 => github.com/golang/text v0.3.2
	golang.org/x/tools v0.0.0-20190530001615-b97706b7f64d => github.com/golang/tools v0.0.0-20190530001615-b97706b7f64d
)
