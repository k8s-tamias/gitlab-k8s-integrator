package glk8smain

import (
	"fmt"
	"gitlab.informatik.haw-hamburg.de/icc/gl-k8s-integrator/webhooklistener"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func Main() {
	log.Println("Gitlab K8s Integrator starting up!")
	quit := make(chan int)

	// Handle System signals
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	go func() {
		s := <-sigc
		switch s {
		case os.Interrupt:
			quit <- 0
		case syscall.SIGTERM:
			quit <- 0
		}
	}()

	// listen in sep. routine
	go webhooklistener.Listen(quit)
	log.Println("Gitlab K8s Integrator listening!")
	// Wait until server signals quit
	select {
	case <-quit:
		fmt.Println("Goodbye!")
	}

}
