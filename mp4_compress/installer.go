package main

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

//go:embed internal/mp4_compress.exe internal/mp4_compress.ps1
var embeddedFiles embed.FS

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
)

func init() {
	// Enable ANSI color support on Windows
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	setConsoleMode := kernel32.NewProc("SetConsoleMode")
	handle, _ := syscall.GetStdHandle(syscall.STD_OUTPUT_HANDLE)
	setConsoleMode.Call(uintptr(handle), 0x0001|0x0002|0x0004)
}

func colorPrint(color, text string) {
	fmt.Print(color + text + colorReset)
}

func colorPrintln(color, text string) {
	fmt.Println(color + text + colorReset)
}

func isAdmin() bool {
	var sid *windows.SID
	err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid)
	if err != nil {
		return false
	}
	defer windows.FreeSid(sid)

	token := windows.Token(0)
	member, err := token.IsMember(sid)
	if err != nil {
		return false
	}
	return member
}

func runAsAdmin() error {
	verb := "runas"
	exe, _ := os.Executable()
	cwd, _ := os.Getwd()
	args := strings.Join(os.Args[1:], " ")

	verbPtr, _ := syscall.UTF16PtrFromString(verb)
	exePtr, _ := syscall.UTF16PtrFromString(exe)
	cwdPtr, _ := syscall.UTF16PtrFromString(cwd)
	argPtr, _ := syscall.UTF16PtrFromString(args)

	var showCmd int32 = 1

	err := windows.ShellExecute(0, verbPtr, exePtr, argPtr, cwdPtr, showCmd)
	if err != nil {
		return err
	}
	return nil
}

