
[Setup]
AppName=Ollama Portable
AppVersion=1.0.3
DefaultDirName={src}\OllamaPortable
DisableDirPage=yes
DisableProgramGroupPage=yes
Uninstallable=no
OutputDir=..\dist
OutputBaseFilename=OllamaPortable_Installer
Compression=lzma2/ultra64
SolidCompression=yes

[Files]
Source: ".\app.ico"; DestDir: "{app}"; Flags: ignoreversion
; Copy the main executable
Source: "..\dist\windows-amd64\ollama.exe"; DestDir: "{app}"; Flags: ignoreversion
; Copy the entire lib folder (CUDA, ROCm, CPU backends)
Source: "..\dist\windows-amd64\lib\*"; DestDir: "{app}\lib"; Flags: ignoreversion recursesubdirs createallsubdirs

[Icons]
; Optional: Create a shortcut on the desktop to start the server
Name: "{commondesktop}\Ollama Server"; Filename: "{app}\ollama.exe"; Parameters: "serve"; IconFilename: "{app}\app.ico"