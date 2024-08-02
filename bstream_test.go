package bstream

import (
	"context"
	"strconv"
	"testing"
)

func TestMap(t *testing.T) {

	ctx := context.Background()

	err := StreamFlow[string](ctx,
		func(errs chan error) error {
			stream := ArrayToStream[string]([]string{"1", "2", "3", "4"})
			ints := Map(context.Background(), stream, StringToInteger, errs)
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

	err := StreamFlow[string](ctx,
		func(errs chan error) error {
			stream := ArrayToStream[string]([]string{"1", "2", "3", "4"})
			ints := Map(ctx, stream, StringToInteger, errs)
			filteredInts := Filter(ctx, ints, OnlyAllowEvens, errs)
			for v := range filteredInts {
				println(v)
			}
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

func OnlyAllowEvens(v int) (bool, error) {
	if v%2 == 0 {
		return true, nil
	}
	return false, nil
}
