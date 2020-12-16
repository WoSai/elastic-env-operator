package util

import (
	"gotest.tools/assert"
	"testing"
)

func TestMergeStringMap(t *testing.T) {
	assert.Equal(t, MergeStringMap(nil, map[string]string{"a": "1"})["a"], "1")
}

func TestMergeStringMap_1(t *testing.T) {
	assert.Equal(t, MergeStringMap(map[string]string{"a": "1"}, nil)["a"], "1")
}
