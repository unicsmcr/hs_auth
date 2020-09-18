package v2

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	v2 "github.com/unicsmcr/hs_auth/authorization/v2"
	"github.com/unicsmcr/hs_auth/entities"
	"github.com/unicsmcr/hs_auth/routers/api/models"
	"github.com/unicsmcr/hs_auth/services"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

// POST: /api/v2/users/login
// x-www-form-urlencoded
// Request:  email string
//           password string
// Response: token string
// Headers:  Authorization <- token
func (r *apiV2Router) Login(ctx *gin.Context) {
	var req struct {
		Email    string `form:"email"`
		Password string `form:"password"`
	}
	ctx.Bind(&req)

	if len(req.Email) == 0 {
		r.logger.Debug("email was not provided")
		models.SendAPIError(ctx, http.StatusBadRequest, "email must be provided")
		return
	}

	if len(req.Password) == 0 {
		r.logger.Debug("password was not provided")
		models.SendAPIError(ctx, http.StatusBadRequest, "password must be provided")
		return
	}

	user, err := r.userService.GetUserWithEmailAndPwd(ctx, req.Email, req.Password)
	if err != nil {
		switch errors.Cause(err) {
		case services.ErrNotFound:
			r.logger.Debug("user not found", zap.String("email", req.Email), zap.Error(err))
			models.SendAPIError(ctx, http.StatusUnauthorized, "user not found")
		default:
			r.logger.Error("could not fetch user", zap.Error(err))
			models.SendAPIError(ctx, http.StatusInternalServerError, "something went wrong")
		}
		return
	}

	token, err := r.authorizer.CreateUserToken(user.ID, r.cfg.AuthTokenLifetime+r.timeProvider.Now().Unix())
	if err != nil {
		r.logger.Error("could not create JWT", zap.Error(err))
		models.SendAPIError(ctx, http.StatusInternalServerError, "something went wrong")
		return
	}

	ctx.Header(authTokenHeader, token)
	ctx.JSON(http.StatusOK, loginRes{
		Token: token,
	})
}

// POST: /api/v2/users
// x-www-form-urlencoded
// Request:  name string
//           email string
//           password string
// Response:
func (r *apiV2Router) Register(ctx *gin.Context) {
	var req struct {
		Name     string `form:"name"`
		Email    string `form:"email"`
		Password string `form:"password"`
	}
	ctx.Bind(&req)

	if len(req.Name) == 0 || len(req.Email) == 0 || len(req.Password) == 0 {
		r.logger.Debug("one of name, email or password not specified", zap.String("name", req.Name), zap.String("email", req.Email), zap.Int("password length", len(req.Password)))
		models.SendAPIError(ctx, http.StatusBadRequest, "request must include the user's name, email and password")
		return
	}

	_, err := r.userService.CreateUser(ctx, req.Name, req.Email, req.Password)
	if err != nil {
		switch errors.Cause(err) {
		case services.ErrEmailTaken:
			r.logger.Debug("email taken", zap.String("email", req.Email), zap.Error(err))
			models.SendAPIError(ctx, http.StatusBadRequest, "user with given email already exists")
		default:
			r.logger.Error("could not create user", zap.Error(err))
			models.SendAPIError(ctx, http.StatusInternalServerError, "something went wrong")
		}
		return
	}

	// TODO: add email verification (https://github.com/unicsmcr/hs_auth/issues/87)

	ctx.Status(http.StatusOK)
}

