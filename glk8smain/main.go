/*
	Copyright 2017 by Christian Hüning (christianhuening@googlemail.com).

	Licensed under the Apache License, Version 2.0 (the "License");
	you may not use this file except in compliance with the License.
	You may obtain a copy of the License at
		http://www.apache.org/licenses/LICENSE-2.0

	Unless required by applicable law or agreed to in writing, software
	distributed under the License is distributed on an "AS IS" BASIS,
	WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
	See the License for the specific language governing permissions and
	limitations under the License.
*/

package glk8smain

import (
	"fmt"
	"gitlab.informatik.haw-hamburg.de/icc/gl-k8s-integrator/usecases"
	"gitlab.informatik.haw-hamburg.de/icc/gl-k8s-integrator/webhooklistener"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func Main() {
	log.Println("Gitlab K8s Integrator starting up!")
	if os.Getenv("GITLAB_HOSTNAME") == "" {
		log.Fatalln("Please provide GITLAB_HOSTNAME env!")
	}
	if os.Getenv("GITLAB_API_VERSION") == "" {
		log.Fatalln("Please provide GITLAB_API_VERSION env!")
	}
	if os.Getenv("GITLAB_PRIVATE_TOKEN") == "" {
		log.Fatalln("Please provide GITLAB_PRIVATE_TOKEN env!")
	}

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
	go usecases.StartRecurringSyncTimer()
	log.Println("Gitlab K8s Integrator listening!")
	// Wait until server signals quit
	select {
	case <-quit:
		fmt.Println("Goodbye!")
	}

}
