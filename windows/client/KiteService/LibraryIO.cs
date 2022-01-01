using System;
using System.Collections.Generic;
using System.Diagnostics;
using System.Text;
using System.Threading;
using System.IO;
using System.Net;
using System.Reflection;
using System.Security.Cryptography;
using System.Security.Cryptography.X509Certificates;
using System.Security.Cryptography.Xml;
using System.Xml;
using Microsoft.Win32;

namespace KiteService {

    internal enum CommonDirectories {
        CurrentExecutingDirectory,
        LocalAppData,
        Temp
    }

    internal static partial class LibraryIO {

        internal static bool CanWriteToDirectory(string dirPath) {
            try {
                if (!Directory.Exists(dirPath)) {
                    Directory.CreateDirectory(dirPath);
                }

                var tmpFilePath = Path.Combine(dirPath, new Random().Next().ToString());
                File.Create(tmpFilePath).Dispose();
                File.Delete(tmpFilePath);
                return true;
            } catch {
                return false;
            }
        }

        internal static string FindWritableDirectory(params CommonDirectories[] preferenceOrdering) {
            foreach (var candidate in preferenceOrdering) {
                string candidatePath;
                switch (candidate) {
                    case CommonDirectories.CurrentExecutingDirectory:
                        candidatePath = Path.GetDirectoryName(Assembly.GetExecutingAssembly().Location);
                        break;

                    case CommonDirectories.LocalAppData:
                        candidatePath = Path.Combine(Environment.GetFolderPath(
                            Environment.SpecialFolder.LocalApplicationData), "Kite");
                        break;

                    case CommonDirectories.Temp:
                        candidatePath = Path.GetTempPath();
                        break;

                    default:
                        // I'd ordinarily throw an exception here, but I'm weary to because of
                        // this function's use in the updater.  Just skip unrecognized 
                        // CommonDirectories.
                        continue;
                }

                if (LibraryIO.CanWriteToDirectory(candidatePath)) {
                    return candidatePath;
                }
            }
            return null;
        }
    
    }
}
