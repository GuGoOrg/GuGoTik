package main

import (
	"GuGoTik/src/extra/tracing"
	"GuGoTik/src/models"
	"GuGoTik/src/utils/logging"
	"context"
	"sync"
)

func SummaryVideo(ctx context.Context, video models.RawVideo, wg *sync.WaitGroup, out chan<- []string) {
	defer wg.Done()

	ctx, span := tracing.Tracer.Start(ctx, "VideoSummaryService")
	defer span.End()
	logger := logging.LogService("VideoSummaryService").WithContext(ctx)
	logger.Debugf("Process start")

	// TODO: speech-to-text, summary, keywords

	return
}

func speech2Text() {

}

func text2Summary() {

}

func text2Keywords() {

}
