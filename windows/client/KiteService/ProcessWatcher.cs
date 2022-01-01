using System;
using System.IO;
using System.IO.Compression;
using System.Management;
using System.Net;
using System.Runtime.InteropServices;
using System.Text;

namespace KiteService
{
    internal partial class ProcessWatcher
    {
        private static readonly string k_reportCrashUrl = "https://alpha.kite.com/windowscrash?platform={0}&machineid={1}&filename={2}";
        private static ManagementEventWatcher processStartEvent;
        private static ManagementEventWatcher processStopEvent;
        private static string k_machineID;
        private static UInt32 k_sessionID;

        internal static void Watch()
        {
            k_machineID = MachineID.GetMachineIDAndCreateIfNecessary();
            processStartEvent = new ManagementEventWatcher("SELECT * FROM Win32_ProcessStartTrace WHERE ProcessName = 'kited.exe'");
            processStopEvent = new ManagementEventWatcher("SELECT * FROM Win32_ProcessStopTrace WHERE ProcessName = 'kited.exe'");
            processStartEvent.EventArrived += new EventArrivedEventHandler(processStartEvent_EventArrived);
            processStartEvent.Start();
            processStopEvent.EventArrived += new EventArrivedEventHandler(processStopEvent_EventArrived);
            processStopEvent.Start();
        }
 
        internal static void processStartEvent_EventArrived(object sender, EventArrivedEventArgs e)
        {
            try {
                string processName = e.NewEvent.Properties["ProcessName"].Value.ToString();
                string processID = Convert.ToInt32(e.NewEvent.Properties["ProcessID"].Value).ToString();
                k_sessionID = Convert.ToUInt32(e.NewEvent.Properties["SessionID"].Value);
                Log.LogMessage(string.Format("Process started. Name: {0}, ID: {1}, Session ID: {2}", processName, processID, k_sessionID));
            } catch (Exception ex)
            {
                Log.LogError("Error listening for process start event", ex);
            }
}
 
        internal static void processStopEvent_EventArrived(object sender, EventArrivedEventArgs e)
        {
            try
            {
                string processName = e.NewEvent.Properties["ProcessName"].Value.ToString();
                string processID = Convert.ToInt32(e.NewEvent.Properties["ProcessID"].Value).ToString();
                Int32 exitStatus = Convert.ToInt32(e.NewEvent.Properties["ExitStatus"].Value);
                if (exitStatus > 1)
                {
                    Log.LogMessage(string.Format("Process crashed. Name: {0}, ID: {1}, ExitStatus: {2}", processName, processID, exitStatus));

                    // Create filename for sending client log
                    DateTime ts = DateTime.UtcNow;
                    var filename = string.Format("client.log.{0}.bak", ts.ToString("yyyy-MM-dd_hh-mm-ss-tt"));
                    Log.LogMessage(filename);

                    // Get the user token from stored session id. Note the session could have been closed, etc.
                    IntPtr userToken;
                    if (!WTSQueryUserToken(k_sessionID, out userToken))
                    {
                        Log.LogMessage("Query user token failed");
                        return;
                    }

                    // Get LocalAppData folder location
                    string appDir;
                    IntPtr pPath;
                    if (SHGetKnownFolderPath(LocalAppData, 0, userToken, out pPath) == 0)
                    {
                        appDir = System.Runtime.InteropServices.Marshal.PtrToStringUni(pPath);
                        System.Runtime.InteropServices.Marshal.FreeCoTaskMem(pPath);
                    }
                    else
                    {
                        Log.LogMessage("Get known folder path failed");
                        return;
                    }

                    // Try to find the log file
                    string logPath = Path.Combine(appDir, "Kite");
                    logPath = Path.Combine(logPath, "logs");
                    logPath = Path.Combine(logPath, "client.log");
                    if (!File.Exists(logPath))
                    {
                        Log.LogMessage(string.Format("Could not find log file {0}", logPath));
                        return;
                    }

                    string postData;
                    try
                    {
                        using (StreamReader sr = new StreamReader(logPath))
                        {
                            postData = sr.ReadToEnd();
                        }
                    }
                    catch (IOException ex)
                    {
                        Log.LogWarning("Exception while trying to read crash log.", ex);
                        return;
                    }

                    // Compress log file contents
                    byte[] buffer = Encoding.UTF8.GetBytes(postData);
                    var memoryStream = new MemoryStream();
                    using (var gZipStream = new GZipStream(memoryStream, CompressionMode.Compress, true))
                    {
                        gZipStream.Write(buffer, 0, buffer.Length);
                        gZipStream.Close();
                    }

                    memoryStream.Position = 0;
                    var compressedData = new byte[memoryStream.Length];
                    memoryStream.Read(compressedData, 0, compressedData.Length);

                    var gZipBuffer = new byte[compressedData.Length];
                    Buffer.BlockCopy(compressedData, 0, gZipBuffer, 0, compressedData.Length);

                    // make the HTTP POST request with the client log
                    var request = (HttpWebRequest)WebRequest.Create(
                    String.Format(k_reportCrashUrl, "windows", k_machineID, filename));
                    request.KeepAlive = false;

                    request.Method = "POST";
                    request.ContentType = "application/text";
                    request.ContentLength = gZipBuffer.Length;
                    request.Headers.Add("Content-Encoding", "gzip");

                    try
                    {
                        using (var stream = request.GetRequestStream())
                        {
                            stream.Write(gZipBuffer, 0, gZipBuffer.Length);
                            stream.Close();
                        }
                        using (var responseObject = request.GetResponse())
                        using (var responseReader = new StreamReader(responseObject.GetResponseStream()))
                        {
                            Log.LogMessage(responseReader.ReadToEnd());
                        }
                    }
                    catch (Exception ex)
                    {
                        Log.LogWarning("Exception while trying to send crash log.", ex);
                    }
                }
                else
                {
                    Log.LogMessage(string.Format("Process stopped normally. Name: {0}, ID: {1}, ExitStatus: {2}", processName, processID, exitStatus));
                };
            } catch (Exception ex)
            {
                Log.LogError("Error listening for process stop event", ex);
            }
        }

        public static readonly Guid LocalAppData = new Guid("F1B32785-6FBA-4FCF-9D55-7B8E7F157091");

        [DllImport("wtsapi32.dll", SetLastError = true)]
        private static extern bool WTSQueryUserToken(UInt32 sessionId, out IntPtr Token);

        [DllImport("shell32.dll")]
        static extern int SHGetKnownFolderPath(
            [MarshalAs(UnmanagedType.LPStruct)] Guid rfid,
            uint dwFlags,
            IntPtr hToken,
            out IntPtr pszPath  // API uses CoTaskMemAlloc
        );
    }
}