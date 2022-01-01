using System;
using System.Collections.Generic;
using System.ComponentModel;
using System.Data;
using System.Drawing;
using System.Linq;
using System.Text;
using System.Windows.Forms;
using System.IO;
using System.Security.Cryptography;
using System.Security.Cryptography.X509Certificates;
using System.Security.Cryptography.Xml;
using System.Xml;

namespace KiteUpdateSigner {

    public partial class Form1 : Form {

        public Form1() {
            InitializeComponent();
        }

        private void Form1_Load(object sender, EventArgs e) {
            // try to find the path to the private key
            foreach (var driveletter in "a,b,c,d,e,f,g,h,i,j,k,l,m,n,o,p,q,r,s,t,u,v,w,x,y,z".Split(',')) {
                var potentialPath = driveletter + @":\client_update_key";
                if (Directory.Exists(potentialPath)) {
                    txtPrivateKeyFilePath.Text = Path.Combine(potentialPath, @"PROD\certificate-with-encrypted-key.p12");
                }
            }
            if (txtPrivateKeyFilePath.Text.Length == 0) {
                btnUseTestKey_Click(null, EventArgs.Empty);
            }

            // try to find the path to the latest build's KiteUpdater.exe, and also use that
            // to populate a reasonable value for the sig output file and download URL
            var curPath = Environment.CurrentDirectory;
            while (curPath.Length > 0 && !Directory.Exists(Path.Combine(curPath, "installer"))) {
                curPath = Path.GetDirectoryName(curPath); // go one level up
            }

            var prevRelease = "";

            if (curPath.Length > 0) {
                curPath = Path.Combine(curPath, "installer");

                var buildDirs = Directory.GetDirectories(Path.Combine(curPath, "builds"));
                var keys = new long[buildDirs.Length];
                for (int i = 0; i < buildDirs.Length; i++) {
                    buildDirs[i] = buildDirs[i].Substring(Path.GetDirectoryName(buildDirs[i]).Length + 1);
                    keys[i] = long.Parse(buildDirs[i].Split('.')[1] + buildDirs[i].Split('.')[2].PadLeft(4, '0') + buildDirs[i].Split('.')[3].PadLeft(2, '0'));
                }
                Array.Sort(keys, buildDirs);
                var targetBuild = buildDirs[buildDirs.Length - 1];

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
                Console.WriteLine(patchUpdaterFilename);

                prevRelease = patchUpdaterFilename.Substring("KitePatchUpdater".Length,
                                      patchUpdaterFilename.LastIndexOf("-") - "KitePatchUpdater".Length);
                Console.WriteLine(prevRelease);

                var patchUpdaterDownloadFilename = String.Format("KitePatchUpdater{0}.exe", prevRelease);

                txtUpdateExecutableFilePath.Text = Path.Combine(curPath, @"builds\" + targetBuild + string.Format(@"\{0}", patchUpdaterFilename));
                txtOutFilePath.Text = Path.Combine(Path.GetDirectoryName(txtUpdateExecutableFilePath.Text),
                    string.Format("KitePatchUpdateInfo{0}.xml", prevRelease));
                txtDownloadUrl.Text = String.Format("https://windows.kite.com/windows/{0}/{1}", targetBuild, patchUpdaterDownloadFilename);
            } else {
                MessageBox.Show("Couldn't guess at the build directory path.");
            }

            if(Environment.GetCommandLineArgs().Length > 1 && Environment.GetCommandLineArgs()[1] == "--sign-with-test") {
                if(txtUpdateExecutableFilePath.Text.Length == 0) {
                    // already showed the MessageBox above
                    Application.Exit();
                }

                txtOutFilePath.Text = Path.Combine(Path.GetDirectoryName(txtUpdateExecutableFilePath.Text),
                    String.Format("KitePatchUpdateInfo{0}-SignedWithTESTKey.xml", prevRelease));
                btnUseTestKey_Click(null, EventArgs.Empty);
                butGo_Click(null, EventArgs.Empty);
                Application.Exit();
            }
        }

        private void butGo_Click(object sender, EventArgs e) {
            var xmlDoc = CreateUpdateXmlDoc(txtUpdateExecutableFilePath.Text,
                txtDownloadUrl.Text);

            AddSignatureToXmlDocument(xmlDoc, new X509Certificate2(txtPrivateKeyFilePath.Text,
                txtPassword.Text));

            using (var xmltw = new XmlTextWriter(txtOutFilePath.Text, new UTF8Encoding(false))) {
                xmlDoc.WriteTo(xmltw);
                xmltw.Close();
            }

            if (Environment.GetCommandLineArgs().Length == 1 || Environment.GetCommandLineArgs()[1] != "--sign-with-test") {
                MessageBox.Show("Done!");
            }
        }

        private static void AddSignatureToXmlDocument(XmlDocument toSign, X509Certificate2 cert) {
            var signedXml = new SignedXml(toSign);
            signedXml.SigningKey = cert.PrivateKey;

            var reference = new Reference();
            reference.Uri = "";
            reference.AddTransform(new XmlDsigEnvelopedSignatureTransform());
            signedXml.AddReference(reference);

            signedXml.ComputeSignature();
            var xmlDigitalSignature = signedXml.GetXml();
            toSign.DocumentElement.AppendChild(toSign.ImportNode(xmlDigitalSignature, true));
            if (toSign.FirstChild is XmlDeclaration) {
                toSign.RemoveChild(toSign.FirstChild);
            }
        }

        private static XmlDocument CreateUpdateXmlDoc(string updateExecutableFilePath, string downloadUrl) {
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

        private void btnUseTestKey_Click(object sender, EventArgs e) {
            // try walking up until we find `keys\client_update_key\TEST\certificate-with-encrypted-key.p12`
            var curDir = Path.GetFullPath(".");
            while(Path.GetPathRoot(curDir) != curDir) {
                var candidateKeyPath = Path.Combine(curDir, @"keys\client_update_key\TEST\certificate-with-encrypted-key.p12");
                if(File.Exists(candidateKeyPath)) {
                    txtPrivateKeyFilePath.Text = candidateKeyPath;
                    txtPassword.Text = "test";
                    return;
                }

                curDir = Path.GetFullPath(Path.Combine(curDir, ".."));
            }

            MessageBox.Show("Couldn't guess at the key file path.");
        }

        public class FunctionalComparer<T> : IComparer<T> {
            private Func<T, T, int> comparer;
            public FunctionalComparer(Func<T, T, int> comparer) {
                this.comparer = comparer;
            }
            public static IComparer<T> Create(Func<T, T, int> comparer) {
                return new FunctionalComparer<T>(comparer);
            }
            public int Compare(T x, T y) {
                return comparer(x, y);
            }
        }
    }
}
