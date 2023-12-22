package repo

import (
	"context"
	"reflect"
	"testing"
)

func Test_CommitFromContext(t *testing.T) {
	cases := map[string]struct {
		ctx  context.Context
		want Commit
	}{
		"default": {
			ctx:  context.TODO(),
			want: headRef,
		},
		"with commit": {
			ctx:  WithCommit(context.TODO(), "test"),
			want: "test",
		},
	}
	for name := range cases {
		tc := cases[name]
		t.Run(name, func(t *testing.T) {
			if got := CommitFromContext(tc.ctx); got != tc.want {
				t.Fatalf("want: %s, got: %s", got, tc.want)
			}
		})
	}
}

func Test_DiffFromContext(t *testing.T) {
	cases := map[string]struct {
		ctx  context.Context
		want DiffOption
	}{
		"default": {
			ctx:  context.TODO(),
			want: DiffOption{},
		},
		"with from to": {
			ctx:  WithDiff(context.TODO(), "src", "dst"),
			want: DiffOption{From: "src", To: "dst"},
		},
	}
	for name := range cases {
		tc := cases[name]
		t.Run(name, func(t *testing.T) {
			if got := DiffFromContext(tc.ctx); !reflect.DeepEqual(tc.want, got) {
				t.Fatalf("want: %v, got: %v", got, tc.want)
			}
		})
	}
}
