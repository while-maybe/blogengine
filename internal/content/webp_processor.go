package content

import (
	"blogengine/internal/storage"
	"bytes"
	"context"
	"fmt"
	"image"
	"io"
	"log/slog"
	"sync"

	"github.com/kolesa-team/go-webp/encoder"
	"github.com/kolesa-team/go-webp/webp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/image/draw"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

type ImageJob struct {
	SourcePath string
	ID         string
	Width      int
	ParentSpan trace.SpanContext
}

type Processor struct {
	jobs     chan ImageJob
	wg       sync.WaitGroup
	logger   *slog.Logger
	inFlight sync.Map
	store    storage.Provider
	tracer   trace.Tracer
}

var _ ImageProcessorService = (*Processor)(nil)

func NewProcessor(ctx context.Context, store storage.Provider, sourcesDir string, workercount int, logger *slog.Logger) (*Processor, error) {
	p := &Processor{
		jobs:   make(chan ImageJob, 25),
		logger: logger,
		store:  store,
		tracer: otel.Tracer("blogengine/content/processor"),
	}
	for i := range workercount {
		p.wg.Go(func() {
			p.worker(ctx, i)
		})
	}

	go func() {
		<-ctx.Done()
		p.logger.Info("image processor received shutdown signal")
		close(p.jobs)
		p.wg.Wait()
		p.logger.Info("image processor shutdown complete")
	}()

	return p, nil
}

func (p *Processor) worker(ctx context.Context, id int) {
	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-p.jobs:
			if !ok {
				return
			}
			key := fmt.Sprintf("%s_%d.webp", job.ID, job.Width)

			p.ProcessJob(ctx, id, job)
			p.inFlight.Delete(key)
		}
	}
}

func (p *Processor) ProcessJob(ctx context.Context, id int, job ImageJob) {
	link := trace.Link{
		SpanContext: job.ParentSpan,
	}

	ctx, span := p.tracer.Start(ctx, "ProcessJob",
		trace.WithAttributes(
			attribute.String("image.id", job.ID),
			attribute.Int("image.width", job.Width),
		),
		trace.WithLinks(link),
	)
	defer span.End()

	destKey := fmt.Sprintf("%s_%d.webp", job.ID, job.Width)

	p.logger.Info("worker processing image variants", "worker_id", id, "uuid", job.ID, "variant", job.Width)

	// any other worker has done this?
	if p.store.Exists(ctx, destKey) {
		return
	}

	if ctx.Err() != nil {
		return
	}

	reader, err := p.store.Open(ctx, job.SourcePath)
	if err != nil {
		p.logger.Error("failed to download source", "key", job.SourcePath, "err", err)
		return
	}
	defer reader.Close()

	_, cpuSpan := p.tracer.Start(ctx, "GenerateVariant.CPU")
	processedBuffer, err := p.generateVariant(ctx, reader, job.Width)
	cpuSpan.End()
	if err != nil {
		p.logger.Error("variant failed", "worker", id, "variant", job.Width, "err", err)
		return
	}

	// finally save to bucket here
	if err := p.store.Save(ctx, destKey, processedBuffer); err != nil {
		p.logger.Error("failed to upload variant", "key", destKey, "err", err)
	}
}

func (p *Processor) generateVariant(ctx context.Context, r io.Reader, width int) (io.ReadSeeker, error) {
	img, _, err := image.Decode(r)
	if err != nil {
		return nil, fmt.Errorf("decode error: %w", err)
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	if img.Bounds().Dx() > width {
		img = p.resizeImage(img, width)
	}

	var buf bytes.Buffer
	options, err := encoder.NewLossyEncoderOptions(encoder.PresetDefault, 75)
	if err != nil {
		return nil, fmt.Errorf("encoding options: %w", err)
	}

	if err := webp.Encode(&buf, img, options); err != nil {
		return nil, fmt.Errorf("encode error: %w", err)
	}

	return bytes.NewReader(buf.Bytes()), nil
}

func (p *Processor) Enqueue(ctx context.Context, job ImageJob) error {
	key := fmt.Sprintf("%s_%d", job.ID, job.Width)

	// no duplicated jobs
	if _, loaded := p.inFlight.LoadOrStore(key, struct{}{}); loaded {
		return nil
	}

	select {
	case <-ctx.Done():
		// should a caller's request timeout
		p.inFlight.Delete(key)
		return ctx.Err()
	case p.jobs <- job:
		return nil
	default:
		p.inFlight.Delete(key)
		return fmt.Errorf("image processor queue full")
	}
}

func (p *Processor) resizeImage(source image.Image, maxWidth int) image.Image {
	b := source.Bounds()
	currentWidth := b.Dx()

	// ensure scales down only
	if currentWidth <= maxWidth {
		return source
	}

	newHeight := (b.Dy() * maxWidth) / currentWidth

	dest := image.NewRGBA(image.Rect(0, 0, maxWidth, newHeight))

	// bilinear has a good quality / speed tradeoff
	draw.BiLinear.Scale(dest, dest.Bounds(), source, source.Bounds(), draw.Over, nil)

	return dest
}
