package bstream

import (
	"context"
	"errors"
	"sync"

	"golang.org/x/sync/errgroup"
)

func ArrayToStream[T any](values []T) chan T {
	out := make(chan T)
	go func() {
		for _, v := range values {
			out <- v
		}
		close(out)
	}()
	return out
}

func Map[I any, O any](
	ctx context.Context,
	in chan I,
	f func(i I) (O, error),
	errs chan error,
) chan O {

	out := make(chan O)

	go func() {
		for {
			select {
			case v, ok := <-in:
				if ok {
					v, err := f(v)
					if err != nil {
						errs <- err
						close(out)
						return
					}
					out <- v
				} else {
					close(out)
					return
				}
			case <-ctx.Done():
				close(out)
				return
			}
		}
	}()
	return out
}

func Filter[I any](
	ctx context.Context,
	in chan I,
	f func(i I) (bool, error),
	errs chan error,
) chan I {
	out := make(chan I)
	go func() {
		for {
			select {
			case v, ok := <-in:
				if ok {
					good, err := f(v)
					if err != nil {
						errs <- err
						close(out)
						return
					}
					if good {
						out <- v
					}
				} else {
					close(out)
					return
				}
			case <-ctx.Done():
				close(out)
				return
			}
		}
	}()
	return out
}

func ErrorLoop(ctx context.Context, errs chan error) error {
	var e error
	var done bool
	for {
		select {
		case <-ctx.Done():
			done = true
		case err, ok := <-errs:
			if ok {
				e = err
			}
			done = true
		}
		if done {
			break
		}
	}
	return e
}

func Stream[T any](ctx context.Context, f func(errs chan error) error) error {

	errs := make(chan error)
	eg := errgroup.Group{}

	ctx, cancel := context.WithCancel(ctx)

	eg.Go(func() error {
		err := ErrorLoop(ctx, errs)
		if err != nil {
			cancel()
		}
		return err
	})
	eg.Go(func() error {
		err := f(errs)
		close(errs)
		return err
	})
	return eg.Wait()
}

func FanOut[T any](ctx context.Context, in chan T, concurrency int, errs chan error) []chan T {

	if concurrency <= 0 {
		errs <- errors.New("concurrency must be greater than zero")
		return []chan T{}
	}

	streams := make([]chan T, concurrency)
	for index, _ := range streams {
		streams[index] = make(chan T)
	}
	go func() {
		streamIndex := 0
		for {
			select {
			case v, ok := <-in:
				if ok {
					streams[streamIndex] <- v
					streamIndex++
					if streamIndex >= concurrency {
						streamIndex = concurrency - 1
					}
				} else {
					//input stream closed
					for _, stream := range streams {
						close(stream)
					}
					return
				}
			case <-ctx.Done():
				for _, stream := range streams {
					close(stream)
				}
				return
			}
		}
	}()
	return streams
}

func FanIn[T any](ctx context.Context, streams []chan T, errs chan error) chan T {
	out := make(chan T)
	go func() {
		wg := sync.WaitGroup{}
		for _, stream := range streams {
			wg.Add(1)
			go func() {
				for {
					select {
					case v, ok := <-stream:
						if ok {
							out <- v
						} else {
							wg.Done()
							return
						}
					case <-ctx.Done():
						wg.Done()
						return
					}
				}
			}()
		}
		wg.Wait()
		close(out)
	}()
	return out
}

func ConcurrentMap[I, O any](ctx context.Context, in chan I, concurrency int,
	f func(i I) (O, error), errs chan error,
) chan O {
	iStreams := FanOut(ctx, in, 2, errs)
	oStreams := make([]chan O, len(iStreams))
	for index, stream := range iStreams {
		oStreams[index] = Map(ctx, stream, f, errs)
	}
	out := FanIn(ctx, oStreams, errs)
	return out
}
