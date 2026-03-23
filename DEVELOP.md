# Kiddo Development Guide

This guide is for developers contributing to or extending the Kiddo service.

## Project Structure

```
kiddo/
├── main.go                      # Service entry point
├── internal/
│   ├── app/
│   │   └── program.go          # Main service loop & orchestration
│   ├── config/
│   │   ├── config.go           # Configuration & rule parsing
│   │   └── manager.go          # GitHub polling & caching
│   ├── enforcer/
│   │   └── enforcer.go         # User login time enforcement
│   ├── scheduler/
│   │   └── scheduler.go        # PC shutdown scheduling
│   ├── cleanup/
│   │   └── cleanup.go          # Unauthorized user removal
│   └── logger/
│       └── logger.go           # Logging setup
├── go.mod                       # Go module definition
├── Makefile                     # Build targets (cross-platform)
├── build.bat                    # Windows build script
├── config.example.json          # Example configuration
├── schedules.example.csv        # Example schedule
├── README.md                    # User documentation
├── INSTALL.md                   # Installation guide
└── DEVELOP.md                   # Developer guide (this file)
```

## Development Setup

### Prerequisites

- Go 1.21 or later
- Git
- Windows 10/11 (for testing service features)

### Clone & Setup

```bash
git clone https://github.com/yourusername/kiddo.git
cd kiddo

# Download dependencies
go mod download

# Run tests
go test ./...

# Build for Windows
GOOS=windows GOARCH=amd64 go build -o bin/kiddo.exe .

# Or on Windows:
.\build.bat
```

## Component Guide

### 1. Logger (`internal/logger/`)

**File**: `logger.go`

Handles all logging. Configuration:
- Output: `C:\ProgramData\Kiddo\logs\kiddo.log`
- Format: JSON (structured logging)
- Rotation: Daily, 7-day retention
- Init: Must call `logger.Init()` in main.go

**Usage**:
```go
import "github.com/look/kiddo/internal/logger"

var log = logger.Get()

// Methods: Debugf, Infof, Warnf, Errorf, Fatalf
log.Infof("User %s logged in", username)
log.Errorf("Failed to fetch config: %v", err)
```

### 2. Config (`internal/config/`)

**Files**:
- `config.go` - Structures, parsing, file I/O
- `manager.go` - GitHub polling

**Key Types**:
```go
type Rule struct {
    Username  string
    DayOfWeek string
    StartHour int
    EndHour   int
    ParsedAt  time.Time
}

type AppConfig struct {
    GitHubToken string
    GitHubRepo  string
    // ... see config.go for full struct
}
```

**Key Functions**:
- `LoadConfig(path)` - Load JSON config
- `ParseRules(csv)` - Parse CSV schedule
- `Manager.Fetch()` - Poll GitHub with ETag caching
- `LoadRulesFromCache()` - Fallback to local cache

**GitHub Fetch Logic**:
1. Construct raw GitHub URL
2. Add If-None-Match header with cached ETag
3. If 304 Not Modified → use cache
4. If 200 → parse rules, save ETag, update cache
5. On error → fall back to cache

### 3. Policy Enforcer (`internal/enforcer/`)

**File**: `enforcer.go`

Enforces login time restrictions.

**Key Functions**:
- `Enforce()` - Check and apply current rules
- `SetUserLoginHours(username, startHour, endHour)` - Set allowed hours
- `DisableUser(username)` - Disable account
- `EnableUser(username)` - Enable account
- `GetLocalUsers()` - List local users

**Future Enhancement Ideas**:
- Group Policy Object (GPO) integration for login hours
- Profile management per user
- Real-time user session monitoring

### 4. Scheduler (`internal/scheduler/`)

**File**: `scheduler.go`

Manages PC shutdown scheduling.

**Key Functions**:
- `ScheduleShutdowns()` - Create scheduled tasks
- `CheckAndExecuteShutdowns()` - Check if shutdown time reached
- `scheduleUserShutdown(username, endHour)` - Create single task

**Implementation Details**:
- Uses Windows Task Scheduler (`schtasks.exe`)
- Creates daily recurring task per user
- `shutdown /s /t 300` = 5 minute warning

**Example Task**:
```
schtasks /create \
  /tn "Kiddo_Shutdown_Janek" \
  /tr "shutdown /s /t 300" \
  /sc daily \
  /st 21:00 \
  /f
```

### 5. User Cleanup (`internal/cleanup/`)

**File**: `cleanup.go`

Removes unauthorized user accounts.

**Key Functions**:
- `Clean()` - Run cleanup cycle
- `getLocalUsers()` - Enumerate users
- `logoffUser(username)` - Force logoff
- `deleteUser(username)` - Delete account

**Protected Accounts**:
Will never delete (hardcoded list):
- Administrator
- Guest
- System
- Network Service
- WDAGUtilityAccount

**Example Workflow**:
1. Build list of authorized users from rules
2. Enumerate all local users
3. For each unauthorized user:
   - Log them off (`logoff username`)
   - Delete account (`net user username /delete`)

### 6. Main Program (`internal/app/`)

**File**: `program.go`

The service orchestrator implementing `service.Service` interface.

**Key Methods**:
- `Start(svc)` - Initialize and start service loop
- `Stop(svc)` - Gracefully stop service
- `run()` - Main event loop

**Event Loop Timing**:
```
Config Check: Every 60s (configurable)
Cleanup Run:  Every 300s (configurable)
Enforcement:  With each check
```

