package testutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"go-auth-boilerplate/internal/database"
	"go-auth-boilerplate/internal/models"
	"go-auth-boilerplate/internal/routes"

	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	containerName = "go-auth-boilerplate-test"
	networkName   = "test-network"
)

var (
	testInstance *TestInstance
	setupOnce    sync.Once
	setupError   error
)

type TestInstance struct {
	pool     *dockertest.Pool
	network  *dockertest.Network
	DB       *gorm.DB
	RedisURL string
	resource *dockertest.Resource
}

func removeContainerIfExists(pool *dockertest.Pool, containerName string) error {
	if container, err := pool.Client.InspectContainer(containerName); err == nil {
		if container.State.Running {
			if err := pool.Client.StopContainer(containerName, 10); err != nil {
				return fmt.Errorf("could not stop container %s: %v", containerName, err)
			}
		}
		removeOptions := docker.RemoveContainerOptions{
			ID:            containerName,
			Force:         true,
			RemoveVolumes: true,
		}
		if err := pool.Client.RemoveContainer(removeOptions); err != nil {
			return fmt.Errorf("could not remove container %s: %v", containerName, err)
		}
	}
	return nil
}

func removeNetworkIfExists(pool *dockertest.Pool, networkName string) error {
	networks, err := pool.Client.ListNetworks()
	if err != nil {
		return fmt.Errorf("could not list networks: %v", err)
	}

	for _, network := range networks {
		if network.Name == networkName {
			containers, err := pool.Client.ListContainers(docker.ListContainersOptions{
				All: true,
				Filters: map[string][]string{
					"network": {networkName},
				},
			})
			if err != nil {
				return fmt.Errorf("could not list containers in network: %v", err)
			}

			for _, container := range containers {
				if err := pool.Client.DisconnectNetwork(network.ID, docker.NetworkConnectionOptions{
					Container: container.ID,
					Force:     true,
				}); err != nil {
					return fmt.Errorf("could not disconnect container from network: %v", err)
				}
			}

			if err := pool.Client.RemoveNetwork(network.ID); err != nil {
				return fmt.Errorf("could not remove network %s: %v", networkName, err)
			}
			break
		}
	}
	return nil
}

func NewTestInstance(t *testing.T) *TestInstance {
	setupOnce.Do(func() {
		testInstance, setupError = setupTestInfrastructure(t)
	})
	require.NoError(t, setupError)
	return testInstance
}

