package cognito

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws/awserr"

	// "github.com/davecgh/go-spew/spew"
	"github.com/gobuffalo/envy"
)

var CFG = &AppClientConfig{}
var cognitoID = ""

func init() {
	r, err := envy.MustGet("REGION")
	p, err := envy.MustGet("POOL_ID")
	c, err := envy.MustGet("CLIENT_ID")
	if err != nil {
		os.Exit(1)
	}

	CFG.Region = r
	CFG.PoolID = p
	CFG.ClientID = c
}

func TestNewAppClient(t *testing.T) {

	client, err := NewAppClient(CFG)
	assert.Nil(t, err, "Error not nil")
	assert.Equal(t, CFG.ClientID, client.ClientID)
}

func TestAuthenticateUserPassword(t *testing.T) {
	// Setup the credentials from Environment:
	u, err := envy.MustGet("USERNAME")
	p, err := envy.MustGet("PASSWORD")
	cid, err := envy.MustGet("COGNITO_ID")
	assert.Nil(t, err, "Could not get credentials from .env")

	credentials := &Credentials{
		Username: u,
		Password: p,
	}
	failCredentials := &Credentials{
		Username: u + "fail",
		Password: p + "fail",
	}
	cognitoID = cid

	client, err := NewAppClient(CFG)
	assert.Nil(t, err, "Error not nil")

	response, err := client.AuthenticateUserPassword(credentials)
	assert.Nil(t, err, "Error not nil")
	assert.Equal(t, cognitoID, response)

	response, err = client.AuthenticateUserPassword(failCredentials)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			assert.Equal(t, awsErr.Message(), "User does not exist.")
		}
	}

}