**State Management**:
```go
type Program struct {
    cfg       *config.AppConfig      // Configuration
    cfgMgr    *config.Manager        // GitHub polling
    enf       *enforcer.Enforcer     // Policy enforcement
    sched     *scheduler.Scheduler   // Shutdown scheduling
    clean     *cleanup.Cleaner       // User cleanup
    stopChan  chan struct{}          // Graceful shutdown
    ticker    *time.Ticker           // Check timer
}
```

## Testing

### Unit Tests

```bash
go test ./internal/config -v
go test ./internal/enforcer -v
go test ./...
```

### Test Cases to Add

1. **Config Parsing**
   - Valid CSV with various day formats
   - Invalid CSV (missing fields, bad hours)
   - ETag caching / cache hit detection

2. **Enforcement**
   - Time matching for different days
   - User disable/enable commands
   - Edge cases (midnight, hours 0-23)

3. **Cleanup**
   - Protected account filtering
   - User enumeration parsing
   - Account deletion

### Integration Testing

1. **Mock GitHub Responses**
   ```go
   // Use httptest.Server to mock GitHub API
   ```

2. **Test Service Loop**
   - Verify tick intervals
   - Verify component updates

3. **Windows-Specific Testing**
   - User account creation/deletion
   - Scheduled task creation
   - Service startup/stop

## Common Development Tasks

### Add New Configuration Option

1. Add field to `AppConfig` struct in `config/config.go`
2. Update `DefaultConfig()` with default value
3. Add to JSON example in `config.example.json`

### Add New Enforcement Rule

1. Modify `Rule` struct in `config/config.go`
2. Update `ParseRules()` to handle new format
3. Update enforcer logic in `enforcer/enforcer.go`
4. Add tests

### Improve Logging

1. All logging uses `logger.Get()` pattern
2. Change log level in config to "debug" for verbose output
3. Logs are JSON — use `jq` to parse:
   ```bash
   cat logs/kiddo.log | jq '.level' | sort | uniq -c
   ```

### Cross-Platform Testing

```bash
# Build for Windows (from Linux/Mac)
GOOS=windows GOARCH=amd64 go build -o bin/kiddo.exe .

# Build for Linux (for testing non-Windows behavior)
go build -o bin/kiddo .
```

## Code Style

- Follow Go idioms and `gofmt`
- Use named return types for error cases
- Prefer explicit error handling over panics
- Log important actions at INFO level
- Use DEBUG level for frequent/verbose logs

## Dependencies

Current minimal dependencies:

- `github.com/kardianos/service` - Windows service wrapper
- `github.com/sirupsen/logrus` - Structured logging
- `github.com/natefinch/lumberjack` - Log rotation

All other functionality uses Go standard library.

## Performance Considerations

- **Memory**: Service runs <30MB resident (very lightweight)
- **CPU**: Mostly idle, brief spikes every 60s during GitHub check
- **Disk I/O**: Only on config updates (~1KB for rules file)
- **Network**: ~1KB request every 60s, ETag caching prevents overhead

## Security Best Practices

1. **Secrets Management**
   - Store GitHub token in environment variable or Windows Registry
   - Never log or display tokens
   - Rotate tokens regularly

2. **Logging**
   - Don't log sensitive data (passwords, tokens)
   - Logs stored in protected directory (admin-only)

3. **Process Privileges**
   - Service requires SYSTEM privileges for user management
   - Always validate input (rules, usernames)

4. **Windows Integration**
   - Use Windows APIs when available via `golang.org/x/sys`
   - Fallback gracefully if APIs unavailable

## Roadmap Ideas

- [ ] Group Policy Object (GPO) integration
- [ ] Admin dashboard/UI for remote config
- [ ] Prometheus metrics export
- [ ] Enhanced user activity logging
- [ ] Mobile app for parent control
- [ ] Time limit notifications (e.g., 30 min warning)
- [ ] Exception management (sick days, special events)
- [ ] Activity reporting

## Debugging

### Enable Debug Logging

```json
{
  "log_level": "debug"
}
```

Then check logs:
```powershell
Get-Content "C:\ProgramData\Kiddo\logs\kiddo.log" -Tail 50
```

### Run in Console Mode

```powershell
# Stop service first
Stop-Service Kiddo

# Run in foreground
.\bin\kiddo.exe

# Logs will appear in console
# Press Ctrl+C to stop
```

### Inspect State

```powershell
# Check loaded rules (look for "rules" in logs)
Get-Content "C:\ProgramData\Kiddo\logs\kiddo.log" | Select-String "rules"

# Check GitHub connectivity
Get-Content "C:\ProgramData\Kiddo\logs\kiddo.log" | Select-String "github\|fetch"

# Check enforcement
Get-Content "C:\ProgramData\Kiddo\logs\kiddo.log" | Select-String "enforce\|user"
```

## Contributing

When contributing:
1. Write tests for new functionality
2. Follow Go conventions (gofmt, etc.)
3. Document public functions
4. Add to README if user-facing
5. Test on Windows before submitting PR

## Resources

- Go Documentation: https://golang.org/doc
- Windows API Reference: https://docs.microsoft.com/en-us/windows/win32/api/
- Golang Windows Systems: https://golang.org/x/sys
- Service Package Docs: https://pkg.go.dev/github.com/kardianos/service

---

**Happy coding!** 🚀
