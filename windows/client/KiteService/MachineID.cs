using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using Microsoft.Win32;

namespace KiteService {

    internal static class MachineID {

        // general order of precedence for MachineID (first to last):
        //   64-bit HKLM
        //   32-bit HKLM
        // we tried to implement 32-bit registry views using the win32 apis but they weren't working, nor was sample code from 
        //   the web, so we'll stick to only reading/writing 64-bit reg here.
        // in general, avoid using HKCU for MachineID, since it can confuse things / create multiple MachineIDs for the machine

        internal static string GetMachineIDAndCreateIfNecessary() {
            try {
                // registry value should have originally been set during first install, by the NSIS file 
                //   GenerateMachineIDIfAppropriate.nsh.
                // but if it's not then we'll generate and save a new one
                var currentMachineID = ReadMachineID();
                if (currentMachineID != null) {
                    try {
                        // the following WriteMachineID() call may throw an exception.
                        // if there is a problem writing, then we still want to return currentMachineID because it came from
                        //   the registry, so it's written there somewhere, should be (somewhat) stable.
                        WriteMachineID(currentMachineID);
                    } catch (Exception e) {
                        Log.LogError("Exception while writing MachineID to registry that was read from the registry; using read value", e);
                    }
                    return currentMachineID;
                }

                var tentativeMachineID = GuidToString(Guid.NewGuid());
                try {
                    WriteMachineID(tentativeMachineID);
                } catch (Exception e) {
                    Log.LogError("Exception writing new MachineID to registry.  Throwing it away and using all zeros", e);
                    return GuidToString(Guid.Empty);
                }
                return tentativeMachineID;

            } catch (Exception e) {
                Log.LogError("Exception reading or building MachineID from service", e);
                return string.Empty;
            }
        }

        private static string ReadMachineID() {
            // note: don't catch top level / random exceptions here; let exceptions bubble up to our class's internal functions

            using (var regKey = Registry.LocalMachine.CreateSubKey(KiteService.PermanentRegistryPath)) {
                if (regKey != null) {
                    var regValue = regKey.GetValue("MachineID") as string;
                    if (regValue != null) {
                        try {
                            // make sure regValue parses as a Guid
                            var stringValue = GuidToString(new Guid(regValue));
                            if (GuidStringIsValid(stringValue)) {
                                return stringValue;
                            }
                        } catch (Exception e) {
                            Log.LogError("Exception while trying to parse MachineID", e);
                        }
                    }
                }
            }

            return null;
        }

        private static void WriteMachineID(string machineID) {
            // don't write to HKCU, because we're running as SYSTEM anyway, and just isn't reliable

            using (var regKey = Registry.LocalMachine.CreateSubKey(KiteService.PermanentRegistryPath)) {
                if (regKey.GetValue("MachineID") as string != machineID) {
                    // 64 bit registry
                    regKey.SetValue("MachineID", machineID);

                    // try to read/verify
                    var readMachineID64 = regKey.GetValue("MachineID") as string;
                    Log.LogMessage(string.Format("Tentative machine id is {0}.  Read machine id after writing to 64bit hklm is {1}.",
                        machineID, readMachineID64));

                    if (readMachineID64 != machineID) {
                        throw new Exception("Writing MachineID to 64bit hklm failed.");
                    }

                    Log.LogMessage("Writing MachineID to 64bit hklm successful.");
                }
            }
        }

        private static string GuidToString(Guid x) {
            return x.ToString().Replace("-", string.Empty).ToLowerInvariant();
        }

        private static bool GuidStringIsValid(string machineID) {
            return machineID.Length == 32 && machineID != "00000000000000000000000000000000";
        }
    }
}
