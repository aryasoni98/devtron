/*
 * Copyright (c) 2020-2024. Devtron Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package user

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	util2 "github.com/devtron-labs/devtron/pkg/auth/user/util"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/devtron-labs/authenticator/middleware"
	casbin2 "github.com/devtron-labs/devtron/pkg/auth/authorisation/casbin"
	userBean "github.com/devtron-labs/devtron/pkg/auth/user/bean"
	"github.com/devtron-labs/devtron/pkg/auth/user/repository"
	"github.com/go-pg/pg"

	"github.com/caarlos0/env"
	"github.com/coreos/go-oidc/v3/oidc"
	session2 "github.com/devtron-labs/devtron/client/argocdServer/session"
	"github.com/devtron-labs/devtron/internal/constants"
	"github.com/devtron-labs/devtron/internal/util"
	"github.com/golang-jwt/jwt/v4"
	"github.com/gorilla/sessions"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

type UserAuthService interface {
	HandleLoginWithClientIp(ctx context.Context, username, password, clientIp string) (string, error)
	HandleLogin(username string, password string) (string, error)
	HandleDexCallback(w http.ResponseWriter, r *http.Request)
	HandleRefresh(w http.ResponseWriter, r *http.Request)

	CreateRole(roleData *userBean.RoleData) (bool, error)
	AuthVerification(r *http.Request) (bool, string, error)
	DeleteRoles(entityType string, entityName string, tx *pg.Tx, envIdentifier string, workflowName string) error
}

type UserAuthServiceImpl struct {
	userAuthRepository repository.UserAuthRepository
	// sessionClient is being used for argocd username-password login proxy
	sessionClient       session2.ServiceClient
	logger              *zap.SugaredLogger
	userRepository      repository.UserRepository
	sessionManager      *middleware.SessionManager
	roleGroupRepository repository.RoleGroupRepository
	userService         UserService
}

var (
	cStore         *sessions.CookieStore
	dexOauthConfig *oauth2.Config
	// googleOauthConfig *oauth2.Config
	oauthStateString     = randToken()
	idTokenVerifier      *oidc.IDTokenVerifier
	jwtKey               = randKey()
	CookieExpirationTime int
	JwtExpirationTime    int
)

type User struct {
	email  string
	groups []string
}

var Claims struct {
	Email    string   `json:"email"`
	Verified bool     `json:"email_verified"`
	Groups   []string `json:"groups"`
	Token    string   `json:"token"`
	Roles    []string `json:"roles"`
	jwt.StandardClaims
}

type DexConfig struct {
	RedirectURL          string `env:"DEX_RURL" envDefault:"http://127.0.0.1:8080/callback" description:"Dex redirect URL(http://argocd-dex-server.devtroncd:8080/callback)"`
	ClientID             string `env:"DEX_CID" envDefault:"example-app" description:"dex client id "`
	ClientSecret         string `env:"DEX_SECRET" description:"Dex secret"`
	DexURL               string `env:"DEX_URL" description:"Dex service endpoint with dex path(http://argocd-dex-server.devtroncd:5556/dex)"`
	DexJwtKey            string `env:"DEX_JWTKEY" description:"DEX JWT key.  "`
	CStoreKey            string `env:"DEX_CSTOREKEY" description:"DEX CSTOREKEY."`
	CookieExpirationTime int    `env:"CExpirationTime" envDefault:"600" description:"Caching expiration time."`
	JwtExpirationTime    int    `env:"JwtExpirationTime" envDefault:"120" description:"JWT expiration time."`
}

type WebhookToken struct {
	WebhookToken string `env:"WEBHOOK_TOKEN" envDefault:"" description:"If you want to continue using jenkins for CI then please provide this for authentication of requests"`
}

func NewUserAuthServiceImpl(userAuthRepository repository.UserAuthRepository, sessionManager *middleware.SessionManager,
	client session2.ServiceClient, logger *zap.SugaredLogger, userRepository repository.UserRepository,
	roleGroupRepository repository.RoleGroupRepository, userService UserService) *UserAuthServiceImpl {
	serviceImpl := &UserAuthServiceImpl{
		userAuthRepository:  userAuthRepository,
		sessionManager:      sessionManager,
		sessionClient:       client,
		logger:              logger,
		userRepository:      userRepository,
		roleGroupRepository: roleGroupRepository,
		userService:         userService,
	}
	cStore = sessions.NewCookieStore(randKey())
	return serviceImpl
}

func GetConfig() (*DexConfig, error) {
	cfg := &DexConfig{}
	err := env.Parse(cfg)
	return cfg, err
}

func GetWebhookToken() (*WebhookToken, error) {
	cfg := &WebhookToken{}
	err := env.Parse(cfg)
	return cfg, err
}

/* #nosec */
func randToken() string {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		util.GetLogger().Error(err)
	}
	return base64.StdEncoding.EncodeToString(b)
}

