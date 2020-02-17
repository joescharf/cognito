// Adapted from tmaiaroto/aegis/framework/cognito_client.go

package cognito

import (
	"bytes"
	"context"
	"crypto/rsa"
	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"

	// "github.com/davecgh/go-spew/spew"
	"github.com/dgrijalva/jwt-go"
	"github.com/lestrrat-go/jwx/jwk"
	// log "github.com/sirupsen/logrus"
)

// AppClient is an interface for working with AWS Cognito
type AppClient struct {
	AWSAccessKey             string
	AWSSecretAccessKey       string
	Region                   string
	UserPoolID               string
	ClientID                 string
	ClientSecret             string
	Domain                   string
	WellKnownJWKs            *jwk.Set
	BaseURL                  string
	HostedLoginURL           string
	HostedLogoutURL          string
	HostedSignUpURL          string
	RedirectURI              string
	LogoutRedirectURI        string
	TokenEndpoint            string
	Base64BasicAuthorization string
}

// AppClientConfig defines required info to build a new AppClient
type AppClientConfig struct {
	AWSAccessKey       string
	AWSSecretAccessKey string
	Region             string                 `json:"region"`
	PoolID             string                 `json:"poolId"`
	Domain             string                 `json:"domain"`
	ClientID           string                 `json:"clientId"`
	ClientSecret       string                 `json:"clientSecret"`
	RedirectURI        string                 `json:"redirectUri"`
	LogoutRedirectURI  string                 `json:"logoutRedirectUri"`
	TraceContext       context.Context        `json:"-"`
	AWSClientTracer    func(c *client.Client) `json:"-"`
}

