package trace

import (
	"context"
	"github.com/wsx864321/kim/pkg/log"
	"sync"

	"github.com/wsx864321/kim/pkg/krpc/config"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

var (
	tp   *tracesdk.TracerProvider
	once sync.Once
)

// StartAgent 开启trace collector
func StartAgent() {
	once.Do(func() {
		exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(config.GetTraceCollectionUrl())))
		if err != nil {
			log.Info(context.Background(), "trace start agent err", log.String("err", err.Error()))
			return
		}

		tp = tracesdk.NewTracerProvider(
			tracesdk.WithSampler(tracesdk.TraceIDRatioBased(config.GetTraceSampler())),
			tracesdk.WithBatcher(exp),
			tracesdk.WithResource(resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String(config.GetTraceServiceName()),
			)),
		)

		otel.SetTracerProvider(tp)
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{}, propagation.Baggage{}))
	})
}

// StopAgent 关闭trace collector,在服务停止时调用StopAgent，不然可能造成trace数据的丢失
func StopAgent() {
	_ = tp.Shutdown(context.TODO())
}
