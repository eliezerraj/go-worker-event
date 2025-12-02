package service

import (
	"time"
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
			Ctx(ctx).
			Str("func","Stat").Send()

	return s.workerRepository.Stat(ctx)
}

// About check health service
func (s * WorkerService) HealthCheck(ctx context.Context) error {
	s.logger.Info().
			Ctx(ctx).
			Str("func","HealthCheck").Send()

	ctx, span := tracerProvider.SpanCtx(ctx, "service.HealthCheck")
	defer span.End()

	// Check database health
	_, spanDB := tracerProvider.SpanCtx(ctx, "DatabasePG.Ping")
	err := s.workerRepository.DatabasePG.Ping()
	if err != nil {
		s.logger.Error().
				Ctx(ctx).
				Err(err).Msg("*** Database HEALTH FAILED ***")
		return erro.ErrHealthCheck
	}
	spanDB.End()

	s.logger.Info().
			Ctx(ctx).
			Str("func","HealthCheck").
			Msg("*** Database HEALTH SUCCESSFULL ***")

	return nil
}

func (s *WorkerService) ClearanceReconciliacion(ctx context.Context, reconciliation *model.Reconciliation) error {
	// trace and log
	s.logger.Info().
			Ctx(ctx).
			Str("func","ClearanceReconciliacion").Send()

	ctx, span := tracerProvider.SpanCtx(ctx, "service.ClearanceReconciliacion")
	defer span.End()

	// prepare database
	tx, conn, err := s.workerRepository.DatabasePG.StartTx(ctx)
	if err != nil {
		return err
	}

	// handle connection
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		} else {
			tx.Commit(ctx)
		}
		s.workerRepository.DatabasePG.ReleaseTx(conn)
		span.End()
	}()

	// prepare data
	now := time.Now()
	reconciliation.CreatedAt = now
	reconciliation.Status = "CLEARANCE:RECEIVED"
	reconciliation.Currency	= "BRL"
	reconciliation.Amount = reconciliation.Payment.Amount - reconciliation.Order.Amount		   
	
	// Create payment
	_, err = s.workerRepository.ClearanceReconciliacion(ctx, tx, reconciliation)
	if err != nil {
		return err
	}
	
	return nil
}