package cognito

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
)

func (c *AppClient) AddUserToGroup(username, group string) error {
	input := &cognitoidentityprovider.AdminAddUserToGroupInput{
		Username:   aws.String(username),
		GroupName:  aws.String(group),
		UserPoolId: &c.UserPoolID,
	}

	// Create the CognitoIdentityProvider
	cip, err := c.NewCIP()
	if err != nil {
		return err
	}

	// Add the user to the group:
	_, err = cip.AdminAddUserToGroup(input)

	if err != nil {
		return err
	}

	return nil
}

func (c *AppClient) GetUserGroups(username string) ([]*cognitoidentityprovider.GroupType, error) {
	input := &cognitoidentityprovider.AdminListGroupsForUserInput{
		Username:   aws.String(username),
		UserPoolId: &c.UserPoolID,
	}

	// Create the CognitoIdentityProvider
	cip, err := c.NewCIP()
	if err != nil {
		return nil, err
	}

	// Get the groups
	out, err := cip.AdminListGroupsForUser(input)

	if err != nil {
		return nil, err
	}

	return out.Groups, err
}

func (c *AppClient) InGroup(groupType []*cognitoidentityprovider.GroupType, group string) bool {
	for _, gt := range groupType {
		if *gt.GroupName == group {
			return true
		}
	}
	return false
}

func (c *AppClient) ConfirmUser(username string) error {
	input := &cognitoidentityprovider.AdminConfirmSignUpInput{
		Username:   aws.String(username),
		UserPoolId: &c.UserPoolID,
	}

	// Create the CognitoIdentityProvider
	cip, err := c.NewCIP()
	if err != nil {
		return err
	}

	// Confirm the signup
	_, err = cip.AdminConfirmSignUp(input)
	if err != nil {
		return err
	}

	return nil
}
func (c *AppClient) SetUserPassword(username, password string, permanent bool) error {
	input := &cognitoidentityprovider.AdminSetUserPasswordInput{
		Username:   aws.String(username),
		Password:   aws.String(password),
		Permanent:  aws.Bool(permanent),
		UserPoolId: &c.UserPoolID,
	}

	// Create the CognitoIdentityProvider
	cip, err := c.NewCIP()
	if err != nil {
		return err
	}

	// Set the password
	_, err = cip.AdminSetUserPassword(input)
	if err != nil {
		return err
	}

	return nil
}

// RegisterNewUserEmailPass creates a new user in cognito based on a username (email) and password
// If password is null, then cognito will create the temporary password for you.
// Requires a AWS session with developer credentials
func (c *AppClient) RegisterNewUserEmailPass(username, password string) (cognitoID string, err error) {

	var input *cognitoidentityprovider.AdminCreateUserInput

	emailAt := &cognitoidentityprovider.AttributeType{
		Name:  aws.String("email"),
		Value: aws.String(username),
	}
	emailVerifiedAt := &cognitoidentityprovider.AttributeType{
		Name:  aws.String("email_verified"),
		Value: aws.String("true"),
	}

	if password != "" {
		input = &cognitoidentityprovider.AdminCreateUserInput{
			Username:          aws.String(username),
			TemporaryPassword: aws.String(password),
			UserPoolId:        &c.UserPoolID,
			UserAttributes:    []*cognitoidentityprovider.AttributeType{emailAt, emailVerifiedAt},
		}
	} else {
		input = &cognitoidentityprovider.AdminCreateUserInput{
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
