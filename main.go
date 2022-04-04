package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
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

const (
	banner = `Krateo Control Plane
┏┓        ┏┓          ┏┓       ┏┓   ┏┓
┃┃┏┓ ┏┓┏┓ ┃┗━┓ ┏━━┓   ┃┗━┓ ┏━┓ ┗┛ ┏━┛┃ ┏━━┓ ┏━━┓ 
┃┗┛┛ ┃┃┃┃ ┃┏┓┃ ┃┃━┫   ┃┏┓┃ ┃┏┛ ┏┓ ┃┏┓┃ ┃┏┓┃ ┃┃━┫ 
┃┏┓┓ ┃┗┛┃ ┃┗┛┃ ┃┃━┫   ┃┗┛┃ ┃┃  ┃┃ ┃┗┛┃ ┃┗┛┃ ┃┃━┫ 
┗┛┗┛ ┗━━┛ ┗━━┛ ┗━━┛   ┗━━┛ ┗┛  ┗┛ ┗━━┛ ┗━┓┃ ┗━━┛ ver: VERSION
Kubernetes Bridge Component            ┗━━┛      cid: BUILD`
)

var (
	Version string
	Build   string
)

func main() {
	// Flags
	defaultKubeconfig := os.Getenv(clientcmd.RecommendedConfigPathEnvVar)
	kubeconfig := flag.String(clientcmd.RecommendedConfigPathFlag,
		defaultKubeconfig, "absolute path to the kubeconfig file")

	debug := flag.Bool("debug", support.EnvBool("KUBE_BRIDGE_DEBUG", false), "dump verbose output")

	loggerServiceUrl := flag.String("logger-service-url", support.EnvString("LOGGER_SERVICE_URL", ""),
		"logger service url")

	bootstrap := flag.Bool("bootstrap", support.EnvBool("KUBE_BRIDGE_BOOTSTRAP", true), "enable/disable Krateo runtime bootstrap")

	servicePort := flag.Int("port", support.EnvInt("KUBE_BRIDGE_PORT", 8171), "port to listen on")

	flag.Usage = func() {
		printBanner()
		fmt.Fprintln(flag.CommandLine.Output(), "Flags:")
		flag.PrintDefaults()
	}

	flag.Parse()

	// Initialize the logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	// Default level for this log is info, unless debug flag is present
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	log := zerolog.New(os.Stdout).With().
		Str("service", support.ServiceName).
		Timestamp().
		Logger()

	if log.Debug().Enabled() {
		log.Debug().
			Str("bootstrap", fmt.Sprintf("%t", *bootstrap)).
			Str("debug", fmt.Sprintf("%t", *debug)).
			Str("loggerServiceUrl", *loggerServiceUrl).
			Str("port", fmt.Sprintf("%d", *servicePort)).
			Msg("configuration values")
	}

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
	if *bootstrap {
		err = boot.Run(boot.BootOptions{
			Config:  cfg,
			Verbose: *debug,
			Bus:     bus,
		})
		if err != nil {
			bus.Publish(support.ErrorNotification(err))
			log.Fatal().Err(err).Msg("booting krateo required deps")
		}
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
		Addr:         fmt.Sprintf(":%d", *servicePort),
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

func printBanner() {
	res := strings.Replace(banner, "VERSION", Version, 1)
	res = strings.Replace(res, "BUILD", Build, 1)
	fmt.Fprintf(flag.CommandLine.Output(), "%s:\n\n", res)
}
