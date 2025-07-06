package valueobject

import (
	"errors"
	"regexp"
	"strings"
)

var (
	ErrInvalidActorID    = errors.New("invalid actor id")
	ErrInvalidActorName  = errors.New("invalid actor name")
	ErrInvalidActorEmail = errors.New("invalid actor email")
)

type Actor struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func NewActor(id, name, email string) (Actor, error) {
	actor := Actor{
		ID:    strings.TrimSpace(id),
		Name:  strings.TrimSpace(name),
		Email: strings.TrimSpace(email),
	}

	if err := actor.IsValid(); err != nil {
		return Actor{}, err
	}

	return actor, nil
}

func (a Actor) IsValid() error {
	if a.ID == "" {
		return ErrInvalidActorID
	}
	if a.Name == "" {
		return ErrInvalidActorName
	}
	if !isValidEmail(a.Email) {
		return ErrInvalidActorEmail
	}
	return nil
}

func isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}
