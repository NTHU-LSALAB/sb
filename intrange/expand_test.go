package intrange

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpandNothing(t *testing.T) {
	assert.Equal(t, MustExpand("hello"), []string{"hello"})
}

func TestExpandEmptyRange(t *testing.T) {
	assert.Equal(t, MustExpand("hello[]world"), []string{"helloworld"})
}

func TestExpandSingleBasicRange(t *testing.T) {
	assert.Equal(t, MustExpand("hello[10]world"), []string{"hello10world"})
}

func TestExpandSingleRange(t *testing.T) {
	assert.Equal(t, MustExpand("hello[3-7]world"), []string{"hello3world", "hello4world", "hello5world", "hello6world", "hello7world"})
	assert.Equal(t, MustExpand("hello[7-11]world"), []string{"hello7world", "hello8world", "hello9world", "hello10world", "hello11world"})
	assert.Equal(t, MustExpand("hello[07-11]world"), []string{"hello07world", "hello08world", "hello09world", "hello10world", "hello11world"})
}

func TestExpandMultipeRanges(t *testing.T) {
	assert.Equal(t, MustExpand("[1]hello[2]world[3]"), []string{"1hello2world3"})
	assert.Equal(t, MustExpand("[1-2]hello[3,5]world[9-11]"), []string{
		"1hello3world9",
		"1hello3world10",
		"1hello3world11",
		"1hello5world9",
		"1hello5world10",
		"1hello5world11",
		"2hello3world9",
		"2hello3world10",
		"2hello3world11",
		"2hello5world9",
		"2hello5world10",
		"2hello5world11",
	})
}
