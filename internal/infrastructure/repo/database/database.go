package database

import (
	"fmt"
	"context"

	"github.com/rs/zerolog"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/codes"

	"github.com/go-worker-event/internal/domain/model"

	go_core_otel_trace "github.com/eliezerraj/go-core/v2/otel/trace"
	go_core_db_pg "github.com/eliezerraj/go-core/v2/database/postgre"
)

// WorkerRepository struct
type WorkerRepository struct {
	DatabasePG 		*go_core_db_pg.DatabasePGServer
	logger			*zerolog.Logger
	tracerProvider 	*go_core_otel_trace.TracerProvider
}

// Above new worker
func NewWorkerRepository(databasePG *go_core_db_pg.DatabasePGServer,
						appLogger *zerolog.Logger,
						tracerProvider *go_core_otel_trace.TracerProvider) *WorkerRepository{
	logger := appLogger.With().
						Str("package", "repo.database").
						Logger()
	logger.Info().
		Str("func","NewWorkerRepository").Send()

	return &WorkerRepository{
		DatabasePG: databasePG,
		logger: &logger,
		tracerProvider: tracerProvider,
	}
}

// Above get stats from database
func (w *WorkerRepository) Stat(ctx context.Context) (go_core_db_pg.PoolStats){
	w.logger.Info().
		Ctx(ctx).
		Str("func","Stat").Send()
	
	stats := w.DatabasePG.Stat()

	resPoolStats := go_core_db_pg.PoolStats{
		AcquireCount:         stats.AcquireCount(),
		AcquiredConns:        stats.AcquiredConns(),
		CanceledAcquireCount: stats.CanceledAcquireCount(),
		ConstructingConns:    stats.ConstructingConns(),
		EmptyAcquireCount:    stats.EmptyAcquireCount(),
		IdleConns:            stats.IdleConns(),
		MaxConns:             stats.MaxConns(),
		TotalConns:           stats.TotalConns(),
	}

	return resPoolStats
}

// About create a clearance
func (w* WorkerRepository) ClearanceReconciliacion(ctx context.Context, 
													tx pgx.Tx, 
													reconciliation *model.Reconciliation) (*model.Reconciliation, error){
	w.logger.Info().
		Ctx(ctx).
		Str("func","ClearanceReconciliacion").Send()
			
	// trace
	ctx, span := w.tracerProvider.SpanCtx(ctx, "database.ClearanceReconciliacion", trace.SpanKindInternal)
	defer span.End()

	//Prepare
	var id int

	// Query Execute
	query := `INSERT INTO order_clearance_reconciliacion ( 	fk_clearance_id,
															fk_order_id,
															transaction_id,
															clearance_status,
															clearance_currency,
															clearance_amount,
															order_status,
															order_currency,
															order_amount,
															reconciliacion_type,
															reconciliacion_status,
															reconciliacion_currency,
															reconciliacion_amount,
															created_at) 
				VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14) RETURNING id`

	row := tx.QueryRow(	ctx, 
						query,
						reconciliation.Payment.ID,
						reconciliation.Order.ID,
						reconciliation.Transaction,

						reconciliation.Payment.Status,
						reconciliation.Payment.Currency,	
						reconciliation.Payment.Amount,	

						reconciliation.Order.Status,
						reconciliation.Order.Currency,
						reconciliation.Order.Amount,
						
						reconciliation.Type,
						reconciliation.Status,
						reconciliation.Currency,
						reconciliation.Amount,
						reconciliation.CreatedAt)
						
	if err := row.Scan(&id); err != nil {
		span.RecordError(err) 
        span.SetStatus(codes.Error, err.Error())
		w.logger.Error().
			Ctx(ctx).
			Err(err).Send()
		return nil, fmt.Errorf("FAILED to scan reconciliation ID: %w", err)
	}

	// Set PK
	reconciliation.ID = id
	
	return reconciliation , nil
}
