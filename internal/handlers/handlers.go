package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	huma "github.com/danielgtaylor/huma/v2"
)

type contextKey string

// Context keys
const (
	PoolKey   = contextKey("dbPool")
	KeyGenKey = contextKey("keyGen")
)

// Error responses
var (
	ErrPoolNotFound   = errors.New("database connection pool not found in context")
	ErrKeyGenNotFound = errors.New("key generator not found in context")
)

// The type definitions and functions that follow are used to
// mock the crypto/rand.Read function for testing purposes.
type RandomKeyGenerator interface {
	RandomKey(len int) (key string, err error)
}

type StandardKeyGen struct{}

func (s StandardKeyGen) RandomKey(len int) (string, error) {
	b := make([]byte, len)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// AddRoutes adds all the routes to the API
func AddRoutes(pool *pgxpool.Pool, keyGen RandomKeyGenerator, api huma.API) error {
	err := RegisterManifestRoutes(pool, api)
	if err != nil {
		fmt.Printf("    Unable to register Manifest routes: %v\n", err)
		return err
	}
	err = RegisterUsersRoutes(pool, keyGen, api)
	if err != nil {
		fmt.Printf("    Unable to register Users routes: %v\n", err)
		return err
	}
	err = RegisterProjectsRoutes(pool, api)
	if err != nil {
		fmt.Printf("    Unable to register Projects routes: %v\n", err)
		return err
	}
	err = RegisterEmbeddingsRoutes(pool, api)
	if err != nil {
		fmt.Printf("    Unable to register Embeddings routes: %v\n", err)
		return err
	}
	err = RegisterAPIStandardsRoutes(pool, api)
	if err != nil {
		fmt.Printf("    Unable to register API standards routes: %v\n", err)
		return err
	}
	err = RegisterInstancesRoutes(pool, api)
	if err != nil {
		fmt.Printf("    Unable to register Instances routes: %v\n", err)
		return err
	}
	err = RegisterDefinitionsRoutes(pool, api)
	if err != nil {
		fmt.Printf("    Unable to register Definitions routes: %v\n", err)
		return err
	}
	err = RegisterSimilarRoutes(pool, api)
	if err != nil {
		fmt.Printf("    Unable to register Similar routes: %v\n", err)
		return err
	}
	err = RegisterAdminRoutes(pool, api)
	if err != nil {
		fmt.Printf("    Unable to register Admin routes: %v\n", err)
		return err
	}
	return nil
}

// Middleware to add the connection pool to the context
func addPoolToContext[I any, O any](pool *pgxpool.Pool, next func(context.Context, *I) (*O, error)) func(context.Context, *I) (*O, error) {
	return func(ctx context.Context, input *I) (*O, error) {
		if pool == nil {
			return nil, fmt.Errorf("provided pool is nil")
		}
		ctx = context.WithValue(ctx, PoolKey, pool)
		return next(ctx, input)
	}
}

// Middleware to add the key generator to the context
func addKeyGenToContext[I any, O any](keyGen RandomKeyGenerator, next func(context.Context, *I) (*O, error)) func(context.Context, *I) (*O, error) {
	return func(ctx context.Context, input *I) (*O, error) {
		if keyGen == nil {
			return nil, fmt.Errorf("provided keyGen is nil")
		}
		ctx = context.WithValue(ctx, KeyGenKey, keyGen)
		return next(ctx, input)
	}
}

// Get the database connection pool from the context
// (exported helper function so that blackbox testing can access it)
func GetDBPool(ctx context.Context) (*pgxpool.Pool, error) {
	pool, ok := ctx.Value(PoolKey).(*pgxpool.Pool)
	if !ok {
		return nil, huma.NewError(http.StatusInternalServerError, ErrPoolNotFound.Error())
	}
	return pool, nil
}

// Get the key generator from the context
// (exported helper function so that blackbox testing can access it)
func GetKeyGen(ctx context.Context) (RandomKeyGenerator, error) {
	keyGen, ok := ctx.Value(KeyGenKey).(RandomKeyGenerator)
	if !ok {
		return nil, huma.NewError(http.StatusInternalServerError, ErrKeyGenNotFound.Error())
	}
	return keyGen, nil
}
