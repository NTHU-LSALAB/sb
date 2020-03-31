package intrange

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSimpleValue(t *testing.T) {
	assert.Equal(t, MustRange("48"), []string{"48"})
}

func TestBasicRange(t *testing.T) {
	assert.Equal(t, MustRange("7-9"), []string{"7", "8", "9"})
}

func TestFormatRange(t *testing.T) {
	assert.Equal(t, MustRange("7-11"), []string{"7", "8", "9", "10", "11"})
	assert.Equal(t, MustRange("07-11"), []string{"07", "08", "09", "10", "11"})
	assert.Equal(t, MustRange("007-11"), []string{"007", "008", "009", "010", "011"})
}

func TestMultipleRanges(t *testing.T) {
	assert.Equal(t, MustRange("1,3-4"), []string{"1", "3", "4"})
	assert.Equal(t, MustRange("1,2,3,4"), []string{"1", "2", "3", "4"})
	assert.Equal(t, MustRange("1-3,7-11"), []string{"1", "2", "3", "7", "8", "9", "10", "11"})
	assert.Equal(t, MustRange("01-3,7-11"), []string{"01", "02", "03", "7", "8", "9", "10", "11"})
}
