package generate_test

import (
	"testing"

	"github.com/kalo-build/plugin-openapi-ai-context/pkg/generate"
	"github.com/stretchr/testify/suite"
)

type GenerateTestSuite struct {
	suite.Suite
}

func TestGenerateTestSuite(t *testing.T) {
	suite.Run(t, new(GenerateTestSuite))
}

func (s *GenerateTestSuite) specBytes() []byte {
	return []byte(`openapi: "3.1.0"
info:
  title: "Test API"
  version: "1.0.0"
servers:
  - url: /api/v1
security:
  - bearerAuth: []
components:
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
  schemas:
    User:
      type: object
      properties:
        id:
          type: string
        name:
          type: string
      required: [id, name]
    UserCreate:
      type: object
      properties:
        name:
          type: string
      required: [name]
paths:
  /users:
    get:
      operationId: listUsers
      tags: [users]
      parameters:
        - name: q
          in: query
          schema:
            type: string
      responses:
        "200":
          description: "OK"
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/User'
    post:
      operationId: createUser
      tags: [users]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/UserCreate'
      responses:
        "201":
          description: "Created"
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/User'
  /login:
    post:
      operationId: login
      tags: [auth]
      security: []
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/UserCreate'
      responses:
        "200":
          description: "OK"
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/User'
`)
}

func (s *GenerateTestSuite) TestBuildContracts_BasePath() {
	contracts, err := generate.BuildContracts(s.specBytes())
	s.NoError(err)
	s.Equal("/api/v1", contracts.BasePath)
}

func (s *GenerateTestSuite) TestBuildContracts_Auth() {
	contracts, err := generate.BuildContracts(s.specBytes())
	s.NoError(err)
	s.Equal("Bearer JWT", contracts.Auth)
}

func (s *GenerateTestSuite) TestBuildContracts_EndpointCount() {
	contracts, err := generate.BuildContracts(s.specBytes())
	s.NoError(err)

	s.Len(contracts.Endpoints, 2, "expected two tag groups: auth, users")
	s.Len(contracts.Endpoints["auth"], 1)
	s.Len(contracts.Endpoints["users"], 2)
}

func (s *GenerateTestSuite) TestBuildContracts_AuthDisabled() {
	contracts, err := generate.BuildContracts(s.specBytes())
	s.NoError(err)

	loginEP := contracts.Endpoints["auth"][0]
	s.Require().NotNil(loginEP.Auth)
	s.False(*loginEP.Auth, "login endpoint should have auth disabled")
}

func (s *GenerateTestSuite) TestBuildContracts_RequestBody() {
	contracts, err := generate.BuildContracts(s.specBytes())
	s.NoError(err)

	createEP := contracts.Endpoints["users"][1]
	s.Equal("POST", createEP.Method)
	s.Equal("UserCreate", createEP.Body)
}

func (s *GenerateTestSuite) TestBuildContracts_ResponseDirect() {
	contracts, err := generate.BuildContracts(s.specBytes())
	s.NoError(err)

	createEP := contracts.Endpoints["users"][1]
	s.Equal("User", createEP.Response)
}

func (s *GenerateTestSuite) TestBuildContracts_ResponseArray() {
	contracts, err := generate.BuildContracts(s.specBytes())
	s.NoError(err)

	listEP := contracts.Endpoints["users"][0]
	s.Equal("GET", listEP.Method)
	s.Equal("User[]", listEP.Response)
}

func (s *GenerateTestSuite) TestBuildContracts_Filters() {
	contracts, err := generate.BuildContracts(s.specBytes())
	s.NoError(err)

	listEP := contracts.Endpoints["users"][0]
	s.Equal([]string{"q"}, listEP.Filters)
}

func (s *GenerateTestSuite) TestBuildContracts_EndpointSorting() {
	contracts, err := generate.BuildContracts(s.specBytes())
	s.NoError(err)

	users := contracts.Endpoints["users"]
	s.Equal("/users", users[0].Path)
	s.Equal("GET", users[0].Method)
	s.Equal("/users", users[1].Path)
	s.Equal("POST", users[1].Method)
}

func (s *GenerateTestSuite) TestBuildContracts_NoServers() {
	spec := []byte(`openapi: "3.1.0"
info:
  title: "Minimal"
  version: "0.1.0"
paths: {}
`)
	contracts, err := generate.BuildContracts(spec)
	s.NoError(err)
	s.Empty(contracts.BasePath)
	s.Empty(contracts.Auth)
	s.Empty(contracts.Endpoints)
}

func (s *GenerateTestSuite) TestBuildContracts_WrappedArrayResponse() {
	spec := []byte(`openapi: "3.1.0"
info:
  title: "Wrapped"
  version: "1.0.0"
paths:
  /items:
    get:
      operationId: listItems
      tags: [items]
      responses:
        "200":
          description: "OK"
          content:
            application/json:
              schema:
                type: object
                properties:
                  data:
                    type: array
                    items:
                      $ref: '#/components/schemas/Item'
components:
  schemas:
    Item:
      type: object
      properties:
        id:
          type: string
`)
	contracts, err := generate.BuildContracts(spec)
	s.NoError(err)
	s.Len(contracts.Endpoints["items"], 1)
	s.Equal("Item[]", contracts.Endpoints["items"][0].Response)
}

func (s *GenerateTestSuite) TestBuildContracts_NoResponseBody() {
	spec := []byte(`openapi: "3.1.0"
info:
  title: "NoBody"
  version: "1.0.0"
paths:
  /items/{id}:
    delete:
      operationId: deleteItem
      tags: [items]
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
      responses:
        "204":
          description: "Deleted"
`)
	contracts, err := generate.BuildContracts(spec)
	s.NoError(err)
	s.Len(contracts.Endpoints["items"], 1)
	s.Empty(contracts.Endpoints["items"][0].Response)
	s.Empty(contracts.Endpoints["items"][0].Body)
}

func (s *GenerateTestSuite) TestBuildContracts_ApiKeyAuth() {
	spec := []byte(`openapi: "3.1.0"
info:
  title: "ApiKey"
  version: "1.0.0"
components:
  securitySchemes:
    apiKey:
      type: apiKey
      in: header
      name: X-API-Key
paths: {}
`)
	contracts, err := generate.BuildContracts(spec)
	s.NoError(err)
	s.Equal("API key via header X-API-Key", contracts.Auth)
}

func (s *GenerateTestSuite) TestMarshalContracts() {
	contracts := &generate.APIContracts{
		BasePath: "/api",
		Auth:     "Bearer JWT",
		Endpoints: map[string][]generate.Endpoint{
			"users": {
				{Method: "GET", Path: "/users", Response: "User[]"},
			},
		},
	}
	out, err := generate.MarshalContracts(contracts)
	s.NoError(err)
	s.Contains(string(out), "base_path: /api")
	s.Contains(string(out), "auth: Bearer JWT")
	s.Contains(string(out), "method: GET")
}

func (s *GenerateTestSuite) TestConfig_Resolve() {
	cfg := generate.Config{}
	cfg.Resolve()
	s.Equal("openapi.yaml", cfg.SpecFileName)
}
