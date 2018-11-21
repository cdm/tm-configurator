package main

import (
	"github.com/cdm/tm-configurator/core"
	log "github.com/sirupsen/logrus"
	"os"
)

func main() {
	
	core.New().Run()
}

func init() {
	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)

	// Only log the warning severity or above.
	log.SetLevel(log.InfoLevel)
}
