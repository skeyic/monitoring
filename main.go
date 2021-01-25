package main

import (
	"flag"
	"github.com/golang/glog"
	"github.com/skeyic/monitoring/app/service"
)

func main() {
	flag.Parse()

	var (
		err error
	)

	service.TheFutuCollector.AddFilter(service.NewRateFutuMsgFilter())
	err = service.TheFutuCollector.Start()
	if err != nil {
		glog.V(4).Infof("Start Futu collector failed, ERR: %v\n", err)
		return
	}

	<-make(chan struct{}, 1)
}
