namespace ReadyToRestartTestServer {
    partial class frmMain {
        /// <summary>
        /// Required designer variable.
        /// </summary>
        private System.ComponentModel.IContainer components = null;

        /// <summary>
        /// Clean up any resources being used.
        /// </summary>
        /// <param name="disposing">true if managed resources should be disposed; otherwise, false.</param>
        protected override void Dispose(bool disposing) {
            if (disposing && (components != null)) {
                components.Dispose();
            }
            base.Dispose(disposing);
        }

        #region Windows Form Designer generated code

        /// <summary>
        /// Required method for Designer support - do not modify
        /// the contents of this method with the code editor.
        /// </summary>
        private void InitializeComponent() {
            this.cached409Server = new System.Windows.Forms.Button();
            this.btnUnexpectedCodeServer = new System.Windows.Forms.Button();
            this.btnAlwaysNoServer = new System.Windows.Forms.Button();
            this.btnHangingServer = new System.Windows.Forms.Button();
            this.btnNonRespondingServer = new System.Windows.Forms.Button();
            this.btnRedirectToCnnDotCom = new System.Windows.Forms.Button();
            this.SuspendLayout();
            // 
            // cached409Server
            // 
            this.cached409Server.Location = new System.Drawing.Point(12, 128);
            this.cached409Server.Name = "cached409Server";
            this.cached409Server.Size = new System.Drawing.Size(124, 23);
            this.cached409Server.TabIndex = 10;
            this.cached409Server.Text = "cached 409 server";
            this.cached409Server.UseVisualStyleBackColor = true;
            this.cached409Server.Click += new System.EventHandler(this.cached409Server_Click);
            // 
            // btnUnexpectedCodeServer
            // 
            this.btnUnexpectedCodeServer.Location = new System.Drawing.Point(12, 99);
            this.btnUnexpectedCodeServer.Name = "btnUnexpectedCodeServer";
            this.btnUnexpectedCodeServer.Size = new System.Drawing.Size(124, 23);
            this.btnUnexpectedCodeServer.TabIndex = 9;
            this.btnUnexpectedCodeServer.Text = "unexp code server";
            this.btnUnexpectedCodeServer.UseVisualStyleBackColor = true;
            this.btnUnexpectedCodeServer.Click += new System.EventHandler(this.btnUnexpectedCodeServer_Click);
            // 
            // btnAlwaysNoServer
            // 
            this.btnAlwaysNoServer.Location = new System.Drawing.Point(12, 70);
            this.btnAlwaysNoServer.Name = "btnAlwaysNoServer";
            this.btnAlwaysNoServer.Size = new System.Drawing.Size(124, 23);
            this.btnAlwaysNoServer.TabIndex = 8;
            this.btnAlwaysNoServer.Text = "always-no server";
            this.btnAlwaysNoServer.UseVisualStyleBackColor = true;
            this.btnAlwaysNoServer.Click += new System.EventHandler(this.btnAlwaysNoServer_Click);
            // 
            // btnHangingServer
            // 
            this.btnHangingServer.Location = new System.Drawing.Point(12, 41);
            this.btnHangingServer.Name = "btnHangingServer";
            this.btnHangingServer.Size = new System.Drawing.Size(124, 23);
            this.btnHangingServer.TabIndex = 7;
            this.btnHangingServer.Text = "hanging server";
            this.btnHangingServer.UseVisualStyleBackColor = true;
            this.btnHangingServer.Click += new System.EventHandler(this.btnHangingServer_Click);
            // 
            // btnNonRespondingServer
            // 
            this.btnNonRespondingServer.Location = new System.Drawing.Point(12, 12);
            this.btnNonRespondingServer.Name = "btnNonRespondingServer";
            this.btnNonRespondingServer.Size = new System.Drawing.Size(124, 23);
            this.btnNonRespondingServer.TabIndex = 6;
            this.btnNonRespondingServer.Text = "non-responding server";
            this.btnNonRespondingServer.UseVisualStyleBackColor = true;
            this.btnNonRespondingServer.Click += new System.EventHandler(this.btnNonRespondingServer_Click);
            // 
            // btnRedirectToCnnDotCom
            // 
            this.btnRedirectToCnnDotCom.Location = new System.Drawing.Point(12, 157);
            this.btnRedirectToCnnDotCom.Name = "btnRedirectToCnnDotCom";
            this.btnRedirectToCnnDotCom.Size = new System.Drawing.Size(124, 23);
            this.btnRedirectToCnnDotCom.TabIndex = 11;
            this.btnRedirectToCnnDotCom.Text = "redirect to cnn.com";
            this.btnRedirectToCnnDotCom.UseVisualStyleBackColor = true;
            this.btnRedirectToCnnDotCom.Click += new System.EventHandler(this.btnRedirectToCnnDotCom_Click);
            // 
            // frmMain
            // 
            this.AutoScaleDimensions = new System.Drawing.SizeF(6F, 13F);
            this.AutoScaleMode = System.Windows.Forms.AutoScaleMode.Font;
            this.ClientSize = new System.Drawing.Size(152, 192);
            this.Controls.Add(this.btnRedirectToCnnDotCom);
            this.Controls.Add(this.cached409Server);
            this.Controls.Add(this.btnUnexpectedCodeServer);
            this.Controls.Add(this.btnAlwaysNoServer);
            this.Controls.Add(this.btnHangingServer);
            this.Controls.Add(this.btnNonRespondingServer);
            this.MaximizeBox = false;
            this.Name = "frmMain";
            this.Text = "T";
            this.ResumeLayout(false);

        }

        #endregion

        private System.Windows.Forms.Button cached409Server;
        private System.Windows.Forms.Button btnUnexpectedCodeServer;
        private System.Windows.Forms.Button btnAlwaysNoServer;
        private System.Windows.Forms.Button btnHangingServer;
        private System.Windows.Forms.Button btnNonRespondingServer;
        private System.Windows.Forms.Button btnRedirectToCnnDotCom;
    }
}

