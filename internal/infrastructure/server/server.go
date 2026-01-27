package server

import(
	"time"
	"sync"
	"context"
	"encoding/json"

	"github.com/rs/zerolog"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"

	go_core_event "github.com/eliezerraj/go-core/v2/event/kafka" 
	go_core_otel_trace 	"github.com/eliezerraj/go-core/v2/otel/trace"
	go_core_middleware "github.com/eliezerraj/go-core/v2/middleware"

	"github.com/go-worker-event/internal/domain/model"
	"github.com/go-worker-event/internal/domain/service"

	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel"
)

type EventAppServer struct {
	appServer			*model.AppServer
	consumerWorker 		*go_core_event.ConsumerWorker
	workerService		*service.WorkerService
	logger 				*zerolog.Logger
	tracerProvider 		*go_core_otel_trace.TracerProvider	
}

// About create a consumer worker event
func NewEventAppServer(	appServer *model.AppServer,
						workerService *service.WorkerService,
						appLogger *zerolog.Logger,
						tracerProvider *go_core_otel_trace.TracerProvider) (*EventAppServer, error) {
	logger := appLogger.With().
						Str("package", "infrastructure.server").
						Logger()
	
	logger.Info().
			Str("func","NewEventAppServer").Send()

	consumerWorker, err := go_core_event.NewConsumerWorker(appServer.KafkaConfigurations,
														   &logger)	
	if err != nil {
		logger.Error().
				Err(err).Send()
		return nil, err
	}

	return &EventAppServer{
		appServer: appServer,
		workerService: workerService,
		consumerWorker: consumerWorker,
		logger: &logger,
		tracerProvider: tracerProvider,
	}, nil
}

// About consume messages from kafka
func (e *EventAppServer) Consumer(ctx context.Context,
								  wg *sync.WaitGroup) {
	e.logger.Info().
			 Ctx(ctx).
			 Str("func","Consumer").Send()

	// Ensure WaitGroup is marked done when this goroutine exits and log on exit
	defer wg.Done()
	defer func() {
		e.logger.Info().
			Ctx(ctx).
			Msg("Exiting consumer KAFKA SUCCESSFULL")
	}()

	messages := make(chan go_core_event.Message)

	go e.consumerWorker.Consumer(*e.appServer.Topics, messages)
	
	for msg := range messages {

		e.logger.Info().Msg("=============== BEGIN - KAFKA CONSUMER- BEGIN =============")
		e.logger.Info().Interface("msg", msg).Send()
		e.logger.Info().Msg("=============== END - KAFKA CONSUMER-- END ==================")

		// otel trace
		kafkaHeaderCarrier := KafkaHeaderCarrier{}
		kafkaHeaders := kafkaHeaderCarrier.MapToKafkaHeaders(*msg.Header)
		appCarrier := KafkaHeaderCarrier{Headers: &kafkaHeaders}

		// Process each message in its own scope so per-message defers run immediately
		func() {
			msgCtx := ctx

			// Extract all headers from appCarrier and add to context (do not mutate outer ctx)
			for _, key := range appCarrier.Keys() {
				value := appCarrier.Get(key)

				switch key {
				case string(go_core_middleware.RequestIDKey):
					msgCtx = context.WithValue(msgCtx, go_core_middleware.RequestIDKey, value)
				default:
					msgCtx = context.WithValue(msgCtx, key, value) // optional
				}
			}

			// per-message timeout (use configured value if available)
			timeoutSec := e.appServer.Server.CtxTimeout
			if e.appServer != nil && e.appServer.Server.CtxTimeout > 0 {
				timeoutSec = e.appServer.Server.CtxTimeout
			}

			msgCtx, cancel := context.WithTimeout(msgCtx, time.Duration(timeoutSec)*time.Second)
			defer cancel()

			msgCtx = otel.GetTextMapPropagator().Extract(msgCtx, appCarrier)

			msgCtx, span := e.tracerProvider.SpanCtx(msgCtx, e.appServer.Application.Name, trace.SpanKindConsumer)
			defer span.End()

			// Decode payload
			event := model.Event{}
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				e.logger.Error().
					Ctx(msgCtx).
					Err(err).Send()
				return
			}

			// call service
			if event.Type == "cleareance.order" {
				// convert interface to bytes
				paymentBytes, err := json.Marshal(event.EventData)
				if err != nil {
					e.logger.Error().
						Ctx(msgCtx).
						Err(err).Send()
					return
				}

				// convert bytes to struct
				payment := model.Payment{}
				if err := json.Unmarshal(paymentBytes, &payment); err != nil {
					e.logger.Error().
						Ctx(msgCtx).
						Err(err).Send()
					return
				}

				reconciliation := model.Reconciliation{
					Transaction: payment.Transaction,
					Type:        "RECONCILIATION:RECEIVED",
					Payment:     &payment,
					Order:       payment.Order,
				}

				e.logger.Info().
					Ctx(msgCtx).
					Interface("reconciliation", reconciliation).Send()

				if err := e.workerService.ClearanceReconciliacion(msgCtx, &reconciliation); err != nil {
					e.logger.Error().
						Ctx(msgCtx).
						Err(err).Send()
					return
				}
			}

			// commit transaction only on success (commit specific message)
			if msg.Raw != nil {
				if err := e.consumerWorker.CommitMessage(msg.Raw); err != nil {
					e.logger.Error().
						Ctx(msgCtx).
						Err(err).
						Msg("KAFKA CONSUMER COMMIT FAILED")
				} else {
					e.logger.Info().
						Ctx(msgCtx).
						Msg("KAFKA CONSUMER COMMIT SUCCESSFUL !!!")
				}
			} else {
				// fallback to generic commit if raw message not available
				e.consumerWorker.Commit()
				e.logger.Info().
					Ctx(msgCtx).
					Msg("KAFKA CONSUMER COMMIT SUCCESSFUL (fallback) !!!")
			}
		}()
	}
}

// ----------------------------------------------
// Helper kafka header OTEL
// ----------------------------------------------

type KafkaHeaderCarrier struct {
	Headers *[]kafka.Header
}

func (c KafkaHeaderCarrier) Get(key string) string {
	for _, h := range *c.Headers {
		if h.Key == key {
			return string(h.Value)
		}
	}
	return ""
}

func (c KafkaHeaderCarrier) Set(key string, value string) {
	// remove existing key
	newHeaders := make([]kafka.Header, 0)
	for _, h := range *c.Headers {
		if h.Key != key {
			newHeaders = append(newHeaders, h)
		}
	}
	// append new key
	newHeaders = append(newHeaders, kafka.Header{
		Key:   key,
		Value: []byte(value),
	})
	*c.Headers = newHeaders
}

func (c KafkaHeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(*c.Headers))
	for _, h := range *c.Headers {
		keys = append(keys, h.Key)
	}
	return keys
}

func (c KafkaHeaderCarrier) MapToKafkaHeaders(m map[string]string) []kafka.Header {
    hdrs := make([]kafka.Header, 0, len(m))
    for k, v := range m {
        hdrs = append(hdrs, kafka.Header{
            Key:   k,
            Value: []byte(v),
        })
    }
    return hdrs
}