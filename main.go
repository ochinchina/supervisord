package main

import(
	"os"
)

func main() {
        s := NewSupervisor( os.Args[1] )
        s.Reload()
        NewXmlRPC().Start(":9000", s )
}

