package testenv

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type Environment struct {
	RedisAddr string
	PGHost    string
	PGPort    string
	PGDB      string
	PGUser    string
	PGPass    string
	RedisPass string
	RedisDB   int

	pgContainer    testcontainers.Container
	redisContainer testcontainers.Container
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

		pgContainer, err := testcontainers.Run(
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
			return
		}

		redisContainer, err := testcontainers.Run(
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
			_ = pgContainer.Terminate(context.Background())
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

	if err := os.Setenv("PG_HOST", instance.PGHost); err != nil {
		return fmt.Errorf("failed to set PG_HOST: %w", err)
	}
	if err := os.Setenv("PG_PORT", instance.PGPort); err != nil {
		return fmt.Errorf("failed to set PG_PORT: %w", err)
	}
	if err := os.Setenv("PG_DB", instance.PGDB); err != nil {
		return fmt.Errorf("failed to set PG_DB: %w", err)
	}
	if err := os.Setenv("PG_USER", instance.PGUser); err != nil {
		return fmt.Errorf("failed to set PG_USER: %w", err)
	}
	if err := os.Setenv("PG_PASSWORD", instance.PGPass); err != nil {
		return fmt.Errorf("failed to set PG_PASSWORD: %w", err)
	}
	if err := os.Setenv("REDIS_ADDRESS", instance.RedisAddr); err != nil {
		return fmt.Errorf("failed to set REDIS_ADDRESS: %w", err)
	}
	if err := os.Setenv("REDIS_PASSWORD", instance.RedisPass); err != nil {
		return fmt.Errorf("failed to set REDIS_PASSWORD: %w", err)
	}
	if err := os.Setenv("REDIS_DB", strconv.Itoa(instance.RedisDB)); err != nil {
		return fmt.Errorf("failed to set REDIS_DB: %w", err)
	}

	return nil
}
