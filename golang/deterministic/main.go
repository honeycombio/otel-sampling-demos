 package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
	"math/rand"

	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/exporters/stdout"

	httptrace "go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	honeycombsamplers "github.com/honeycombio/opentelemetry-samplers-go/honeycombsamplers"

)

func main() {
	exp, err := stdout.NewExporter([]stdout.Option{
		stdout.WithQuantiles([]float64{0.5, 0.9, 0.99}),
		stdout.WithPrettyPrint(),
	}...)
	if err != nil {
		log.Fatal(err)
	}
	sample, err := honeycombsamplers.DeterministicSampler(2)
	if err != nil {
		log.Fatal(err)
	}
	config := sdktrace.Config{
		DefaultSampler: sample,
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithConfig(config), sdktrace.WithSyncer(exp))

	global.SetTracerProvider(tp)
	tracer := global.Tracer("sampling-service")
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		attrs, _, spanCtx := httptrace.Extract(ctx, r)
		ctx, span := tracer.Start(
			trace.ContextWithRemoteSpanContext(ctx, spanCtx),
			"parent-span",
			trace.WithAttributes(attrs...),
		)
		upper := rand.Intn(10)

		for i := 0; i < upper; i++ {
			_, iSpan := tracer.Start(ctx, fmt.Sprintf("Sample-%d", i))
			log.Printf("Doing really hard work (%d / %d)\n", i + 1, upper)
			<-time.After(time.Duration(rand.Intn(1000)) * time.Millisecond)
			iSpan.End()
		}

		defer span.End()
		fmt.Fprintf(w, "Hello, World")
	})

	log.Fatal(http.ListenAndServe(":8000", mux))
}