// Token defines a token struct for JSON responses from Cognito TOKEN endpoint
type Token struct {
	IDToken      string `json:"id_token"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Error        string `json:"error"`
}

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// NewAppClient returns a new AppClient interface configured for the given Cognito user pool and client
func NewAppClient(cfg *AppClientConfig) (*AppClient, error) {
	var err error
	c := &AppClient{
		AWSAccessKey:       cfg.AWSAccessKey,
		AWSSecretAccessKey: cfg.AWSSecretAccessKey,
		Region:             cfg.Region,
		UserPoolID:         cfg.PoolID,
		ClientID:           cfg.ClientID,
		ClientSecret:       cfg.ClientSecret,
		Domain:             cfg.Domain,
		RedirectURI:        cfg.RedirectURI,
		LogoutRedirectURI:  cfg.LogoutRedirectURI,
	}

	if c.ClientSecret != "" {
		// Set the Base64 <client_id>:<client_secret> for basic authorization header
		var buffer bytes.Buffer
		buffer.WriteString(c.ClientID)
		buffer.WriteString(":")
		buffer.WriteString(c.ClientSecret)
		base64AuthStr := b64.StdEncoding.EncodeToString(buffer.Bytes())
		buffer.Reset()

		buffer.WriteString("Basic ")
		buffer.WriteString(base64AuthStr)
		c.Base64BasicAuthorization = buffer.String()
		buffer.Reset()

		// Set up login and signup URLs, if there is a domain available
		c.getURLs()
	}

	// Set the well known JSON web token key sets
	err = c.getWellKnownJWTKs()
	if err != nil {
		log.Println("Error getting well known JWTKs", err)
	}

	return c, err
}

// getWellKnownJWTKs gets the well known JSON web token key set for this client's user pool
func (c *AppClient) getWellKnownJWTKs() error {
	// https://cognito-idp.<region>.amazonaws.com/<pool_id>/.well-known/jwks.json
	var buffer bytes.Buffer
	buffer.WriteString("https://cognito-idp.")
	buffer.WriteString(c.Region)
	buffer.WriteString(".amazonaws.com/")
	buffer.WriteString(c.UserPoolID)
	buffer.WriteString("/.well-known/jwks.json")
	wkjwksURL := buffer.String()
	buffer.Reset()

	// Use this cool package
	set, err := jwk.Fetch(wkjwksURL)
	if err == nil {
		c.WellKnownJWKs = set
	} else {
		log.Println("There was a problem getting the well known JSON web token key set")
		log.Println(err)
	}
	return err
}

// getURLs gets all of the URLs and endpoints for the Cognito client, AWS hosted login/signup pages, token endpoints for oauth2, etc.
func (c *AppClient) getURLs() {
	if c.Domain != "" {
		// Get the base URL
		var buffer bytes.Buffer
		buffer.WriteString("https://")
		buffer.WriteString(c.Domain)
		buffer.WriteString(".auth.")
		buffer.WriteString(c.Region)
		buffer.WriteString(".amazoncognito.com")
		baseURL := buffer.String()
		c.BaseURL = baseURL
		buffer.Reset()

		// Set the HostedLoginURL
		buffer.WriteString(baseURL)
		buffer.WriteString("/login?response_type=code&client_id=")
		buffer.WriteString(c.ClientID)
		buffer.WriteString("&redirect_uri=")
		buffer.WriteString(c.RedirectURI)
		c.HostedLoginURL = buffer.String()
		buffer.Reset()

		// Set the HostedLogoutURL
		buffer.WriteString(baseURL)
		buffer.WriteString("/logout?response_type=code&client_id=")
		buffer.WriteString(c.ClientID)
		buffer.WriteString("&redirect_uri=")
		buffer.WriteString(c.RedirectURI)
		c.HostedLogoutURL = buffer.String()
		buffer.Reset()

		// Set the HostedSignUpURL
		buffer.WriteString(baseURL)
		buffer.WriteString("/signup?response_type=code&client_id=")
		buffer.WriteString(c.ClientID)
		buffer.WriteString("&redirect_uri=")
		buffer.WriteString(c.RedirectURI)
		c.HostedSignUpURL = buffer.String()
		buffer.Reset()

		// Set the authorization token URL
		buffer.WriteString(c.BaseURL)
		buffer.WriteString("/oauth2/token")
		c.TokenEndpoint = buffer.String()
		buffer.Reset()
	}
}

// GetTokens will make a POST request to the Cognito TOKEN endpoint to exchange a code for an access token
func (c *AppClient) GetTokens(code string, scope []string) (Token, error) {
	var token Token

	hc := http.Client{}
	// set the url-encoded payload
	form := url.Values{}
	form.Set("code", code)
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", c.ClientID)
	form.Set("redirect_uri", c.RedirectURI)
	if len(scope) > 0 {
		form.Set("scope", strings.Join(scope, " "))
	}
	// request
	req, err := http.NewRequest("POST", c.TokenEndpoint, strings.NewReader(form.Encode()))
	if err == nil {
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		// This should be a string like: Basic XXXXXXXXXX
		req.Header.Add("Authorization", c.Base64BasicAuthorization)

		resp, err := hc.Do(req)
		if err != nil {
			log.Println("Could not make request to Cognito TOKEN endpoint")
			return token, err
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println("Could not read response body from Cognito TOKEN endpoint")
			return token, err
		}

		err = json.Unmarshal(body, &token)
		if err != nil {
			log.Println("Could not unmarshal token response from Cognito TOKEN endpoint")
		}
	} else {
		log.Println("Error making HTTP request", err)
	}
	return token, err
}

// ParseAndVerifyJWT will parse and verify a JWT, if an error is returned the token is invalid,
// only a valid token will be returned
//
// https://github.com/awslabs/aws-support-tools/tree/master/Cognito/decode-verify-jwt
// Amazon Cognito returns three tokens: the ID token, access token, and refresh token—the ID token
// contains the user fields defined in the Amazon Cognito user pool.
//
// To verify the signature of an Amazon Cognito JWT, search for the key with a key ID that matches
// the key ID of the JWT, then use libraries to decode the token and verify the signature.
//
// Be sure to also verify that:
//  - The token is not expired.
//  - The audience ("aud") in the payload matches the app client ID created in the Cognito user pool.
func (c *AppClient) ParseAndVerifyJWT(t string) (*jwt.Token, error) {
	// 3 tokens are returned from the Cognito TOKEN endpoint; "id_token" "access_token" and "refresh_token"
	token, err := jwt.Parse(t, func(token *jwt.Token) (interface{}, error) {
		// Looking up the key id will return an array of just one key
		keys := c.WellKnownJWKs.LookupKeyID(token.Header["kid"].(string))
		if len(keys) == 0 {
			log.Println("Failed to look up JWKs")
			return nil, errors.New("could not find matching `kid` in well known tokens")
		}
		// Build the public RSA key
		key, err := keys[0].Materialize()
		if err != nil {
			log.Printf("Failed to create public key: %s", err)
			return nil, err
		}
		rsaPublicKey := key.(*rsa.PublicKey)
		return rsaPublicKey, nil
	})

	// Populated when you Parse/Verify a token
	// First verify the token itself is a valid format
	if err == nil && token.Valid {
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			// Then check time based claims; exp, iat, nbf
			err = claims.Valid()
			if err == nil {
				// Then check that `aud` matches the app client id
				// (if `aud` even exists on the token, second arg is a "required" option)
				if claims.VerifyAudience(c.ClientID, false) {
					return token, nil
				} else {
					err = errors.New("token audience does not match client id")
					log.Println("Invalid audience for id token")
				}
			} else {
				log.Println("Invalid claims for id token")
				log.Println(err)
			}
		}
	} else {
		log.Println("Invalid token:", err)
	}

	return nil, err
}

func (c *AppClient) NewCIP() (cip *cognitoidentityprovider.CognitoIdentityProvider, err error) {

	var ses *session.Session

	// Setup the AWS session with or without AWS credentials:
	if c.AWSAccessKey != "" && c.AWSSecretAccessKey != "" {
		credentials := credentials.NewStaticCredentials(c.AWSAccessKey, c.AWSSecretAccessKey, "")
		ses, err = session.NewSession(&aws.Config{
			Credentials: credentials,
			Region:      aws.String(c.Region),
		})
	} else {
		ses, err = session.NewSession(&aws.Config{
			Region: aws.String(c.Region),
		})
	}
	if err != nil {
		return cip, err
	}

	// Create the CognitoIdentityProvider Client
	cip = cognitoidentityprovider.New(ses)
	return cip, err
}

func (c *AppClient) AuthenticateUserPassword(credentials *Credentials) (cognitoID string, err error) {
	username := aws.String(credentials.Username)
	password := aws.String(credentials.Password)
	clientID := aws.String(c.ClientID)

	params := &cognitoidentityprovider.InitiateAuthInput{
		AuthFlow: aws.String("USER_PASSWORD_AUTH"),
		AuthParameters: map[string]*string{
			"USERNAME": username,
			"PASSWORD": password,
		},
		ClientId: clientID,
	}

	// Create the CognitoIdentityProvider
	cip, err := c.NewCIP()
	if err != nil {
		return
	}

	// Authenticate
	token, err := cip.InitiateAuth(params)
	if err != nil {
		return
	}

	// Now we need to get the cognito id from the IDToken Claims (sub)
	idToken, err := c.ParseAndVerifyJWT(*token.AuthenticationResult.IdToken)
	if idClaims, ok := idToken.Claims.(jwt.MapClaims); ok && idToken.Valid {
		cognitoID = idClaims["sub"].(string)
	}

	return cognitoID, err

}

// RegisterNewUserEmailPass creates a new user in cognito based on a username (email) and password
// If password is null, then cognito will create the temporary password for you.
// Requires a AWS session with developer credentials
func (c *AppClient) RegisterNewUserEmailPass(username, password string) (cognitoID string, err error) {

	emailAt := &cognitoidentityprovider.AttributeType{
		Name:  aws.String("email"),
		Value: aws.String(username),
	}
	emailVerifiedAt := &cognitoidentityprovider.AttributeType{
		Name:  aws.String("email_verified"),
		Value: aws.String("true"),
	}

	if password != "" {
		input := &cognitoidentityprovider.AdminCreateUserInput{
			Username:          aws.String(username),
			TemporaryPassword: aws.String(password),
			UserPoolId:        &c.UserPoolID,
			UserAttributes:    []*cognitoidentityprovider.AttributeType{emailAt, emailVerifiedAt},
		}
	} else {
		input := &cognitoidentityprovider.AdminCreateUserInput{
			Username:       aws.String(username),
			UserPoolId:     &c.UserPoolID,
			UserAttributes: []*cognitoidentityprovider.AttributeType{emailAt, emailVerifiedAt},
		}
	}

	// Create the CognitoIdentityProvider
	cip, err := c.NewCIP()
	if err != nil {
		return
	}
	out, err := cip.AdminCreateUser(input)
	if err != nil {
		return
	}
	cognitoID = *out.User.Username

	return
}

func (c *AppClient) DeleteUser(username string) error {

	input := &cognitoidentityprovider.AdminDeleteUserInput{
		Username:   aws.String(username),
		UserPoolId: &c.UserPoolID,
	}

	// Create the CognitoIdentityProvider
	cip, err := c.NewCIP()
	if err != nil {
		return err
	}
	_, err = cip.AdminDeleteUser(input)

	return err

}
