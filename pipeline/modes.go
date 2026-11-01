package pipeline

import (
	"runtime"
	"sync"
)

// runSequential processes one image at a time, all four stages, in the
// caller's goroutine: the "without goroutines" baseline.
func runSequential(s stages, names []string, collect func(job)) {
	for index, name := range names {
		collect(s.process(newJob(index, name)))
	}
}

// runPipeline is CODEHEIM's design: one goroutine per stage, joined by
// unbuffered channels. While one image is being resized the next can be
// loading, so up to four images are in flight. Throughput is limited by the
// slowest stage (Spike 001).
func runPipeline(s stages, names []string, collect func(job)) {
	loaded := s.loadAll(names)
	resized := startStage(loaded, s.resize)
	grayed := startStage(resized, s.grayscale)
	saved := startStage(grayed, s.save)
	for finished := range saved {
		collect(finished)
	}
}

// loadAll is the pipeline's first stage: a goroutine that loads each named
// image in turn and sends it on, closing its channel after the last one.
func (s stages) loadAll(names []string) <-chan job {
	loaded := make(chan job)
	go func() {
		defer close(loaded)
		for index, name := range names {
			loaded <- s.load(newJob(index, name))
		}
	}()
	return loaded
}

// startStage starts a goroutine that applies step to every job from input and
// sends the result on. It closes its output when input is closed, so closing
// spreads down the pipeline and every stage goroutine ends.
func startStage(input <-chan job, step func(job) job) <-chan job {
	output := make(chan job)
	go func() {
		defer close(output)
		for j := range input {
			output <- step(j)
		}
	}()
	return output
}

// runWorkers starts a pool of goroutines; each takes the next image and runs
// all four stages on it. Unlike the pipeline, its speed-up is not capped by
// the slowest stage: it grows with the number of workers until the CPUs (or
// memory and disk) are saturated. Images finish in any order; the index in
// each job lets Run put the results back in input order.
func runWorkers(s stages, names []string, collect func(job)) {
	pending := make(chan job)
	finished := make(chan job)
	var running sync.WaitGroup
	for range s.workerCount() {
		running.Add(1)
		go func() {
			defer running.Done()
			for j := range pending {
				finished <- s.process(j)
			}
		}()
	}
	go func() { // feed the pool, then tell the workers there is no more work
		defer close(pending)
		for index, name := range names {
			pending <- newJob(index, name)
		}
	}()
	go func() { // close finished only after the last worker has sent its job
		running.Wait()
		close(finished)
	}()
	for j := range finished {
		collect(j)
	}
}

// workerCount is Config.Workers, or one worker per CPU when it is below 1.
func (s stages) workerCount() int {
	if s.cfg.Workers < 1 {
		return runtime.NumCPU()
	}
	return s.cfg.Workers
}
