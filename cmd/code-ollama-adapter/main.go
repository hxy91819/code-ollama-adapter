package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"code-ollama-adapter/internal/proxy"
)

type stringList []string

func (s *stringList) String() string {
	return strings.Join(*s, ",")
}

func (s *stringList) Set(value string) error {
	if value == "" {
		return errors.New("value must not be empty")
	}
	*s = append(*s, value)
	return nil
}

func main() {
	var aliases stringList
	var maps stringList
	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	host := flags.String("host", "127.0.0.1", "listen host")
	port := flags.Int("port", 11435, "listen port")
	upstream := flags.String("upstream", "http://127.0.0.1:11434", "Ollama base URL")
	target := flags.String("model-target", "glm-5.2:cloud", "model name sent to Ollama")
	defaultReasoning := flags.String("default-reasoning-effort", "", "reasoning effort to inject when the client omits one")
	timeout := flags.Duration("timeout", 3000*time.Second, "upstream request timeout")
	flags.Var(&aliases, "model-alias", "client model alias to rewrite; repeatable")
	flags.Var(&maps, "reasoning-map", "reasoning effort mapping in FROM=TO form; repeatable")
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintln(flags.Output(), "Local Codex/Claude adapter for Ollama Cloud.")
		fmt.Fprintln(flags.Output(), "\nOptions:")
		flags.PrintDefaults()
		fmt.Fprintln(flags.Output(), "\nExamples:")
		fmt.Fprintf(flags.Output(), "  %s --port 11435\n", os.Args[0])
		fmt.Fprintf(flags.Output(), "  %s --reasoning-map xhigh=max --model-alias glm-5.2\n", os.Args[0])
	}
	if err := flags.Parse(os.Args[1:]); err != nil {
		log.Fatal(err)
	}

	if len(aliases) == 0 {
		aliases = []string{"glm-5.2", "glm-5.2:cloud[1m]"}
	}
	if len(maps) == 0 {
		maps = []string{"xhigh=max"}
	}

	config, err := proxy.NewConfig(proxy.ConfigInput{
		Upstream:               *upstream,
		ModelAliases:           aliases,
		ModelTarget:            *target,
		ReasoningMaps:          maps,
		DefaultReasoningEffort: *defaultReasoning,
		Timeout:                *timeout,
	})
	if err != nil {
		log.Fatal(err)
	}

	handler := proxy.NewHandler(config, log.Default())
	server := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", *host, *port),
		Handler:           handler,
		ReadHeaderTimeout: 15 * time.Second,
	}

	aliasList := append([]string(nil), aliases...)
	sort.Strings(aliasList)
	log.Printf(
		"listening on http://%s:%d upstream=%s aliases=%s target=%s reasoning_map=%v default_reasoning=%s",
		*host,
		*port,
		*upstream,
		strings.Join(aliasList, ","),
		*target,
		config.ReasoningMap,
		config.DefaultReasoningEffort,
	)

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

	select {
	case sig := <-signalCh:
		log.Printf("received %s, shutting down", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Fatal(err)
		}
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}
}
