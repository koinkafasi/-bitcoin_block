package repo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClampTierWeight(t *testing.T) {
	assert.Equal(t, 0, clampTierWeight(-5))
	assert.Equal(t, 0, clampTierWeight(0))
	assert.Equal(t, 3, clampTierWeight(3))
	assert.Equal(t, 5, clampTierWeight(5))
	assert.Equal(t, 5, clampTierWeight(99))
}
