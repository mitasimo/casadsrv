// Release сборка
//go build -ldflags "-w -s"

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"simd/winservice"
	"strconv"
	"strings"
)

const (
	serviceName        = "casadsrv"
	serviceDisplayName = "CAS AD Rest service"
	serviceDescription = "REST сервис взаимодействия с весами CAS AD, подключенными к последовательному (COM) порту"
)

const (
	paramHost           = "host"
	paramSerialName     = "sn"
	paramSerialBaudRate = "sbr"
)

func main() {
	var err error

	host := flag.String(paramHost, ":1133", "listen ip:port")
	serialName := flag.String(paramSerialName, "COM1", "serial port name")
	serialBaudRate := flag.Int(paramSerialBaudRate, 9600, "serial port baud rate")

	// parse command line
	flag.Parse()

	// check that programm runs by user or by service manager

	if winservice.StartsFromCommandLine() {
		args := flag.Args()
		if len(args) == 0 {
			usage()
			os.Exit(1)
		}
		cmd := strings.ToUpper(args[0])
		switch cmd {
		case "INSTALL":
			err = winservice.Install(serviceName, serviceDisplayName, serviceDescription, "-"+paramHost+"="+*host, "-"+paramSerialName+"="+*serialName, "-"+paramSerialBaudRate+"="+strconv.Itoa(*serialBaudRate))
		case "UNINSTALL":
			err = winservice.Remove(serviceName)
		case "START":
			err = winservice.Start(serviceName)
		case "STOP":
			err = winservice.Stop(serviceName)
		default:
			err = fmt.Errorf("unkown command %s", cmd)
		}

		if err != nil {
			log.Fatalf(`Error execute command %s: %v`, cmd, err)
		}

		return
	}

	// programm runs by service manager
	device, _ := NewSerialCasadDevice(*serialName, *serialBaudRate)
	service := NewService(*host, device)
	winservice.Run2(serviceName, &winservice.Service2{Handle: service})

}

func usage() {
	fmt.Fprintln(os.Stderr,
		`Usage: casadsrv [flags] command
command:
	install		- install service. 
	uninstall	- uninstall service
	start		- start service
	stop		- stop service

flags (used with command install or when started in non interative mode):
	-sn		- serial port name
	-sbr	- serial port baud rate
	-host	- ip address and port listened by http-server`)
}
