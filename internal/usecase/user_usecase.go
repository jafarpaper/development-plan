package usecase

type User struct {
    ID    string `json:"_key,omitempty"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

type UserRepository interface {
    GetByID(ctx context.Context, id string) (*User, error)
}

type UserUsecase struct {
    Repo UserRepository
}

func (u *UserUsecase) GetUserByID(ctx context.Context, id string) (*User, error) {
    return u.Repo.GetByID(ctx, id)
}