// GET: /api/v2/users
// Response: users []entities.User
// Headers:  Authorization -> token
func (r *apiV2Router) GetUsers(ctx *gin.Context) {
	var (
		users []entities.User
		err   error
	)
	if ctx.Query("team") != "" {
		// TODO: https://github.com/unicsmcr/hs_auth/issues/68
		users, err = r.getTeamMembersCtxAware(ctx, ctx.Query("team"))
	} else {
		users, err = r.userService.GetUsers(ctx)
	}

	if err != nil {
		switch errors.Cause(err) {
		case v2.ErrInvalidToken:
			r.logger.Debug("invalid token", zap.Error(err))
			r.HandleUnauthorized(ctx)
		case v2.ErrInvalidTokenType:
			r.logger.Debug("invalid token type", zap.Error(err))
			models.SendAPIError(ctx, http.StatusBadRequest, "provided token is of invalid type for the requested operation")
		case services.ErrInvalidID:
			r.logger.Debug("invalid id", zap.Error(err))
			models.SendAPIError(ctx, http.StatusBadRequest, "invalid id")
		case services.ErrNotFound:
			r.logger.Debug("user not found", zap.Error(err))
			models.SendAPIError(ctx, http.StatusNotFound, "user not found")
		case services.ErrUserNotInTeam:
			r.logger.Debug("user is not in a team", zap.Error(err))
			models.SendAPIError(ctx, http.StatusBadRequest, "user not found")
		default:
			r.logger.Error("could not fetch user", zap.Error(err))
			models.SendAPIError(ctx, http.StatusInternalServerError, "something went wrong")
		}
		return
	}

	ctx.JSON(http.StatusOK, getUsersRes{Users: users})
}

// GET: /api/v2/users/(:id|me)
// Response: user entities.User
// Headers:  Authorization -> token
func (r *apiV2Router) GetUser(ctx *gin.Context) {
	var (
		user *entities.User
		err  error
	)
	user, err = r.getUserCtxAware(ctx, ctx.Param("id"))
	if err != nil {
		switch errors.Cause(err) {
		case v2.ErrInvalidToken:
			r.logger.Debug("invalid token", zap.Error(err))
			r.HandleUnauthorized(ctx)
		case v2.ErrInvalidTokenType:
			r.logger.Debug("invalid token type", zap.Error(err))
			models.SendAPIError(ctx, http.StatusBadRequest, "provided token is of invalid type for the requested operation")
		case services.ErrInvalidID:
			r.logger.Debug("invalid user id", zap.Error(err))
			models.SendAPIError(ctx, http.StatusBadRequest, "invalid user id")
		case services.ErrNotFound:
			r.logger.Debug("user not found", zap.Error(err))
			models.SendAPIError(ctx, http.StatusNotFound, "user not found")
		default:
			r.logger.Error("could not fetch user", zap.Error(err))
			models.SendAPIError(ctx, http.StatusInternalServerError, "something went wrong")
		}
		return
	}

	ctx.JSON(http.StatusOK, getUserRes{
		User: *user,
	})
}

// PUT: /api/v2/users/:id/team
// x-www-form-urlencoded
// Request:  team primitive.ObjectId
// Headers:  Authorization -> token
func (r *apiV2Router) SetTeam(ctx *gin.Context) {
	teamId := ctx.PostForm("team")
	if len(teamId) == 0 {
		r.logger.Debug("team id not provided")
		models.SendAPIError(ctx, http.StatusBadRequest, "team id must be provided")
		return
	}

	userId := ctx.Param("id")
	if userId == "me" {
		userIdObj, err := r.authorizer.GetUserIdFromToken(r.GetAuthToken(ctx))
		if err != nil {
			switch errors.Cause(err) {
			case v2.ErrInvalidToken:
				r.logger.Debug("invalid token", zap.Error(err))
				r.HandleUnauthorized(ctx)
			case v2.ErrInvalidTokenType:
				r.logger.Debug("invalid token type", zap.Error(err))
				models.SendAPIError(ctx, http.StatusBadRequest, "provided token is of invalid type for the requested operation")
			default:
				r.logger.Error("could not extract token type", zap.Error(err))
				models.SendAPIError(ctx, http.StatusInternalServerError, "something went wrong")
			}
			return
		}

		userId = userIdObj.Hex()
	}

	err := r.teamService.AddUserWithIDToTeamWithID(ctx, userId, teamId)
	if err != nil {
		switch errors.Cause(err) {
		case services.ErrInvalidID:
			r.logger.Debug("user or team id is invalid", zap.Error(err))
			models.SendAPIError(ctx, http.StatusBadRequest, "user or team id is invalid")
		case services.ErrNotFound:
			r.logger.Debug("user or team not found", zap.Error(err))
			models.SendAPIError(ctx, http.StatusNotFound, "user or team with given id not found")
		case services.ErrUserInTeam:
			r.logger.Debug("user is already in team", zap.Error(err))
			models.SendAPIError(ctx, http.StatusBadRequest, "user is already in a team")
		default:
			r.logger.Error("could not add user to team", zap.Error(err))
			models.SendAPIError(ctx, http.StatusInternalServerError, "something went wrong")
		}
		return
	}

	ctx.Status(http.StatusNoContent)
}