func setupTestInfrastructure(t *testing.T) (*TestInstance, error) {
	if os.Getenv("DB_USER") == "" {
		os.Setenv("DB_USER", "postgres")
	}
	if os.Getenv("DB_PASSWORD") == "" {
		os.Setenv("DB_PASSWORD", "postgres")
	}
	if os.Getenv("DB_NAME") == "" {
		os.Setenv("DB_NAME", "testdb")
	}
	if os.Getenv("DB_PORT") == "" {
		os.Setenv("DB_PORT", "5432")
	}
	if os.Getenv("REDIS_PORT") == "" {
		os.Setenv("REDIS_PORT", "6379")
	}

	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, fmt.Errorf("could not connect to docker: %v", err)
	}

	pool.MaxWait = 60 * time.Second

	if err := removeContainerIfExists(pool, containerName); err != nil {
		return nil, err
	}

	if err := removeNetworkIfExists(pool, networkName); err != nil {
		return nil, err
	}

	network, err := pool.CreateNetwork(networkName)
	if err != nil {
		return nil, fmt.Errorf("could not create network: %v", err)
	}

	instance := &TestInstance{
		pool:    pool,
		network: network,
	}

	postgresContainer, err := pool.RunWithOptions(&dockertest.RunOptions{
		Name:       "go_auth_boilerplate_postgres",
		Repository: "postgres",
		Tag:        "16-alpine",
		NetworkID:  network.Network.ID,
		Env: []string{
			fmt.Sprintf("POSTGRES_USER=%s", os.Getenv("DB_USER")),
			fmt.Sprintf("POSTGRES_PASSWORD=%s", os.Getenv("DB_PASSWORD")),
			fmt.Sprintf("POSTGRES_DB=%s", os.Getenv("DB_NAME")),
		},
		ExposedPorts: []string{"5432/tcp"},
		PortBindings: map[docker.Port][]docker.PortBinding{
			"5432/tcp": {{HostIP: "0.0.0.0", HostPort: os.Getenv("DB_PORT")}},
		},
	})
	if err != nil {
		network.Close()
		return nil, fmt.Errorf("could not start postgres container: %v", err)
	}

	redisContainer, err := pool.RunWithOptions(&dockertest.RunOptions{
		Name:         "go_auth_boilerplate_redis",
		Repository:   "redis",
		Tag:          "7-alpine",
		NetworkID:    network.Network.ID,
		Cmd:          []string{"redis-server", "--appendonly", "yes"},
		ExposedPorts: []string{"6379/tcp"},
		PortBindings: map[docker.Port][]docker.PortBinding{
			"6379/tcp": {{HostIP: "0.0.0.0", HostPort: os.Getenv("REDIS_PORT")}},
		},
	})
	if err != nil {
		postgresContainer.Close()
		network.Close()
		return nil, fmt.Errorf("could not start redis container: %v", err)
	}

	instance.resource = postgresContainer

	instance.RedisURL = fmt.Sprintf("redis://localhost:%s", os.Getenv("REDIS_PORT"))

	if err := pool.Retry(func() error {
		exitCode, err := postgresContainer.Exec(
			[]string{"pg_isready", "-U", os.Getenv("DB_USER")},
			dockertest.ExecOptions{},
		)
		if err != nil || exitCode != 0 {
			return fmt.Errorf("postgres is not ready: %v, exit code: %d", err, exitCode)
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("postgres failed to become ready: %v", err)
	}

	if err := pool.Retry(func() error {
		exitCode, err := redisContainer.Exec(
			[]string{"redis-cli", "ping"},
			dockertest.ExecOptions{},
		)
		if err != nil || exitCode != 0 {
			return fmt.Errorf("redis is not ready: %v, exit code: %d", err, exitCode)
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("redis failed to become ready: %v", err)
	}

	dsn := fmt.Sprintf("host=localhost port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
	)

	var db *gorm.DB
	if err := pool.Retry(func() error {
		var err error
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			return err
		}
		sqlDB, err := db.DB()
		if err != nil {
			return err
		}
		return sqlDB.Ping()
	}); err != nil {
		return nil, fmt.Errorf("could not connect to postgres: %v", err)
	}

	if err := db.AutoMigrate(&models.User{}, &models.Post{}); err != nil {
		return nil, fmt.Errorf("could not migrate database: %v", err)
	}

	instance.DB = db

	redisClient := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("localhost:%s", os.Getenv("REDIS_PORT")),
	})
	if err := pool.Retry(func() error {
		return redisClient.Ping(redisClient.Context()).Err()
	}); err != nil {
		return nil, fmt.Errorf("could not connect to redis: %v", err)
	}
	redisClient.Close()

	return instance, nil
}

type TestServer struct {
	App   *fiber.App
	DB    *gorm.DB
	Redis string
}

func SetupRouter(db *gorm.DB, redisURL string) *fiber.App {
	app := fiber.New()
	routes.SetupRoutes(app, db, redisURL)
	return app
}

func NewTestServer(t *testing.T) *TestServer {
	os.Setenv("JWT_SECRET", "test_secret")

	instance := NewTestInstance(t)

	opt, err := redis.ParseURL(instance.RedisURL)
	require.NoError(t, err)
	database.RedisClient = redis.NewClient(opt)

	app := SetupRouter(instance.DB, instance.RedisURL)

	return &TestServer{
		App:   app,
		DB:    instance.DB,
		Redis: instance.RedisURL,
	}
}

func (ts *TestServer) Close(t *testing.T) {
	ts.DB.Exec("TRUNCATE users, posts CASCADE")
}

type TestResponse struct {
	*http.Response
	Body []byte
}

func (ts *TestServer) SendRequest(t *testing.T, method, path string, body interface{}, headers map[string]string) *TestResponse {
	var reqBody []byte
	var err error
	if body != nil {
		reqBody, err = json.Marshal(body)
		require.NoError(t, err)
	}

	req := httptest.NewRequest(method, path, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := ts.App.Test(req)
	require.NoError(t, err)

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()

	return &TestResponse{
		Response: &http.Response{
			StatusCode: resp.StatusCode,
			Header:     resp.Header,
		},
		Body: respBody,
	}
}

func (tr *TestResponse) DecodeBody(v interface{}) error {
	return json.Unmarshal(tr.Body, v)
}
