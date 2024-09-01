package bstream

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"testing"
)

func TestMap(t *testing.T) {

	ctx := context.Background()

	err := Stream[string](ctx,
		func(errs chan error) error {
			stream := ArrayToStream[string]([]string{"1", "2", "3", "4"})
			ints := ConcurrentMap(ctx, stream, 2, StringToInteger, errs)
			for v := range ints {
				println(v)
			}
			return nil
		},
	)
	if err != nil {
		t.Error(err)
	}
}

func TestFilter(t *testing.T) {
	ctx := context.Background()

	err := Stream[string](ctx,
		func(errs chan error) error {
			stream := ArrayToStream[int]([]int{1, 2, 3, 4})
			streams := Clone[int](ctx, stream, 2, errs)

			wg := sync.WaitGroup{}
			wg.Add(2)
			//deal with stream 1
			go func() {
				stream := streams[0]
				filteredInts := Filter(ctx, stream, OnlyAllowEvens, errs)
				for v := range filteredInts {
					fmt.Printf("stream 1: %d\n", v)
				}
				wg.Done()
			}()

			//deal with stream 2
			go func() {
				stream := streams[1]
				ss := Map(ctx, stream, IntegerToString, errs)
				for v := range ss {
					fmt.Printf("stream 2: %s\n", v)
				}
				wg.Done()
			}()

			wg.Wait()

			return nil
		},
	)
	if err != nil {
		t.Error(err)
	}
}

func StringToInteger(str string) (int, error) {
	return strconv.Atoi(str)
}

func IntegerToString(i int) (string, error) {
	return strconv.FormatInt(int64(i), 10), nil
}

func OnlyAllowEvens(v int) (bool, error) {
	if v%2 == 0 {
		return true, nil
	}
	return false, nil
}
