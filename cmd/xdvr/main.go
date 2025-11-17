package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
)

var (
	cfg               *Config
	log               zerolog.Logger
	logFile           *os.File
	logPath, logLevel string
	debug, prettyLog  bool
	timeout           time.Duration
)

func main() {
	rootCMD := newRootCMD()
	rootCMD.AddCommand(newConfigCMD())
	rootCMD.AddCommand(newSyncCMD())

	if err := rootCMD.Execute(); err != nil {
		panic(err)
	}
}

func newRootCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "xdvr",
		Short:   "Download clips and screenshots from your Xbox account.",
		Version: "0.1.2",
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			if logPath == "" {
				logPath = "xdvr.log"
			}

			var err error

			logFile, err = os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
			if err != nil {
				fmt.Printf("Unable to create log file: %s: %v\n", logPath, err)
				os.Exit(1)
			}

			logLevelParsed, err := zerolog.ParseLevel(strings.ToLower(logLevel))
			if err != nil {
				fmt.Printf("Unable to parse log level: %s: %v\n", logLevel, err)
				os.Exit(1)
			}

			if debug {
				logLevelParsed = zerolog.DebugLevel

				// Start pprof in the background.
				go func() {
					log.Info().Msg("Starting pprof on port 6060")

					const idleTimeout = 30 * time.Second
					const readTimeout = 10 * time.Second
					const writeTimeout = 10 * time.Second

					srv := &http.Server{Addr: ":6060", IdleTimeout: idleTimeout, ReadTimeout: readTimeout, WriteTimeout: writeTimeout}

					if err := srv.ListenAndServe(); err != nil {
						log.Error().Err(err).Msg("Pprof server failed")
					}
				}()
			}

			newLogger(logFile, logLevelParsed)

			configDir, err := os.UserHomeDir()
			if err != nil {
				log.Fatal().Err(err).Msg("Unable to retrieve user's home directory")
			}

			cfg, err = newConfig(filepath.Join(configDir, ".config", "xbox-dvr"))
			if err != nil {
				log.Fatal().Err(err).Msg("Unable to initialize config")
			}

			if debug {
				log.Info().Msg("Debug mode enabled")
				log.Info().Msg("Auto delete: " + cast.ToString(cfg.AutoDelete))
				log.Info().Msg("Save path: " + cfg.SavePath)
				log = log.Level(zerolog.DebugLevel)
			}
		},
		PersistentPostRun: func(_ *cobra.Command, _ []string) {
			if logFile != nil {
				if err := logFile.Close(); err != nil {
					fmt.Println("Failed to gracefully close log file")
					panic(err)
				}
			}
		},
	}

	cmd.PersistentFlags().BoolVarP(&debug, "debug", "d", cast.ToBool(getEnvFallback("DEBUG", "false")), "Print debug logs")
	cmd.PersistentFlags().StringVar(&logLevel, "logLevel", getEnvFallback("LOG_LEVEL", "info"), "Set log level (debug, info, warn, error, fatal, panic)")
	cmd.PersistentFlags().StringVar(&logPath, "logPath", getEnvFallback("LOG_PATH", ""), "Set log path (defaults to the current directory")
	cmd.PersistentFlags().BoolVar(&prettyLog, "prettyLog", cast.ToBool(getEnvFallback("PRETTY_LOG", "true")), "Pretty print logs to console")
	cmd.PersistentFlags().DurationVarP(&timeout, "timeout", "t", cast.ToDuration(getEnvFallback("TIMEOUT", "60000000000")), "Timeout duration for queries")

	return cmd
}

func getEnvFallback(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}

	return fallback
}

func newLogger(logFile *os.File, logLevel zerolog.Level) {
	var logWriter io.Writer

	if prettyLog {
		logWriter = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	} else {
		logWriter = os.Stdout
	}

	if logFile == nil {
		log = zerolog.New(logWriter).With().Timestamp().Logger().Level(logLevel)
	}

	log = zerolog.New(zerolog.MultiLevelWriter(logWriter, logFile)).With().Timestamp().Logger().Level(logLevel)
}
