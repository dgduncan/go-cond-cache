//go:build !integration

package local_test

import (
	"reflect"
	"testing"

	local "github.com/dgduncan/go-cond-cache/caches/local"
)

func TestNewBasicCache(t *testing.T) {
	tests := []struct {
		name string
		want local.BasicCache
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := local.NewBasicCache(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewBasicCache() = %v, want %v", got, tt.want)
			}
		})
	}
}
