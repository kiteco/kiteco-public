using System;
using System.Collections.Generic;
using System.Diagnostics;
using System.Text;
using System.Threading;
using System.IO;
using System.Net;
using System.Net.Cache;
using System.Reflection;
using System.Security.Cryptography;
using System.Security.Cryptography.X509Certificates;
using System.Security.Cryptography.Xml;
using System.Xml;
using Microsoft.Win32;

namespace KiteService {

    internal delegate void VoidDelegate();

    internal static class UpdateChecker {

        private static readonly string k_updateUrl = "https://windows.kite.com/windows/kite-app/update-check?version={0}&machine-id={1}&will-attempt-update-if-provided={2}&user-wants-beta-builds={3}";
        private static readonly TimeSpan k_checkFrequency = TimeSpan.FromMinutes(5);

        private static readonly string k_kitedReadyToRestartUrl = "http://127.0.0.1:46624/clientapi/update/readyToRestart";
        private static readonly TimeSpan k_maxKitedWaitDuration = TimeSpan.FromMinutes(120);  // NOTE: 2 hours
        private static readonly TimeSpan k_kitedWaitPollInterval = TimeSpan.FromSeconds(30);  // NOTE: 30 seconds

        private static readonly object k_checkForAndRunUpdatesLockObject = new object();

        private static System.Threading.Timer s_checkTimer;

        internal static void StartUpdateCheckerTimer() {
            var timeUntilFirstCheck = new TimeSpan((long)(k_checkFrequency.Ticks * new Random().NextDouble()));

            if (IsImmediateUpdateCheckFlagSet()) {
                timeUntilFirstCheck = TimeSpan.FromSeconds(10);
                Log.LogMessage("ImmediateUpdateCheck flag is set.");
            }

            s_checkTimer = new System.Threading.Timer(CheckForAndRunUpdates, null,
                timeUntilFirstCheck, k_checkFrequency);

            Log.LogMessage(String.Format("First update check scheduled for t-minus {0} minutes",
                timeUntilFirstCheck.TotalMinutes));

            WebRequest.DefaultCachePolicy = new RequestCachePolicy(RequestCacheLevel.NoCacheNoStore);
        }

        internal static void ForceUpdateCheckNow() {
            s_checkTimer.Change(TimeSpan.Zero, k_checkFrequency);
        }

