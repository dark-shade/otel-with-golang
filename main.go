package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/gorilla/mux"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/propagation"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

const (
	serviceName      = "hello-app"
	serviceVersion   = "1.0"
	metricPrefix     = "custom.metric."
	numberOfExecName = metricPrefix + "number.of.exec"
	numberOfExecDesc = "Count the number of executions."
	heapMemoryName   = metricPrefix + "heap.memory"
	heapMemoryDesc   = "Reports heap memory utilization."
)

var (
	tracer             trace.Tracer
	meter              metric.Meter
	numberOfExecutions metric.BoundInt64Counter
)

func main() {

	ctx := context.Background()
	endpoint := os.Getenv("EXPORTER_ENDPOINT")

	// Resource for traces and metrics
	res0urce, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String(serviceVersion),
		),
	)
	if err != nil {
		log.Fatalf("%s: %v", "failed to create resource", err)
	}

	// Setup the tracing
	traceExporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithDialOption(grpc.WithBlock()),
	)
	if err != nil {
		log.Fatalf("%s: %v", "failed to create exporter", err)
	}

	otel.SetTracerProvider(sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res0urce),
		sdktrace.WithSpanProcessor(
			sdktrace.NewBatchSpanProcessor(traceExporter)),
	))

	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.Baggage{},
			propagation.TraceContext{},
		),
	)

	// Setup the metrics
	metricExporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithInsecure(),
		otlpmetricgrpc.WithEndpoint(endpoint),
		otlpmetricgrpc.WithDialOption(grpc.WithBlock()),
	)
	if err != nil {
		log.Fatalf("%s: %v", "failed to create exporter", err)
	}

	pusher := controller.New(
		processor.New(
			simple.NewWithExactDistribution(),
			metricExporter,
		),
		controller.WithResource(res0urce),
		controller.WithExporter(metricExporter),
		controller.WithCollectPeriod(5*time.Second),
	)
	err = pusher.Start(ctx)
	if err != nil {
		log.Fatalf("%s: %v", "failed to start the controller", err)
	}
	defer func() { _ = pusher.Stop(ctx) }()

	global.SetMeterProvider(pusher.MeterProvider())

	// Support for programatic traces and metrics
	tracer = otel.Tracer("io.opentelemetry.traces.hello")
	meter = global.Meter("io.opentelemetry.metrics.hello")

	// Metric that is updated manually
	numberOfExecutions = metric.Must(meter).
		NewInt64Counter(
			numberOfExecName,
			metric.WithDescription(numberOfExecDesc),
		).Bind(
		[]attribute.KeyValue{
			attribute.String(
				numberOfExecName,
				numberOfExecDesc)}...)

	// Metric that updates automatically
	_ = metric.Must(meter).
		NewInt64CounterObserver(
			heapMemoryName,
			func(_ context.Context, result metric.Int64ObserverResult) {
				var mem runtime.MemStats
				runtime.ReadMemStats(&mem)
				result.Observe(int64(mem.HeapAlloc),
					attribute.String(heapMemoryName,
						heapMemoryDesc))
			},
			metric.WithDescription(heapMemoryDesc))

	// Start the API with instrumentation
	router := mux.NewRouter()
	router.Use(otelmux.Middleware(serviceName))
	router.HandleFunc("/hello", hello)
	http.ListenAndServe(":8888", router)

}

func hello(writer http.ResponseWriter, request *http.Request) {

	ctx := request.Context()

	ctx, buildResp := tracer.Start(ctx, "buildResponse")
	response := buildResponse(writer)
	buildResp.End()

	// Creating a custom span just for fun...
	_, mySpan := tracer.Start(ctx, "mySpan")
	if response.isValid() {
		log.Print("The response is valid")
	}
	mySpan.End()

	// Updating the number of executions metric...
	numberOfExecutions.Add(ctx, 1)

}

func buildResponse(writer http.ResponseWriter) Response {

	writer.WriteHeader(http.StatusOK)
	writer.Header().Add("Content-Type",
		"application/json")

	response := Response{"Hello World"}
	bytes, _ := json.Marshal(response)
	writer.Write(bytes)
	return response

}

// Response struct
type Response struct {
	Message string `json:"Message"`
}

func (r Response) isValid() bool {
	return true
}
