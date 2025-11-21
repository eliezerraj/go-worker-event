package service

import (
	"fmt"
	"context"
	"sync"
	//"encoding/json"

	"github.com/rs/zerolog"

	"github.com/go-worker-event/shared/erro"
	"github.com/go-worker-event/internal/domain/model"
	"github.com/go-worker-event/internal/infrastructure/repo/database"
	"github.com/go-worker-event/internal/infrastructure/adapter/event"

	go_core_http 		"github.com/eliezerraj/go-core/v2/http"
	go_core_db_pg 		"github.com/eliezerraj/go-core/v2/database/postgre"
	go_core_otel_trace 	"github.com/eliezerraj/go-core/v2/otel/trace"
	"go.opentelemetry.io/otel/trace"
)

var tracerProvider go_core_otel_trace.TracerProvider

type WorkerService struct {
	appServer			*model.AppServer
	workerRepository	*database.WorkerRepository
	logger 				*zerolog.Logger
	workerEvent			*event.WorkerEvent
	httpService			*go_core_http.HttpService
	mutex    			sync.Mutex 	 	
}

// About new worker service
func NewWorkerService(appServer	*model.AppServer,
					  workerRepository *database.WorkerRepository, 
					  workerEvent	*event.WorkerEvent,
					  appLogger *zerolog.Logger) *WorkerService {

	logger := appLogger.With().
						Str("package", "domain.service").
						Logger()
	logger.Info().
			Str("func","NewWorkerService").Send()

	httpService := go_core_http.NewHttpService(&logger)					

	return &WorkerService{
		appServer: appServer,
		workerRepository: workerRepository,
		logger: &logger,
		workerEvent: workerEvent,
		httpService: httpService,
	}
}

// About database stats
func (s *WorkerService) Stat(ctx context.Context) (go_core_db_pg.PoolStats){
	s.logger.Info().
			Str("func","Stat").Send()

	return s.workerRepository.Stat(ctx)
}

// About check health service
func (s * WorkerService) HealthCheck(ctx context.Context) error {
	s.logger.Info().
			Str("func","HealthCheck").Send()

	// Check database health
	err := s.workerRepository.DatabasePG.Ping()
	if err != nil {
		s.logger.Error().
				Err(err).Msg("*** Database HEALTH FAILED ***")
		return erro.ErrHealthCheck
	}

	s.logger.Info().
			Str("func","HealthCheck").
			Msg("*** Database HEALTH SUCCESSFULL ***")

	return nil
}

func (s *WorkerService) Clearance(ctx context.Context) error {
	// trace and log
	ctx, span := tracerProvider.SpanCtx(ctx, "service.Clearance")
	defer span.End()

	s.logger.Info().
			Ctx(ctx).
			Str("func","Clearance").Send()

	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		// Add the OTel Trace ID and Span ID fields to the log event
		//fmt.Printf("===> TraceID: %d \n", spanCtx.TraceID().String())
		//fmt.Printf("===> SpanID: %d \n", spanCtx.SpanID().String())
		fmt.Println("traceID", spanCtx.TraceID().String())
		fmt.Println("spanID", spanCtx.SpanID().String())
	}

	return nil
}