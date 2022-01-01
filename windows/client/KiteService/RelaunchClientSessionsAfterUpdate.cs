using System;
using System.Collections.Generic;
using System.Diagnostics;
using System.IO;
using System.Linq;
using System.Reflection;
using System.Runtime.InteropServices;
using System.Text;
using Microsoft.Win32;

namespace KiteService {

    internal static class ClientAppSessions {

        private static List<uint> GetRunningClientAppSessionIDsOrderedByStartTime() {
            Process[] clientAppProcesses = Process.GetProcessesByName("kited");

            var startTimes = new DateTime[clientAppProcesses.Length];
            for (int i = 0; i < clientAppProcesses.Length; i++) {
                try {
                    startTimes[i] = clientAppProcesses[i].StartTime;
                } catch (Exception e) {
                    Log.LogError("Exception trying to get kited process start time", e);
                    startTimes[i] = DateTime.MaxValue;
                }
            }
            Array.Sort(startTimes, clientAppProcesses);

            List<uint> sessionIDs = new List<uint>();
            foreach (var clientAppProcess in clientAppProcesses) {
                uint sessionID;
                if (!ProcessIdToSessionId((uint)clientAppProcess.Id, out sessionID)) {
                    // some kind of error
                    continue;
                }

                sessionIDs.Add(sessionID);
            }
            return sessionIDs;
        }

        internal static void RememberClientAppSessionIDsForUpdate() {
            // we can't (easily) gracefully shut down the client app since it is running on the user's
            //   desktop so we can't just PostMessage() to it from Session0.
            // but we will save info about its current state so we can try to re-launch is after the update
            // this will work even for multiple users running the app in different desktop sessions

            try {
                var sessionIDs = GetRunningClientAppSessionIDsOrderedByStartTime();
                var sessionIDsAsStrings = new List<string>();
                foreach (var sessionID in sessionIDs) {
                    sessionIDsAsStrings.Add(sessionID.ToString());

                    Log.LogMessage(String.Format(
                        "Remembering client app session information for relaunch after update.  (Session {0})",
                        sessionID));
                }

                using (var regKey = Registry.LocalMachine.CreateSubKey(KiteService.PerInstallationRegistryPath)) {
                    regKey.SetValue("UpdateUserSessionIDs", string.Join(",", sessionIDsAsStrings.ToArray()), RegistryValueKind.String);
                    regKey.SetValue("UpdateUserSessionIDTimestamp", DateTime.UtcNow.Ticks, RegistryValueKind.QWord);
                }
            } catch (Exception e) {
                Log.LogError("Exception trying to record client app session ID", e);
            }
        }

        internal static void RestartAllClientAppSessionsFromUpdate() {
            try {
                // read session IDs to launch kited in from the registry (and delete them from the registry)
                string[] sessionIDStrings;
                long sessionIDsTimestamp;
                using (var regKey = Registry.LocalMachine.OpenSubKey(KiteService.PerInstallationRegistryPath, true)) {
                    if (regKey == null) {
                        return;
                    }

                    sessionIDStrings = ((regKey.GetValue("UpdateUserSessionIDs") as string) ?? string.Empty).Split(new char[] { ',' },
                        StringSplitOptions.RemoveEmptyEntries);
                    sessionIDsTimestamp = (regKey.GetValue("UpdateUserSessionIDTimestamp") as long?).GetValueOrDefault(0L);

                    regKey.DeleteValue("UpdateUserSessionIDs", false);
                    regKey.DeleteValue("UpdateUserSessionIDTimestamp", false);
                }

                // check which session IDs, if any, have kited already running
                var sessionIDsAlreadyRunningKited = GetRunningClientAppSessionIDsOrderedByStartTime();

                // if the same user had multiple instances of kited open, only relaunch one per user.
                // this is a little more convoluted because the HashSet isn't guaranteed to keep a stable
                //   ordering, so we can't use it for enumeration.
                var relaunchedSessionIDs = new HashSet<uint>();
                foreach (var sessionIDString in sessionIDStrings) {
                    uint sessionID;
                    try {
                        sessionID = uint.Parse(sessionIDString);
                    } catch (Exception eParse) {
                        Log.LogError("Exception trying to parse session ID", eParse);
                        continue;
                    }

                    if (sessionIDsAlreadyRunningKited.Contains(sessionID)) {
                        continue;
                    }

                    if (relaunchedSessionIDs.Contains(sessionID)) {
                        continue;
                    }
                    relaunchedSessionIDs.Add(sessionID);

                    RestartClientAppSessionFromUpdate(sessionID, sessionIDsTimestamp);
                }
            } catch (Exception e) {
                Log.LogError("Exception trying to record client app session ID", e);
            }
        }