        // could block for a while, but will execute on a ThreadPool thread.
        // 
        // don't let exceptions bubble past the scope of this function, or else bad things
        //   will happen since it's a ThreadPool thread on .NET 2.0+.
        // 
        // this function protects against concurrent execution on different threads.  because of this,
        //   don't worry that System.Threading.Timer can be reentrant.
        private static void CheckForAndRunUpdates(object nothing) {
            try {
                Log.LogMessage("Entered CheckForAndRunUpdates");

                // if another thread is currently executing here, just return, since there's no point in doing
                //   this twice in a row, back-to-back.  i.e. use TryEnter, not Enter
                if (Monitor.TryEnter(k_checkForAndRunUpdatesLockObject)) {
                    try {
                        // try to download update info
                        Log.LogMessage("Asking server for update...");
                        var updateInfoXml = AskServerForUpdate();
                        if (string.IsNullOrEmpty(updateInfoXml)) {
                            if (updateInfoXml == null) {
                                Log.LogMessage("Server response was null; exiting");
                            } else {
                                Log.LogMessage("Server response was empty; exiting");
                            }

                            return;
                        }

                        // decode and verify update info
                        Log.LogMessage("Verifying server response...");
                        var verifiedUpdateInfo = VerifiedUpdateInfo.FromUpdateXml(updateInfoXml,
                            KiteService.IsServiceDebugFlagSet());
                        Log.LogMessage("Received verified update info from server");

                        // try to download update executable
                        var downloadFileDir = LibraryIO.FindWritableDirectory(CommonDirectories.CurrentExecutingDirectory,
                            CommonDirectories.LocalAppData, CommonDirectories.Temp);
                        if (downloadFileDir == null) {
                            throw new Exception("Cannot find writable directory for update download.");
                        }
                        var downloadFilePath = Path.Combine(downloadFileDir, "KiteUpdater.exe");
                        bool needToDownload = true;
                        if (File.Exists(downloadFilePath)) {
                            // if we've downloaded this specific update exe in a previous session we need to reuse it, 
                            //   because there will exist users with:
                            //     a) appreciable delays between downloading and executing the update (because of waiting on kited),
                            //     b) short duration runs of KiteService (e.g. they restart their machine often), and
                            //     c) have slow internet connections -> long download times.
                            // it's safe to only look in the chosen writable directory vs all three possibilities since the writable
                            //   directory should be stable enough, e.g. we won't have to download more than three times in worst case.

                            string currentFileHash = null;
                            try {
                                currentFileHash = HashFile(downloadFilePath);
                            } catch(Exception e) {
                                // for example, UnauthorizedAccessException
                                Log.LogWarning("Exception while trying to compare current downloaded file to desired hash.  Deleting current file in prep for download...", e);
                            }
                            if (currentFileHash == verifiedUpdateInfo.DownloadHash) {
                                Log.LogMessage("Previous updater download matches hash returned from server; skipping download");
                                needToDownload = false;
                            } else {
                                Log.LogMessage("Existing updater file does not match expected hash; deleting...");
                                if (!TryToDeletePreviousUpdater(downloadFilePath)) {
                                    // couldn't immediate delete `downloadFilePath`
                                    // mark it for deletion on next Windows boot

                                    Log.LogMessage("Marking updater file for deletion on next boot.");

                                    var moveFileResultOldName = MoveFileEx(downloadFilePath, null, MoveFileFlags.MOVEFILE_DELAY_UNTIL_REBOOT);
                                    Log.LogMessage(string.Format("Return code from marking old updater for deletion on next boot: {0}", moveFileResultOldName));

                                    throw new Exception("Cannot delete previous updater.");
                                }
                            }
                        }
                        if (needToDownload) {
                            Log.LogMessage("Downloading updater...");
                            using (var webClient = new WebClient()) {
                                webClient.DownloadFile(verifiedUpdateInfo.DownloadUrl, downloadFilePath);
                            }
                        }

                        // verify downloaded file hash
                        if (HashFile(downloadFilePath) != verifiedUpdateInfo.DownloadHash) {
                            throw new Exception("Updater executable hash is mismatched (after download)");
                        }

                        // verify with kited that it is "readyToRestart", i.e. it isn't in the middle of a UI-facing operation
                        //   or doing a batch file update.
                        // if kited isn't running or returns an error then apply update immediately.
                        // if kited returns "no" for too long then run the update anyway.
                        var kitedWaitDurationSoFar = TimeSpan.Zero;
                        while(kitedWaitDurationSoFar <= k_maxKitedWaitDuration && !IsKitedReadyToRestart()) {
                            Log.LogMessage("kited is NOT ready to restart.  update is in holding pattern; trying again in a bit...");

                            kitedWaitDurationSoFar += k_kitedWaitPollInterval;
                            Thread.Sleep(k_kitedWaitPollInterval);
                        }
                        if(kitedWaitDurationSoFar > TimeSpan.Zero) {
                            // we waited on kited to be "ready to restart"
                            // we're about to run the update
                            // re-check the updater exe hash
                            if (HashFile(downloadFilePath) != verifiedUpdateInfo.DownloadHash) {
                                throw new Exception("Updater executable hash is mismatched (after kited readyToRestart step)");
                            }
                        }
                        Log.LogMessage(string.Format("kited update ready-to-restart step complete (waited a total of {0})", 
                            kitedWaitDurationSoFar));

                        // we're about the run the updater.  if it sees fit it will TerminateProcess() any running kited instances,
                        //   which could be many across multiple user accounts.  it can't restart them all, but KiteService can, so
                        //   before we run the updater remember which user account sessions are running kited, so we know where to
                        //   restart them after the update.
                        try {
                            ClientAppSessions.RememberClientAppSessionIDsForUpdate();
                        } catch (Exception eBeforeExecuting) {
                            Log.LogWarning("Exception calling BeforeExecutingUpdateFile delegate", eBeforeExecuting);
                        }

                        Log.LogMessage(String.Format("Launching update executable {0}", downloadFilePath));

                        Process.Start(downloadFilePath);

                        // if all goes well we will never wake from this sleep, but our next generation offspring will rise!
                        Thread.Sleep(TimeSpan.FromSeconds(15));
                    } finally {
                        Monitor.Exit(k_checkForAndRunUpdatesLockObject);
                    }
                }
            } catch (Exception e) {
                Log.LogError("Exception while checking for and running updates, but will try again soon", e);
            }
        }

