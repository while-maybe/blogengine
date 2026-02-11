package content

import (
	"context"
	"errors"
	"fmt"
	"image"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/kolesa-team/go-webp/encoder"
	"github.com/kolesa-team/go-webp/webp"
	"golang.org/x/image/draw"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

type ImageJob struct {
	SourcePath string
	ID         string
	Width      int
}

type Processor struct {
	jobs     chan ImageJob
	wg       sync.WaitGroup
	logger   *slog.Logger
	root     *os.Root
	inFlight sync.Map
}

const defaultFolderPermissions = 0755

var _ ImageProcessorService = (*Processor)(nil)

func NewProcessor(ctx context.Context, sourcesDir string, workercount int, logger *slog.Logger) (*Processor, error) {
	root, err := os.OpenRoot(sourcesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to open secure root: %w", err)
	}

	p := &Processor{
		jobs:   make(chan ImageJob, 25),
		logger: logger,
		root:   root,
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

			p.inFlight.Store(key, struct{}{})
			p.ProcessJob(ctx, id, job)
			p.inFlight.Delete(key)
		}
	}
}

func (p *Processor) ProcessJob(ctx context.Context, id int, job ImageJob) {
	destName := fmt.Sprintf("%s_%d.webp", job.ID, job.Width)
	destPath := filepath.Join("data", "cache", destName)

	p.logger.Info("worker processing image variants", "worker_id", id, "uuid", job.ID, "variant", job.Width)

	// any other worker has done this?
	if _, err := os.Stat(destPath); !errors.Is(err, os.ErrNotExist) {
		return
	}

	if ctx.Err() != nil {
		return
	}

	sourceFile, err := p.root.Open(job.SourcePath)
	if err != nil {
		p.logger.Error("failed to open source", "worker_id", id, "source", job.SourcePath, "err", err)
		return
	}
	defer sourceFile.Close()

	if err := p.generateVariant(ctx, sourceFile, destPath, job.Width); err != nil {
		p.logger.Error("variant failed", "worker", id, "variant", job.Width, "err", err)

		if err := os.Remove(destPath); err != nil {
			p.logger.Error("remove corrupt file", "path", destPath, "err", err)
		}
	}
}

func (p *Processor) generateVariant(ctx context.Context, r io.Reader, dest string, width int) error {
	img, _, err := image.Decode(r)
	if err != nil {
		return fmt.Errorf("decode error: %w", err)
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	if img.Bounds().Dx() > width {
		img = p.resizeImage(img, width)
	}

	destinationDir := filepath.Dir(dest)
	if err := os.MkdirAll(destinationDir, defaultFolderPermissions); err != nil {
		return fmt.Errorf("could not create directory %q, %w", destinationDir, err)
	}

	f, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("could not create file %q: %w", dest, err)
	}
	defer f.Close()

	options, err := encoder.NewLossyEncoderOptions(encoder.PresetDefault, 75)
	if err != nil {
		return fmt.Errorf("could not encode image: %w", err)
	}

	return webp.Encode(f, img, options)
}

func (p *Processor) Enqueue(ctx context.Context, job ImageJob) error {
	key := fmt.Sprintf("%s_%d", job.ID, job.Width)

	if _, loaded := p.inFlight.LoadOrStore(key, struct{}{}); loaded {
		return nil
	}

	select {
	case <-ctx.Done():
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
