package bstream

import (
	"context"

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

func StreamFlow[T any](ctx context.Context, f func(errs chan error) error) error {

	errs := make(chan error)
	eg := errgroup.Group{}

	eg.Go(func() error {
		return ErrorLoop(ctx, errs)
	})
	eg.Go(func() error {
		err := f(errs)
		close(errs)
		return err
	})
	return eg.Wait()
}
