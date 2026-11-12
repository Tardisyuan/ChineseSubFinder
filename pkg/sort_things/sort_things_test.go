package sort_things

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestSortByModTime(t *testing.T) {
	dir := t.TempDir()
	base := time.Now().Add(-time.Hour)
	var in []string
	for i, name := range []string{"old", "new", "mid"} {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
		mt := base.Add(time.Duration([]int{0, 2, 1}[i]) * time.Minute)
		if err := os.Chtimes(p, mt, mt); err != nil {
			t.Fatal(err)
		}
		in = append(in, p)
	}
	got := SortByModTime(in)
	want := []string{in[1], in[2], in[0]} // newest first
	if !reflect.DeepEqual(got, want) {
		t.Errorf("SortByModTime() = %v, want %v", got, want)
	}
	if in[0] != filepath.Join(dir, "old") {
		t.Errorf("input slice was modified")
	}
}

func TestSortStringSliceByLength(t *testing.T) {
	type args struct {
		m []string
	}
	tests := []struct {
		name string
		args args
		want PathSlices
	}{
		{name: "00", args: args{m: []string{"1", "12", "123"}}, want: PathSlices{
			PathSlice{"123"},
			PathSlice{"12"},
			PathSlice{"1"},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SortStringSliceByLength(tt.args.m); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SortStringSliceByLength() = %v, want %v", got, tt.want)
			}
		})
	}
}
