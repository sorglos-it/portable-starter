# Nach dem Umzug von Programmen in den Unterordner app\: stellt Verknüpfungen,
# Dateitypen und Deinstallationseinträge um, die jetzt ins Leere zeigen.
#   - Programm, das der Starter startet  -> portable_*.exe (öffnet dann portabel)
#   - andere verschobene Datei           -> neuer Ort in app\ (z. B. Uninstall.exe)
#   - Symbol (DefaultIcon, DisplayIcon)  -> neuer Ort in app\, gleiche Nummer
#   - verschwundener alter Starter       -> portable_*.exe
# Fasst nur Einträge an, deren Ziel fehlt. Zeigt erst alles, sichert es und
# ändert nur nach „J“. Adminrechte fordert es nur für systemweite Einträge an.
#
#   verknuepfungen.ps1 [Ordner …]   Programm- oder Sammelordner, z. B. D:\App\PortableApps
[CmdletBinding(PositionalBinding = $false)]   # Ordner ohne Namen gehören immer zu -Ordner
param(
    [Parameter(Position = 0, ValueFromRemainingArguments = $true)] [string[]] $Ordner,
    [switch] $NurZeigen,              # nichts ändern, nur auflisten
    [switch] $Ja,                     # ohne Rückfrage umstellen
    [string[]] $Verknuepfungsorte,    # statt Desktop, Startmenü und Taskleiste
    [string[]] $Registryorte,         # statt Dateitypen und Deinstallationseinträgen
    [string] $Sicherung,              # statt Dokumente\portable-starter-sicherung\<Zeit>
    [switch] $Erhoeht                 # intern: läuft im Admin-Fenster
)

# Das Admin-Fenster bleibt am Ende stehen, damit man das Ergebnis lesen kann.
function Wait-Enter { if ((-not $Ja -and -not $NurZeigen) -or $Erhoeht) { Write-Host ''; [void](Read-Host 'Enter zum Schließen') } }