/* #nosec */
func randKey() []byte {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		util.GetLogger().Error(err)
	}
	return b
}

// authorize verifies a bearer token and pulls user information form the claims.
func authorize(ctx context.Context, bearerToken string) (*User, error) {
	idToken, err := idTokenVerifier.Verify(ctx, bearerToken)
	if err != nil {
		return nil, fmt.Errorf("could not verify bearer token: %v", err)
	}
	if err := idToken.Claims(&Claims); err != nil {
		return nil, fmt.Errorf("failed to parse claims: %v", err)
	}
	if !Claims.Verified {
		return nil, fmt.Errorf("email (%q) in returned claims was not verified", Claims.Email)
	}
	return &User{Claims.Email, Claims.Groups}, nil
}

func (impl UserAuthServiceImpl) HandleRefresh(w http.ResponseWriter, r *http.Request) {
	session, _ := cStore.Get(r, "JWT_TOKEN")
	// Check if user is authenticated
	if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
		http.Redirect(w, r, dexOauthConfig.AuthCodeURL(oauthStateString), http.StatusFound)
	} else {
		jwtToken := session.Values["jwtToken"].(string)
		claims := &Claims

		// Parse the JWT string and store the result in `claims`.
		// Note that we are passing the key in this method as well. This method will return an error
		// if the token is invalid (if it has expired according to the expiry time we set on sign in),
		// or if the signature does not match
		tkn, err := jwt.ParseWithClaims(jwtToken, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})
		if !tkn.Valid {
			session.Options = &sessions.Options{
				MaxAge: -1,
			}
			writeResponse(http.StatusUnauthorized, "TOKEN EXPIRED", w, errors.New("token expired"))
			return
		}
		if err != nil {
			if err == jwt.ErrSignatureInvalid {
				writeResponse(http.StatusUnauthorized, "SignatureInvalid", w, errors.New("SignatureInvalid"))
				return
			}
			writeResponse(http.StatusBadRequest, "StatusBadRequest", w, errors.New("StatusBadRequest"))
			return
		}
		claims.Email = util2.ConvertEmailToLowerCase(claims.Email)
		bearerToken := claims.Token
		user, err := authorize(context.Background(), bearerToken)
		if err != nil {
			fmt.Print("Exception :", err)
		}
		fmt.Print(user)

		// We ensure that a new token is not issued until enough time has elapsed
		// In this case, a new token will only be issued if the old token is within
		// 30 seconds of expiry. Otherwise, return a bad request status
		/*if time.Unix(claims.ExpiresAt, 0).Sub(time.Now()) > 30*time.Second {
			w.WriteHeader(http.StatusBadRequest)
			return
		}*/

		dbUser, err := impl.userRepository.FetchUserDetailByEmail(Claims.Email)
		if err != nil {
			impl.logger.Errorw("Exception while fetching user from db", "err", err)
		}
		if dbUser.Id > 0 {
			// Do nothing, User already exist in our db. (unique check by email id)
		} else {
			// TODO - need to handle case
		}

		// Now, create a new token for the current use, with a renewed expiration time
		expirationTime := time.Now().Add(time.Duration(JwtExpirationTime) * time.Second)
		// Create the JWT claims, which includes the username and expiry time
		claims.ExpiresAt = expirationTime.Unix()

		claims.Roles = dbUser.Roles
		// Declare the token with the algorithm used for signing, and the claims
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		// Create the JWT string
		tokenString, err := token.SignedString(jwtKey)
		if err != nil {
			// If there is an error in creating the JWT return an internal server error
			writeResponse(http.StatusInternalServerError, "StatusInternalServerError", w, errors.New("unauthorized"))
			return
		}

		// Set some session values.
		session.Values["jwtToken"] = tokenString
		session.Values["authenticated"] = true
		session.Options = &sessions.Options{
			MaxAge: CookieExpirationTime,
		}
		// Save it before we write to the response/return from the handler.
		session.Save(r, w)

		http.Redirect(w, r, "/", http.StatusFound)
	}
}

