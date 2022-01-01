using System;
using System.ServiceProcess;
using System.Text;
using System.Threading;
using Microsoft.Win32;

namespace KiteService {

    public partial class KiteService : ServiceBase {

        internal static readonly string PermanentRegistryPath = @"Software\Kite";
        internal static readonly string PerInstallationRegistryPath = @"Software\Kite\AppData";

        public KiteService() {
            InitializeComponent();
        }

        protected override void OnStart(string[] args) {
            // warning: be careful about putting anything in here -- we've seen instances
            // of the service getting killed by the SCM (service control manager) because
            // it was taking > 30 seconds to return.  that may seem like a lot, but it's not
            // if the machine and 10's of other services are trying to boot at the same
            // time!
            // 
            // note/update: the startup timeout might not have been due to logic in this 
            // function.  it was probably actually due to authenticode verification which
            // took HTTP calls that might have been timing out / no connection.  I fixed
            // this by specifying generatePublisherEvidence in app.config, per these URLs:
            // * http://msdn.microsoft.com/en-us/library/bb629393.aspx (note comment RE 
            //   services)
            // * http://blogs.msdn.com/b/dougste/archive/2008/02/29/should-i-authenticode-sign-my-net-assembly.aspx
            // 
            // nevertheless, it's probably still good practice to return as quickly as 
            // possible.

            // two minutes is supposedly a maximum (http://bit.ly/kZPSUk), but who knows.
            // should be a significant improvement.
            RequestAdditionalTime(120000);

            // execute StartupForReal asynchronously
            ThreadPool.QueueUserWorkItem(StartupForReal, null);
        }

        protected override void OnStop() {
            Log.LogMessage("KiteService stopping");
        }

        private void StartupForReal(object unusedState) {
            // note: the installer starts the service and kited.exe simultaneously.

            string machineID = string.Empty;
            try {
                machineID = MachineID.GetMachineIDAndCreateIfNecessary();
            } catch(Exception e) {
                Log.LogError("Exception while getting MachineID", e);
            }

            Log.LogMessage(string.Format("KiteService started (MachineID is {0})", machineID));

            // Watches kited process and handles crash reporting
            SafeInvokeDelegate(ProcessWatcher.Watch);
            // Sends telemetry
            SafeInvokeDelegate(StatusLogger.StartLoggingStatus);
            // Allows client app to manually trigger update (in theory)
            SafeInvokeDelegate(ClientIpc.StartListeningForEventFromAnyClientApp);

            SafeInvokeDelegate(UpdateChecker.StartUpdateCheckerTimer);
            SafeInvokeDelegate(ClientAppSessions.RestartAllClientAppSessionsFromUpdate);

        }

        private delegate void SimpleDelegate();
        private static void SafeInvokeDelegate(SimpleDelegate x) {
            foreach (var target in x.GetInvocationList()) {
                try {
                    target.DynamicInvoke();
                } catch (Exception e) {
                    Log.LogError("Exception invoking delegate", e);
                }
            }
        }

        internal static bool IsServiceDebugFlagSet() {
#if DEBUG
            return true;
#endif
            try {
                using (var regKey = Registry.LocalMachine.OpenSubKey(PermanentRegistryPath, false)) {
                    if (regKey == null) {
                        return false;
                    }
                    return (((regKey.GetValue("IsDebug") as string) ?? string.Empty).ToLowerInvariant() ==
                        "true");
                }
            } catch {
                return false;
            }
        }

    }
}