# --- Programmordner: enthalten app\ und eine portable_*.exe -----------------
if (-not $Ordner) {
    $pick = (New-Object -ComObject Shell.Application).BrowseForFolder(0, 'Ordner mit den portablen Programmen wählen, z. B. D:\App\PortableApps', 0x51, 17)
    if (-not $pick) { exit }
    $Ordner = @($pick.Self.Path)
}
$bases = @()
foreach ($o in $Ordner) {
    $o = $o.Trim('"').TrimEnd('\')
    if (-not (Test-Path -LiteralPath $o -PathType Container)) { continue }
    foreach ($d in @(Get-Item -LiteralPath $o) + @(Get-ChildItem -LiteralPath $o -Directory -ErrorAction SilentlyContinue)) {
        if ((Test-Path -LiteralPath (Join-Path $d.FullName 'app') -PathType Container) -and
            (Get-ChildItem -LiteralPath $d.FullName -Filter 'portable_*.exe' -File -ErrorAction SilentlyContinue)) {
            $bases += $d.FullName
        }
    }
}
$bases = @($bases | Sort-Object -Unique)
if (-not $bases) {
    Write-Host 'Kein Programmordner mit app\ und portable_*.exe gefunden.' -ForegroundColor Yellow
    Wait-Enter; exit 1
}

# Programme, die der Starter startet (Einträge „exe“ in config_starter.json)
$programs = @{}
foreach ($base in $bases) {
    $list = @()
    $cfg = Join-Path $base 'config_starter.json'
    if (Test-Path -LiteralPath $cfg) {
        try {
            $json = Get-Content -LiteralPath $cfg -Raw -Encoding UTF8 | ConvertFrom-Json
            $list = @($json.programme | ForEach-Object { ([string]$_.exe).Replace('/', '\').Trim().ToLowerInvariant() })
        } catch { }
    }
    $programs[$base] = $list
}

# Get-NewPath liefert für einen fehlenden Pfad unter einem Programmordner den
# neuen Pfad; context „start“ für Aufrufe, „icon“ für Symbole.
function Get-NewPath([string] $path, [string] $context) {
    if (-not $path -or (Test-Path -LiteralPath $path)) { return $null }
    foreach ($base in $bases) {
        if (-not $path.StartsWith($base + '\', [StringComparison]::OrdinalIgnoreCase)) { continue }
        $rel = $path.Substring($base.Length + 1)
        $inApp = Join-Path (Join-Path $base 'app') $rel
        $starter = (Get-ChildItem -LiteralPath $base -Filter 'portable_*.exe' -File | Select-Object -First 1).FullName
        if ($context -eq 'icon') {
            if (Test-Path -LiteralPath $inApp) { return $inApp }
            return $starter
        }
        if ($programs[$base] -contains $rel.ToLowerInvariant()) { return $starter }
        if (Test-Path -LiteralPath $inApp) { return $inApp }
        if ($rel -match '\.(exe|bat|cmd)$') { return $starter }   # alter Starter
    }
    return $null
}

$changes = New-Object System.Collections.Generic.List[object]

# --- Verknüpfungen --------------------------------------------------------
$wsh = New-Object -ComObject WScript.Shell
$machineDirs = @([Environment]::GetFolderPath('CommonDesktopDirectory'), [Environment]::GetFolderPath('CommonStartMenu')) | Where-Object { $_ }
if (-not $Verknuepfungsorte) {
    $Verknuepfungsorte = @([Environment]::GetFolderPath('Desktop'), [Environment]::GetFolderPath('StartMenu'),
        (Join-Path $env:APPDATA 'Microsoft\Internet Explorer\Quick Launch')) + $machineDirs
}
foreach ($place in $Verknuepfungsorte) {
    if (-not $place -or -not (Test-Path -LiteralPath $place)) { continue }
    foreach ($file in Get-ChildItem -LiteralPath $place -Recurse -Filter '*.lnk' -File -ErrorAction SilentlyContinue) {
        $lnk = $wsh.CreateShortcut($file.FullName)
        $new = Get-NewPath $lnk.TargetPath 'start'
        if (-not $new) { continue }
        $icon = $null
        $iconPath = ($lnk.IconLocation -replace ',-?\d+$', '')
        if ($iconPath -and -not (Test-Path -LiteralPath $iconPath)) {
            if ($new -like '*\portable_*.exe') { $icon = "$new,0" }
            else { $n = Get-NewPath $iconPath 'icon'; if ($n) { $icon = $lnk.IconLocation.Replace($iconPath, $n) } }
        }
        $workDir = Split-Path $new
        $machine = [bool]($machineDirs | Where-Object { $file.FullName.StartsWith($_ + '\', [StringComparison]::OrdinalIgnoreCase) })
        $changes.Add([pscustomobject]@{ Art = 'Verknüpfung'; Ort = $file.FullName; Wert = ''; Alt = $lnk.TargetPath; Neu = $new
                Arbeitsordner = $workDir; Icon = $icon; Kind = ''; Maschine = $machine })
    }
}

# --- Registry: Dateitypen, Protokolle, Deinstallation ---------------------
if (-not $Registryorte) {
    $Registryorte = 'HKCU\Software\Classes', 'HKLM\SOFTWARE\Classes',
        'HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall',
        'HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
        'HKLM\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
}
$pathPattern = '(?i)[a-z]:\\[^"<>|?*\r\n]*?\.(?:exe|bat|cmd)\b'
$keys = @{}
foreach ($root in $Registryorte) {
    foreach ($base in $bases) {
        foreach ($line in (& reg.exe query $root /s /f $base /d 2>$null)) {
            if ($line -match '^HKEY_') { $keys[$line.Trim()] = $true }
        }
    }
}
foreach ($key in $keys.Keys) {
    $item = Get-Item -LiteralPath "Registry::$key" -ErrorAction SilentlyContinue
    if (-not $item) { continue }
    foreach ($name in $item.GetValueNames()) {
        $kind = $item.GetValueKind($name)
        if ($kind -ne 'String' -and $kind -ne 'ExpandString') { continue }
        $data = [string]$item.GetValue($name, $null, 'DoNotExpandEnvironmentNames')
        $context = 'start'
        if ($key -match '\\DefaultIcon$' -or $name -eq 'DisplayIcon') { $context = 'icon' }
        $new = [regex]::Replace($data, $pathPattern, [System.Text.RegularExpressions.MatchEvaluator] {
                param($m)
                $n = Get-NewPath $m.Value $context
                if ($n) { $n } else { $m.Value }
            })
        if ($new -ne $data) {
            $changes.Add([pscustomobject]@{ Art = 'Registry'; Ort = $key; Wert = $name; Alt = $data; Neu = $new
                    Arbeitsordner = ''; Icon = $null; Kind = [string]$kind; Maschine = $key.StartsWith('HKEY_LOCAL_MACHINE') })
        }
    }
}

# --- Anzeigen ---------------------------------------------------------------
$isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole(
    [Security.Principal.WindowsBuiltInRole]::Administrator)
if ($changes.Count -eq 0) {
    Write-Host 'Alles in Ordnung - kein Eintrag zeigt ins Leere.' -ForegroundColor Green
    Wait-Enter; exit
}
foreach ($c in $changes) {
    $label = $c.Ort
    if ($c.Wert) { $label += "  [$($c.Wert)]" }
    if ($c.Maschine) { $label += '  (systemweit)' }
    Write-Host $label -ForegroundColor Cyan
    Write-Host "    alt: $($c.Alt)"
    Write-Host "    neu: $($c.Neu)"
}
Write-Host ''
Write-Host "$($changes.Count) Einträge zeigen ins Leere." -ForegroundColor Yellow
if ($NurZeigen) { exit }
if (-not $Ja) {
    if ((Read-Host 'Jetzt umstellen? (J/N)') -notmatch '^[jJyY]') { Write-Host 'Nichts geändert.'; Wait-Enter; exit }
}

# Systemweite Einträge brauchen Adminrechte. Das Admin-Fenster sieht keine
# Netzlaufwerke, deshalb läuft es mit einer Kopie des Skripts aus %TEMP%.
$machineCount = @($changes | Where-Object Maschine).Count
if ($machineCount -gt 0 -and -not $isAdmin -and -not $Erhoeht) {
    $copy = Join-Path $env:TEMP 'portable-starter-verknuepfungen.ps1'
    Copy-Item -LiteralPath $PSCommandPath -Destination $copy -Force
    $argList = @('-NoProfile', '-ExecutionPolicy', 'Bypass', '-File', "`"$copy`"", '-Ja', '-Erhoeht') + ($bases | ForEach-Object { "`"$_`"" })
    try {
        Start-Process powershell.exe -Verb RunAs -ArgumentList $argList -Wait
        exit
    } catch {
        Write-Host "Ohne Adminrechte bleiben $machineCount systemweite Einträge unverändert." -ForegroundColor Yellow
    }
}
$todo = @($changes | Where-Object { $isAdmin -or -not $_.Maschine })

# --- Sichern und umstellen --------------------------------------------------
if (-not $Sicherung) {
    $Sicherung = Join-Path ([Environment]::GetFolderPath('MyDocuments')) ('portable-starter-sicherung\' + (Get-Date -Format 'yyyy-MM-dd_HHmmss'))
}
try {
    New-Item -ItemType Directory -Path $Sicherung -Force -ErrorAction Stop | Out-Null
    $i = 0
    foreach ($c in $todo) {
        $i++
        if ($c.Art -eq 'Verknüpfung') {
            Copy-Item -LiteralPath $c.Ort -Destination (Join-Path $Sicherung ('{0:D2}_{1}' -f $i, (Split-Path $c.Ort -Leaf))) -ErrorAction Stop
        } else {
            & reg.exe export $c.Ort (Join-Path $Sicherung ('{0:D2}.reg' -f $i)) /y | Out-Null
            if ($LASTEXITCODE -ne 0) { throw "reg export $($c.Ort)" }
        }
    }
} catch {
    Write-Host "Sicherung fehlgeschlagen, nichts geändert: $($_.Exception.Message)" -ForegroundColor Red
    Wait-Enter; exit 1
}
$failed = 0
foreach ($c in $todo) {
    try {
        if ($c.Art -eq 'Verknüpfung') {
            $lnk = $wsh.CreateShortcut($c.Ort)
            $lnk.TargetPath = $c.Neu
            $lnk.WorkingDirectory = $c.Arbeitsordner
            if ($c.Icon) { $lnk.IconLocation = $c.Icon }
            $lnk.Save()
        } else {
            $hive, $sub = $c.Ort -split '\\', 2
            $rootKey = [Microsoft.Win32.Registry]::CurrentUser
            if ($hive -eq 'HKEY_LOCAL_MACHINE') { $rootKey = [Microsoft.Win32.Registry]::LocalMachine }
            $rk = $rootKey.OpenSubKey($sub, $true)
            $rk.SetValue($c.Wert, $c.Neu, [Microsoft.Win32.RegistryValueKind]$c.Kind)
            $rk.Close()
        }
    } catch {
        $failed++
        Write-Host "FEHLER $($c.Ort): $($_.Exception.Message)" -ForegroundColor Red
    }
}
if (Test-Path "$env:SystemRoot\System32\ie4uinit.exe") { & "$env:SystemRoot\System32\ie4uinit.exe" -show }
Write-Host ''
Write-Host ("{0} Einträge umgestellt, {1} Fehler. Sicherung: {2}" -f ($todo.Count - $failed), $failed, $Sicherung) -ForegroundColor Green
Wait-Enter
