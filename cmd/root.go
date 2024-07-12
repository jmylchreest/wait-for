package cmd

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"slices"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	urip "github.com/um7a/uri-parser"
)

const defaultLogLevel = "info"

var (
	version          string = "0.0.0"                // set by ldflags
	commit           string = "XXXXX"                // set by ldflags
	date             string = "1970-01-01T00:00:00Z" // set by ldflags
	timeout          int
	quiet            bool
	httpStatus       int
	supportedSchemes = []string{"tcp", "http", "https"}
)

func waitFor(uri *urip.Uri) {

	scheme := string(uri.Scheme)
	host := string(uri.Host)
	port := string(uri.Port)
	timeoutEnd := time.Now().Add(time.Duration(timeout) * time.Second)

	for {
		var err error
		var r *http.Response
		switch scheme {
		case "tcp":
			_, err = net.DialTimeout("tcp", net.JoinHostPort(host, port), time.Second)
		case "http":
			r, _ = http.Get(uri.String())
			if r.StatusCode == httpStatus {
				return
			} else {
				err = fmt.Errorf("HTTP status code %d != %d", r.StatusCode, httpStatus)
			}
		case "https":
			r, err = http.Get(uri.String())
			if err != nil {
				log.Debug().Err(err).Msg("Error fetching HTTP endpoint.")
			} else if r.StatusCode == httpStatus {
				log.Debug().Msgf("HTTP status code %d == %d", r.StatusCode, httpStatus)
				return
			} else {
				err = fmt.Errorf("HTTP status code %d != %d", r.StatusCode, httpStatus)
			}
		}

		if err == nil {
			return
		}

		if timeout != 0 && time.Now().After(timeoutEnd) {
			log.Error().Msgf("Operation timed out")
			os.Exit(1)
		}

		time.Sleep(time.Second)
	}
}

func execCommand(commandArgs []string) {
	var command = ""

	if len(commandArgs) == 0 {
		log.Debug().Msgf("No command specified.")
		return
	}

	command = commandArgs[0]
	commandArgs = commandArgs[1:]

	log.Debug().Msgf("Command specified: %s", command)
	execCmd := exec.Command(command, commandArgs...)
	execCmd.Dir = "."
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	// Run the command
	err := execCmd.Run()
	if err != nil {
		log.Error().Err(err).Msgf("Error executing command: %v", err)
		os.Exit(execCmd.ProcessState.ExitCode())
	}
}

var rootCmd = &cobra.Command{
	Use:   "wait-for tcp://host:port|http://host:port|https://host:port [-- command]",
	Short: "Wait for a TCP port or HTTP endpoint to be available",
	Long: `Wait for a TCP port to become available or an HTTP endpoint to respond.
Can optionally run a command after the port/endpoint is ready.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		// Parse the provided URI
		uriArg := args[0]
		uri, err := urip.Parse([]byte(uriArg))
		if err != nil {
			log.Error().Err(err).Msgf("Error parsing URI: %s", err)
			os.Exit(1)
		}

		// check its one we support
		if !slices.Contains(supportedSchemes, string(uri.Scheme)) {
			log.Error().Err(err).Msgf("Unsupported protocol '%s'", uri.Scheme)
			os.Exit(1)
		}

		// Must define port for TCP
		if bytes.Equal([]byte("tcp"), uri.Scheme) && len(uri.Port) == 0 {
			log.Error().Err(err).Msgf("TCP protocol requires a port to be specified")
			os.Exit(1)
		}

		// Wait for the port to become available
		waitFor(uri)

		// Fetch any post-wait command args and execute them
		if cmd.ArgsLenAtDash() > 0 {
			commandArgs := args[cmd.ArgsLenAtDash():]
			execCommand(commandArgs)
		}
	},
}

func init() {
	logLevel := os.Getenv("LOG_LEVEL")
	level, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		log.Warn().Msgf("Invalid log level %q, defaulting to '%s'", logLevel, defaultLogLevel)
		level, _ = zerolog.ParseLevel(defaultLogLevel)
	}
	zerolog.SetGlobalLevel(level)

	rootCmd.Flags().IntVarP(&timeout, "timeout", "t", 15, "Timeout in seconds, zero for no timeout")
	rootCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Do not output any status messages")
	rootCmd.Flags().BoolP("version", "v", false, "Show the version of this tool")
	rootCmd.Flags().IntVarP(&httpStatus, "http-status", "s", 200, "HTTP status code to wait for")
	rootCmd.Version = fmt.Sprintf("%s-%s [%s]", version, commit, date)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Error().Msgf("Execute error: %s", err)
		os.Exit(1)
	}
}