func (impl UserAuthServiceImpl) HandleLoginWithClientIp(ctx context.Context, username, password, clientIp string) (string, error) {
	impl.logger.Info("login with client ip")
	token, err := impl.HandleLogin(username, password)
	if err == nil {
		id, _, err := impl.userService.GetUserByToken(ctx, token)
		if err != nil {
			impl.logger.Errorw("error occurred while getting user by token", "err", err)
		} else {
			impl.userService.SaveLoginAudit("", clientIp, id)
		}
	}
	return token, err
}

func (impl UserAuthServiceImpl) HandleLogin(username string, password string) (string, error) {
	return impl.sessionClient.Create(context.Background(), username, password)
}

func (impl UserAuthServiceImpl) HandleDexCallback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	session, _ := cStore.Get(r, "JWT_TOKEN")
	fmt.Print(state)
	// Verify state.

	oauth2Token, err := dexOauthConfig.Exchange(context.Background(), r.URL.Query().Get("code"))
	if err != nil {
		// handle error
	}

	// Extract the ID Token from OAuth2 token.
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		// handle missing token
	}

	// Parse and verify ID Token payload.
	idToken, err := idTokenVerifier.Verify(context.Background(), rawIDToken)
	if err != nil {
		// handle error
	}

	if err := idToken.Claims(&Claims); err != nil {
		// handle error
	}

	dbConnection := impl.userRepository.GetConnection()
	tx, err := dbConnection.Begin()
	if err != nil {
		return
	}
	// Rollback tx on error.
	defer tx.Rollback()

	Claims.Email = util2.ConvertEmailToLowerCase(Claims.Email)
	dbUser, err := impl.userRepository.FetchUserDetailByEmail(Claims.Email)
	if err != nil {
		impl.logger.Errorw("Exception while fetching user from db", "err", err)
	}
	if dbUser.Id > 0 {
		// Do nothing, User already exist in our db. (unique check by email id)
	} else {
		// create new user in our db on d basis of info got from google api or hex. assign a basic role
		model := &repository.UserModel{
			EmailId:     Claims.Email,
			AccessToken: rawIDToken,
		}
		_, err := impl.userRepository.CreateUser(model, tx)
		if err != nil {
			log.Println(err)
		}
		dbUser, err = impl.userRepository.FetchUserDetailByEmail(Claims.Email)
	}
	err = tx.Commit()
	if err != nil {
		return
	}

	// Declare the expiration time of the token
	// here, we have kept it as 5 minutes
	expirationTime := time.Now().Add(time.Duration(JwtExpirationTime) * time.Second)
	// Create the JWT claims, which includes the username and expiry time
	claims := &Claims
	claims.Email = dbUser.EmailId
	claims.Verified = dbUser.Exist
	claims.ExpiresAt = expirationTime.Unix()
	claims.Token = rawIDToken
	claims.Roles = dbUser.Roles
	// Declare the token with the algorithm used for signing, and the claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// Create the JWT string
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		// If there is an error in creating the JWT return an internal server error
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Set some session values.
	session.Values["jwtToken"] = tokenString
	session.Values["authenticated"] = true
	session.Options = &sessions.Options{
		MaxAge: CookieExpirationTime,
	}
	// Save it before we write to the response/return from the handler.
	session.Save(r, w)
	fmt.Print()

	http.Redirect(w, r, "/", http.StatusFound)
}

