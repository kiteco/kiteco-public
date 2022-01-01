using System;
using System.Collections.Generic;
using System.ComponentModel;
using System.Data;
using System.Drawing;
using System.Linq;
using System.Net;
using System.Text;
using System.Threading;
using System.Windows.Forms;

namespace ReadyToRestartTestServer {
    public partial class frmMain : Form {
        public frmMain() {
            InitializeComponent();
        }

        private void btnNonRespondingServer_Click(object sender, EventArgs e) {
            var prefixes = new string[] { "http://127.0.0.1:46624/" };
            if (prefixes == null || prefixes.Length == 0) {
                throw new ArgumentException("prefixes");
            }

            // Create a listener.
            HttpListener listener = new HttpListener();
            // Add the prefixes.
            foreach (string s in prefixes) {
                listener.Prefixes.Add(s);
            }
            listener.Start();
            // Note: The GetContext method blocks while waiting for a request. 
            while (true) {
                HttpListenerContext context = listener.GetContext();
                HttpListenerRequest request = context.Request;

                Thread.Sleep(TimeSpan.FromHours(1));
            }
        }

        private void btnHangingServer_Click(object sender, EventArgs e) {
            var prefixes = new string[] { "http://127.0.0.1:46624/" };
            if (prefixes == null || prefixes.Length == 0) {
                throw new ArgumentException("prefixes");
            }

            // Create a listener.
            HttpListener listener = new HttpListener();
            // Add the prefixes.
            foreach (string s in prefixes) {
                listener.Prefixes.Add(s);
            }
            listener.Start();
            // Note: The GetContext method blocks while waiting for a request. 
            while (true) {
                Thread.Sleep(TimeSpan.FromHours(1));
            }
        }

        private void btnAlwaysNoServer_Click(object sender, EventArgs e) {
            var prefixes = new string[] { "http://127.0.0.1:46624/" };
            if (prefixes == null || prefixes.Length == 0) {
                throw new ArgumentException("prefixes");
            }

            // Create a listener.
            HttpListener listener = new HttpListener();
            // Add the prefixes.
            foreach (string s in prefixes) {
                listener.Prefixes.Add(s);
            }
            listener.Start();
            // Note: The GetContext method blocks while waiting for a request. 
            while (true) {
                HttpListenerContext context = listener.GetContext();
                HttpListenerRequest request = context.Request;

                context.Response.StatusCode = 409;
                context.Response.ContentLength64 = 0;
                context.Response.OutputStream.Close();
            }
        }

        private void btnUnexpectedCodeServer_Click(object sender, EventArgs e) {
            var prefixes = new string[] { "http://127.0.0.1:46624/" };
            if (prefixes == null || prefixes.Length == 0) {
                throw new ArgumentException("prefixes");
            }

            // Create a listener.
            HttpListener listener = new HttpListener();
            // Add the prefixes.
            foreach (string s in prefixes) {
                listener.Prefixes.Add(s);
            }
            listener.Start();
            // Note: The GetContext method blocks while waiting for a request. 
            while (true) {
                HttpListenerContext context = listener.GetContext();
                HttpListenerRequest request = context.Request;

                context.Response.StatusCode = 404;
                context.Response.ContentLength64 = 0;
                context.Response.OutputStream.Close();
            }
        }

        private void cached409Server_Click(object sender, EventArgs e) {
            var prefixes = new string[] { "http://127.0.0.1:46624/" };
            if (prefixes == null || prefixes.Length == 0) {
                throw new ArgumentException("prefixes");
            }

            // Create a listener.
            HttpListener listener = new HttpListener();
            // Add the prefixes.
            foreach (string s in prefixes) {
                listener.Prefixes.Add(s);
            }
            listener.Start();
            // Note: The GetContext method blocks while waiting for a request. 
            while (true) {
                HttpListenerContext context = listener.GetContext();
                HttpListenerRequest request = context.Request;

                context.Response.AddHeader("Cache-Control", "public, max-age=31536000");
                context.Response.StatusCode = 409;
                context.Response.ContentLength64 = 0;
                context.Response.OutputStream.Close();
            }
        }

        private void btnRedirectToCnnDotCom_Click(object sender, EventArgs e) {
            var prefixes = new string[] { "http://127.0.0.1:46624/" };
            if (prefixes == null || prefixes.Length == 0) {
                throw new ArgumentException("prefixes");
            }

            // Create a listener.
            HttpListener listener = new HttpListener();
            // Add the prefixes.
            foreach (string s in prefixes) {
                listener.Prefixes.Add(s);
            }
            listener.Start();
            // Note: The GetContext method blocks while waiting for a request. 
            while (true) {
                HttpListenerContext context = listener.GetContext();
                HttpListenerRequest request = context.Request;

                context.Response.Redirect("http://cnn.com/");
                context.Response.OutputStream.Close();
            }
        }
    }
}
