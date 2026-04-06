# rtm-housekeeper

A CLI tool that automatically organizes tasks in [Remember The Milk](https://www.rememberthemilk.com/).

## Features

### Archive stale someday tasks

Moves incomplete tasks in the specified lists that haven't been updated for a given number of days (default: 60) to the `archive` list. Target lists and the threshold can be configured via flags.

### Archive tagged tasks

Moves incomplete tasks with the `archive` tag to the `archive` list. Just tag any task you want to archive.

### Delete tagged tasks

Deletes incomplete tasks with the `trash` tag.

### Note on the archive list

You can archive the `archive` list using RTM's built-in "Archive list" feature to hide it from searches and the "All Tasks" view. Moving tasks to an archived list via the API works without issues.

## Setup

### 1. Get an API key

Apply for a non-commercial API key from the [Remember The Milk API key page](https://www.rememberthemilk.com/services/api/keys.rtm).

### 2. Set environment variables

Set the following environment variables with your API key and Shared Secret.

| Variable | Description |
|---|---|
| `RTM_API_KEY` | Your API application key |
| `RTM_SHARED_SECRET` | Your API shared secret |
| `RTM_AUTH_TOKEN` | Auth token (obtained in step 3) |

### 3. Authenticate

```sh
go run ./cmd/auth
```

Follow the browser prompt to authorize the app, then set the displayed token as `RTM_AUTH_TOKEN`.

## Usage

```sh
# Run (actually moves/deletes tasks)
go run .

# Dry run (shows affected tasks without making changes)
go run . -dry-run

# Specify multiple someday lists
go run . -someday-lists someday,work_someday

# Change the stale threshold (default: 60 days)
go run . -stale-days 30
```

### Flags

| Flag | Default | Description |
|---|---|---|
| `-dry-run` | `false` | Show affected tasks without making changes |
| `-someday-lists` | `someday` | Comma-separated list names to auto-archive stale tasks from |
| `-stale-days` | `60` | Number of days since last update to consider a task stale |

## License

MIT
