using System;
using System.Collections.Generic;
using System.Linq;
using System.Security.AccessControl;
using System.Security.Principal;
using System.Text;
using System.Threading;
using Microsoft.Win32;

namespace KiteService {

    internal class ClientIpc {

        private static EventWaitHandle s_downloadAndApplyUpdateIfPossibleEvent;

        private static EventWaitHandle s_enableUserWantsBetaBuilds;
        private static EventWaitHandle s_disableUserWantsBetaBuilds;

        internal static void StartListeningForEventFromAnyClientApp() {
            try {
                // Note: don't change these names without also changing them in the client app

                s_downloadAndApplyUpdateIfPossibleEvent = CreateGlobalEventWaitHandle(
                    @"Global\Kite-DownloadAndApplyUpdateIfPossible");

                s_enableUserWantsBetaBuilds = CreateGlobalEventWaitHandle(
                    @"Global\Kite-EnableUserWantsBetaBuilds");
                s_disableUserWantsBetaBuilds = CreateGlobalEventWaitHandle(
                    @"Global\Kite-DisableUserWantsBetaBuilds");

                ThreadPool.QueueUserWorkItem(WaitForEventFromAnyClient, s_downloadAndApplyUpdateIfPossibleEvent);

                ThreadPool.QueueUserWorkItem(WaitForEventFromAnyClient, s_enableUserWantsBetaBuilds);
                ThreadPool.QueueUserWorkItem(WaitForEventFromAnyClient, s_disableUserWantsBetaBuilds);
            } catch (Exception e) {
                Log.LogError("Error starting listening for automatic update signaling events", e);
            }
        }

        private static void WaitForEventFromAnyClient(object eventWaitHandleToProcess) {
            try {
                EventWaitHandle handle = (EventWaitHandle)eventWaitHandleToProcess;
                while (true) {
                    handle.WaitOne();

                    if (handle == s_downloadAndApplyUpdateIfPossibleEvent) {
                        Log.LogMessage("Received IPC signal to download and apply update immediately, if possible");
                        UpdateChecker.ForceUpdateCheckNow();

                    } else if (handle == s_enableUserWantsBetaBuilds || handle == s_disableUserWantsBetaBuilds) {
                        bool userWantsBetaBuilds = (handle == s_enableUserWantsBetaBuilds);

                        Log.LogMessage(string.Format("Received IPC signal re user wants beta builds ({0})",
                            userWantsBetaBuilds));

                        using (var regKey = Registry.LocalMachine.CreateSubKey(KiteService.PerInstallationRegistryPath)) {
                            if (regKey != null) {
                                regKey.SetValue("UserWantsBetaBuilds", userWantsBetaBuilds ? 1 : 0, RegistryValueKind.DWord);
                            }
                        }

                    } else {
                        throw new Exception("Unrecognized event handle.");
                    }
                }
            } catch (Exception e) {
                Log.LogError("Error listening for automatic update signaling event", e);
            }
        }

        private static EventWaitHandle CreateGlobalEventWaitHandle(string name) {
            var users = new SecurityIdentifier(WellKnownSidType.AuthenticatedUserSid, null);
            var rule = new EventWaitHandleAccessRule(users, EventWaitHandleRights.Synchronize |
                EventWaitHandleRights.Modify, AccessControlType.Allow);
            var security = new EventWaitHandleSecurity();
            security.AddAccessRule(rule);

            bool createdNew;
            return new EventWaitHandle(false, EventResetMode.AutoReset, name, out createdNew, security);
        }
    }
}
