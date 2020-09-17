package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
	"math/rand"

	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	sampler "github.com/honeycombio/opentelemetry-sampler-go/honeycomb"

	"github.com/honeycombio/opentelemetry-exporter-go/honeycomb"

)

func main() {
	exp, err := honeycomb.NewExporter(
		honeycomb.Config{
			APIKey: os.Getenv("HONEYCOMB_WRITE_KEY"),
		},
		honeycomb.TargetingDataset(os.Getenv("HONEYCOMB_DATASET")),
		honeycomb.WithServiceName("sampling-service-otel-go"),
	)
	if err != nil {
		log.Fatal(err)
	}
	sample, err := sampler.NewDeterministicSampler(5)
	if err != nil {
		log.Fatal(err)
	}
	config := sdktrace.Config{
		DefaultSampler: sample,
	}
	tp, err := sdktrace.NewProvider(sdktrace.WithConfig(config), sdktrace.WithSyncer(exp))
	if err != nil {
		log.Fatal(err)
	}
	global.SetTraceProvider(tp)
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
