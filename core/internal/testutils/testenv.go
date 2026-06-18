package testutils

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/moby/moby/api/types/network"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type testContainer interface {
	Host(context.Context) (string, error)
	MappedPort(context.Context, string) (network.Port, error)
	Terminate(context.Context, ...testcontainers.TerminateOption) error
}

var startContainer = func(ctx context.Context, image string, opts ...testcontainers.ContainerCustomizer) (testContainer, error) {
	return testcontainers.Run(ctx, image, opts...)
}

type Environment struct {
	RedisAddr string
	PGHost    string
	PGPort    string
	PGDB      string
	PGUser    string
	PGPass    string
	RedisPass string
	RedisDB   int

	pgContainer    testContainer
	redisContainer testContainer
}

var (
	instance *Environment
	once     sync.Once
)

// Start initializes the test environment with PostgreSQL and Redis containers.
func Start(ctx context.Context) error {
	var initErr error
	once.Do(func() {
		pgEnv := map[string]string{
			"POSTGRES_DB":       "testdb",
			"POSTGRES_USER":     "testuser",
			"POSTGRES_PASSWORD": "testpass",
		}

		redisContainer, err := startContainer(
			ctx,
			"redis:7.4-alpine",
			testcontainers.WithExposedPorts("6379/tcp"),
			testcontainers.WithWaitStrategy(
				wait.ForLog("Ready to accept connections"),
				wait.ForListeningPort("6379/tcp"),
			),
		)
		if err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				fmt.Println("Start: Redis container startup deadline exceeded")
			}
			initErr = fmt.Errorf("failed to start Redis container: %w", err)
			return
		}

		pgContainer, err := startContainer(
			ctx,
			"postgres:16-alpine",
			testcontainers.WithExposedPorts("5432/tcp"),
			testcontainers.WithEnv(pgEnv),
			testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections"),
				wait.ForListeningPort("5432/tcp"),
			),
		)
		if err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				fmt.Println("Start: PostgreSQL container startup deadline exceeded")
			}
			initErr = fmt.Errorf("failed to start PostgreSQL container: %w", err)
			_ = redisContainer.Terminate(context.Background())
			return
		}

		pgHost, err := pgContainer.Host(ctx)
		if err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				fmt.Println("Start: PostgreSQL host resolution deadline exceeded")
			}
			initErr = fmt.Errorf("failed to resolve PostgreSQL host: %w", err)
			_ = pgContainer.Terminate(context.Background())
			_ = redisContainer.Terminate(context.Background())
			return
		}

		pgMappedPort, err := pgContainer.MappedPort(ctx, "5432/tcp")
		if err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				fmt.Println("Start: PostgreSQL port resolution deadline exceeded")
			}
			initErr = fmt.Errorf("failed to resolve PostgreSQL mapped port: %w", err)
			_ = pgContainer.Terminate(context.Background())
			_ = redisContainer.Terminate(context.Background())
			return
		}

		redisHost, err := redisContainer.Host(ctx)
		if err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				fmt.Println("Start: Redis host resolution deadline exceeded")
			}
			initErr = fmt.Errorf("failed to resolve Redis host: %w", err)
			_ = pgContainer.Terminate(context.Background())
			_ = redisContainer.Terminate(context.Background())
			return
		}

		redisMappedPort, err := redisContainer.MappedPort(ctx, "6379/tcp")
		if err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				fmt.Println("Start: Redis port resolution deadline exceeded")
			}
			initErr = fmt.Errorf("failed to resolve Redis mapped port: %w", err)
			_ = pgContainer.Terminate(context.Background())
			_ = redisContainer.Terminate(context.Background())
			return
		}

		pgPort := pgMappedPort.Port()
		redisAddr := fmt.Sprintf("%s:%s", redisHost, redisMappedPort.Port())

		instance = &Environment{
			RedisAddr:      redisAddr,
			PGHost:         pgHost,
			PGPort:         pgPort,
			PGDB:           "testdb",
			PGUser:         "testuser",
			PGPass:         "testpass",
			RedisPass:      "",
			RedisDB:        0,
			pgContainer:    pgContainer,
			redisContainer: redisContainer,
		}
	})
	return initErr
}

// Get returns the initialized test environment instance.
func Get() *Environment {
	return instance
}

// Stop stops and removes all containers.
func Stop(ctx context.Context) error {
	if instance == nil {
		return nil
	}

	var errs []error
	if err := instance.pgContainer.Terminate(ctx); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			fmt.Println("Stop: PostgreSQL container termination deadline exceeded")
		}
		errs = append(errs, fmt.Errorf("failed to terminate PostgreSQL container: %w", err))
	}
	if err := instance.redisContainer.Terminate(ctx); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			fmt.Println("Stop: Redis container termination deadline exceeded")
		}
		errs = append(errs, fmt.Errorf("failed to terminate Redis container: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("stop errors: %v", errs)
	}

	instance = nil
	once = sync.Once{}
	return nil
}

// SetEnv sets OS-level environment variables based on the environment configuration.
func SetEnv() error {
	if instance == nil {
		return fmt.Errorf("environment not initialized, call Start() first")
	}
	os.Setenv("PG_HOST", instance.PGHost)
	os.Setenv("PG_PORT", instance.PGPort)
	os.Setenv("PG_DB", instance.PGDB)
	os.Setenv("PG_USER", instance.PGUser)
	os.Setenv("PG_PASSWORD", instance.PGPass)
	os.Setenv("REDIS_ADDRESS", instance.RedisAddr)
	os.Setenv("REDIS_PASSWORD", instance.RedisPass)
	os.Setenv("REDIS_DB", strconv.Itoa(instance.RedisDB))

	return nil
}
