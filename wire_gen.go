// Code generated by Wire. DO NOT EDIT.

//go:generate wire
//+build !wireinject

package main

import (
	"github.com/unicsmcr/hs_auth/authorization/v2"
	"github.com/unicsmcr/hs_auth/config"
	"github.com/unicsmcr/hs_auth/environment"
	"github.com/unicsmcr/hs_auth/repositories"
	"github.com/unicsmcr/hs_auth/routers"
	"github.com/unicsmcr/hs_auth/routers/api/v1"
	v2_2 "github.com/unicsmcr/hs_auth/routers/api/v2"
	"github.com/unicsmcr/hs_auth/routers/frontend"
	"github.com/unicsmcr/hs_auth/services/mongo"
	"github.com/unicsmcr/hs_auth/services/sendgrid"
	"github.com/unicsmcr/hs_auth/utils"
)

// Injectors from wire.go:

func InitializeServer() (Server, error) {
	logger, err := utils.NewLogger()
	if err != nil {
		return Server{}, err
	}
	env := environment.NewEnv(logger)
	appConfig, err := config.NewAppConfig(env)
	if err != nil {
		return Server{}, err
	}
	database, err := utils.NewDatabase(logger, env)
	if err != nil {
		return Server{}, err
	}
	userRepository, err := repositories.NewUserRepository(database)
	if err != nil {
		return Server{}, err
	}
	userService := mongo.NewMongoUserService(logger, env, appConfig, userRepository)
	client := utils.NewSendgridClient(env)
	emailService, err := sendgrid.NewSendgridEmailService(logger, appConfig, env, client, userService)
	if err != nil {
		return Server{}, err
	}
	teamRepository, err := repositories.NewTeamRepository(database)
	if err != nil {
		return Server{}, err
	}
	teamService := mongo.NewMongoTeamService(logger, env, teamRepository, userService)
	apiv1Router := v1.NewAPIV1Router(logger, appConfig, env, userService, emailService, teamService)
	timeProvider := utils.NewTimeProvider()
	authorizer := v2.NewAuthorizer(timeProvider, env)
	apiv2Router := v2_2.NewAPIV2Router(logger, authorizer)
	router := frontend.NewRouter(logger, appConfig, env, userService, teamService, emailService)
	mainRouter := routers.NewMainRouter(logger, apiv1Router, apiv2Router, router)
	server := NewServer(mainRouter, env)
	return server, nil
}
