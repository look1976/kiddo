# Kiddo - Windows Parental Control Service

Kiddo is a Windows service that controls how much time your kids spend using their computers. It subscribes to schedules stored in a GitHub repository and enforces login time restrictions automatically.

## Features

✓ **Runs as Windows Service** - Starts automatically at boot, runs with SYSTEM privileges  
✓ **GitHub-based Configuration** - Control schedules from a GitHub repository (private or public)  
✓ **Automatic Policy Enforcement** - Enforces user login times via Windows commands  
✓ **Scheduled PC Shutdowns** - Automatically shuts down at configured times via Task Scheduler  
✓ **User Cleanup** - Removes unauthorized user accounts periodically  
✓ **Structured Logging** - JSON logs with daily rotation for troubleshooting  
✓ **ETag Caching** - Efficient GitHub polling (only downloads when file changes)  
✓ **Robust Error Handling** - Falls back to cached rules if GitHub is unreachable  

## Architecture

The service consists of 4 independent components running in coordination:

- **Config Manager** - Polls GitHub every 60s for schedule updates (configurable)
- **Policy Enforcer** - Checks and enforces login hour restrictions  
- **Scheduler** - Creates Windows scheduled tasks for automatic PC shutdowns  
- **User Cleanup** - Periodically removes authorized-only accounts  

See [functionalities.md](functionalities.md) for complete requirements.

## Configuration

### 1. Create Configuration Directory

```powershell
New-Item -Path "C:\ProgramData\Kiddo" -ItemType Directory -Force
```

### 2. Create Configuration File

Copy `config.example.json` to `C:\ProgramData\Kiddo\config.json` and edit:

```json
{
  "github_token": "ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
  "github_repo": "yourname/schedules-repo",
  "github_branch": "main",
  "github_config_file": "schedules.csv",
  "check_interval_seconds": 60,
  "cleanup_interval_seconds": 300,
  "log_level": "info"
}
```

**Configuration Details:**

| Setting | Description | Default |
|---------|-------------|---------|
| `github_token` | GitHub Personal Access Token (read-only scope) | (from env) |
| `github_repo` | Repository in format `owner/repo` | required |
| `github_branch` | Branch name | main |
| `github_config_file` | Path to CSV file in repo | schedules.csv |
| `check_interval_seconds` | How often to poll GitHub | 60 |
| `cleanup_interval_seconds` | How often to clean unauthorized users | 300 |
| `log_level` | Logging level (debug/info/warn/error) | info |

### 3. Create GitHub Repository

Create a CSV file in your GitHub repository with the schedule. Example:

```csv
# Schedule file
# Format: Username, Day-of-week, Start-hour-End-hour

Janek,Mon,16-21
Janek,Fri,17-23
Janek,Sat,13-23

Igor,Mon,17-20
Igor,Thu,17-20
Igor,Sat,14-20
```

Supported day abbreviations: `Mon`, `Tue`, `Wed`, `Thu`, `Fri`, `Sat`, `Sun`

### 4. Create GitHub Personal Access Token

1. Go to GitHub → Settings → Developer settings → Personal access tokens
2. Create token with `repo` scope (read-only)
3. Copy token to config file or set environment variable:

```powershell
[Environment]::SetEnvironmentVariable("KIDDO_GITHUB_TOKEN", "ghp_...", "Machine")
```

## Build Instructions

### Prerequisites

- Windows 10/11
- Go 1.21 or later
- Administrator privileges for service installation

### Build on Windows

```powershell
# Using batch script
.\build.bat

# Or using Go directly
go build -o bin\kiddo.exe .
```

### Build on Linux/Mac (cross-compile)

```bash
make build
# Creates: bin/kiddo.exe
```

## Installation

### Option 1: Automatic Installation

```powershell
# Run as Administrator
.\bin\kiddo.exe install

# Start the service
net start Kiddo

# Check status
Get-Service Kiddo

# View logs
Get-Content "C:\ProgramData\Kiddo\logs\kiddo.log"
```

### Option 2: Manual Installation with Services UI

1. Install: `.\bin\kiddo.exe install`
2. Open Services (services.msc)
3. Find "Kiddo Parental Control Service"
4. Set startup type to "Automatic"
5. Start the service

### Verify Installation

```powershell
# Check service status
Get-Service Kiddo

# Check logs
Get-Content "C:\ProgramData\Kiddo\logs\kiddo.log" -Tail 20

# Verify scheduled tasks
Get-ScheduledTask -TaskName "Kiddo_Shutdown_*"
```

## Uninstall

