package main

import (
	"encoding/json"
	"flag"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const defaultLogLevel = log.InfoLevel
const defaultContentType = "application/json"

var flagHealthcheckCommand = flag.String("command", "", "(Required) Command to run to perform healthcheck")
var flagHealthcheckCommandTimeout = flag.Int("timeout", 3, "Timeout in seconds for healthcheck command")
var flagHealthcheckPeriodicity = flag.String("periodicity", "@every 1s", "Healthcheck periodicity, must be robfig/cron compliant")
var flagLogLevel = flag.String("log-level", "info", "Log level")
var flagBind = flag.String("bind", ":8080", "Address to bind to")
var flagResourceName = flag.String("resource-name", "/health", "Name of the HTTP resource that delivers healthcheck results")

var currentCheck atomic.Value
var mutex sync.Mutex

type healthCheckResult struct {
	Check bool `json:"-"`
	Command string `json:"command"`
	Timestamp string `json:"timestamp"`
}

func init() {
	flag.Parse()

	if *flagHealthcheckCommand == "" {
        flag.PrintDefaults()
        os.Exit(1)
    }

	formatter := &log.TextFormatter{
    	FullTimestamp: true,
	}
	log.SetFormatter(formatter)

	if logLevel, err := log.ParseLevel(*flagLogLevel); err != nil {
		log.Warningf("Invalid log level %s, using default %s", *flagLogLevel, defaultLogLevel)
		log.SetLevel(defaultLogLevel)
	} else {
		log.SetLevel(logLevel)
	}
	log.Debugf("Log level set to %s", log.GetLevel())

	currentCheck.Store(healthCheckResult{})
}

func healthHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", defaultContentType)
	result := currentCheck.Load().(healthCheckResult)
	if result.Check{
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
	json.NewEncoder(w).Encode(result)
}

func performHealthCheck() {
	mutex.Lock()
	defer mutex.Unlock()

	commandArgs := strings.Fields(*flagHealthcheckCommand)
	currentTime := time.Now().Format(time.RFC3339)
	timeoutReached := false

    commandHasResult := make(chan string, 1)
	go func() {
		command := exec.Command(commandArgs[0], commandArgs[1:]...)
		if err := command.Run(); err != nil {
			log.Debugf("Health check with command \"%s\", result: KO", *flagHealthcheckCommand)
			currentCheck.Store(healthCheckResult{Check: false, Command: *flagHealthcheckCommand, Timestamp: currentTime})
		} else {
			if !timeoutReached {
				log.Debugf("Health check with command \"%s\", result: OK", *flagHealthcheckCommand)
				currentCheck.Store(healthCheckResult{Check: true, Command: *flagHealthcheckCommand, Timestamp: currentTime})
			} else {
				currentCheck.Store(healthCheckResult{Check: false, Command: *flagHealthcheckCommand, Timestamp: currentTime})
			}

		}
		commandHasResult <- "has result"
	}()
	select {
		case <-commandHasResult:
			log.Debugf("Command %s had results", *flagHealthcheckCommand)
		case <-time.After(time.Duration(*flagHealthcheckCommandTimeout) * time.Second):
			log.Debugf("Command %s reached timeout", *flagHealthcheckCommand)
			timeoutReached = true
    }
}

func main() {
	// Trap SIGINT
	trapSig := make(chan os.Signal, 1)                                       
	signal.Notify(trapSig, os.Interrupt)                                     
	go func() {                                                        
	  for sig := range trapSig {                                             
		log.Infof("Received signal %s, exiting", sig)
		os.Exit(0)                                                     
	  }                                                                
	}()   

	log.Infof("Starting ualive, listening on %s", *flagBind)
	log.Infof("Healthcheck resource name is %s", *flagResourceName)

	http.HandleFunc(*flagResourceName, healthHandler)
	go http.ListenAndServe(*flagBind, nil)

	c := cron.New()
	c.AddFunc(*flagHealthcheckPeriodicity, performHealthCheck)
	c.Start()
	defer c.Stop()

	// Keep running until we get a SIGINT
	select {}
}
