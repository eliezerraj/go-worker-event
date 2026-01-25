package server

import(
	"sync"
	"context"
	"encoding/json"
	"github.com/google/uuid"

	"github.com/rs/zerolog"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"

	go_core_event "github.com/eliezerraj/go-core/v2/event/kafka" 
	go_core_otel_trace 	"github.com/eliezerraj/go-core/v2/otel/trace"

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

// Set a trace-i inside the context
func (e *EventAppServer) setContextTraceId(ctx context.Context, trace_id string) context.Context {
	e.logger.Info().
			Ctx(ctx).
			Str("func","setContextTraceId").Send()

	var traceUUID string

	if trace_id == "" {
		traceUUID = uuid.New().String()
		trace_id = traceUUID
		
		e.logger.Info().
			Ctx(ctx).
			Str("func","setContextTraceId").
			Msg("Create a new trace_id !!!")
	}

	ctx = context.WithValue(ctx, "request-id",  trace_id  )
	return ctx
}

// About consume messages from kafka
func (e *EventAppServer) Consumer(ctx context.Context,
								  wg *sync.WaitGroup) {
	e.logger.Info().
			 Ctx(ctx).
			 Str("func","Consumer").Send()

	// cancel everything		
	defer func() {
		e.logger.Info().
				Ctx(ctx).
				Msg("Exiting consumer KAFKA SUCCESSFULL")
		
			 defer wg.Done()
	}()

	messages := make(chan go_core_event.Message)

	go e.consumerWorker.Consumer(*e.appServer.Topics, messages)
	
	for msg := range messages {

		e.logger.Info().Msg("=============== BEGIN - MSG FROM KAFKA - BEGIN =============")
		e.logger.Info().Interface("msg", msg).Send()
		e.logger.Info().Msg("=============== END - MSG FROM KAFKA - END ==================")

		// otel trace
		kafkaHeaderCarrier := KafkaHeaderCarrier{}
		kafkaHeaders := kafkaHeaderCarrier.MapToKafkaHeaders(*msg.Header)
		appCarrier := KafkaHeaderCarrier{Headers: &kafkaHeaders }

		// Extract all headers from appCarrier and add to context
		allKeys := appCarrier.Keys()
		for _, key := range allKeys {
			value := appCarrier.Get(key)
			ctx = context.WithValue(ctx, key, value)
		}

		ctx := otel.GetTextMapPropagator().Extract(ctx, appCarrier)
		ctx, span := e.tracerProvider.SpanCtx(ctx, e.appServer.Application.Name, trace.SpanKindConsumer)

		// Decode payload
		event := model.Event{}
		errUnMarshall := json.Unmarshal([]byte(msg.Payload), &event)
		if errUnMarshall != nil {
			e.logger.Error().
					 Ctx(ctx).
					 Err(errUnMarshall).Send()
			continue
		}

		// call service
		var err error
		if event.Type == "cleareance.order" {
			
			// convert interface to bytes
			paymentBytes, errUnMarshall := json.Marshal(event.EventData)
			if errUnMarshall != nil {
				e.logger.Error().
						Ctx(ctx).
						Err(errUnMarshall).Send()
				continue
			}

			// convert bytes to struct
			payment := model.Payment{}
			errUnMarshall = json.Unmarshal(paymentBytes, &payment)
			if errUnMarshall != nil {
				e.logger.Error().
						Ctx(ctx).
						Err(errUnMarshall).Send()
				continue
			}

			reconciliation := model.Reconciliation{ Transaction: payment.Transaction,
													Type: "RECONCILIATION:RECEIVED",
													Payment: &payment,
													Order: payment.Order }	

			e.logger.Info().
					 Ctx(ctx).
					 Interface("reconciliation",reconciliation).Send()

			err = e.workerService.ClearanceReconciliacion(ctx, &reconciliation)
			if err != nil {
				e.logger.Error().
						Ctx(ctx).
						Err(err).Send()
				continue
			}
		}

		// commit transaction
		e.consumerWorker.Commit()
		e.logger.Info().
				Ctx(ctx).
				Msg("KAFKA CONSUMER COMMIT SUCCESSFUL !!!")

		span.End()
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