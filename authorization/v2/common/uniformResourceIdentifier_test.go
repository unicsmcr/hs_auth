package common

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	mock_resources "github.com/unicsmcr/hs_auth/mocks/authorization/v2/common"
	"github.com/unicsmcr/hs_auth/testutils"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

func testHandlerm(*gin.Context) {}
func testUnmarshalYAMLValid(a interface{}) error {
	sl := reflect.ValueOf(a).Elem()
	sl.Set(reflect.Append(sl, reflect.ValueOf("hs:hs_auth")))
	return nil
}
func testUnmarshalYAMLInvalid(interface{}) error {
	return errors.New("random error")
}

func TestNewUriFromRequest(t *testing.T) {
	w := httptest.NewRecorder()
	testCtx, _ := gin.CreateTestContext(w)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	postFormParams := url.Values{}
	postFormParams.Add("name", "Bob the Tester")
	req := httptest.NewRequest(http.MethodPost, "/test?name=RobTheTester", bytes.NewBufferString(postFormParams.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")
	req.PostForm = postFormParams
	testCtx.Request = req
	testutils.AddUrlParamsToCtx(testCtx, map[string]string{"name": "Bill the Tester"})

	mockRouterResource := mock_resources.NewMockRouterResource(ctrl)
	mockRouterResource.EXPECT().GetResourcePath().Return("test_router").Times(1)

	uri := NewUriFromRequest(mockRouterResource, testHandlerm, testCtx)

	assert.Equal(t, "test_router:testHandlerm", uri.path)
	assert.Equal(t, map[string]string{
		"path_name":     "Bill the Tester",
		"query_name":    "RobTheTester",
		"postForm_name": "Bob the Tester",
	}, uri.arguments)
	assert.Nil(t, uri.metadata)
}

func Test_NewURIFromString__should_return_correct_URI(t *testing.T) {
	tests := []struct {
		name        string
		uri         string
		expectedURI UniformResourceIdentifier
	}{
		{
			name: "with only path",
			uri:  "hs:hs_auth:api:v2:provide_access_to_uri",
			expectedURI: UniformResourceIdentifier{
				path: "hs:hs_auth:api:v2:provide_access_to_uri",
			},
		},
		{
			name: "with path and arguments",
			uri:  "hs:hs_auth:api:v2:provide_access_to_uri?allowed_uri%3Dhs%3Ahs_application%3A%2A",
			expectedURI: UniformResourceIdentifier{
				path:      "hs:hs_auth:api:v2:provide_access_to_uri",
				arguments: map[string]string{"allowed_uri": "hs:hs_application:*"},
			},
		},
		{
			name: "with path, arguments and metadata",
			uri:  "hs:hs_auth:api:v2:provide_access_to_uri?allowed_uri%3Dhs%3Ahs_application%3A%2A#until%3D21392103",
			expectedURI: UniformResourceIdentifier{
				path:      "hs:hs_auth:api:v2:provide_access_to_uri",
				arguments: map[string]string{"allowed_uri": "hs:hs_application:*"},
				metadata:  map[string]string{"until": "21392103"},
			},
		},
		{
			name: "with path and metadata",
			uri:  "hs:hs_auth:api:v2:provide_access_to_uri#until%3D21392103",
			expectedURI: UniformResourceIdentifier{
				path:     "hs:hs_auth:api:v2:provide_access_to_uri",
				metadata: map[string]string{"until": "21392103"},
			},
		},
		{
			name: "with url encoded metadata rune in metadata",
			uri:  "hs:hs_auth:api:v2:provide_access_to_uri?test=ok%23#test2=ok",
			expectedURI: UniformResourceIdentifier{
				path:      "hs:hs_auth:api:v2:provide_access_to_uri",
				arguments: map[string]string{"test": "ok#"},
				metadata:  map[string]string{"test2": "ok"},
			},
		},
		{
			name: "with path and arguments provided with key and no value",
			uri:  "hs:hs_auth:api:v2:provide_access_to_uri?test=",
			expectedURI: UniformResourceIdentifier{
				path:      "hs:hs_auth:api:v2:provide_access_to_uri",
				arguments: map[string]string{"test": ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualURI, err := NewURIFromString(tt.uri)
			assert.NoError(t, err)

			assert.Equal(t, actualURI, tt.expectedURI)
		})
	}
}

func Test_NewURIFromString__should_throw_error(t *testing.T) {
	tests := []struct {
		name string
		uri  string
	}{
		{
			name: "when malformed arguments provided, with key only",
			uri:  "hs:hs_auth:api:v2:provide_access_to_uri?test_arg",
		},
		{
			name: "when more than one argument rune provided",
			uri:  "hs:hs_auth:api:v2:provide_access_to_uri??",
		},
		{
			name: "when malformed metadata provided",
			uri:  "hs:hs_auth:api:v2:provide_access_to_uri#test_arg_metadata",
		},
		{
			name: "when malformed uri provided",
			uri:  "hs:hs_auth:api:v2:provide_access_to_uri#test_arg_metadata%3Dtest1#test_arg2%3Dtest2",
		},
		{
			name: "when malformed url encoded arguments provided",
			uri:  "hs:hs_auth:api:v2:provide_access_to_uri?test_arg%3Dtest1%ZZ",
		},
		{
			name: "when malformed url encoded metadata provided",
			uri:  "hs:hs_auth:api:v2:provide_access_to_uri#test_arg%3Dtest1%NN%UU",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testURI := tt.uri
			_, err := NewURIFromString(testURI)
			assert.Error(t, err)
		})
	}
}

func Test_MarshalJSON__should_return_correct_string(t *testing.T) {
	tests := []struct {
		name           string
		uri            UniformResourceIdentifier
		expectedResult string
	}{
		{
			name: "with only path",
			uri: UniformResourceIdentifier{
				path: "hs:hs_auth:api:v2:provide_access_to_uri",
			},
			expectedResult: "\"hs:hs_auth:api:v2:provide_access_to_uri\"",
		},
		{
			name: "with path and arguments",
			uri: UniformResourceIdentifier{
				path:      "hs:hs_auth:api:v2:provide_access_to_uri",
				arguments: map[string]string{"allowed_uri": "hs:hs_application:*"},
			},
			expectedResult: "\"hs:hs_auth:api:v2:provide_access_to_uri?allowed_uri%3Dhs%3Ahs_application%3A%2A\"",
		},
		{
			name: "with path and metadata",
			uri: UniformResourceIdentifier{
				path:     "hs:hs_auth:api:v2:provide_access_to_uri",
				metadata: map[string]string{"test_arg": "test1"},
			},
			expectedResult: "\"hs:hs_auth:api:v2:provide_access_to_uri#test_arg%3Dtest1\"",
		},
		{
			name: "with path, arguments and metadata",
			uri: UniformResourceIdentifier{
				path:      "hs:hs_auth:api:v2:provide_access_to_uri",
				arguments: map[string]string{"test_arg": "test1"},
				metadata:  map[string]string{"until": "21392103"},
			},
			expectedResult: "\"hs:hs_auth:api:v2:provide_access_to_uri?test_arg%3Dtest1#until%3D21392103\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.uri.MarshalJSON()
			assert.NoError(t, err)

			assert.Equal(t, tt.expectedResult, string(result))
		})
	}
}

func Test_UnmarshalJSON__should_return_correct_URI(t *testing.T) {
	tests := []struct {
		name           string
		uriString      string
		expectedResult UniformResourceIdentifier
	}{
		{
			name:           "with no uri",
			uriString:      "null",
			expectedResult: UniformResourceIdentifier{},
		},
		{
			name:      "with only path",
			uriString: "\"hs:hs_auth:api:v2:provide_access_to_uri\"",
			expectedResult: UniformResourceIdentifier{
				path: "hs:hs_auth:api:v2:provide_access_to_uri",
			},
		},
		{
			name:      "with path and arguments",
			uriString: "\"hs:hs_auth:api:v2:provide_access_to_uri?allowed_uri%3Dhs%3Ahs_application%3A%2A\"",
			expectedResult: UniformResourceIdentifier{
				path:      "hs:hs_auth:api:v2:provide_access_to_uri",
				arguments: map[string]string{"allowed_uri": "hs:hs_application:*"},
			},
		},
		{
			name:      "with path and metadata",
			uriString: "\"hs:hs_auth:api:v2:provide_access_to_uri#test_arg%3Dtest1\"",
			expectedResult: UniformResourceIdentifier{
				path:     "hs:hs_auth:api:v2:provide_access_to_uri",
				metadata: map[string]string{"test_arg": "test1"},
			},
		},
		{
			name:      "with path, arguments and metadata",
			uriString: "\"hs:hs_auth:api:v2:provide_access_to_uri?test_arg%3Dtest1#until%3D21392103\"",
			expectedResult: UniformResourceIdentifier{
				path:      "hs:hs_auth:api:v2:provide_access_to_uri",
				arguments: map[string]string{"test_arg": "test1"},
				metadata:  map[string]string{"until": "21392103"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			identifier := UniformResourceIdentifier{}
			err := identifier.UnmarshalJSON([]byte(tt.uriString))
			assert.NoError(t, err)

			assert.Equal(t, tt.expectedResult, identifier)
		})
	}
}

func Test_UniformResourceIdentifiers_MarshalJSON__should_return_correct_string(t *testing.T) {
	tests := []struct {
		name           string
		uris           UniformResourceIdentifiers
		expectedResult string
	}{
		{
			name: "with only path",
			uris: UniformResourceIdentifiers{{
				path: "hs:hs_auth:api:v2:provide_access_to_uri",
			}},
			expectedResult: "[\"hs:hs_auth:api:v2:provide_access_to_uri\"]",
		},
		{
			name: "with only path and multiple uris",
			uris: UniformResourceIdentifiers{
				{
					path: "hs:hs_auth:api:v2:provide_access_to_uri",
				},
				{
					path: "hs:hs_auth:api:v2:users",
				}},
			expectedResult: "[\"hs:hs_auth:api:v2:provide_access_to_uri\",\"hs:hs_auth:api:v2:users\"]",
		},
		{
			name: "with path and arguments",
			uris: UniformResourceIdentifiers{{
				path:      "hs:hs_auth:api:v2:provide_access_to_uri",
				arguments: map[string]string{"allowed_uri": "hs:hs_application:*"},
			}},
			expectedResult: "[\"hs:hs_auth:api:v2:provide_access_to_uri?allowed_uri%3Dhs%3Ahs_application%3A%2A\"]",
		},
		{
			name: "with path and metadata",
			uris: UniformResourceIdentifiers{{
				path:     "hs:hs_auth:api:v2:provide_access_to_uri",
				metadata: map[string]string{"test_arg": "test1"},
			}},
			expectedResult: "[\"hs:hs_auth:api:v2:provide_access_to_uri#test_arg%3Dtest1\"]",
		},
		{
			name: "with path, arguments and metadata",
			uris: UniformResourceIdentifiers{{
				path:      "hs:hs_auth:api:v2:provide_access_to_uri",
				arguments: map[string]string{"test_arg": "test1"},
				metadata:  map[string]string{"until": "21392103"},
			}},
			expectedResult: "[\"hs:hs_auth:api:v2:provide_access_to_uri?test_arg%3Dtest1#until%3D21392103\"]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.uris.MarshalJSON()
			assert.NoError(t, err)

			assert.Equal(t, tt.expectedResult, string(result))
		})
	}
}

func Test_UniformResourceIdentifiers_UnmarshalJSON__should_return_correct_uri(t *testing.T) {
	tests := []struct {
		name           string
		uriString      string
		expectedResult UniformResourceIdentifiers
	}{
		{
			name:           "with no uris",
			uriString:      "null",
			expectedResult: nil,
		},
		{
			name:      "with uri array, single element",
			uriString: "[\"hs:hs_auth:api:v2:provide_access_to_uri?test_arg%3Dtest1#until%3D21392103\"]",
			expectedResult: UniformResourceIdentifiers{{
				path:      "hs:hs_auth:api:v2:provide_access_to_uri",
				arguments: map[string]string{"test_arg": "test1"},
				metadata:  map[string]string{"until": "21392103"},
			}},
		},
		{
			name:      "with uri array, multiple elements",
			uriString: "[\"hs:hs_auth:api:v2:provide_access_to_uri\",\"hs:hs_notify\"]",
			expectedResult: UniformResourceIdentifiers{
				{
					path: "hs:hs_auth:api:v2:provide_access_to_uri",
				},
				{
					path: "hs:hs_notify",
				}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			identifier := UniformResourceIdentifiers(nil)
			err := identifier.UnmarshalJSON([]byte(tt.uriString))
			assert.NoError(t, err)

			assert.Equal(t, tt.expectedResult, identifier)
		})
	}
}

func Test_UniformResourceIdentifiers_UnmarshalJSON__should_return_error_with_invalid_uri(t *testing.T) {
	testURIString := "[\"hs:hs_auth:api:v2:user?test_arg%3Dtest1##until%3D21392103\"]"

	identifier := UniformResourceIdentifiers{}
	err := identifier.UnmarshalJSON([]byte(testURIString))
	assert.Error(t, err)
}

func Test_UnmarshalYAML__should_unmarshal_with_valid_uri(t *testing.T) {
	uriSequence := &UniformResourceIdentifiers{}
	err := uriSequence.UnmarshalYAML(testUnmarshalYAMLValid)
	assert.NoError(t, err)

	expectedURI, _ := NewURIFromString("hs:hs_auth")
	assert.Equal(t, expectedURI, (*uriSequence)[0])
}

func Test_UnmarshalYAML__should_return_err_with_invalid_uri(t *testing.T) {
	uriSequence := &UniformResourceIdentifiers{}
	err := uriSequence.UnmarshalYAML(testUnmarshalYAMLInvalid)
	assert.Error(t, err)
}

func Test_MarshalBSONValue_should_marshal_with_valid_uri(t *testing.T) {
	expectedURI := "hs:test"
	var allURIs = UniformResourceIdentifiers{
		{
			path: expectedURI,
		},
	}

	bsonType, data, err := allURIs.MarshalBSONValue()
	actualResult, _, _ := bsoncore.ReadString(data)
	assert.NoError(t, err)
	assert.Equal(t, expectedURI, actualResult)
	assert.Equal(t, bsontype.String, bsonType)
}

func Test_UnmarshalBSONValue_should_unmarshal_with_valid_uri(t *testing.T) {
	testURI := "hs:test"
	uriBytes := bsoncore.AppendString(nil, testURI)
	var allURIs = UniformResourceIdentifiers{}

	err := allURIs.UnmarshalBSONValue(bsontype.String, uriBytes)
	assert.NoError(t, err)
	assert.Equal(t, testURI, allURIs[0].path)
}

func Test_UnmarshalBSONValue_should_return_error_with_invalid_uri(t *testing.T) {
	testURI := "#hs:test??####"
	uriBytes := bsoncore.AppendString(nil, testURI)
	var allURIs = UniformResourceIdentifiers{}

	err := allURIs.UnmarshalBSONValue(bsontype.String, uriBytes)
	assert.Error(t, err)
}

func Test_isSupersetOf__should_return_true_with_source_in_target_set(t *testing.T) {
	tests := []struct {
		name   string
		source UniformResourceIdentifier
		target UniformResourceIdentifier
	}{
		{
			name: "only path of same lengths",
			source: UniformResourceIdentifier{
				path: "hs:hs_auth:api:v2:GetUser",
			},
			target: UniformResourceIdentifier{
				path: "hs:hs_auth:api:v2:GetUser",
			},
		},
		{
			name: "path of same lengths, and same arguments",
			source: UniformResourceIdentifier{
				path:      "hs:hs_auth:api:v2:GetUser",
				arguments: map[string]string{"path_id": "me"},
			},
			target: UniformResourceIdentifier{
				path:      "hs:hs_auth:api:v2:GetUser",
				arguments: map[string]string{"path_id": "me"},
			},
		},
		{
			name: "path of same lengths, and arguments",
			source: UniformResourceIdentifier{
				path: "hs:hs_auth:api:v2:GetUser",
			},
			target: UniformResourceIdentifier{
				path:      "hs:hs_auth:api:v2:GetUser",
				arguments: map[string]string{"path_id": "me"},
			},
		},
		{
			name: "only path",
			source: UniformResourceIdentifier{
				path: "hs:hs_auth:api:v2",
			},
			target: UniformResourceIdentifier{
				path: "hs:hs_auth:api:v2:provide_access_to_uri",
			},
		},
		{
			name: "path and arguments",
			source: UniformResourceIdentifier{
				path:      "hs:hs_auth:api:v2:provide_access_to_uri",
				arguments: map[string]string{"allowed_uri": "hs:hs_application:*"},
			},
			target: UniformResourceIdentifier{
				path:      "hs:hs_auth:api:v2:provide_access_to_uri",
				arguments: map[string]string{"allowed_uri": "hs:hs_application:checkin:*"},
			},
		},
		{
			name: "path, arguments and metadata",
			source: UniformResourceIdentifier{
				path:      "hs:hs_auth:api:v2:provide_access_to_uri",
				arguments: map[string]string{"allowed_uri": "hs:hs_application:*"},
				metadata:  map[string]string{"until": "21392103"},
			},
			target: UniformResourceIdentifier{
				path:      "hs:hs_auth:api:v2:provide_access_to_uri",
				arguments: map[string]string{"allowed_uri": "hs:hs_application:checkin:*"},
				metadata:  map[string]string{"until": "21392103"},
			},
		},
		{
			name: "target has more argument limitations than source",
			source: UniformResourceIdentifier{
				path: "hs:hs_auth:frontend:ResetPassword",
				arguments: map[string]string{
					"postForm_userId": "5f759cc023a05c9953542c62",
				},
			},
			target: UniformResourceIdentifier{
				path: "hs:hs_auth:frontend:ResetPassword",
				arguments: map[string]string{
					"postForm_userId":          "5f759cc023a05c9953542c62",
					"postForm_password":        "asdasd",
					"postForm_passwordConfirm": "asdasd",
				},
			},
		},
		{
			name: "target does not have the argument that is limited to an empty string",
			source: UniformResourceIdentifier{
				path: "hs:hs_auth:frontend:ResetPassword",
				arguments: map[string]string{
					"postForm_userId": "",
				},
			},
			target: UniformResourceIdentifier{
				path: "hs:hs_auth:frontend:ResetPassword",
				arguments: map[string]string{
					"postForm_password":        "asdasd",
					"postForm_passwordConfirm": "asdasd",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := tt.source.isSupersetOf(tt.target)

			assert.Equal(t, true, valid)
		})
	}
}

func Test_isSupersetOf__should_return_false_with_source_not_in_target_set(t *testing.T) {
	tests := []struct {
		name   string
		source UniformResourceIdentifier
		target UniformResourceIdentifier
	}{
		{
			name: "only path",
			source: UniformResourceIdentifier{
				path: "hs:hs_application:user:@me",
			},
			target: UniformResourceIdentifier{
				path: "hs:hs_application:user",
			},
		},
		{
			name: "only path with target longer than source",
			source: UniformResourceIdentifier{
				path: "hs:hs_auth:api:v2:GetUser",
			},
			target: UniformResourceIdentifier{
				path: "hs:hs_auth:api:v2:GetUsers",
			},
		},
		{
			name: "only path with source longer than target",
			source: UniformResourceIdentifier{
				path: "hs:hs_application:teams",
			},
			target: UniformResourceIdentifier{
				path: "hs:hs_application:user",
			},
		},
		{
			name: "path of same lengths and different argument values",
			source: UniformResourceIdentifier{
				path:      "hs:hs_auth:api:v2:GetUser",
				arguments: map[string]string{"path_id": "123"},
			},
			target: UniformResourceIdentifier{
				path:      "hs:hs_auth:api:v2:GetUser",
				arguments: map[string]string{"path_id": "me"},
			},
		},
		{
			name: "path and arguments with regex",
			source: UniformResourceIdentifier{
				path:      "hs:hs_auth:api:v2:provide_access_to_uri",
				arguments: map[string]string{"allowed_uri": "hs:hs_application:checkin:*"},
				metadata:  nil,
			},
			target: UniformResourceIdentifier{
				path:      "hs:hs_auth:api:v2:provide_access_to_uri",
				arguments: map[string]string{"allowed_uri": "hs:hs_application:*"},
			},
		},
		{
			name: "empty string in arguments",
			source: UniformResourceIdentifier{
				path:      "hs:hs_auth:api:v2:provide_access_to_uri",
				arguments: map[string]string{"allowed_uri": ""},
				metadata:  nil,
			},
			target: UniformResourceIdentifier{
				path:      "hs:hs_auth:api:v2:provide_access_to_uri",
				arguments: map[string]string{"allowed_uri": "hs:hs_application:*"},
			},
		},
		{
			name: "missing argument in target URI for non-empty argument limiter in source",
			source: UniformResourceIdentifier{
				path:      "hs:hs_auth:api:v2:provide_access_to_uri",
				arguments: map[string]string{"allowed_uri": "non-empty string"},
				metadata:  nil,
			},
			target: UniformResourceIdentifier{
				path:      "hs:hs_auth:api:v2:provide_access_to_uri",
				arguments: map[string]string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := tt.source.isSupersetOf(tt.target)

			assert.Equal(t, false, valid)
		})
	}
}

func Test_IsSupersetOfAtLeastOne__should_return_true_when_last_target_matches(t *testing.T) {
	testSource := UniformResourceIdentifier{
		path:      "hs:hs_auth",
		arguments: map[string]string{"test": "1"},
	}
	testTargets := []UniformResourceIdentifier{
		{
			path: "hs:hs_application",
		},
		{
			path:      "hs:hs_auth",
			arguments: map[string]string{"test": "1"},
		},
	}

	valid := testSource.IsSupersetOfAtLeastOne(testTargets)
	assert.Equal(t, valid, true)
}

func Test_IsSupersetOfAtLeastOne__should_return_false_when_no_targets(t *testing.T) {
	testSource := UniformResourceIdentifier{
		path:      "hs:hs_auth",
		arguments: map[string]string{"test": "1"},
	}

	valid := testSource.IsSupersetOfAtLeastOne(nil)
	assert.Equal(t, valid, false)
}

func Test_IsSupersetOfAtLeastOne__should_return_false_when_path_doesnt_match(t *testing.T) {
	testSource := UniformResourceIdentifier{
		path: "hs:hs_auth1",
	}
	testTargets := []UniformResourceIdentifier{
		{
			path: "hs:hs_autb1",
		},
	}

	valid := testSource.IsSupersetOfAtLeastOne(testTargets)
	assert.Equal(t, valid, false)
}

func Test_GetAllSupersets__should_return_valid_uri_set(t *testing.T) {
	tests := []struct {
		name         string
		source       UniformResourceIdentifier
		targets      []UniformResourceIdentifier
		expectedUris []UniformResourceIdentifier
	}{
		{
			name: "with one target uri that is a subset of source",
			source: UniformResourceIdentifier{
				path: "hs:hs_auth:api:v2",
			},
			targets: []UniformResourceIdentifier{
				{path: "hs:hs_auth:api:v1:SetUser"},
				{path: "hs:hs_auth:api:v2:GetUser"},
				{path: "hs:hs_application"},
			},
			expectedUris: []UniformResourceIdentifier{
				{path: "hs:hs_auth:api:v2:GetUser"},
			},
		},
		{
			name: "with many target uris that is are subset of source",
			source: UniformResourceIdentifier{
				path: "hs:hs_auth:api:v2",
			},
			targets: []UniformResourceIdentifier{
				{path: "hs:hs_auth:api:v1:SetUser"},
				{path: "hs:hs_auth:api:v2"},
				{path: "hs:hs_application"},
				{path: "hs:hs_auth:api:v2:GetUser:test"},
			},
			expectedUris: []UniformResourceIdentifier{
				{path: "hs:hs_auth:api:v2"},
				{path: "hs:hs_auth:api:v2:GetUser:test"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validTargets := tt.source.GetAllSupersets(tt.targets)
			assert.Equal(t, tt.expectedUris, validTargets)
		})
	}
}

func TestUniformResourceIdentifier_GetMetadata(t *testing.T) {
	testUri := UniformResourceIdentifier{
		metadata: map[string]string{"testKey": "testValue"},
	}

	assert.Equal(t, map[string]string{"testKey": "testValue"}, testUri.GetMetadata())
}
