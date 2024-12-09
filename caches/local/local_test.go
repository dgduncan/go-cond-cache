//go:build !integration

package local

import (
	"reflect"
	"testing"
)

func TestNewBasicCache(t *testing.T) {
	tests := []struct {
		name string
		want BasicCache
	}{
		{},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewBasicCache(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewBasicCache() = %v, want %v", got, tt.want)
			}
		})
	}
}
