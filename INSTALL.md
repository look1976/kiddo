# Kiddo Installation & Deployment Guide

## Quick Start (5 Minutes)

### Step 1: Prepare GitHub Repository

Create a new GitHub repository with your schedule file:

**Repository name**: `kiddo-schedules` (any name works)

**File**: `schedules.csv`

```csv
# Your kids' computer usage schedules
Janek,Mon,16-21
Janek,Fri,17-23
Janek,Sat,13-23

Igor,Mon,17-20
Igor,Fri,18-22
Igor,Sat,14-20
```

### Step 2: Create GitHub Token

1. Go to [GitHub Settings → Developer settings → Personal access tokens](https://github.com/settings/tokens)
2. Click "Generate new token"
3. Name: `Kiddo Service`
4. Scope: Check only `repo` (private repositories only)
5. Expiration: 30 days (rotate regularly)
6. Copy the token: `ghp_xxxxxxxxxxxxxxxxxxxx`

### Step 3: Create Windows Configuration

Open PowerShell as Administrator:

```powershell
# Create directory
New-Item -Path "C:\ProgramData\Kiddo" -ItemType Directory -Force

# Create config.json
$config = @{
    github_token = "ghp_xxxxxxxxxxxxxxxxxxxx"
    github_repo = "yourusername/kiddo-schedules"
    github_branch = "main"
    github_config_file = "schedules.csv"
    check_interval_seconds = 60
    cleanup_interval_seconds = 300
    log_level = "info"
} | ConvertTo-Json

$config | Out-File -FilePath "C:\ProgramData\Kiddo\config.json" -Encoding UTF8
icacls "C:\ProgramData\Kiddo\config.json" /inheritance:r /grant:r "$($env:USERNAME):F" "SYSTEM:F"
```

### Step 4: Build the Service

On Windows machine with Go 1.21+:

```powershell
git clone [your-repo-url]
cd kiddo

# Option A: Using batch script
.\build.bat

# Option B: Using Go directly
go build -o bin\kiddo.exe .
```

### Step 5: Install Service

```powershell
.\bin\kiddo.exe install
```

Expected output:
```
Service installed successfully
```

### Step 6: Start Service

```powershell
Start-Service -Name Kiddo

# Verify it started
Get-Service -Name Kiddo

# Check logs
Get-Content "C:\ProgramData\Kiddo\logs\kiddo.log"
```

**Done!** Service is now running and will:
- Check GitHub every 60 seconds for schedule updates
- Enforce login times automatically
- Schedule PC shutdowns at configured times
- Clean up unauthorized user accounts every 5 minutes

---

## Detailed Setup Guide

### Prerequisites

- Windows 10/11 (Pro or Home, any edition)
- Administrator account access
- Internet connection
- GitHub account with private repository OR public repository

### Installation Steps

#### 1. Create Schedule Repository on GitHub

**For Private Repository** (Recommended):

1. Go to [github.com/new](https://github.com/new)
2. Repository name: `kiddo-schedules`
3. Private: ✓ (checked)
4. Do NOT initialize with README
5. Create repository

Add your schedule file:

- Go to repository
- Click "Create new file"
- Filename: `schedules.csv`
- Content (example):

```
# Weekday schedules
Janek,Mon,16-21
Janek,Tue,16-21
Janek,Wed,16-21
Janek,Thu,16-21
Janek,Fri,17-23

# Weekend schedules
Janek,Sat,13-23
Janek,Sun,13-21

# Second user
Igor,Mon,17-20
Igor,Fri,18-22
Igor,Sat,14-20
Igor,Sun,14-19
```

- Commit directly to main

#### 2. Generate GitHub Personal Access Token

1. Login to GitHub
2. Go to [Settings → Developer settings → Personal access tokens → Tokens (classic)](https://github.com/settings/tokens)
3. Click "Generate new token (classic)"
4. Fill in form:
   - **Note**: `Kiddo Service`
   - **Expiration**: 30 days
   - **Scopes**: Check ONLY `repo` (allows access to private repos)
5. Click "Generate token"
6. Copy the token immediately (won't be shown again)

#### 3. Download and Build

On your Windows machine with Go installed:

```powershell
# Clone or download this repository
git clone https://github.com/yourusername/kiddo.git
cd kiddo

# Build
.\build.bat
```

Output should show:
```
========================================
Build successful!
Output: bin\kiddo.exe
```

#### 4. Create Configuration

Create `C:\ProgramData\Kiddo\config.json` with your settings:

```powershell
# Run as Administrator
$token = "ghp_xxxxxxxxxxxxxxxxxxxx"  # Your GitHub token
$repo = "yourusername/kiddo-schedules"

$config = @{
    github_token = $token
    github_repo = $repo
    github_branch = "main"
    github_config_file = "schedules.csv"
    check_interval_seconds = 60
    cleanup_interval_seconds = 300
    log_level = "info"
} | ConvertTo-Json

New-Item -Path "C:\ProgramData\Kiddo" -ItemType Directory -Force | Out-Null
$config | Out-File "C:\ProgramData\Kiddo\config.json" -Encoding UTF8

# Restrict permissions (admin only)
icacls "C:\ProgramData\Kiddo\config.json" /inheritance:r /grant:r "SYSTEM:F" "BUILTIN\Administrators:F"
```

#### 5. Install as Windows Service

```powershell
# Run as Administrator
.\bin\kiddo.exe install

# Verify installation
Get-Service -Name Kiddo
```

#### 6. Start Service

```powershell
# Start service
Start-Service -Name Kiddo

# Wait a moment
Start-Sleep -Seconds 2

# Verify running
Get-Service -Name Kiddo

# Check initial logs
Get-Content "C:\ProgramData\Kiddo\logs\kiddo.log" -Tail 30
```

#### 7. Enable Automatic Startup

Service should auto-start on boot, but verify:

```powershell
# Set to automatic
Set-Service -Name Kiddo -StartupType Automatic

# Verify
Get-Service -Name Kiddo | Select-Object Name, StartType
```

---

## Configuration Reference

### config.json Fields

```json
{
  "github_token": "ghp_...",          // GitHub Personal Access Token (or env var)
  "github_repo": "user/repo-name",    // REQUIRED - Repository to check
  "github_branch": "main",             // Branch name (default: main)
  "github_config_file": "schedules.csv", // CSV file in repo (default: schedules.csv)
  "check_interval_seconds": 60,        // Check frequency (default: 60)
  "cleanup_interval_seconds": 300,     // Cleanup frequency (default: 300 = 5 min)
  "log_level": "info"                  // Log level (debug/info/warn/error)
}
```

### Schedule CSV Format

Column format: `Username, Day-of-week, Start-hour-End-hour`

Example:
```csv
Janek,Mon,16-21
Janek,Tue,16-21
Igor,Sat,10-18
```

**Valid day values**:
- Full: Monday, Tuesday, Wednesday, Thursday, Friday, Saturday, Sunday
- Short: Mon, Tue, Wed, Thu, Fri, Sat, Sun

**Hours are 24-hour format**:
- Midnight: 0
- Noon: 12
- 9 PM: 21

---

## Verification Checklist

After installation, verify with this checklist:

- [ ] Service installed: `Get-Service Kiddo`
- [ ] Service is running: `Get-Service Kiddo | Select Status`
- [ ] Service set to auto-start: `Get-Service Kiddo | Select StartType`
- [ ] Config file exists: `Test-Path C:\ProgramData\Kiddo\config.json`
- [ ] Logs directory exists: `Test-Path C:\ProgramData\Kiddo\logs`
- [ ] Recent logs present: `Get-ChildItem C:\ProgramData\Kiddo\logs`
- [ ] Service can read config: `Get-Content C:\ProgramData\Kiddo\logs\kiddo.log | Select-String "Configuration loaded"`
- [ ] GitHub connection works: `Get-Content C:\ProgramData\Kiddo\logs\kiddo.log | Select-String "Successfully fetched"`

---

## Troubleshooting Installation

### Service fails to install

**Error**: "Access Denied"
- **Solution**: Run PowerShell as Administrator

**Error**: "Service already exists"
- **Solution**: Uninstall first: `.\bin\kiddo.exe uninstall`

### Service won't start

Check logs:
```powershell
Get-Content C:\ProgramData\Kiddo\logs\kiddo.log -Tail 50
```

Common issues:
- **"Configuration loaded" missing** → config.json path or format wrong
- **"GitHub token not configured"** → Token not in config or env var
- **Permission denied** → Run service as Administrator or rebuild config

### Test in Debug Mode

```powershell
# Stop running service first
Stop-Service -Name Kiddo

# Run in console to see live errors
C:\path\to\kiddo.exe

# Press Ctrl+C to stop
```

### Scheduled shutdown not working

1. Verify Windows Task Scheduler service is running:
   ```powershell
   Get-Service -Name Schedule | Select Status
   ```

2. Check created tasks:
   ```powershell
   Get-ScheduledTask -TaskName "Kiddo_Shutdown_*" | Select TaskName, State
   ```

3. View task details:
   ```powershell
   Get-ScheduledTask -TaskName "Kiddo_Shutdown_USERNAME" | Get-ScheduledTaskInfo
   ```

---

## Updating Schedules

After schedules are deployed and running:

1. Edit `schedules.csv` in your GitHub repository
2. Commit and push changes
3. Service checks GitHub every 60 seconds
4. Service automatically applies new rules within 60 seconds
5. Changes take effect immediately (no restart needed)

Monitor live:
```powershell
Get-Content "C:\ProgramData\Kiddo\logs\kiddo.log" -Tail 20 -Wait
```

---

## Uninstall

```powershell
# Run as Administrator
Stop-Service -Name Kiddo
.\bin\kiddo.exe uninstall

# Verify removed
Get-Service Kiddo
# Should show: "Cannot find a service with name 'Kiddo'"
```

Optionally clean up files:
```powershell
Remove-Item -Path "C:\ProgramData\Kiddo" -Recurse -Force
```

---

## Support

For issues:

1. **Check logs first**: `Get-Content C:\ProgramData\Kiddo\logs\kiddo.log`
2. **Review configuration**: `Get-Content C:\ProgramData\Kiddo\config.json`
3. **Test GitHub connection**: Verify token and repository path
4. **Run in debug mode**: `.\bin\kiddo.exe` (console mode)
5. **Check Event Viewer**: `eventvwr.msc` → Windows Logs → System

Common log messages:
- `"Logger initialized"` → Service started OK
- `"Successfully fetched"` → GitHub fetch worked
- `"Rules unchanged"` → No new changes detected
- `"updated X rules"` → New rules applied
