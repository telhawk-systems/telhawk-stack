# TelHawk CLI User Management

The `thawk` CLI provides complete user management capabilities for TelHawk Stack.

## Prerequisites

Login to the system first:
```bash
./scripts/thawk auth login -u admin -p admin123 --auth-url http://auth:8080
```

The configuration is now persisted in a Docker volume, so you only need to login once.

## User Management Commands

### List All Users
```bash
./scripts/thawk user list
```

Output in table format by default. Use `--output json` for JSON format.

### Get User Details
```bash
./scripts/thawk user get <user-id>
```

### Create a New User
```bash
./scripts/thawk user create \
  -u username \
  -e email@example.com \
  -p password123 \
  -r admin,analyst,viewer
```

Available roles: `admin`, `analyst`, `viewer`, `ingester`

### Update User
```bash
# Update email
./scripts/thawk user update <user-id> -e newemail@example.com

# Update roles
./scripts/thawk user update <user-id> -r analyst,viewer

# Enable user
./scripts/thawk user update <user-id> --enabled

# Disable user
./scripts/thawk user update <user-id> --disabled
```

### Reset User Password
```bash
./scripts/thawk user reset-password <user-id> -p newpassword123
```

### Delete User
```bash
./scripts/thawk user delete <user-id> --force
```

Note: `--force` flag is required to confirm deletion.

## Other Authentication Commands

### Check Current User
```bash
./scripts/thawk auth whoami
```

### Logout
```bash
./scripts/thawk auth logout
```

## Configuration Persistence

The CLI configuration is stored in a Docker volume named `telhawk-stack_thawk-config`. This persists your authentication tokens between CLI invocations.

To clear the configuration:
```bash
docker volume rm telhawk-stack_thawk-config
```

## Examples

### Create an analyst user
```bash
./scripts/thawk user create \
  -u jdoe \
  -e jdoe@company.com \
  -p SecurePass123! \
  -r analyst,viewer
```

### Disable a user account
```bash
./scripts/thawk user update 7f317d44-9b9e-402d-a5c0-d50d026485e5 --disabled
```

### List all users in JSON format
```bash
./scripts/thawk user list --output json
```
