# Run developer commands inside an owned, hidden console. Unlike CreateNoWindow,
# a hidden parent console is inherited by console children which do not set their
# own creation flags (notably the Go toolchain and older test fixtures).
# Usage:
#   . .\scripts\run-hidden.ps1
#   Invoke-HiddenTask -ScriptPath tmp\checks.ps1 -LogPath tmp\checks.log
# The script is passed as one literal argv item, never interpolated as shell code.
Set-StrictMode -Version Latest

if (-not ('WhaleIsle.Dev.HiddenTask' -as [type])) {
    Add-Type -TypeDefinition @'
using System;
using System.ComponentModel;
using System.Runtime.InteropServices;
using System.Text;

namespace WhaleIsle.Dev {
    public static class HiddenTask {
        const uint CREATE_NEW_CONSOLE = 0x10;
        const uint CREATE_SUSPENDED = 0x4;
        const uint STARTF_USESTDHANDLES = 0x100;
        const uint STARTF_USESHOWWINDOW = 1;
        const uint INFINITE = 0xffffffff;
        const uint WAIT_TIMEOUT = 258;
        const uint JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE = 0x2000;
        [StructLayout(LayoutKind.Sequential)] struct SECURITY_ATTRIBUTES {
            public int length; public IntPtr descriptor; public int inherit;
        }
        [StructLayout(LayoutKind.Sequential, CharSet=CharSet.Unicode)] struct STARTUPINFO {
            public int cb; public string reserved; public string desktop; public string title;
            public uint x,y,xSize,ySize,xCountChars,yCountChars,fillAttribute,flags;
            public ushort showWindow,reserved2; public IntPtr reservedPtr;
            public IntPtr input,output,error;
        }
        [StructLayout(LayoutKind.Sequential)] struct PROCESS_INFORMATION {
            public IntPtr process,thread; public uint processId,threadId;
        }
        [StructLayout(LayoutKind.Sequential)] struct BASIC_LIMITS {
            public long processTime,jobTime; public uint flags;
            public UIntPtr minWorkingSet,maxWorkingSet; public uint activeProcesses;
            public UIntPtr affinity; public uint priority,scheduling;
        }
        [StructLayout(LayoutKind.Sequential)] struct IO_COUNTERS {
            public ulong readOps,writeOps,otherOps,readBytes,writeBytes,otherBytes;
        }
        [StructLayout(LayoutKind.Sequential)] struct EXTENDED_LIMITS {
            public BASIC_LIMITS basic; public IO_COUNTERS io;
            public UIntPtr processMemory,jobMemory,peakProcessMemory,peakJobMemory;
        }
        [DllImport("kernel32.dll", CharSet=CharSet.Unicode, SetLastError=true)]
        static extern IntPtr CreateFileW(string name,uint access,uint share,ref SECURITY_ATTRIBUTES security,uint creation,uint flags,IntPtr template);
        [DllImport("kernel32.dll", CharSet=CharSet.Unicode, SetLastError=true)]
        static extern bool CreateProcessW(string app,StringBuilder command,IntPtr processSecurity,IntPtr threadSecurity,bool inherit,uint flags,IntPtr environment,string directory,ref STARTUPINFO startup,out PROCESS_INFORMATION process);
        [DllImport("kernel32.dll", SetLastError=true)] static extern IntPtr CreateJobObjectW(IntPtr security,IntPtr name);
        [DllImport("kernel32.dll", SetLastError=true)] static extern bool SetInformationJobObject(IntPtr job,int kind,ref EXTENDED_LIMITS info,int size);
        [DllImport("kernel32.dll", SetLastError=true)] static extern bool AssignProcessToJobObject(IntPtr job,IntPtr process);
        [DllImport("kernel32.dll", SetLastError=true)] static extern uint ResumeThread(IntPtr thread);
        [DllImport("kernel32.dll", SetLastError=true)] static extern uint WaitForSingleObject(IntPtr handle,uint timeout);
        [DllImport("kernel32.dll", SetLastError=true)] static extern bool GetExitCodeProcess(IntPtr process,out uint code);
        [DllImport("kernel32.dll", SetLastError=true)] static extern bool TerminateProcess(IntPtr process,uint code);
        [DllImport("kernel32.dll", SetLastError=true)] static extern bool TerminateJobObject(IntPtr job,uint code);
        [DllImport("kernel32.dll")] static extern bool CloseHandle(IntPtr handle);
        [DllImport("kernel32.dll")] static extern IntPtr GetConsoleWindow();
        [DllImport("user32.dll")] static extern bool IsWindowVisible(IntPtr handle);
        static string Quote(string value) {
            if (value.IndexOf('\0') >= 0) throw new ArgumentException("NUL in process argument");
            var result = new StringBuilder("\""); int slashes = 0;
            foreach (char c in value) {
                if (c == '\\') { slashes++; continue; }
                if (c == '"') result.Append('\\',slashes*2+1).Append(c);
                else result.Append('\\',slashes).Append(c);
                slashes = 0;
            }
            return result.Append('\\',slashes*2).Append('"').ToString();
        }
        public static string ConsoleState() {
            IntPtr console=GetConsoleWindow();
            return "console="+console.ToInt64()+" visible="+(console!=IntPtr.Zero&&IsWindowVisible(console));
        }
        public static int Run(string executable,string script,string directory,string log,int timeoutMs) {
            var security = new SECURITY_ATTRIBUTES { length=Marshal.SizeOf<SECURITY_ATTRIBUTES>(),inherit=1 };
            IntPtr output=IntPtr.Zero,input=IntPtr.Zero,job=IntPtr.Zero;
            PROCESS_INFORMATION process = new PROCESS_INFORMATION();
            bool assigned=false;
            try {
                output=CreateFileW(log,0x40000000,3,ref security,2,0x80,IntPtr.Zero);
                if(output==new IntPtr(-1)) throw new Win32Exception(Marshal.GetLastWin32Error(),"Create hidden task log");
                input=CreateFileW("NUL",0x80000000,3,ref security,3,0x80,IntPtr.Zero);
                if(input==new IntPtr(-1)) throw new Win32Exception(Marshal.GetLastWin32Error(),"Open hidden task input");
                job=CreateJobObjectW(IntPtr.Zero,IntPtr.Zero);
                if(job==IntPtr.Zero) throw new Win32Exception(Marshal.GetLastWin32Error(),"Create hidden task job");
                var limits=new EXTENDED_LIMITS(); limits.basic.flags=JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE;
                if(!SetInformationJobObject(job,9,ref limits,Marshal.SizeOf<EXTENDED_LIMITS>())) throw new Win32Exception(Marshal.GetLastWin32Error(),"Set hidden task job limits");
                var startup = new STARTUPINFO { cb=Marshal.SizeOf<STARTUPINFO>(),flags=STARTF_USESTDHANDLES|STARTF_USESHOWWINDOW,showWindow=0,input=input,output=output,error=output,title="WhaleIsle hidden checks" };
                var command=new StringBuilder(Quote(executable)+" -NoLogo -NoProfile -NonInteractive -File "+Quote(script));
                if(!CreateProcessW(executable,command,IntPtr.Zero,IntPtr.Zero,true,CREATE_NEW_CONSOLE|CREATE_SUSPENDED,IntPtr.Zero,directory,ref startup,out process)) throw new Win32Exception(Marshal.GetLastWin32Error(),"Create hidden task");
                if(!AssignProcessToJobObject(job,process.process)) throw new Win32Exception(Marshal.GetLastWin32Error(),"Assign hidden task job");
                assigned=true;
                if(ResumeThread(process.thread)==0xffffffff) throw new Win32Exception(Marshal.GetLastWin32Error(),"Resume hidden task");
                uint wait=WaitForSingleObject(process.process,timeoutMs<=0?INFINITE:(uint)timeoutMs);
                if(wait==WAIT_TIMEOUT) { TerminateJobObject(job,124); WaitForSingleObject(process.process,5000); return 124; }
                if(wait!=0) throw new Win32Exception(Marshal.GetLastWin32Error(),"Wait hidden task");
                uint code; if(!GetExitCodeProcess(process.process,out code)) throw new Win32Exception(Marshal.GetLastWin32Error(),"Read hidden task exit");
                return unchecked((int)code);
            } finally {
                if(process.process!=IntPtr.Zero&&!assigned) TerminateProcess(process.process,125);
                if(job!=IntPtr.Zero) CloseHandle(job);
                if(process.thread!=IntPtr.Zero) CloseHandle(process.thread);
                if(process.process!=IntPtr.Zero) CloseHandle(process.process);
                if(input!=IntPtr.Zero&&input!=new IntPtr(-1)) CloseHandle(input);
                if(output!=IntPtr.Zero&&output!=new IntPtr(-1)) CloseHandle(output);
            }
        }
    }
}
'@
}

