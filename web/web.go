package web

import (
	"context"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	"net/http"
	"os"
	"os/signal"
	"time"
)

type Web struct {
	Server   *http.Server
	shutdown chan os.Signal
}

func (w *Web) listenAndServe() error {
	if w.shutdown == nil {
		w.shutdown = make(chan os.Signal)
	}
	if err := w.Server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Debug().Err(err).Msg("Server error")
		return err
	}
	<-w.shutdown
	return nil
}

func (w *Web) waitForShutdown() error {
	signals := make(chan os.Signal)
	signal.Notify(signals, os.Interrupt, os.Kill)
	<-signals
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := w.Server.Shutdown(ctx)
	log.Debug().Err(err).Msg("Server shutdown")
	defer close(w.shutdown)
	return err
}

func (w *Web) Init(port int, handler http.Handler) {
	w.Server = &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%d", port),
		Handler: handler,
	}
}

func (w *Web) Run() error {
	if w.Server == nil {
		return fmt.Errorf("server must be initialised")
	}
	g := new(errgroup.Group)
	g.Go(func() error {
		return w.listenAndServe()
	})
	g.Go(func() error {
		return w.waitForShutdown()
	})
	return g.Wait()
}
