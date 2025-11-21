package database

import (
		"context"
		//"errors"
		//"database/sql"

		"github.com/rs/zerolog"
		//"github.com/jackc/pgx/v5"

		//"github.com/go-worker-event/shared/erro"
		//"github.com/go-worker-event/internal/domain/model"

		go_core_otel_trace "github.com/eliezerraj/go-core/v2/otel/trace"
		go_core_db_pg "github.com/eliezerraj/go-core/v2/database/postgre"
)

var tracerProvider go_core_otel_trace.TracerProvider

type WorkerRepository struct {
	DatabasePG *go_core_db_pg.DatabasePGServer
	logger		*zerolog.Logger
}

// Above new worker
func NewWorkerRepository(databasePG *go_core_db_pg.DatabasePGServer,
						appLogger *zerolog.Logger) *WorkerRepository{
	logger := appLogger.With().
						Str("package", "repo.database").
						Logger()
	logger.Info().
			Str("func","NewWorkerRepository").Send()

	return &WorkerRepository{
		DatabasePG: databasePG,
		logger: &logger,
	}
}

// Above get stats from database
func (w *WorkerRepository) Stat(ctx context.Context) (go_core_db_pg.PoolStats){
	w.logger.Info().
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
