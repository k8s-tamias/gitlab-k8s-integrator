package main

import "gitlab.informatik.haw-hamburg.de/icc/gl-k8s-integrator/graylog"

func main() {
	graylog.DeleteAllRulesForAllStreams()
}

