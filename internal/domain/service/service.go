package service

import (
	"time"
	"context"

	"github.com/rs/zerolog"

	"github.com/go-worker-event/shared/erro"
	"github.com/go-worker-event/internal/domain/model"
	"github.com/go-worker-event/internal/infrastructure/repo/database"

	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/codes"
	
	go_core_db_pg 		"github.com/eliezerraj/go-core/v2/database/postgre"
	go_core_otel_trace 	"github.com/eliezerraj/go-core/v2/otel/trace"
)

type WorkerService struct {
	workerRepository 	*database.WorkerRepository
	logger 				*zerolog.Logger
	tracerProvider 		*go_core_otel_trace.TracerProvider	
}

// About new worker service
func NewWorkerService(	workerRepository *database.WorkerRepository, 
						appLogger 		*zerolog.Logger,
						tracerProvider 	*go_core_otel_trace.TracerProvider,
						) *WorkerService {

	logger := appLogger.With().
						Str("package", "domain.service").
						Logger()
	logger.Info().
		Str("func","NewWorkerService").Send()

	return &WorkerService{
		workerRepository: workerRepository,
		logger: &logger,
		tracerProvider: tracerProvider,
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

	ctx, span := s.tracerProvider.SpanCtx(ctx, "service.HealthCheck", trace.SpanKindServer)
	defer span.End()

	// Check database health
	ctx, spanDB := s.tracerProvider.SpanCtx(ctx, "DatabasePG.Ping", trace.SpanKindInternal)
	err := s.workerRepository.DatabasePG.Ping()
	spanDB.End()

	if err != nil {
		span.RecordError(err) 
        span.SetStatus(codes.Error, err.Error())
		s.logger.Error().
			Ctx(ctx).
			Err(err).Msg("*** Database HEALTH FAILED ***")
		return erro.ErrHealthCheck
	}
	spanDB.End()

	s.logger.Info().
		Ctx(ctx).
		Msg("*** Database HEALTH SUCCESSFULL ***")

	return nil
}

func (s *WorkerService) ClearanceReconciliacion(ctx context.Context, reconciliation *model.Reconciliation) error {
	// trace and log
	s.logger.Info().
		Ctx(ctx).
		Str("func","ClearanceReconciliacion").Send()

	ctx, span := s.tracerProvider.SpanCtx(ctx, "service.ClearanceReconciliacion", trace.SpanKindServer)
	defer span.End()

	// prepare database
	tx, conn, err := s.workerRepository.DatabasePG.StartTx(ctx)
	if err != nil {
		span.RecordError(err) 
        span.SetStatus(codes.Error, err.Error())
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
	reconciliation.Type = "ORDER"
	reconciliation.Amount = reconciliation.Payment.Amount - reconciliation.Order.Amount		   
	
	if reconciliation.Payment.Currency	== "" {
		err = erro.ErrCurrencyRequired
		span.RecordError(err) 
        span.SetStatus(codes.Error, err.Error())
		s.logger.Error().
			Ctx(ctx).
			Err(err).Send()
		return err
	}

	reconciliation.Currency = reconciliation.Payment.Currency
	
	// Create payment
	_, err = s.workerRepository.ClearanceReconciliacion(ctx, tx, reconciliation)
	if err != nil {
		return err
	}
	
	return nil
}