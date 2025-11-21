package server

import(
	"sync"
	"context"
	"encoding/json"
	"github.com/google/uuid"

	"github.com/rs/zerolog"

	go_core_event "github.com/eliezerraj/go-core/v2/event/kafka" 
	go_core_otel_trace 	"github.com/eliezerraj/go-core/v2/otel/trace"

	"github.com/go-worker-event/internal/domain/model"
	"github.com/go-worker-event/internal/domain/service"

	"go.opentelemetry.io/otel"
)

var appTracerProvider 	go_core_otel_trace.TracerProvider

type EventAppServer struct {
	appServer		*model.AppServer
	consumerWorker 	*go_core_event.ConsumerWorker
	workerService	*service.WorkerService
	logger			*zerolog.Logger
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

	ctx = context.WithValue(ctx, "trace-request-id",  trace_id  )
	return ctx
}

// About create a consumer worker event
func NewEventAppServer(	appServer *model.AppServer,
						workerService *service.WorkerService,
						appLogger *zerolog.Logger) (*EventAppServer, error) {
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
	}, nil
}

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

	go e.consumerWorker.Consumer(e.appServer.Topics, messages)
	var tracerProvider 	go_core_otel_trace.TracerProvider

	for msg := range messages {

		e.logger.Info().Msg("=============== BEGIN - MSG FROM KAFKA - BEGIN ==================")
		e.logger.Info().Interface("msg", msg).Send()
		e.logger.Info().Msg("=============== END - MSG FROM KAFKA - END ==================")

		kafkaHeaderCarrier := go_core_otel_trace.KafkaHeaderCarrier{}
		kafkaHeaders := kafkaHeaderCarrier.MapToKafkaHeaders(*msg.Header)
		appCarrier := go_core_otel_trace.KafkaHeaderCarrier{Headers: &kafkaHeaders }
		ctx := otel.GetTextMapPropagator().Extract(ctx, appCarrier)

		ctx, span := tracerProvider.SpanCtx(ctx, 
								  			e.appServer.Application.Name)

		// Decode payload
		event := model.Event{}
		json.Unmarshal([]byte(msg.Payload), &event)

		// call service
		var err error
		if event.Type == "cleareance.order" {
			err = e.workerService.Clearance(ctx,)
			if err != nil {
				e.logger.Error().
						Ctx(ctx).
						Err(err).Send()
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

/*
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

func MapToKafkaHeaders(m map[string]string) []kafka.Header {
    hdrs := make([]kafka.Header, 0, len(m))
    for k, v := range m {
        hdrs = append(hdrs, kafka.Header{
            Key:   k,
            Value: []byte(v),
        })
    }
    return hdrs
}
	*/