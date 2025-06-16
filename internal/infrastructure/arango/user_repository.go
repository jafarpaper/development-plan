package arango

import (
    "context"
    "go-clean-arango/internal/usecase"

    driver "github.com/arangodb/go-driver"
)

type ArangoUserRepo struct {
    col driver.Collection
}

func NewArangoUserRepo(db driver.Database) *ArangoUserRepo {
    col, _ := db.Collection(context.Background(), "users")
    return &ArangoUserRepo{col: col}
}

func (repo *ArangoUserRepo) GetByID(ctx context.Context, id string) (*usecase.User, error) {
    var user usecase.User
    _, err := repo.col.ReadDocument(ctx, id, &user)
    if err != nil {
        return nil, err
    }
    return &user, nil
}
