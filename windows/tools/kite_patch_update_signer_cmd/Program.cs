using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using System.Security.Cryptography;
using System.Security.Cryptography.X509Certificates;
using System.Security.Cryptography.Xml;
using System.Text;
using System.Windows.Forms;
using System.Xml;

namespace KiteUpdateSignerCmd {
    struct BuildParams
    {
        public string ExecutablePath;
        public string XMLPath;
        public string DownloadURL;
    }

    static class Program {
        /// <summary>
        /// The main entry point for the application.
        /// </summary>
        [STAThread]
        static void Main(string[] args) {
            if (args.Length != 1)
            {
                Console.WriteLine("Please provide password");
                Environment.Exit(1);
            }

            BuildParams bp = buildParams();

            var xmlDoc = CreateUpdateXmlDoc(bp.ExecutablePath, bp.DownloadURL);

            var privateKeyPath = privateKeyFilePath();
            if (privateKeyPath.Length == 0)
            {
                Console.WriteLine("no private key file found. mount the key please.");
                Environment.Exit(1);
            }
            AddSignatureToXmlDocument(xmlDoc, new X509Certificate2(privateKeyPath, args[0]));

            using (var xmltw = new XmlTextWriter(bp.XMLPath, new UTF8Encoding(false)))
            {
                xmlDoc.WriteTo(xmltw);
                xmltw.Close();
            }
        }

        private static string privateKeyFilePath()
        {
            // try to find the path to the private key
            foreach (var driveletter in "a,b,c,d,e,f,g,h,i,j,k,l,m,n,o,p,q,r,s,t,u,v,w,x,y,z".Split(','))
            {
                var potentialPath = driveletter + @":\client_update_key";
                if (Directory.Exists(potentialPath))
                {
                    return Path.Combine(potentialPath, @"PROD\certificate-with-encrypted-key.p12");
                }
            }
            return "";
        }

        private static BuildParams buildParams()
        {
            // try to find the path to the latest build's KiteUpdater.exe, and also use that
            // to populate a reasonable value for the sig output file and download URL
            var curPath = Environment.CurrentDirectory;
            while (curPath.Length > 0 && !Directory.Exists(Path.Combine(curPath, "installer")))
            {
                curPath = Path.GetDirectoryName(curPath); // go one level up
            }

            if (curPath.Length == 0)
            {
                Console.WriteLine("Incorrect working directory");
                Environment.Exit(1);
            }

            curPath = Path.Combine(curPath, "installer");

            var buildDirs = Directory.GetDirectories(Path.Combine(curPath, "builds"));
            var keys = new long[buildDirs.Length];
            for (int i = 0; i < buildDirs.Length; i++)
            {
                buildDirs[i] = buildDirs[i].Substring(Path.GetDirectoryName(buildDirs[i]).Length + 1);
                keys[i] = long.Parse(buildDirs[i].Split('.')[1] + buildDirs[i].Split('.')[2].PadLeft(4, '0') + buildDirs[i].Split('.')[3].PadLeft(2, '0'));
            }
            Array.Sort(keys, buildDirs);
            if (buildDirs.Length < 1)
            {
                Console.WriteLine("no builds");
                Environment.Exit(1);
            }
            var targetBuild = buildDirs[buildDirs.Length - 1];

            Console.WriteLine("target build: {0}", targetBuild);

            var patchUpdaterFiles = Directory.GetFiles(Path.Combine(curPath, "builds", targetBuild), "KitePatchUpdater*.exe");
            if (patchUpdaterFiles.Length < 1)
            {
                Console.WriteLine("no patch updater executable found");
                Environment.Exit(1);
            }
            if (patchUpdaterFiles.Length != 1)
            {
                Console.WriteLine("too many patch updater executables found");
                Environment.Exit(1);
            }

            var patchUpdaterFilename = Path.GetFileName(patchUpdaterFiles[0]);
            Console.WriteLine("patch updater: {0}", patchUpdaterFilename);

            var prevRelease = patchUpdaterFilename.Substring("KitePatchUpdater".Length,
                                  patchUpdaterFilename.LastIndexOf("-") - "KitePatchUpdater".Length);
            Console.WriteLine("source build: {0}", prevRelease);

            BuildParams bp = new BuildParams();
            bp.ExecutablePath = Path.Combine(curPath, @"builds\" + targetBuild + string.Format(@"\{0}", patchUpdaterFilename));
            bp.XMLPath = Path.Combine(Path.GetDirectoryName(bp.ExecutablePath),
                string.Format(@"KitePatchUpdateInfo{0}.xml", prevRelease));
            bp.DownloadURL = String.Format("https://windows.kite.com/windows/{0}/{1}", targetBuild, patchUpdaterFilename);

            Console.WriteLine(bp.XMLPath);
            Console.WriteLine(bp.DownloadURL);
            return bp;
        }

        private static void AddSignatureToXmlDocument(XmlDocument toSign, X509Certificate2 cert)
        {
            var signedXml = new SignedXml(toSign);
            signedXml.SigningKey = cert.PrivateKey;

            var reference = new Reference();
            reference.Uri = "";
            reference.AddTransform(new XmlDsigEnvelopedSignatureTransform());
            signedXml.AddReference(reference);

            signedXml.ComputeSignature();
            var xmlDigitalSignature = signedXml.GetXml();
            toSign.DocumentElement.AppendChild(toSign.ImportNode(xmlDigitalSignature, true));
            if (toSign.FirstChild is XmlDeclaration)
            {
                toSign.RemoveChild(toSign.FirstChild);
            }
        }

        private static XmlDocument CreateUpdateXmlDoc(string updateExecutableFilePath, string downloadUrl)
        {
            XmlDocument document = new XmlDocument();

            var rootNode = document.CreateNode(XmlNodeType.Element, "UpdateInfo", string.Empty);
            document.AppendChild(rootNode);

            var downloadUrlNode = document.CreateNode(XmlNodeType.Element, "DownloadUrl", string.Empty);
            downloadUrlNode.InnerText = downloadUrl;
            rootNode.AppendChild(downloadUrlNode);

            var downloadHashNode = document.CreateNode(XmlNodeType.Element, "", "DownloadHash", "");
            downloadHashNode.InnerText = Convert.ToBase64String(new SHA512CryptoServiceProvider().ComputeHash(
                File.ReadAllBytes(updateExecutableFilePath)));
            rootNode.AppendChild(downloadHashNode);

            return document;
        }

    }
}
