package cmd

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/kikeuf/chirpstack-simulator/internal/as"
	"github.com/kikeuf/chirpstack-simulator/internal/config"
	"github.com/kikeuf/chirpstack-simulator/internal/ns"
	"github.com/kikeuf/chirpstack-simulator/internal/simulator"
	"github.com/kikeuf/chirpstack-simulator/internal/listener"
)

func run(cnd *cobra.Command, args []string) error {
	tasks := []func(context.Context, *sync.WaitGroup) error{
		setLogLevel,
		printStartMessage,
		setupASAPIClient,
		setupASIntegration,
		setupNSIntegration,
		setupPrometheus,
		setupListener,
		startSimulator,
	}

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	for _, t := range tasks {
		if err := t(ctx, &wg); err != nil {
			log.Fatal(err)
		}
	}

	exitChan := make(chan struct{})
	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
	go func() {
		cancel()
		wg.Wait()
		exitChan <- struct{}{}
	}()
	cancel()
	select {
	case <-exitChan:
	case s := <-sigChan:
		log.WithField("signal", s).Info("signal received, terminating")
	}

	return nil
}

func setLogLevel(ctx context.Context, wg *sync.WaitGroup) error {
	log.SetLevel(log.Level(uint8(config.C.General.LogLevel)))
	return nil
}

func printStartMessage(ctx context.Context, wg *sync.WaitGroup) error {
	log.WithFields(log.Fields{
		"version": version,
		"docs":    "https://www.chirpstack.io/",
	}).Info("starting ChirpStack Simulator")
	return nil
}

func setupASAPIClient(ctx context.Context, wg *sync.WaitGroup) error {
	return as.Setup(config.C)
}

func setupASIntegration(ctx context.Context, wg *sync.WaitGroup) error {
	return nil
}

func setupNSIntegration(ctx context.Context, wg *sync.WaitGroup) error {
	return ns.Setup(config.C)
}

func setupPrometheus(ctx context.Context, wg *sync.WaitGroup) error {
	log.WithFields(log.Fields{
		"bind": config.C.Prometheus.Bind,
	}).Info("starting Prometheus endpoint server")

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	server := http.Server{
		Handler: mux,
		Addr:    config.C.Prometheus.Bind,
	}

	go func() {
		err := server.ListenAndServe()
		log.WithError(err).Error("prometheus endpoint server error")
	}()

	return nil
}

func setupListener(ctx context.Context, wg *sync.WaitGroup) error {
	if config.C.Listener.Activate {	
		if (config.C.Listener.Port==uint64(0)) { config.C.Listener.Port=uint64(8090) }		
		go listener.Listen(config.C.Listener.Port,config.C.Listener.JSON)	
		
	}

	return nil
}

func startSimulator(ctx context.Context, wg *sync.WaitGroup) error {
	if err := simulator.Start(ctx, wg, config.C); err != nil {
		return errors.Wrap(err, "start simulator error")
	}

	return nil
}
