package auth

import (
	"fmt"
	"testing"

	"github.com/unicsmcr/hs_auth/utils/auth/common"

	"github.com/dgrijalva/jwt-go"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/stretchr/testify/assert"

	"github.com/unicsmcr/hs_auth/entities"
)

func Test_NewJWT__should_throw_error_when_secret_empty(t *testing.T) {
	testUser := entities.User{}

	_, err := NewJWT(testUser, 100, []byte{})
	assert.Error(t, err)
}

func Test_NewJWT__should_return_correct_JWT(t *testing.T) {
	testUser := entities.User{
		ID:        primitive.NewObjectID(),
		AuthLevel: 3,
	}
	testSecret := []byte(`test_secret`)

	expectedToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, common.AuthClaims{
		StandardClaims: jwt.StandardClaims{
			Id:       testUser.ID.Hex(),
			IssuedAt: 100,
		},
		AuthLevel: 3,
	}).SignedString(testSecret)
	assert.NoError(t, err)

	actualToken, err := NewJWT(testUser, 100, testSecret)
	assert.NoError(t, err)

	assert.Equal(t, expectedToken, actualToken)
}

func Test_IsValidJWT__should_return_true_for_valid_JWT(t *testing.T) {
	testUser := entities.User{
		ID:        primitive.NewObjectID(),
		AuthLevel: 3,
	}
	testSecret := []byte(`test_secret`)

	token, err := NewJWT(testUser, 101, testSecret)
	fmt.Println(token)
	assert.NoError(t, err)

	assert.True(t, IsValidJWT(token, testSecret))
}
func Test_IsValidJWT__should_return_false_for_invalid_JWT(t *testing.T) {
	// token with an increased auth_level in claims (signed with the secret "test_secret")
	invalidToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9eyJqdGkiOiI1ZDZlYzA2Nzg4ODJhMTFhYmE0ZjMzODEiLCJpYXQiOjEwMSwiYXV0aF9sZXZlbCI6NH0HbBIrZiQxexzKrnU+GCM8VCs3ZwxaMg=="

	testSecret := []byte(`test_secret`)
	assert.False(t, IsValidJWT(invalidToken, testSecret))
}