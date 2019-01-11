package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net"
	"runtime"
	"time"

	kb "github.com/lsds/KungFu/srcs/go/kungfubase"
	"github.com/lsds/KungFu/srcs/go/plan"
	"github.com/lsds/KungFu/srcs/go/runner"
	sch "github.com/lsds/KungFu/srcs/go/scheduler"
	"github.com/lsds/KungFu/srcs/go/utils"
)

var (
	np         = flag.Int("np", runtime.NumCPU(), "number of tasks")
	hostList   = flag.String("H", plan.DefaultHostSpec().String(), "comma separated list of <hostname>:<nslots>[,<public addr>]")
	selfHost   = flag.String("self", "", "")
	timeout    = flag.Duration("timeout", 10*time.Second, "timeout")
	verboseLog = flag.Bool("v", true, "show task log")
	niName     = flag.String("ni", "", "network interface name, for infer self host")
	algo       = flag.String("algo", "", "algorithm")
)

func init() {
	log.SetPrefix("[kungfu-prun] ")
	flag.Parse()
	utils.LogArgs()
	utils.LogKungfuEnv()
}

func main() {
	selfIP := func() string {
		switch {
		case len(*selfHost) > 0:
			return *selfHost
		case len(*niName) > 0:
			return inferIP(*niName)
		}
		return "127.0.0.1"
	}()
	log.Printf("Using selfHost=%s", selfIP)
	restArgs := flag.Args()
	if len(restArgs) < 1 {
		utils.ExitErr(errors.New("missing program name"))
	}
	prog := restArgs[0]
	args := restArgs[1:]
	log.Printf("will parallel run multiple %s with %q", prog, args)

	jc := sch.JobConfig{
		TaskCount: *np,
		HostList:  *hostList,
		Prog:      prog,
		Args:      args,
	}

	ps, err := jc.CreateProcs(kb.ParseAlgo(*algo))
	if err != nil {
		utils.ExitErr(err)
	}
	myPs := sch.ForHost(selfIP, ps)
	if len(myPs) <= 0 {
		log.Print("No task to run on this node")
		return
	}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, *timeout)
	defer cancel()
	d, err := utils.Measure(func() error { return runner.LocalRunAll(ctx, myPs, *verboseLog) })
	log.Printf("all %d/%d local tasks finished, took %s", len(myPs), len(ps), d)
	if err != nil && err != context.DeadlineExceeded {
		utils.ExitErr(err)
	}
}

func inferIP(niName string) string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "127.0.0.1"
	}
	for _, i := range ifaces {
		if i.Name != niName {
			continue
		}
		addrs, err := i.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip.To4() != nil {
				return ip.String()
			}
		}
	}
	return "127.0.0.1"
}
