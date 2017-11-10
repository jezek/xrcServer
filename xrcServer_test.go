package main

import (
	"fmt"
	"reflect"
	"testing"
)

func TestRun(t *testing.T) {
	testCases := []struct {
		runners []runner
		out     map[int]error
	}{
		{},
		{
			func() []runner {
				return []runner{}
			}(),
			nil,
		},
		{
			func() []runner {
				return []runner{
					{
						func() error {
							return nil
						},
						func() error {
							return nil
						},
					},
				}
			}(),
			map[int]error{},
		},
		{
			func() []runner {
				count := make(chan int, 2)
				for i := 0; i < 2; i++ {
					count <- i
				}
				return []runner{
					{
						func() error {
							return fmt.Errorf("%d run", <-count)
						},
						func() error {
							return fmt.Errorf("%d stop", <-count)
						},
					},
				}
			}(),
			map[int]error{0: fmt.Errorf("0 run")},
		},
		{
			func() []runner {
				r1ch := make(chan struct{})
				return []runner{
					{
						func() error {
							return fmt.Errorf("run")
						},
						func() error {
							return fmt.Errorf("stop")
						},
					},
					{
						func() error {
							<-r1ch
							return fmt.Errorf("run")
						},
						func() error {
							defer close(r1ch)
							return nil
						},
					},
				}
			}(),
			map[int]error{
				0: fmt.Errorf("run"),
				1: runStopErr{fmt.Errorf("run")},
			},
		},
		{
			func() []runner {
				r1ch := make(chan struct{})
				r2ch := make(chan struct{})
				return []runner{
					{
						func() error {
							return fmt.Errorf("run")
						},
						func() error {
							return fmt.Errorf("stop")
						},
					},
					{
						func() error {
							<-r1ch
							return fmt.Errorf("run")
						},
						func() error {
							defer close(r1ch)
							return nil
						},
					},
					{
						func() error {
							<-r2ch
							return fmt.Errorf("run")
						},
						func() error {
							defer close(r2ch)
							return fmt.Errorf("stop")
						},
					},
				}
			}(),
			map[int]error{
				0: fmt.Errorf("run"),
				1: runStopErr{fmt.Errorf("run")},
				2: fmt.Errorf("stop"),
			},
		},
	}

	for _, tc := range testCases {
		out := run(tc.runners...)
		if !reflect.DeepEqual(out, tc.out) {
			t.Errorf("run(%#v) = %#v, want %#v", tc.runners, out, tc.out)
		}
	}
}