        private static string AskServerForUpdate() {
            var currentVersion = Assembly.GetExecutingAssembly().GetName().Version;
            string machineID = MachineID.GetMachineIDAndCreateIfNecessary();
            var userWantsBetaBuilds = false;
            try {
                using (var regKey = Registry.LocalMachine.OpenSubKey(KiteService.PerInstallationRegistryPath)) {
                    if (regKey != null) {
                        var regValue = regKey.GetValue("UserWantsBetaBuilds", 0);
                        userWantsBetaBuilds = regValue is int && ((int)regValue) == 1;
                    }
                }
            } catch (Exception eBetaBuilds) {
                Log.LogWarning("Exception while reading UserWantsBetaBuilds value", eBetaBuilds);
            }

            // make the HTTP GET request
            var url = String.Format(k_updateUrl, currentVersion, machineID, true, userWantsBetaBuilds);
            Log.LogMessage("Update request URL is: " + url);
            var request = (HttpWebRequest)WebRequest.Create(url);
            request.KeepAlive = false;
            try {
                using (var responseObject = request.GetResponse())
                using (var responseReader = new StreamReader(responseObject.GetResponseStream())) {
                    HttpWebResponse hwr = responseObject as HttpWebResponse;
                    if (hwr == null) {
                        // this should never happen, but being careful
                        Log.LogMessage("HWR is null.");
                    } else {
                        Log.LogMessage("Response status code: " + hwr.StatusCode);
                        Log.LogMessage("Response content length: " + hwr.ContentLength);
                    }

                    return responseReader.ReadToEnd();
                }
            } catch (Exception e) {
                Log.LogWarning("Exception while trying to check for update info.", e);
                return null;
            }
        }

        private static bool IsKitedReadyToRestart() {
            // for all error cases, return true

            var request = (HttpWebRequest)WebRequest.Create(k_kitedReadyToRestartUrl);
            request.KeepAlive = false;
            request.AllowAutoRedirect = false;
            request.Timeout = 10000;  // 10 seconds (the default is 100 seconds)

            // if there is a machine-wide/IE proxy, ignore it
            // note, we should honor the Proxy where applicable for non-localhost hosts like windows.kite.com
            request.Proxy = GlobalProxySelection.GetEmptyWebProxy();

            try {
                using (var responseObject = GetResponseNoExceptionOnErrorStatusCode(request) as HttpWebResponse) {
                    if(responseObject == null) {
                        // something odd is happening, e.g. HttpWebRequest.GetResponse() isn't returning an 
                        //   HttpWebResponse.
                        Log.LogWarning("Kited ready-to-restart check returned null; proceeding with update.");
                        return true;
                    }

                    // expected response codes: 409 (not ready), 200 (ready)
                    // for all others, return true
                    return responseObject.StatusCode != HttpStatusCode.Conflict;
                }
            } catch (Exception e) {
                Log.LogWarning("Exception while checking kited ready-to-restart; proceeding with update", e);
                return true;
            }
        }

        private static WebResponse GetResponseNoExceptionOnErrorStatusCode(HttpWebRequest request) {
            try {
                return request.GetResponse();
            } catch(WebException we) {
                var response = we.Response as WebResponse;
                if(response == null) {
                    // there are many ways this may happen, e.g. DNS failure
                    // see https://msdn.microsoft.com/en-us/library/es54hw8e(v=vs.110).aspx
                    throw;
                }
                return response;
            }
        }

        private static bool TryToDeletePreviousUpdater(string previousUpdaterFilePath) {
            // this function returns true if the old updater was successfully deleted, and otherwise (a) marks the old updater
            //   for deletion on the next boot of Windows, and (b) returns false.
            // the try/catches used to catch IOException but there are cases related to "file in use" that aren't IOException,
            //   e.g. UnauthorizedAccessException if file is in use and it's in C:\Program Files\Kite.
            // thus the catch bodies need to be okay if the reason for the exception is something other than "file in use", 
            //   which they are.
            try {
                File.Delete(previousUpdaterFilePath);
            } catch (Exception) {
                // this is probably an old update that didn't terminate for some reason.
                // terminateprocess() and then try to delete it again.

                Log.LogMessage("An updater seems to be running already.  Trying to kill it...");

                int nKilled = 0;
                foreach (var runningUpdater in Process.GetProcessesByName("KiteUpdater")) {
                    runningUpdater.Kill();
                    try {
                        runningUpdater.WaitForExit(5000);
                    } catch (Exception eWaitForExit) {
                        // this wasn't in the original DesktopBootstrap code, so wrap it in an exception handler
                        Log.LogWarning("Exception waiting for old updater exe to terminate", eWaitForExit);
                    }
                    nKilled++;
                }

                Log.LogMessage(String.Format("Killed {0} already-running updaters", nKilled));

                // try to delete it again.
                try {
                    File.Delete(previousUpdaterFilePath);
                } catch (Exception) {
                    Log.LogMessage("Still having trouble deleting old updater exe; sleeping for a brief moment...");
                    Thread.Sleep(10000);

                    try {
                        File.Delete(previousUpdaterFilePath);
                    } catch (Exception) {
                        Log.LogMessage("Third attempt to delete old updater exe failed.");
                        return false;
                    }
                }
            }

            return true;
        }