// DELETE: /api/v2/users/:id/team
// x-www-form-urlencoded
// Headers:  Authorization -> token
func (r *apiV2Router) RemoveFromTeam(ctx *gin.Context) {
	userId := ctx.Param("id")
	if userId == "me" {
		userIdObj, err := r.authorizer.GetUserIdFromToken(r.GetAuthToken(ctx))
		if err != nil {
			switch errors.Cause(err) {
			case v2.ErrInvalidToken:
				r.logger.Debug("invalid token", zap.Error(err))
				r.HandleUnauthorized(ctx)
			case v2.ErrInvalidTokenType:
				r.logger.Debug("invalid token type", zap.Error(err))
				models.SendAPIError(ctx, http.StatusBadRequest, "provided token is of invalid type for the requested operation")
			default:
				r.logger.Error("could not extract token type", zap.Error(err))
				models.SendAPIError(ctx, http.StatusInternalServerError, "something went wrong")
			}
			return
		}

		userId = userIdObj.Hex()
	}

	err := r.teamService.RemoveUserWithIDFromTheirTeam(ctx, userId)
	if err != nil {
		switch errors.Cause(err) {
		case services.ErrInvalidID:
			r.logger.Debug("user id is invalid", zap.Error(err))
			models.SendAPIError(ctx, http.StatusBadRequest, "user id is invalid")
		case services.ErrNotFound:
			r.logger.Debug("user or team not found", zap.Error(err))
			models.SendAPIError(ctx, http.StatusNotFound, "user or team with given id not found")
		case services.ErrUserNotInTeam:
			r.logger.Debug("user is not in a team", zap.Error(err))
			models.SendAPIError(ctx, http.StatusBadRequest, "user is not in a team")
		default:
			r.logger.Error("could not add user to team", zap.Error(err))
			models.SendAPIError(ctx, http.StatusInternalServerError, "something went wrong")
		}
		return
	}

	ctx.Status(http.StatusNoContent)
}

// PUT: /api/v2/users/(:id|me)/password
// x-www-form-urlencoded
// Request:  password string
// Response:
// Headers:  Authorization -> token
func (r *apiV2Router) SetPassword(ctx *gin.Context) {
	var req struct {
		Password string `form:"password"`
	}
	_ = ctx.Bind(&req)
	if len(req.Password) == 0 {
		r.logger.Debug("password not specified", zap.Int("password length", len(req.Password)))
		models.SendAPIError(ctx, http.StatusBadRequest, "request must include the new password")
		return
	}

	user, err := r.getUserCtxAware(ctx, ctx.Param("id"))
	if err != nil {
		switch errors.Cause(err) {
		case v2.ErrInvalidToken:
			r.logger.Debug("invalid token", zap.Error(err))
			r.HandleUnauthorized(ctx)
		case v2.ErrInvalidTokenType:
			r.logger.Debug("invalid token type", zap.Error(err))
			models.SendAPIError(ctx, http.StatusBadRequest, "provided token is of invalid type for the requested operation")
		case services.ErrInvalidID:
			r.logger.Debug("invalid user id", zap.Error(err))
			models.SendAPIError(ctx, http.StatusBadRequest, "invalid user id")
		case services.ErrNotFound:
			r.logger.Debug("user not found", zap.Error(err))
			models.SendAPIError(ctx, http.StatusNotFound, "user not found")
		default:
			r.logger.Error("could not fetch user", zap.Error(err))
			models.SendAPIError(ctx, http.StatusInternalServerError, "something went wrong")
		}
		return
	}

	err = r.userService.ResetPasswordForUserWithIDAndEmail(ctx, user.ID.Hex(), user.Email, req.Password)
	if err != nil {
		switch errors.Cause(err) {
		case services.ErrInvalidID:
			r.logger.Debug("invalid user id", zap.Error(err))
			models.SendAPIError(ctx, http.StatusBadRequest, "invalid user id")
		case services.ErrNotFound:
			r.logger.Debug("user not found", zap.Error(err))
			models.SendAPIError(ctx, http.StatusNotFound, "user does not exist")
		default:
			r.logger.Error("could not update user", zap.Error(err))
			models.SendAPIError(ctx, http.StatusInternalServerError, "something went wrong")
		}
	}

	ctx.Status(http.StatusOK)
}

