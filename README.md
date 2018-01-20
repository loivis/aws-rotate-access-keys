# aws-rotate-access-keys

AWS lambda function written in go to rotate access keys. Resources are created with terraform.

## TODO

+ [x] check access keys for all iam users

+ [x] delete keys created (expiration + 30) days ago

+ [x] deactive keys created expiration(90 by default) days ago

+ [x] send slack notification if key was created (expiration - 7) days ago

- [ ] list keys for slack user

- [ ] generate new key for slack user

- [ ] delete keys for slack user

## bash

```
GOOS=linux go build -o main main.go && build-lambda-zip -o aws-rotate-access-keys.zip main
```

## fish

```
set -x GOOS linux; and go build -o main main.go; and build-lambda-zip -o aws-rotate-access-keys.zip main
```
