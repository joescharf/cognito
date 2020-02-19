package cognito

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/stretchr/testify/assert"

	"github.com/gobuffalo/envy"
)

type IntegrationTests struct{ Test *testing.T }

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

// go test -run Integration
func TestIntegration(t *testing.T) {
	t.Run("Integration Tests", func(t *testing.T) {
		test := IntegrationTests{Test: t}
		test.TestNewAppClient()
		test.TestRegisterNewUserEmailPass()
		test.TestSetUserPassword()
		test.TestAddUserToGroup()

		time.Sleep(5 * time.Second)
		test.TestGetUserGroups()
		time.Sleep(10 * time.Second)
		test.TestAuthenticateUserPassword()
		test.TestDeleteUser()
	})
}

func (t *IntegrationTests) TestNewAppClient() {
	fmt.Println("1. TestNewAppClient")
	client, err := NewAppClient(CFG)
	assert.Nil(t.Test, err, "Error not nil")
	assert.Equal(t.Test, CFG.ClientID, client.ClientID)
}

func (t *IntegrationTests) TestRegisterNewUserEmailPass() {
	fmt.Println("2. TestRegisterNewUserEmailPass")
	// Setup the credentials from Environment:
	u, err := envy.MustGet("USERNAME")
	p, err := envy.MustGet("PASSWORD")
	assert.Nil(t.Test, err, "Could not get credentials from .env")

	client, err := NewAppClient(CFG)
	assert.Nil(t.Test, err, "Error not nil")

	_, err = client.RegisterNewUserEmailPass(u, p)
	assert.Nil(t.Test, err, "Error not nil")
}

func (t *IntegrationTests) TestSetUserPassword() {
	fmt.Println("3. TestSetUserPassword")
	// Setup the credentials from Environment:
	u, err := envy.MustGet("USERNAME")
	p, err := envy.MustGet("PASSWORD")
	assert.Nil(t.Test, err, "Could not get credentials from .env")

	client, err := NewAppClient(CFG)
	assert.Nil(t.Test, err, "Error not nil")

	// Set permanent password
	err = client.SetUserPassword(u, p, true)
	assert.Nil(t.Test, err, "Error setting permanent user password")

}

func (t *IntegrationTests) TestConfirmUser() {
	// Setup the credentials from Environment:
	u, err := envy.MustGet("USERNAME")
	assert.Nil(t.Test, err, "Could not get credentials from .env")

	client, err := NewAppClient(CFG)
	assert.Nil(t.Test, err, "Error not nil")

	err = client.ConfirmUser(u)
	assert.Nil(t.Test, err, "Error confirming User")
}

func (t *IntegrationTests) TestAuthenticateUserPassword() {
	fmt.Println("6. TestAuthenticateUserPassword")
	// Setup the credentials from Environment:
	u, err := envy.MustGet("USERNAME")
	p, err := envy.MustGet("PASSWORD")
	assert.Nil(t.Test, err, "Could not get credentials from .env")

	credentials := &Credentials{
		Username: u,
		Password: p,
	}
	failCredentials := &Credentials{
		Username: u + "fail",
		Password: p + "fail",
	}

	client, err := NewAppClient(CFG)
	assert.Nil(t.Test, err, "Error not nil")

	_, err = client.AuthenticateUserPassword(credentials)
	assert.Nil(t.Test, err, "Error not nil")

	_, err = client.AuthenticateUserPassword(failCredentials)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			assert.Equal(t.Test, "Incorrect username or password.", awsErr.Message())
		}
	}
}
func (t *IntegrationTests) TestAddUserToGroup() {
	fmt.Println("4. TestAddUserToGroup")
	// Setup the credentials from Environment:
	u, err := envy.MustGet("USERNAME")
	g, err := envy.MustGet("GROUP")
	assert.Nil(t.Test, err, "Could not get credentials from .env")

	client, err := NewAppClient(CFG)
	assert.Nil(t.Test, err, "Error not nil")

	err = client.AddUserToGroup(u, g)
	assert.Nil(t.Test, err, "Error adding user to group")
}

func (t *IntegrationTests) TestGetUserGroups() {
	fmt.Println("5. TestGetUserGroups")
	// Setup the credentials from Environment:
	u, err := envy.MustGet("USERNAME")
	g, err := envy.MustGet("GROUP")
	assert.Nil(t.Test, err, "Could not get credentials from .env")

	client, err := NewAppClient(CFG)
	assert.Nil(t.Test, err, "Error not nil")

	groups, err := client.GetUserGroups(u)
	assert.Nil(t.Test, err, "Error adding user to group")
	assert.True(t.Test, client.InGroup(groups, g), "Group not found")
}

func (t *IntegrationTests) TestDeleteUser() {
	fmt.Println("7. TestDeleteUser")
	// Setup the credentials from Environment:
	u, err := envy.MustGet("USERNAME")
	assert.Nil(t.Test, err, "Could not get credentials from .env")

	client, err := NewAppClient(CFG)
	assert.Nil(t.Test, err, "Error not nil")

	err = client.DeleteUser(u)
	assert.Nil(t.Test, err, "Error deleting User")

}
