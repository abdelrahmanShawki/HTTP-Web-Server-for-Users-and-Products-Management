package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// this server in simple words , setup to handle graceful shutdown in another gorouyine

func (app *application) serve() error {
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", app.config.port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ErrorLog:     log.New(app.logger, "", 0), //non
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	shutDownError := make(chan error)
	go func() {
		shutDown := make(chan os.Signal, 1)
		signal.Notify(shutDown, syscall.SIGTTIN, syscall.SIGTERM)
		s := <-shutDown

		app.logger.PrintInfo("shutdown server signal received ", map[string]string{
			"signal": s.String(),
		})
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := srv.Shutdown(ctx)
		if err != nil {
			shutDownError <- err
		}

		app.logger.PrintInfo("wait .... completing background tasks", map[string]string{
			"addr": srv.Addr,
		})

		app.waitGroup.Wait()
		shutDownError <- nil

	}()
	app.logger.PrintInfo("starting server ", map[string]string{
		"addr": srv.Addr,
		"env":  app.config.env,
	})
	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	err = <-shutDownError
	if err != nil {
		return err
	}
	app.logger.PrintInfo("server stopped", map[string]string{
		"addr": srv.Addr,
	})
	return nil
}
