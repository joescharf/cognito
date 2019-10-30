joescharf/cognito

## Intro

A quick and dirty cognito module for authenticating JWTs and doing
Cognito Username / Password authentication 

Adapted from tmaiaroto/aegis/framework/cognito_client.go

## Testing 

requires .env file with following settings:

```
REGION:   "aws-region-name"
POOL_ID:   "pool_id"
CLIENT_ID: "client_id"

USERNAME: "valid user in cognito"
PASSWORD: "password for user in cognito"
COGNITO_ID: "cognito id of user"
```