// GET: /api/v2/users/(:id|me)/password/resetEmail
// Response:
// Headers:  Authorization -> token
func (r *apiV2Router) GetPasswordResetEmail(ctx *gin.Context) {
	user, err := r.getUserCtxAware(ctx, ctx.Param("id"))
	if err != nil {
		switch errors.Cause(err) {
		case v2.ErrInvalidToken:
			r.logger.Debug("invalid token", zap.Error(err))
			r.HandleUnauthorized(ctx)
		case v2.ErrInvalidTokenType:
			r.logger.Debug("invalid token type", zap.Error(err))
			models.SendAPIError(ctx, http.StatusBadRequest, "provided token is of invalid type for the requested operation")
		case services.ErrInvalidID:
			r.logger.Debug("invalid user id", zap.Error(err))
			models.SendAPIError(ctx, http.StatusBadRequest, "invalid user id")
		case services.ErrNotFound:
			r.logger.Debug("user not found", zap.Error(err))
			models.SendAPIError(ctx, http.StatusNotFound, "user not found")
		default:
			r.logger.Error("could not fetch user", zap.Error(err))
			models.SendAPIError(ctx, http.StatusInternalServerError, "something went wrong")
		}
		return
	}

	err = r.emailService.SendPasswordResetEmail(*user)
	if err != nil {
		r.logger.Error("could not send password reset email", zap.Error(err))
		models.SendAPIError(ctx, http.StatusInternalServerError, "something went wrong")
	}

	ctx.Status(http.StatusOK)
}

// getUserCtxAware fetches user with the given id. If id is "me", getUserCtxAware tries to extract the user from the ctx
func (r *apiV2Router) getUserCtxAware(ctx *gin.Context, userId string) (*entities.User, error) {
	if userId == "me" {
		token := r.GetAuthToken(ctx)
		userIdObj, err := r.authorizer.GetUserIdFromToken(token)
		if err != nil {
			return nil, errors.Wrap(err, "could not extract user id from auth token")
		}

		userId = userIdObj.Hex()
	}

	user, err := r.userService.GetUserWithID(ctx, userId)
	if err != nil {
		return nil, errors.Wrap(err, "could not fetch user")
	}

	return user, nil
}

// getTeamMembersCtxAware fetches team members for team with the given id.
// If id is "me", getTeamMembersCtxAware tries to extract the team from the ctx
func (r *apiV2Router) getTeamMembersCtxAware(ctx *gin.Context, teamId string) ([]entities.User, error) {
	var (
		members []entities.User
		err     error
	)
	if teamId == "me" {
		token := r.GetAuthToken(ctx)
		var userIdObj primitive.ObjectID
		userIdObj, err = r.authorizer.GetUserIdFromToken(token)
		if err != nil {
			return nil, errors.Wrap(err, "could not extract user id from auth token")
		}

		members, err = r.userService.GetTeamMembersForUserWithID(ctx, userIdObj.Hex())
	} else {
		members, err = r.userService.GetUsersWithTeam(ctx, teamId)
	}

	if err != nil {
		return nil, errors.Wrap(err, "could not fetch team members")
	}

	return members, nil
}