func WhitelistChecker(url string) bool {
	urls := []string{
		"/health",
		"/metrics",
		"/orchestrator/webhook/ci/gocd/artifact",
		"/orchestrator/auth/login",
		"/orchestrator/auth/callback",
		"/orchestrator/api/v1/session",
		"/orchestrator/app/ci-pipeline/github-webhook/trigger",
		"/orchestrator/webhook/msg/nats",
		"/orchestrator/devtron/auth/verify",
		"/orchestrator/security/policy/verify/webhook",
		"/orchestrator/sso/list",
		"/",
		"/orchestrator/dashboard-event/dashboardAccessed",
		"/orchestrator/dashboard-event/dashboardLoggedIn",
		"/orchestrator/self-register/check",
		"/orchestrator/self-register",
		"/orchestrator/telemetry/summary",
	}
	for _, a := range urls {
		if a == url {
			return true
		}
	}
	prefixUrls := []string{
		"/orchestrator/api/vi/pod/exec/ws",
		"/orchestrator/k8s/pod/exec/sockjs/ws",
		"/orchestrator/api/dex",
		"/orchestrator/auth/callback",
		"/orchestrator/auth/login",
		"/dashboard",
		"/orchestrator/webhook/git",
	}
	for _, a := range prefixUrls {
		if strings.Contains(url, a) {
			return true
		}
	}
	return false
}

func writeResponse(status int, message string, w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	type Response struct {
		Code   int              `json:"code,omitempty"`
		Status string           `json:"status,omitempty"`
		Result interface{}      `json:"result,omitempty"`
		Errors []*util.ApiError `json:"errors,omitempty"`
	}
	response := Response{}
	response.Code = status
	response.Result = message
	b, err := json.Marshal(response)
	if err != nil {
		b = []byte("OK")
		util.GetLogger().Errorw("Unexpected error in apiError", "err", err)
	}
	_, err = w.Write(b)
	if err != nil {
		util.GetLogger().Errorw("error", "err", err)
	}
}

func (impl UserAuthServiceImpl) CreateRole(roleData *userBean.RoleData) (bool, error) {
	roleModel := &repository.RoleModel{
		Role:        roleData.Role,
		Team:        roleData.Team,
		EntityName:  roleData.EntityName,
		Environment: roleData.Environment,
		Action:      roleData.Action,
		Cluster:     roleData.Cluster,
		Namespace:   roleData.Namespace,
		Group:       roleData.Group,
		Kind:        roleData.Kind,
		Resource:    roleData.Resource,
	}
	roleModel, err := impl.userAuthRepository.CreateRole(roleModel)
	if err != nil || roleModel == nil {
		return false, err
	}
	return true, nil
}

func (impl UserAuthServiceImpl) AuthVerification(r *http.Request) (bool, string, error) {
	token := r.Header.Get("token")
	if token == "" {
		impl.logger.Infow("no token provided")
		err := &util.ApiError{
			HttpStatusCode:  http.StatusUnauthorized,
			Code:            constants.UserNoTokenProvided,
			InternalMessage: "no token provided",
		}
		return false, "", err
	}

	_, err := impl.sessionManager.VerifyToken(token)
	if err != nil {
		impl.logger.Errorw("failed to verify token", "error", err)
		err := &util.ApiError{
			HttpStatusCode:  http.StatusUnauthorized,
			Code:            constants.UserNoTokenProvided,
			InternalMessage: "failed to verify token",
			UserMessage:     "token verification failed while getting logged in user",
		}
		return false, "", err
	}
	emailId, version, err := impl.userService.GetEmailAndVersionFromToken(token)
	if err != nil {
		impl.logger.Errorw("AuthVerification failed ", "error", err)
		return false, "", err
	}
	exists := impl.userService.UserExists(emailId)
	if !exists {
		err = &util.ApiError{
			HttpStatusCode:  http.StatusUnauthorized,
			Code:            constants.UserNotFoundForToken,
			InternalMessage: "user does not exist",
			UserMessage:     "active user does not exist",
		}
		return false, "", err
	}
	// checking length of version, to ensure backward compatibility as earlier we did not
	// have version for api-tokens
	// therefore, for tokens without version we will skip the below part
	if strings.HasPrefix(emailId, userBean.API_TOKEN_USER_EMAIL_PREFIX) && len(version) > 0 {
		err := impl.userService.CheckIfTokenIsValid(emailId, version)
		if err != nil {
			impl.logger.Errorw("token is not valid", "error", err, "token", token)
			return false, "", err
		}
	}

	//TODO - extends for other purpose
	return true, emailId, nil
}

