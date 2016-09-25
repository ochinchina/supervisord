package main

func main() {
        s := supervisor.NewSupervisor( os.Args[1] )
        s.Reload()
        supervisor.NewXmlRPC().Start(":9000", s )
}