func checkCommand(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

func installFFmpeg() error {
	colorPrintln(colorYellow, "Installing ffmpeg via winget...")
	cmd := exec.Command("winget", "install", "--id", "Gyan.FFmpeg", "--silent", "--accept-package-agreements", "--accept-source-agreements")
	err := cmd.Run()
	if err != nil {
		return err
	}
	colorPrintln(colorGreen, "ffmpeg installed successfully!")
	return nil
}

func setExecutionPolicy() {
	fmt.Print("Setting PowerShell execution policy...")
	cmd := exec.Command("powershell", "-Command", "Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser -Force")
	err := cmd.Run()
	if err != nil {
		colorPrintln(colorYellow, " FAILED")
	} else {
		colorPrintln(colorGreen, " DONE")
	}
}

func createInstallDir(installDir string) error {
	fmt.Print("Creating installation directory...")
	err := os.MkdirAll(installDir, 0755)
	if err != nil {
		colorPrintln(colorRed, " FAILED")
		return fmt.Errorf("cannot create installation directory: %v", err)
	}
	colorPrintln(colorGreen, " DONE")
	return nil
}

func extractEmbeddedFiles(installDir string) error {
	fmt.Print("Extracting files...")

	files := map[string]string{
		"internal/mp4_compress.exe": "mp4_compress.exe",
		"internal/mp4_compress.ps1": "mp4_compress.ps1",
	}

	for embeddedPath, outputName := range files {
		data, err := embeddedFiles.ReadFile(embeddedPath)
		if err != nil {
			colorPrintln(colorRed, " FAILED")
			return fmt.Errorf("failed to read embedded file %s: %v", embeddedPath, err)
		}

		outputPath := filepath.Join(installDir, outputName)
		err = os.WriteFile(outputPath, data, 0755)
		if err != nil {
			colorPrintln(colorRed, " FAILED")
			return fmt.Errorf("failed to write file %s: %v", outputPath, err)
		}
	}

	colorPrintln(colorGreen, " DONE")
	return nil
}

func registerContextMenu(installDir string) error {
	fmt.Print("Registering right-click context menu...")

	regPath := `Software\Classes\SystemFileAssociations\.mp4\shell\Compress Video`
	key, _, err := registry.CreateKey(registry.CURRENT_USER, regPath, registry.SET_VALUE)
	if err != nil {
		colorPrintln(colorRed, " FAILED")
		return fmt.Errorf("failed to create registry key: %v", err)
	}
	defer key.Close()

	err = key.SetStringValue("", "Compress Video")
	if err != nil {
		colorPrintln(colorRed, " FAILED")
		return fmt.Errorf("failed to set registry value: %v", err)
	}

	commandPath := regPath + `\command`
	cmdKey, _, err := registry.CreateKey(registry.CURRENT_USER, commandPath, registry.SET_VALUE)
	if err != nil {
		colorPrintln(colorRed, " FAILED")
		return fmt.Errorf("failed to create command registry key: %v", err)
	}
	defer cmdKey.Close()

	commandValue := fmt.Sprintf(`powershell.exe -ExecutionPolicy Bypass -WindowStyle Hidden -File "%s\mp4_compress.ps1" "%%1"`, installDir)
	err = cmdKey.SetStringValue("", commandValue)
	if err != nil {
		colorPrintln(colorRed, " FAILED")
		return fmt.Errorf("failed to set command registry value: %v", err)
	}

	colorPrintln(colorGreen, " DONE")
	return nil
}

func createUninstallScript(installDir string) error {
	uninstallScript := `Write-Host "Uninstalling MP4 Video Compressor..." -ForegroundColor Yellow

$regPath = "HKCU:\Software\Classes\SystemFileAssociations\.mp4\shell\Compress Video"
if (Test-Path $regPath) {
    Remove-Item -Path $regPath -Recurse -Force
    Write-Host "Context menu removed" -ForegroundColor Green
}

$installDir = "$env:LOCALAPPDATA\mp4_compress"
if (Test-Path $installDir) {
    Set-Location $env:TEMP
    Remove-Item -Path $installDir -Recurse -Force
    Write-Host "Installation directory removed" -ForegroundColor Green
}

Write-Host "Uninstall complete!" -ForegroundColor Green
Read-Host "Press Enter to close"
`

	uninstallPath := filepath.Join(installDir, "uninstall.ps1")
	err := os.WriteFile(uninstallPath, []byte(uninstallScript), 0644)
	if err != nil {
		return fmt.Errorf("failed to create uninstall script: %v", err)
	}
	return nil
}

func verifyInstallation(installDir string) bool {
	fmt.Print("\nVerifying installation...")

	exePath := filepath.Join(installDir, "mp4_compress.exe")
	ps1Path := filepath.Join(installDir, "mp4_compress.ps1")

	if _, err := os.Stat(exePath); os.IsNotExist(err) {
		colorPrintln(colorRed, " FAILED")
		colorPrintln(colorRed, "\nInstallation incomplete! mp4_compress.exe is missing.")
		return false
	}

	if _, err := os.Stat(ps1Path); os.IsNotExist(err) {
		colorPrintln(colorRed, " FAILED")
		colorPrintln(colorRed, "\nInstallation incomplete! mp4_compress.ps1 is missing.")
		return false
	}

	colorPrintln(colorGreen, " OK")
	return true
}

func main() {
	if !isAdmin() {
		colorPrintln(colorYellow, "Requesting administrator privileges...")
		err := runAsAdmin()
		if err != nil {
			colorPrintln(colorRed, "Failed to elevate privileges. Please run as administrator.")
			fmt.Print("\nPress Enter to exit...")
			fmt.Scanln()
			os.Exit(1)
		}
		os.Exit(0)
	}

	colorPrintln(colorCyan, "MP4 Video Compressor")
	fmt.Println()

	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		colorPrintln(colorRed, "FATAL: LOCALAPPDATA environment variable not set")
		fmt.Print("\nPress Enter to exit...")
		fmt.Scanln()
		os.Exit(1)
	}
	installDir := filepath.Join(localAppData, "mp4_compress")

	fmt.Print("Checking for ffmpeg...")
	if !checkCommand("ffmpeg") {
		colorPrintln(colorYellow, " NOT FOUND")
		err := installFFmpeg()
		if err != nil {
			colorPrintln(colorRed, "FATAL: Failed to install ffmpeg automatically.")
			colorPrintln(colorRed, "Please install ffmpeg manually: winget install ffmpeg")
			fmt.Print("\nPress Enter to exit...")
			fmt.Scanln()
			os.Exit(1)
		}
	} else {
		colorPrintln(colorGreen, " FOUND")
	}

	setExecutionPolicy()

	err := createInstallDir(installDir)
	if err != nil {
		colorPrintln(colorRed, fmt.Sprintf("FATAL: %v", err))
		fmt.Print("\nPress Enter to exit...")
		fmt.Scanln()
		os.Exit(1)
	}

	err = extractEmbeddedFiles(installDir)
	if err != nil {
		colorPrintln(colorRed, fmt.Sprintf("FATAL: %v", err))
		fmt.Print("\nPress Enter to exit...")
		fmt.Scanln()
		os.Exit(1)
	}

	err = registerContextMenu(installDir)
	if err != nil {
		colorPrintln(colorYellow, fmt.Sprintf("Warning: %v", err))
		colorPrintln(colorYellow, "You may need to register the context menu manually.")
	}

	err = createUninstallScript(installDir)
	if err != nil {
		colorPrintln(colorYellow, fmt.Sprintf("Warning: %v", err))
	}

	if verifyInstallation(installDir) {
		fmt.Println()
		colorPrintln(colorGreen, "Installation Complete!")
		fmt.Println()
		colorPrintln(colorCyan, "Installed to: "+installDir)
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  Right-click any .mp4 file and select 'Compress Video'")
		fmt.Println()
		fmt.Println("To uninstall:")
		fmt.Println("  Run: " + filepath.Join(installDir, "uninstall.ps1"))
		fmt.Println()
	} else {
		colorPrintln(colorYellow, "Please check for errors above and try again.")
		fmt.Println()
	}

	fmt.Print("Press Enter to close...")
	fmt.Scanln()
}
