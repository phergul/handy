param([string]$inputPath)
$inputPath | Out-File "$env:TEMP\mp4_compress_debug.txt"
"Test-Path result: $(Test-Path $inputPath)" | Out-File "$env:TEMP\mp4_compress_debug.txt" -Append
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
Add-Type -AssemblyName PresentationFramework

[xml]$Xaml = @"
<Window
    xmlns="http://schemas.microsoft.com/winfx/2006/xaml/presentation"
    xmlns:x="http://schemas.microsoft.com/winfx/2006/xaml"
    x:Name="Window">
</Window>
"@
$Reader = (New-Object System.Xml.XmlNodeReader $Xaml)
$Window = [Windows.Markup.XamlReader]::Load($Reader)
[System.Windows.Forms.Application]::EnableVisualStyles()

$ErrorActionPreference = "Stop"
trap {
    [System.Windows.Forms.MessageBox]::Show("Error: $($_.Exception.Message)", "Compression Error")
	Read-Host "Press Enter to close"
    exit 1
}
if (-not $inputPath -or -not (Test-Path $inputPath)) {
    [System.Windows.Forms.MessageBox]::Show("No valid input file path received.`nInput: $inputPath","Error")
	Read-Host "Press Enter to close"
    exit 1
}

function Get-TargetSizeDialog {
    param(
        [int]$Default = 50,
        [string]$Title = "Compress Video",
        [string]$Prompt = "Enter target file size in MB:"
    )

    $form = New-Object System.Windows.Forms.Form
    $form.Text = $Title
    $form.Size = New-Object System.Drawing.Size(360,160)
    $form.StartPosition = 'CenterScreen'
    $form.FormBorderStyle = 'FixedDialog'
    $form.MaximizeBox = $false
    $form.MinimizeBox = $false
    $form.TopMost = $true

    $label = New-Object System.Windows.Forms.Label
    $label.Text = $Prompt
    $label.AutoSize = $true
    $label.Location = New-Object System.Drawing.Point(12,15)

    $nud = New-Object System.Windows.Forms.NumericUpDown
    $nud.Minimum = 1
    $nud.Maximum = 102400
    $nud.Value = [decimal]$Default
    $nud.Location = New-Object System.Drawing.Point(15,40)
    $nud.Size = New-Object System.Drawing.Size(120,20)

    $ok = New-Object System.Windows.Forms.Button
    $ok.Text = "OK"
    $ok.DialogResult = [System.Windows.Forms.DialogResult]::OK
    $ok.Location = New-Object System.Drawing.Point(160,70)

    $cancel = New-Object System.Windows.Forms.Button
    $cancel.Text = "Cancel"
    $cancel.DialogResult = [System.Windows.Forms.DialogResult]::Cancel
    $cancel.Location = New-Object System.Drawing.Point(240,70)

    $form.Controls.AddRange(@($label,$nud,$ok,$cancel))
    $form.AcceptButton = $ok
    $form.CancelButton = $cancel

    $result = $form.ShowDialog()
    if ($result -eq [System.Windows.Forms.DialogResult]::OK) { return [int]$nud.Value } else { return $null }
}

$targetSize = Get-TargetSizeDialog -Default 50

if (-not $targetSize -or $targetSize -eq "0") {
    [System.Windows.Forms.MessageBox]::Show("No size entered. Cancelling.","Cancelled")
    exit
}

$result = [System.Windows.Forms.MessageBox]::Show(
    "Do you want to select an output file location?",
    "Output Location",
    [System.Windows.Forms.MessageBoxButtons]::YesNo
)

if ($result -eq [System.Windows.Forms.DialogResult]::Yes) {
    $dialog = New-Object System.Windows.Forms.SaveFileDialog
    $dialog.Filter = "MP4 files (*.mp4)|*.mp4"
    $dialog.filename = [System.IO.Path]::GetFileNameWithoutExtension($inputPath) + "_compressed.mp4"
    $dialog.InitialDirectory = [System.IO.Path]::GetDirectoryName($inputPath)
    if ($dialog.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) {
        $outputPath = $dialog.FileName
    } else {
        exit
    }
}
else {
    $dir = [System.IO.Path]::GetDirectoryName($inputPath)
    $base = [System.IO.Path]::GetFileNameWithoutExtension($inputPath)
    $outputPath = Join-Path $dir ($base + "_compressed.mp4")
}

$compressor = Join-Path $env:LOCALAPPDATA "mp4_compress\mp4_compress.exe"

if (-not (Test-Path $compressor)) {
    [System.Windows.Forms.MessageBox]::Show("Compressor not found at $compressor","Error")
    exit
}

Start-Process -NoNewWindow -Wait -FilePath $compressor -ArgumentList "`"$inputPath`"", $targetSize, "`"$outputPath`""

[System.Windows.Forms.MessageBox]::Show("Compression complete:`n$outputPath","Done")	
Read-Host "Press Enter to close"