function Invoke-HiddenTask {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)][string]$ScriptPath,
        [Parameter(Mandatory=$true)][string]$LogPath,
        [string]$WorkingDirectory = (Get-Location).Path,
        [int]$TimeoutSeconds = 600
    )
    $ErrorActionPreference = 'Stop'
    $taskScript = (Resolve-Path -LiteralPath $ScriptPath -ErrorAction Stop).Path
    $taskDirectory = (Resolve-Path -LiteralPath $WorkingDirectory -ErrorAction Stop).Path
    $taskLog = if ([System.IO.Path]::IsPathRooted($LogPath)) { [System.IO.Path]::GetFullPath($LogPath) } else { [System.IO.Path]::GetFullPath((Join-Path $taskDirectory $LogPath)) }
    if ($taskLog -eq $taskScript) { throw 'A task cannot overwrite its own script.' }
    $taskLogDirectory = Split-Path -Parent $taskLog
    [System.IO.Directory]::CreateDirectory($taskLogDirectory) | Out-Null
    $taskPwsh = (Get-Process -Id $PID).Path
    if (-not $taskPwsh) { throw 'The current PowerShell executable is unavailable.' }
    $taskMilliseconds = if ($TimeoutSeconds -le 0) { 0 } else { [Math]::Min([int64]$TimeoutSeconds*1000, [int]::MaxValue) }
    $taskExitCode = [WhaleIsle.Dev.HiddenTask]::Run($taskPwsh,$taskScript,$taskDirectory,$taskLog,[int]$taskMilliseconds)
    Write-Output "Hidden task exit=$taskExitCode log=$taskLog"
    if ($taskExitCode -ne 0) { throw "Hidden task failed with exit code $taskExitCode; inspect $taskLog" }
}