func (impl UserAuthServiceImpl) DeleteRoles(entityType string, entityName string, tx *pg.Tx, envIdentifier string, workflowName string) (err error) {
	var roleModels []*repository.RoleModel
	switch entityType {
	case userBean.PROJECT_TYPE:
		roleModels, err = impl.userAuthRepository.GetRolesForProject(entityName)
	case userBean.ENV_TYPE:
		roleModels, err = impl.userAuthRepository.GetRolesForEnvironment(entityName, envIdentifier)
	case userBean.APP_TYPE:
		roleModels, err = impl.userAuthRepository.GetRolesForApp(entityName)
	case userBean.CHART_GROUP_TYPE:
		roleModels, err = impl.userAuthRepository.GetRolesForChartGroup(entityName)
	case userBean.WorkflowType:
		roleModels, err = impl.userAuthRepository.GetRolesForWorkflow(workflowName, entityName)
	}
	if err != nil {
		impl.logger.Errorw(fmt.Sprintf("error in getting roles by %s", entityType), "err", err, "name", entityName)
		return err
	}

	// deleting policies in casbin and roles
	casbin2.LoadPolicy()

	if len(roleModels) > 0 {
		// get roles and roleIds from roleModels for bulk db operations
		roleIds, roles := impl.getRoleIdsAndRolesFromRoleModels(roleModels)

		// deleting policies in casbin and roles
		casbinDeleteFailed := impl.deleteAllUserMappingsForRoleModels(roles)
		if len(casbinDeleteFailed) > 0 {
			impl.logger.Errorw("error in deleting role for user from casbin", "casbinDeleteFailed", casbinDeleteFailed)
		}

		// removing all policies for the role
		success := casbin2.RemovePoliciesByAllRoles(roles)
		if !success {
			impl.logger.Warnw("error in deleting casbin policy for roles", "roles", roles)
			casbinDeleteFailed = append(casbinDeleteFailed, success)
		}
		//deleting user_roles for this role_id (foreign key constraint)
		err = impl.userAuthRepository.DeleteUserRoleByRoleIds(roleIds, tx)
		if err != nil {
			impl.logger.Errorw("error in deleting user_roles by role ids", "err", err, "roleIds", roleIds)
			return err
		}
		//deleting role_group_role_mapping for this role_id (foreign key constraint)
		err = impl.roleGroupRepository.DeleteRoleGroupRoleMappingByRoleIds(roleIds, tx)
		if err != nil {
			impl.logger.Errorw("error in deleting role_group_role_mapping by role ids", "err", err, "roleIds", roleIds)
			return err
		}
		//deleting roles
		err = impl.userAuthRepository.DeleteRolesByIds(roleIds, tx)
		if err != nil {
			impl.logger.Errorw(fmt.Sprintf("error in deleting roles "), "err", err, "role", roleModels)
			return err
		}
	}

	casbin2.LoadPolicy()
	return nil
}

func (impl UserAuthServiceImpl) deleteAllUserMappingsForRoleModels(roles []string) []bool {
	var casbinDeleteFailed []bool
	// deleting all user_role mapping from casbin by getting all users mapped to the role
	for _, role := range roles {
		allUsersMappedToRoles, err := casbin2.GetUserByRole(role)
		if err != nil {
			impl.logger.Errorw("error in getting all users by roles", "err", err, "role", role)
			continue
		}
		for _, rl := range allUsersMappedToRoles {
			success := casbin2.DeleteRoleForUser(rl, role)
			if !success {
				impl.logger.Warnw("error in deleting casbin policy for role", "role", role)
				casbinDeleteFailed = append(casbinDeleteFailed, success)
			}
		}
	}
	return casbinDeleteFailed
}

func (impl UserAuthServiceImpl) getRoleIdsAndRolesFromRoleModels(roleModels []*repository.RoleModel) ([]int, []string) {
	var roleIds []int
	var roles []string
	// deleting all user_role mapping from casbin by getting all users mapped to the role
	for _, roleModel := range roleModels {
		roleIds = append(roleIds, roleModel.Id)
		roles = append(roles, roleModel.Role)
	}
	return roleIds, roles
}