```powershell
# Run as Administrator
net stop Kiddo
.\bin\kiddo.exe uninstall
```

## Log Locations

- **Service Logs**: `C:\ProgramData\Kiddo\logs\kiddo.log`
- **Cached Rules**: `C:\ProgramData\Kiddo\rules_cache.csv`
- **Config File**: `C:\ProgramData\Kiddo\config.json`
- **ETag Cache**: `C:\ProgramData\Kiddo\.etag`

Logs are rotated daily with 7-day retention.

## Usage Examples

### Example 1: Two Kids, Different Schedules

Schedule file in GitHub:

```csv
Janek,Mon,16-21
Janek,Tue,16-21
Janek,Wed,16-21
Janek,Thu,16-21
Janek,Fri,17-23
Janek,Sat,13-23
Janek,Sun,13-21

Igor,Mon,17-20
Igor,Tue,17-20
Igor,Wed,17-20
Igor,Thu,17-20
Igor,Fri,18-22
Igor,Sat,14-20
Igor,Sun,14-19
```

**Effect:**
- Janek can log in weekdays 4-9pm, weekends 1-11pm
- Igor can log in weekdays 5-8pm, weekends 2-8pm
- PCs auto-shutdown at configured end times
- Any other user accounts are deleted

### Example 2: Weekend-Only Access

```csv
Alice,Sat,10-18
Alice,Sun,10-18
```

**Effect:**
- Alice can only use computer on weekends 10am-6pm
- PC automatically shuts down at 6pm

## Troubleshooting

### Service won't start

```powershell
# Check Windows Event Viewer
eventvwr.msc

# Check Kiddo logs
Get-Content "C:\ProgramData\Kiddo\logs\kiddo.log" -Tail 50

# Run in debug mode (console) to see errors
.\bin\kiddo.exe
```

### Configuration not loading from GitHub

1. Verify GitHub token is set and has `repo` scope
2. Check repository name is correct (case-sensitive)
3. Verify CSV file path in config
4. Check logs for network errors
5. Service will fall back to cached rules if GitHub unavailable

### User accounts not being deleted

1. Service must run with SYSTEM privileges (automatic)
2. Check that usernames in CSV match actual account names (case varies)
3. View logs: `Get-Content "C:\ProgramData\Kiddo\logs\kiddo.log" | grep -i cleanup`
4. Protected accounts (Administrator, Guest, etc.) are never deleted

### Scheduled shutdown not working

1. Check Task Scheduler: `taskschd.msc`
2. Look for tasks starting with `Kiddo_Shutdown_`
3. Verify admin rights to create scheduled tasks
4. Check logs for schtasks errors

### Check Service Status

```powershell
# View current service state
Get-Service Kiddo | Select-Object Status, StartType

# View last 20 log lines
Get-Content "C:\ProgramData\Kiddo\logs\kiddo.log" -Tail 20

# Monitor logs in real-time
Get-Content "C:\ProgramData\Kiddo\logs\kiddo.log" -Tail 20 -Wait
```

## Security Considerations

1. **GitHub Token**
   - Use personal access tokens with minimal scope (`repo` for private repos)
   - Rotate tokens periodically
   - Never commit tokens to Git

2. **Service Permissions**
   - Service runs as SYSTEM (required for user management)
   - Config file restricted to admin-only access
   - Logs directory restricted to admin-only access

3. **Rules File**
   - Can be public or private GitHub repository
   - Consider using private repository for privacy

4. **Audit Trail**
   - All enforcement actions logged with timestamps
   - JSON logs for easy parsing/monitoring

## Testing

### Run Tests

```powershell
go test ./...
```

### Debug Mode (Console)

Running the service in console mode is helpful for development:

```powershell
# Install service first
.\bin\kiddo.exe install

# Run in foreground for testing
.\bin\kiddo.exe
```

### Test GitHub Integration

```powershell
# Manually trigger a config check by restarting service
Restart-Service Kiddo

# Check logs
Get-Content "C:\ProgramData\Kiddo\logs\kiddo.log" -Tail 50 | grep -i github
```

## Build Output

Single statically-compiled executable with no runtime dependencies:

```
kiddo.exe (~15MB)
```

All dependencies are bundled in the binary. No .NET Framework or additional runtime required.

## License

[Add your license here]

## Support

For issues or questions:
1. Check logs: `C:\ProgramData\Kiddo\logs\kiddo.log`
2. Review configuration
3. Verify GitHub repository and file path
4. Check Windows Event Viewer for service errors

---

**Kiddo Service** | Automated Computer Usage Control
