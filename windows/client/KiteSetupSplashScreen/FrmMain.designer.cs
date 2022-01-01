namespace KiteSetupSplashScreen {
    partial class FrmMain {
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
            this.components = new System.ComponentModel.Container();
            this.label1 = new System.Windows.Forms.Label();
            this.timeoutTimer = new System.Windows.Forms.Timer(this.components);
            this.donePollTimer = new System.Windows.Forms.Timer(this.components);
            this.videoElementHost = new System.Windows.Forms.Integration.ElementHost();
            this.lblTrueBackgroundColorCantRelyOnFormBkgrd = new System.Windows.Forms.Label();
            this.SuspendLayout();
            // 
            // label1
            // 
            this.label1.Anchor = ((System.Windows.Forms.AnchorStyles)(((System.Windows.Forms.AnchorStyles.Bottom | System.Windows.Forms.AnchorStyles.Left) 
            | System.Windows.Forms.AnchorStyles.Right)));
            this.label1.BackColor = System.Drawing.Color.White;
            this.label1.Font = new System.Drawing.Font("Microsoft Sans Serif", 12F, System.Drawing.FontStyle.Regular, System.Drawing.GraphicsUnit.Point, ((byte)(0)));
            this.label1.ForeColor = System.Drawing.Color.Black;
            this.label1.Location = new System.Drawing.Point(2, 229);
            this.label1.Name = "label1";
            this.label1.Size = new System.Drawing.Size(280, 28);
            this.label1.TabIndex = 0;
            this.label1.Text = "Installing Kite, please wait...";
            this.label1.TextAlign = System.Drawing.ContentAlignment.MiddleCenter;
            // 
            // timeoutTimer
            // 
            this.timeoutTimer.Tick += new System.EventHandler(this.timeoutTimer_Tick);
            // 
            // donePollTimer
            // 
            this.donePollTimer.Tick += new System.EventHandler(this.donePollTimer_Tick);
            // 
            // videoElementHost
            // 
            this.videoElementHost.Anchor = ((System.Windows.Forms.AnchorStyles)((((System.Windows.Forms.AnchorStyles.Top | System.Windows.Forms.AnchorStyles.Bottom) 
            | System.Windows.Forms.AnchorStyles.Left) 
            | System.Windows.Forms.AnchorStyles.Right)));
            this.videoElementHost.BackColor = System.Drawing.Color.White;
            this.videoElementHost.Location = new System.Drawing.Point(2, 2);
            this.videoElementHost.Name = "videoElementHost";
            this.videoElementHost.Size = new System.Drawing.Size(280, 280);
            this.videoElementHost.TabIndex = 2;
            this.videoElementHost.Text = "elementHost1";
            this.videoElementHost.Child = null;
            // 
            // lblTrueBackgroundColorCantRelyOnFormBkgrd
            // 
            this.lblTrueBackgroundColorCantRelyOnFormBkgrd.BackColor = System.Drawing.Color.Gray;
            this.lblTrueBackgroundColorCantRelyOnFormBkgrd.Location = new System.Drawing.Point(0, 0);
            this.lblTrueBackgroundColorCantRelyOnFormBkgrd.Name = "lblTrueBackgroundColorCantRelyOnFormBkgrd";
            this.lblTrueBackgroundColorCantRelyOnFormBkgrd.Size = new System.Drawing.Size(1000, 1000);
            this.lblTrueBackgroundColorCantRelyOnFormBkgrd.TabIndex = 3;
            // 
            // FrmMain
            // 
            this.AutoScaleDimensions = new System.Drawing.SizeF(6F, 13F);
            this.AutoScaleMode = System.Windows.Forms.AutoScaleMode.Font;
            this.BackColor = System.Drawing.Color.Gray;
            this.ClientSize = new System.Drawing.Size(284, 284);
            this.Controls.Add(this.label1);
            this.Controls.Add(this.videoElementHost);
            this.Controls.Add(this.lblTrueBackgroundColorCantRelyOnFormBkgrd);
            this.FormBorderStyle = System.Windows.Forms.FormBorderStyle.None;
            this.Name = "FrmMain";
            this.ShowInTaskbar = false;
            this.Text = "Kite";
            this.ResumeLayout(false);

        }

        #endregion

        private System.Windows.Forms.Label label1;
        private System.Windows.Forms.Timer timeoutTimer;
        private System.Windows.Forms.Timer donePollTimer;
        private System.Windows.Forms.Integration.ElementHost videoElementHost;
        private System.Windows.Forms.Label lblTrueBackgroundColorCantRelyOnFormBkgrd;
    }
}

