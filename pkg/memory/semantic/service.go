package semantic

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type Service struct {
	store  Store
	tracer trace.Tracer
	meter  metric.Meter
}

type ServiceOption func(*Service)

func WithTracer(tracer trace.Tracer) ServiceOption {
	return func(s *Service) { s.tracer = tracer }
}

func WithMeter(meter metric.Meter) ServiceOption {
	return func(s *Service) { s.meter = meter }
}

func NewService(store Store, opts ...ServiceOption) *Service {
	s := &Service{store: store}
	for _, opt := range opts {
		opt(s)
	}
	if s.tracer == nil {
		s.tracer = otel.GetTracerProvider().Tracer("github.com/andrewhowdencom/dux/pkg/memory/semantic")
	}
	if s.meter == nil {
		s.meter = otel.GetMeterProvider().Meter("github.com/andrewhowdencom/dux/pkg/memory/semantic")
	}
	return s
}

func (s *Service) TrackAccess(ctx context.Context, factID string) error {
	ctx, span := s.tracer.Start(ctx, "semantic.TrackAccess",
		trace.WithSpanKind(trace.SpanKindInternal),
	)
	defer span.End()

	span.SetAttributes(attribute.String("fact.id", factID))

	fact, err := s.store.ReadFact(ctx, factID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("failed to read fact for access tracking: %w", err)
	}

	metadata := fact.GetMetadata()
	now := time.Now()

	daysSinceLastAccess := now.Sub(metadata.LastAccessed).Hours() / 24
	metadata.AccessScore += 1.0 / max(daysSinceLastAccess, 1.0)
	metadata.AccessCount++
	metadata.LastAccessed = now

	switch f := fact.(type) {
	case TripleFact:
		f.Metadata = metadata
		err = s.store.WriteTriple(ctx, f)
	case StatementFact:
		f.Metadata = metadata
		err = s.store.WriteStatement(ctx, f)
	}

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("failed to update access stats: %w", err)
	}

	return nil
}

func (s *Service) ValidateFact(ctx context.Context, factID string) error {
	ctx, span := s.tracer.Start(ctx, "semantic.ValidateFact",
		trace.WithSpanKind(trace.SpanKindInternal),
	)
	defer span.End()

	span.SetAttributes(attribute.String("fact.id", factID))

	fact, err := s.store.ReadFact(ctx, factID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("failed to read fact for validation: %w", err)
	}

	metadata := fact.GetMetadata()
	metadata.ValidatedAt = time.Now()

	switch f := fact.(type) {
	case TripleFact:
		f.Metadata = metadata
		err = s.store.WriteTriple(ctx, f)
	case StatementFact:
		f.Metadata = metadata
		err = s.store.WriteStatement(ctx, f)
	}

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("failed to update validated_at: %w", err)
	}

	return nil
}

func (s *Service) CalculateConfidence(metadata FactMetadata, sourceCount int) float64 {
	now := time.Now()

	accessComponent := min(metadata.AccessScore/10.0, 1.0)

	daysSinceValidated := now.Sub(metadata.ValidatedAt).Hours() / 24
	recencyComponent := max(0, 1.0-(daysSinceValidated/365.0))

	sourceComponent := min(float64(sourceCount)/3.0, 1.0)

	return 0.4*accessComponent + 0.4*recencyComponent + 0.2*sourceComponent
}

func (s *Service) WriteTriple(ctx context.Context, fact TripleFact) error {
	return s.store.WriteTriple(ctx, fact)
}

func (s *Service) WriteStatement(ctx context.Context, fact StatementFact) error {
	return s.store.WriteStatement(ctx, fact)
}

func (s *Service) ReadFact(ctx context.Context, id string) (Fact, error) {
	return s.store.ReadFact(ctx, id)
}

func (s *Service) Search(ctx context.Context, query SearchQuery) ([]Fact, error) {
	return s.store.Search(ctx, query)
}

func (s *Service) DeleteFact(ctx context.Context, id string) error {
	return s.store.DeleteFact(ctx, id)
}
