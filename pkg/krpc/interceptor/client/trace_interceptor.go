package client

import (
	"context"

	ktrace "github.com/wsx864321/kim/pkg/krpc/trace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	gcodes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// TraceUnaryClientInterceptor trace interceptor
func TraceUnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.MD{}
		}

		tr := otel.GetTracerProvider().Tracer(ktrace.TraceName)
		name, attrs := ktrace.BuildSpan(method, "")
		ctx, span := tr.Start(ctx, name, trace.WithAttributes(attrs...), trace.WithSpanKind(trace.SpanKindClient))
		defer span.End()

		ktrace.Inject(ctx, otel.GetTextMapPropagator(), &md)
		ctx = metadata.NewOutgoingContext(ctx, md)

		err := invoker(ctx, method, req, reply, cc, opts...)
		if err != nil {
			s, ok := status.FromError(err)
			if ok {
				span.SetStatus(codes.Error, s.Message())
				span.SetAttributes(ktrace.StatusCodeAttr(s.Code()))
			} else {
				span.SetStatus(codes.Error, err.Error())
			}
			return err
		}

		span.SetAttributes(ktrace.StatusCodeAttr(gcodes.OK))
		return nil
	}
}
