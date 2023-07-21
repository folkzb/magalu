MagaLu CLI
==========

This project holds Magalu Cloud CLI. It allows users of the Cloud to control
their resources using a simple command line interface.

## Goals

- [ ] Allow a user to authenticate itself
- [ ] Save user preferences
- [ ] Generate commands at runtime (No need for new binaries)

## Development

### Install

Install dependencies using:

```sh
go install
```

### Build

Build the command line using:

```sh
go build
```

### Execute

To see what commands are currently accepted

```sh
./cli help
```

## Postman

### Setup

To properly setup Postman follow the steps bellow:

- Import the collection file in Postman: `File -> Import...`
- Click on the MGC collection and go to the `Variables` tab
- Add the ClientID, Client Secret and Client UUID
- Go to the `Authorization` tab
- Click on `Create New Token`
  - Authorize all scopes if requested
- On the `Current token` section, select `mgc_access_token` for the Token property
- Enable the `Auto-refresh token` option

Now using the VPN you should be able to make a request to the MGC servers.

### FAQ

**My request is not using the latest token**

Make sure under the MGC Collection `Authorization` tab there is only one token
on the `Token` property in the `Current Token` section. If there is more than
one click on `Manage Tokens` and remove all of them.

## Testing

> TODO
