package valueobject

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewActivityLogID(t *testing.T) {
	id := NewActivityLogID()
	assert.NotEmpty(t, id)
	assert.True(t, id.IsValid())
}

func TestActivityLogID_String(t *testing.T) {
	id := NewActivityLogID()
	str := id.String()
	assert.NotEmpty(t, str)
	assert.Equal(t, string(id), str)
}

func TestActivityLogID_IsValid(t *testing.T) {
	tests := []struct {
		name string
		id   ActivityLogID
		want bool
	}{
		{
			name: "valid id",
			id:   ActivityLogID("valid-id"),
			want: true,
		},
		{
			name: "empty id",
			id:   ActivityLogID(""),
			want: false,
		},
		{
			name: "whitespace only id",
			id:   ActivityLogID("   "),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.id.IsValid()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGenerateID(t *testing.T) {
	id1 := generateID()
	id2 := generateID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
	assert.Len(t, id1, 32)
	assert.Len(t, id2, 32)
}
