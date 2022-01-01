using System;
using System.Collections.Generic;
using System.Diagnostics;
using System.Linq;
using System.Text;
using System.Runtime.InteropServices;

namespace KiteService {

    internal static class Log {

        internal static void LogMessage(string logMessage) {
#if DEBUG
            Console.WriteLine(logMessage);
#endif

            try {
                if (KiteService.IsServiceDebugFlagSet()) {
                    try {
                        var eventSourceName = "KiteService";

                        if (!EventLog.SourceExists(eventSourceName)) {
                            EventLog.CreateEventSource(eventSourceName, "Application");
                        }

                        EventLog.WriteEntry(eventSourceName, logMessage);
                    } catch {
                    }

                    try {
                        OutputDebugString("KiteService: " + logMessage);
                    } catch {
                    }
                }
            } catch { }
        }

        internal static void LogExeptionWithLevel(string logMessage, Exception e, string level) {
            try {
                var parts = new List<string>(new string[] { logMessage, level });
                if (e != null) {
                    parts.AddRange(new string[] {
                        "Message: " + e.Message, "StackTrace: " + e.StackTrace,
                        "InnerMessage: " + (e.InnerException == null ? string.Empty : e.InnerException.Message),
                        "InnerStackTrace: " + (e.InnerException == null ? string.Empty : e.InnerException.StackTrace)
                    });
                }
                LogMessage(string.Join(" -- ", parts.ToArray()));
            } catch { }
        }

        internal static void LogError(string logMessage, Exception e) {
            LogExeptionWithLevel(logMessage, e, "Error");
        }

        internal static void LogError(string logMessage) {
            LogExeptionWithLevel(logMessage, null, "Error");
        }

        internal static void LogWarning(string logMessage, Exception e) {
            LogExeptionWithLevel(logMessage, e, "Warning");
        }

        internal static void LogWarning(string logMessage) {
            LogExeptionWithLevel(logMessage, null, "Warning");
        }

        [DllImport("kernel32.dll", CharSet = CharSet.Unicode)]
        internal static extern void OutputDebugString(string message);
    }
}