        private class VerifiedUpdateInfo {

            internal string DownloadUrl { get; set; }
            internal string DownloadHash { get; set; }

            internal static VerifiedUpdateInfo FromUpdateXml(string updateXmlString, bool acceptTestCertificate) {
                // This could probably be refactored to be more efficient, etc, but I wanted to avoid
                // rethrowing exceptions or making further assumptions about how FromUpdateXml(string, string)'s
                // xml handling and verification works.
                Log.LogMessage("updateXmlString has length " + updateXmlString.Length);
                if (acceptTestCertificate) {
                    Log.LogMessage("Accept test certificate is on.");
                    try {
                        return FromUpdateXml(updateXmlString, k_certificate_PROD);
                    } catch {
                        return FromUpdateXml(updateXmlString, k_certificate_TEST);
                    }
                } else {
                    return FromUpdateXml(updateXmlString, k_certificate_PROD);
                }
            }

            private static VerifiedUpdateInfo FromUpdateXml(string updateXmlString, string certString) {
                var base64Cert = certString.Replace("\r", "").Replace("\n", "").Replace(
                    "-----BEGIN CERTIFICATE-----", "").Replace("-----END CERTIFICATE-----", "");
                X509Certificate2 cert = new X509Certificate2(UTF8Encoding.UTF8.GetBytes(base64Cert));

                var xml = new XmlDocument();
                xml.PreserveWhitespace = true;
                xml.LoadXml(updateXmlString);

                var signedXml = new SignedXml(xml);
                var signatureNodeList = xml.GetElementsByTagName("Signature");
                signedXml.LoadXml((XmlElement)signatureNodeList[0]);

                if (!signedXml.CheckSignature(cert, true)) {
                    throw new Exception("Signature verification failed.");
                }

                return new VerifiedUpdateInfo {
                    DownloadUrl = xml["UpdateInfo"]["DownloadUrl"].InnerText,
                    DownloadHash = xml["UpdateInfo"]["DownloadHash"].InnerText
                };
            }
        }

        #region Miscellaneous Helper Functions

        private static bool IsImmediateUpdateCheckFlagSet() {
            try {
                using (var regKey = Registry.LocalMachine.OpenSubKey(@"Software\Kite", false)) {
                    if (regKey == null) {
                        return false;
                    }
                    return (((regKey.GetValue("ImmediateUpdateCheck") as string) ?? string.Empty).ToLowerInvariant() ==
                        "true");
                }
            } catch {
                return false;
            }
        }

        private static void ExecuteDelegateSwallowExceptions(Delegate callee, params object[] parameters) {
            // if e.g. the logging code throws an exception, don't let it prevent us from updating.

            try {
                callee.DynamicInvoke(parameters);
            } catch { }
        }

        private static string HashFile(string path) {
            // will throw an exception if file is missing, etc.

            string downloadedFileHashBase64;
            using (var fileReader = new FileStream(path, FileMode.Open, FileAccess.Read)) {
                downloadedFileHashBase64 = Convert.ToBase64String(new SHA512CryptoServiceProvider().ComputeHash(fileReader));
            }
            return downloadedFileHashBase64;
        }

        [Flags]
        private enum MoveFileFlags {
            MOVEFILE_REPLACE_EXISTING = 1,
            MOVEFILE_COPY_ALLOWED = 2,
            MOVEFILE_DELAY_UNTIL_REBOOT = 4,
            MOVEFILE_WRITE_THROUGH = 8
        }

        [System.Runtime.InteropServices.DllImportAttribute("kernel32.dll", EntryPoint = "MoveFileEx")]
        private static extern bool MoveFileEx(string lpExistingFileName, string lpNewFileName, MoveFileFlags dwFlags);

        #endregion

        #region Kite update certificates

        private static readonly string k_certificate_PROD =
@"-----BEGIN CERTIFICATE-----
XXXXXXX
-----END CERTIFICATE-----";

        private static readonly string k_certificate_TEST =
@"-----BEGIN CERTIFICATE-----
XXXXXXX
-----END CERTIFICATE-----";

        #endregion
    }
}
