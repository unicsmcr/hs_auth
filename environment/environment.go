package environment

import (
	"os"

	"go.uber.org/zap"
)

// names of env vars
const (
	Environment   = "ENVIRONMENT"
	Port          = "PORT"
	MongoHost     = "MONGO_HOST"
	MongoDatabase = "MONGO_DATABASE"
	MongoUser     = "MONGO_USER"
	MongoPassword = "MONGO_PASSWORD"
	JWTSecret     = "JWT_SECRET"
)

// NewEnv creates an Env with loaded environment variables
func NewEnv(logger *zap.Logger) *Env {
	env := Env{}
	env.vars[Environment] = valueOfEnvVar(logger, Environment)
	env.vars[Port] = valueOfEnvVar(logger, Port)
	env.vars[MongoHost] = valueOfEnvVar(logger, MongoHost)
	env.vars[MongoDatabase] = valueOfEnvVar(logger, MongoDatabase)
	env.vars[MongoUser] = valueOfEnvVar(logger, MongoUser)
	env.vars[JWTSecret] = valueOfEnvVar(logger, JWTSecret)
	return &env
}

// Env is a struct to store environment variables in an immutable collection
type Env struct {
	vars map[string]string
}

// Get returns an environment variable with the specified name
func (env *Env) Get(variableName string) string {
	return env.vars[variableName]
}

func valueOfEnvVar(logger *zap.Logger, varName string) string {
	envVar := os.Getenv(varName)
	if len(envVar) == 0 {
		logger.Warn("expected environment variable not defined", zap.String("var", varName))
	}

	return envVar
}
