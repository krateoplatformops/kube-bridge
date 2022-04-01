package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/krateoplatformops/kube-bridge/pkg/boot"
	"github.com/krateoplatformops/kube-bridge/pkg/eventbus"
	"github.com/krateoplatformops/kube-bridge/pkg/handlers"
	"github.com/krateoplatformops/kube-bridge/pkg/support"
	"github.com/rs/zerolog"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// Flags
	defaultKubeconfig := os.Getenv(clientcmd.RecommendedConfigPathEnvVar)
	kubeconfig := flag.String(clientcmd.RecommendedConfigPathFlag,
		defaultKubeconfig, "absolute path to the kubeconfig file")
	verbose := flag.Bool("verbose", false, "dump verbose output")
	loggerServiceUrl := flag.String("logger-service-url", "", "logger service url")

	flag.Parse()

	// Initialize the logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	// Default level for this log is info, unless debug flag is present
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *verbose {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	log := zerolog.New(os.Stdout).With().
		Str("service", support.ServiceName).
		Timestamp().
		Logger()

	// Kubernetes configuration
	var cfg *rest.Config
	var err error
	if len(*kubeconfig) > 0 {
		cfg, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
	} else {
		cfg, err = rest.InClusterConfig()
	}
	if err != nil {
		log.Fatal().Err(err).Msg("building kube config")
	}

	// Internal event bus for sending notifications
	nd := support.NotificationDispatcher(*loggerServiceUrl, &log)
	bus := eventbus.New()
	eid := bus.Subscribe(support.NotificationEventID, nd)
	defer bus.Unsubscribe(eid)

	// Bootstrap krateo runtime
	err = boot.Run(boot.BootOptions{
		Config:  cfg,
		Verbose: *verbose,
		Bus:     bus,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("booting krateo required deps")
		support.ErrorNotification(err)
	}

	// Server Mux
	mux := http.NewServeMux()

	// HealtZ endpoint
	healthy := int32(0)
	mux.Handle("/healtz", handlers.HealtHandler(&healthy))

	/*
		mux.Handle("/apply", middlewares.Logger(log)(
			middlewares.CorrelationID(
				handlers.ApplyHandler(cfg),
			),
		))
	*/
	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", support.Env("PORT", "8080")),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  20 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), []os.Signal{
		os.Interrupt,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGKILL,
		syscall.SIGHUP,
		syscall.SIGQUIT,
	}...)
	defer stop()

	go func() {
		atomic.StoreInt32(&healthy, 1)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msgf("could not listen on %s", server.Addr)
			support.ErrorNotification(err)
		}
	}()

	// Listen for the interrupt signal.
	log.Info().Msgf("server is ready to handle requests at @ %s", server.Addr)
	<-ctx.Done()

	// Restore default behavior on the interrupt signal and notify user of shutdown.
	stop()
	log.Info().Msg("server is shutting down gracefully, press Ctrl+C again to force")
	atomic.StoreInt32(&healthy, 0)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	server.SetKeepAlivesEnabled(false)
	if err := server.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("server forced to shutdown")
	}

	log.Info().Msg("server gracefully stopped")
}
