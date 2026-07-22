package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/iman/sms-gateway/internal/config"
	"github.com/iman/sms-gateway/internal/repository/memory"
	"github.com/iman/sms-gateway/internal/service"
	httpapi "github.com/iman/sms-gateway/internal/transport/http"
)

func main() {
	cfg := config.Load()

	customerRepo := memory.NewCustomerRepository()
	messageRepo := memory.NewMessageRepository()

	standardOperators := []service.OperatorGateway{
		service.NewSimulatedOperator("operator-a-low-cost", 50*time.Millisecond, 400*time.Millisecond, 0.01),
		service.NewSimulatedOperator("operator-b-standard", 30*time.Millisecond, 250*time.Millisecond, 0.01),
	}
	expressOperators := []service.OperatorGateway{
		service.NewSimulatedOperator("operator-c-express", 10*time.Millisecond, 80*time.Millisecond, 0.005),
	}
	router := service.NewOperatorRouter(standardOperators, expressOperators)

	dispatcher := service.NewDispatcher(router, messageRepo, cfg.NormalWorkers, cfg.ExpressWorkers, cfg.QueueSize)

	smsService := service.NewSMSService(customerRepo, messageRepo, cfg.Prices, dispatcher)

	handler := httpapi.NewRouter(smsService)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	go func() {
		log.Printf("sms-gateway listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("graceful shutdown failed: %v", err)
	}
	log.Println("shutdown complete")
}