        private static void RestartClientAppSessionFromUpdate(uint sessionID, long sessionIDTimestamp) {
            if (sessionIDTimestamp <= 0 ||
                    DateTime.UtcNow.Subtract(new DateTime(sessionIDTimestamp)) > TimeSpan.FromMinutes(5)) {

                Log.LogMessage(string.Format(
                    "Not relaunching client app after an update (sessionID = {0}, timestamp = {1}).",
                    sessionID, sessionIDTimestamp));
                return;
            }

            Log.LogMessage("Trying to relaunch client app after an update...");

            // try to find the path to the Kite client app
            var clientAppPath = GetClientAppPath();
            if (!File.Exists(clientAppPath)) {
                Log.LogMessage("Client app could not be found.");
                return;
            }

            // Get the user token.  Note the session could have been closed, etc.
            IntPtr userToken;
            if (!WTSQueryUserToken((uint)sessionID, out userToken)) {
                Log.LogMessage("Query user token failed");
                return;
            }
            try {
                IntPtr environmentBlock;
                if (!CreateEnvironmentBlock(out environmentBlock, userToken, false)) {
                    Log.LogMessage("Creating environment block failed");
                    return;
                }
                try {
                    var pathWithArgs = "\"" + clientAppPath + "\" --relaunch-after-update";
                    var startupInfo = new STARTUPINFO();
                    startupInfo.lpDesktop = @"winsta0\default";
                    startupInfo.cb = Marshal.SizeOf(startupInfo);
                    PROCESS_INFORMATION processInfo;
                    if (CreateProcessAsUser(userToken, clientAppPath, pathWithArgs, IntPtr.Zero,
                            IntPtr.Zero, false, (uint)CreateProcessFlags.CREATE_UNICODE_ENVIRONMENT,
                            environmentBlock, Path.GetDirectoryName(clientAppPath),
                            ref startupInfo, out processInfo)) {

                        CloseHandle(processInfo.hProcess);
                        CloseHandle(processInfo.hThread);

                    } else {
                        Log.LogMessage("Creating process as user failed");
                        return;
                    }
                } finally {
                    DestroyEnvironmentBlock(environmentBlock);
                }
            } finally {
                CloseHandle(userToken);
            }

            Log.LogMessage("Client app restarted successfully.");
        }

        private static string GetClientAppPath() {
            return Path.Combine(Path.GetDirectoryName(Assembly.GetExecutingAssembly().Location),
                "kited.exe");
        }

        [DllImport("kernel32.dll")]
        private static extern bool ProcessIdToSessionId(uint dwProcessId, out uint pSessionId);

        [DllImport("wtsapi32.dll", SetLastError = true)]
        private static extern bool WTSQueryUserToken(UInt32 sessionId, out IntPtr Token);

        [DllImport("advapi32.dll", SetLastError = true, CharSet = CharSet.Auto)]
        private static extern bool CreateProcessAsUser(IntPtr hToken, string lpApplicationName,
            string lpCommandLine, IntPtr lpProcessAttributes, IntPtr lpThreadAttributes,
            bool bInheritHandles, uint dwCreationFlags, IntPtr lpEnvironment, string lpCurrentDirectory,
            ref STARTUPINFO lpStartupInfo, out PROCESS_INFORMATION lpProcessInformation);
        [StructLayout(LayoutKind.Sequential)]
        private struct PROCESS_INFORMATION {
            public IntPtr hProcess;
            public IntPtr hThread;
            public int dwProcessId;
            public int dwThreadId;
        }
        [StructLayout(LayoutKind.Sequential, CharSet = CharSet.Unicode)]
        private struct STARTUPINFO {
            public Int32 cb;
            public string lpReserved;
            public string lpDesktop;
            public string lpTitle;
            public Int32 dwX;
            public Int32 dwY;
            public Int32 dwXSize;
            public Int32 dwYSize;
            public Int32 dwXCountChars;
            public Int32 dwYCountChars;
            public Int32 dwFillAttribute;
            public Int32 dwFlags;
            public Int16 wShowWindow;
            public Int16 cbReserved2;
            public IntPtr lpReserved2;
            public IntPtr hStdInput;
            public IntPtr hStdOutput;
            public IntPtr hStdError;
        }
        [Flags]
        private enum CreateProcessFlags {
            CREATE_BREAKAWAY_FROM_JOB = 0x01000000,
            CREATE_DEFAULT_ERROR_MODE = 0x04000000,
            CREATE_NEW_CONSOLE = 0x00000010,
            CREATE_NEW_PROCESS_GROUP = 0x00000200,
            CREATE_NO_WINDOW = 0x08000000,
            CREATE_PROTECTED_PROCESS = 0x00040000,
            CREATE_PRESERVE_CODE_AUTHZ_LEVEL = 0x02000000,
            CREATE_SEPARATE_WOW_VDM = 0x00000800,
            CREATE_SHARED_WOW_VDM = 0x00001000,
            CREATE_SUSPENDED = 0x00000004,
            CREATE_UNICODE_ENVIRONMENT = 0x00000400,
            DEBUG_ONLY_THIS_PROCESS = 0x00000002,
            DEBUG_PROCESS = 0x00000001,
            DETACHED_PROCESS = 0x00000008,
            EXTENDED_STARTUPINFO_PRESENT = 0x00080000,
            INHERIT_PARENT_AFFINITY = 0x00010000
        }

        [DllImport("userenv.dll", SetLastError = true)]
        private static extern bool CreateEnvironmentBlock(out IntPtr lpEnvironment, IntPtr hToken, bool bInherit);
        [DllImport("userenv.dll", SetLastError = true)]
        [return: MarshalAs(UnmanagedType.Bool)]
        private static extern bool DestroyEnvironmentBlock(IntPtr lpEnvironment);

        [DllImport("kernel32.dll", SetLastError = true)]
        private static extern bool CloseHandle(IntPtr hHandle);
    }
}
