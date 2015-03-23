; build for windows
[Setup]
AppName=hickwall
AppVersion=0.1.0
DefaultDirName={pf}\hickwall
DefaultGroupName=hickwall
;DisableProgramGroupPage=yes
UninstallDisplayIcon={app}\hickwall.exe
Compression=lzma2
SolidCompression=yes
; OutputDir=

[Files]
Source: "hickwall.exe"; DestDir: "{app}"; Permissions: users-readexec; Flags: overwritereadonly replacesameversion touch
Source: "config.yml.example"; DestDir: "{app}"
Source: "Readme.md"; DestDir: "{app}"
Source: "Readme.html"; DestDir: "{app}"; Flags: isreadme
Source: "start.bat"; DestDir: "{app}"
Source: "stop.bat"; DestDir: "{app}"

[Icons]
Name: "{group}\start hickwall"; Filename: "{app}\start.bat"; WorkingDir: "{app}"
Name: "{group}\stop hickwall"; Filename: "{app}\stop.bat"; WorkingDir: "{app}"
Name: "{group}\Readme.html"; Filename: "{app}\Readme.html"
Name: "{group}\uninstall"; Filename: "{uninstallexe}";

[Run]
Filename: "{app}\hickwall.exe"; Parameters: "service install"; Flags: runhidden; AfterInstall: TheAfterInstall

[UninstallRun]
Filename: "{sys}\taskkill.exe"; Parameters: "/F /FI 'IMAGENAME eq hickwall*'"; Flags: runhidden 
Filename: "{app}\hickwall.exe"; Parameters: "service stop"; Flags: runhidden
Filename: "{app}\hickwall.exe"; Parameters: "service remove"; Flags: runhidden

[Code]
var 
  IsRunningBeforeInstall: Boolean;
  FileName: String;
  WorkingDir: String;

function PrepareToInstall(var NeedsRestart: Boolean): String;
var
  ResultCode: Integer;
begin
  FileName := ExpandConstant('{app}\hickwall.exe')
  WorkingDir := ExpandConstant('{app}')
  IsRunningBeforeInstall := False

  if FileExists(FileName) = True then
    if Exec(FileName, 'service statuscode', WorkingDir, SW_HIDE, ewWaitUntilTerminated, ResultCode) then
    begin
      if ResultCode = 4 then 
      begin
        IsRunningBeforeInstall := True
        //MsgBox('isrunning: ' #13#13 'true' , mbInformation, MB_OK)
      end;
    end;

    if IsRunningBeforeInstall = True then
    begin
      //MsgBox('isrunning: ' #13#13 'true 1' , mbInformation, MB_OK)
      //if not Exec(FileName, 'service stop', WorkingDir, SW_HIDE, ewWaitUntilTerminated, ResultCode) then
      if not Exec(ExpandConstant('{sys}\taskkill.exe'), '/F /FI "IMAGENAME eq hickwall.exe*"', WorkingDir, SW_HIDE, ewWaitUntilTerminated, ResultCode) then
      begin
        //handle failure
        MsgBox('PrepareToInstall' #13#13 'stop service failed ' + IntToStr(ResultCode), mbInformation, MB_OK)
        Result := 'stop service failed'
      end;
    end;
end;


procedure TheAfterInstall();
var
  ResultCode: Integer;
begin
  //MsgBox('isrunning: ' #13#13 'after install' , mbInformation, MB_OK)
 
  if IsRunningBeforeInstall = True then
  begin
    //MsgBox('isrunning: ' #13#13 'true 2' , mbInformation, MB_OK)
    if not Exec(FileName, 'service start', WorkingDir, SW_HIDE, ewWaitUntilTerminated, ResultCode) then
    begin
      MsgBox('start service' #13#13 'start service failed. ' + IntToStr(ResultCode), mbInformation, MB_OK)
    end;
  end;
end;
