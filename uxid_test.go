package uxid

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGenerateExample(t *testing.T) {
	t.Parallel()

	t.Run("UXID generate with prefix cus and size small", func(t *testing.T) {
		got, err := Generate("cus", "small")
		assert.Nil(t, err)
		assert.NotNil(t, got)
		assert.Equal(t, len(got), 18)
	})

	t.Run("UXID generate with prefix txn and size xl", func(t *testing.T) {
		got, err := Generate("txn", "xl")
		assert.Nil(t, err)
		assert.NotNil(t, got)
		assert.Equal(t, len(got), 30)
	})

}

func TestGenerateTShirtSizes(t *testing.T) {
	t.Parallel()

	t.Run("UXID generate with size xs", func(t *testing.T) {
		got, err := Generate("", "xs")
		assert.Nil(t, err)
		assert.NotNil(t, got)
		assert.Equal(t, len(got), 10)
	})

	t.Run("UXID generate with size xsmall", func(t *testing.T) {
		got, err := Generate("", "xsmall")
		assert.Nil(t, err)
		assert.NotNil(t, got)
		assert.Equal(t, len(got), 10)
	})

	t.Run("UXID generate with size s", func(t *testing.T) {
		got, err := Generate("", "s")
		assert.Nil(t, err)
		assert.NotNil(t, got)
		assert.Equal(t, len(got), 14)
	})

	t.Run("UXID generate with size small", func(t *testing.T) {
		got, err := Generate("", "small")
		assert.Nil(t, err)
		assert.NotNil(t, got)
		assert.Equal(t, len(got), 14)
	})

}

func TestGenerateRandSizes(t *testing.T) {
	t.Parallel()

	t.Run("UXID generate with size 0", func(t *testing.T) {
		got, err := Generate("", "0")
		assert.Nil(t, err)
		assert.NotNil(t, got)
		assert.Equal(t, len(got), 10)
	})

	t.Run("UXID generate with size 1", func(t *testing.T) {
		got, err := Generate("", "1")
		assert.Nil(t, err)
		assert.NotNil(t, got)
		assert.Equal(t, len(got), 12)
	})

	t.Run("UXID generate with size 2", func(t *testing.T) {
		got, err := Generate("", "2")
		assert.Nil(t, err)
		assert.NotNil(t, got)
		assert.Equal(t, len(got), 14)
	})

	t.Run("UXID generate with size 3", func(t *testing.T) {
		got, err := Generate("", "3")
		assert.Nil(t, err)
		assert.NotNil(t, got)
		assert.Equal(t, len(got), 15)
	})

	t.Run("UXID generate with size 4", func(t *testing.T) {
		got, err := Generate("", "4")
		assert.Nil(t, err)
		assert.NotNil(t, got)
		assert.Equal(t, len(got), 17)
	})

	t.Run("UXID generate with size 5", func(t *testing.T) {
		got, err := Generate("", "5")
		assert.Nil(t, err)
		assert.NotNil(t, got)
		assert.Equal(t, len(got), 18)
	})

	t.Run("UXID generate with size 6", func(t *testing.T) {
		got, err := Generate("", "6")
		assert.Nil(t, err)
		assert.NotNil(t, got)
		assert.Equal(t, len(got), 20)
	})

	t.Run("UXID generate with size 7", func(t *testing.T) {
		got, err := Generate("", "7")
		assert.Nil(t, err)
		assert.NotNil(t, got)
		assert.Equal(t, len(got), 22)
	})

	t.Run("UXID generate with size 8", func(t *testing.T) {
		got, err := Generate("", "8")
		assert.Nil(t, err)
		assert.NotNil(t, got)
		assert.Equal(t, len(got), 23)
	})

	t.Run("UXID generate with size 9", func(t *testing.T) {
		got, err := Generate("", "9")
		assert.Nil(t, err)
		assert.NotNil(t, got)
		assert.Equal(t, len(got), 25)
	})

	t.Run("UXID generate with size 10", func(t *testing.T) {
		got, err := Generate("", "10")
		assert.Nil(t, err)
		assert.NotNil(t, got)
		assert.Equal(t, len(got), 26)
	})

}
