package repo

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTRMonthShort(t *testing.T) {
	assert.Equal(t, "Oca", trMonthShort(time.January))
	assert.Equal(t, "Şub", trMonthShort(time.February))
	assert.Equal(t, "Ağu", trMonthShort(time.August))
	assert.Equal(t, "Ara", trMonthShort(time.December))
	assert.Equal(t, "", trMonthShort(time.Month(0)))
	assert.Equal(t, "", trMonthShort(time.Month(13)))
}
