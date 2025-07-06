package valueobject

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewActor(t *testing.T) {
	tests := []struct {
		name      string
		id        string
		actorName string
		email     string
		wantErr   error
	}{
		{
			name:      "valid actor",
			id:        "actor1",
			actorName: "John Doe",
			email:     "john@example.com",
			wantErr:   nil,
		},
		{
			name:      "empty id",
			id:        "",
			actorName: "John Doe",
			email:     "john@example.com",
			wantErr:   ErrInvalidActorID,
		},
		{
			name:      "empty name",
			id:        "actor1",
			actorName: "",
			email:     "john@example.com",
			wantErr:   ErrInvalidActorName,
		},
		{
			name:      "invalid email",
			id:        "actor1",
			actorName: "John Doe",
			email:     "invalid-email",
			wantErr:   ErrInvalidActorEmail,
		},
		{
			name:      "empty email",
			id:        "actor1",
			actorName: "John Doe",
			email:     "",
			wantErr:   ErrInvalidActorEmail,
		},
		{
			name:      "whitespace trimmed",
			id:        "  actor1  ",
			actorName: "  John Doe  ",
			email:     "  john@example.com  ",
			wantErr:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actor, err := NewActor(tt.id, tt.actorName, tt.email)

			if tt.wantErr != nil {
				assert.Equal(t, tt.wantErr, err)
				assert.Equal(t, Actor{}, actor)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "actor1", actor.ID)
				assert.Equal(t, "John Doe", actor.Name)
				assert.Equal(t, "john@example.com", actor.Email)
			}
		})
	}
}

func TestActor_IsValid(t *testing.T) {
	tests := []struct {
		name    string
		actor   Actor
		wantErr error
	}{
		{
			name: "valid actor",
			actor: Actor{
				ID:    "actor1",
				Name:  "John Doe",
				Email: "john@example.com",
			},
			wantErr: nil,
		},
		{
			name: "empty id",
			actor: Actor{
				ID:    "",
				Name:  "John Doe",
				Email: "john@example.com",
			},
			wantErr: ErrInvalidActorID,
		},
		{
			name: "empty name",
			actor: Actor{
				ID:    "actor1",
				Name:  "",
				Email: "john@example.com",
			},
			wantErr: ErrInvalidActorName,
		},
		{
			name: "invalid email",
			actor: Actor{
				ID:    "actor1",
				Name:  "John Doe",
				Email: "invalid-email",
			},
			wantErr: ErrInvalidActorEmail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.actor.IsValid()
			if tt.wantErr != nil {
				assert.Equal(t, tt.wantErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  bool
	}{
		{
			name:  "valid email",
			email: "john@example.com",
			want:  true,
		},
		{
			name:  "valid email with subdomain",
			email: "john@mail.example.com",
			want:  true,
		},
		{
			name:  "valid email with plus",
			email: "john+test@example.com",
			want:  true,
		},
		{
			name:  "invalid email without @",
			email: "johnexample.com",
			want:  false,
		},
		{
			name:  "invalid email without domain",
			email: "john@",
			want:  false,
		},
		{
			name:  "invalid email without local part",
			email: "@example.com",
			want:  false,
		},
		{
			name:  "empty email",
			email: "",
			want:  false,
		},
		{
			name:  "invalid email with spaces",
			email: "john doe@example.com",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidEmail(tt.email)
			assert.Equal(t, tt.want, got)
		})
	}
}
