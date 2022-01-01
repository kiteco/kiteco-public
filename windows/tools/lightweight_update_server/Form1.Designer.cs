namespace LightweightUpdateServer {
    partial class Form1 {
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
            this.log = new System.Windows.Forms.ListBox();
            this.txtXml = new System.Windows.Forms.TextBox();
            this.label1 = new System.Windows.Forms.Label();
            this.label2 = new System.Windows.Forms.Label();
            this.txtExe = new System.Windows.Forms.TextBox();
            this.chkboxService = new System.Windows.Forms.CheckBox();
            this.numMachineIDsLabel = new System.Windows.Forms.Label();
            this.txtVersion = new System.Windows.Forms.TextBox();
            this.label3 = new System.Windows.Forms.Label();
            this.txtNumUniqueClientUpdatesRemaining = new System.Windows.Forms.TextBox();
            this.label4 = new System.Windows.Forms.Label();
            this.label5 = new System.Windows.Forms.Label();
            this.chkboxClientApp = new System.Windows.Forms.CheckBox();
            this.SuspendLayout();
            // 
            // log
            // 
            this.log.Anchor = ((System.Windows.Forms.AnchorStyles)((((System.Windows.Forms.AnchorStyles.Top | System.Windows.Forms.AnchorStyles.Bottom) 
            | System.Windows.Forms.AnchorStyles.Left) 
            | System.Windows.Forms.AnchorStyles.Right)));
            this.log.FormattingEnabled = true;
            this.log.Location = new System.Drawing.Point(12, 208);
            this.log.Name = "log";
            this.log.ScrollAlwaysVisible = true;
            this.log.Size = new System.Drawing.Size(804, 407);
            this.log.TabIndex = 0;
            // 
            // txtXml
            // 
            this.txtXml.Anchor = ((System.Windows.Forms.AnchorStyles)(((System.Windows.Forms.AnchorStyles.Top | System.Windows.Forms.AnchorStyles.Left) 
            | System.Windows.Forms.AnchorStyles.Right)));
            this.txtXml.Location = new System.Drawing.Point(129, 22);
            this.txtXml.Name = "txtXml";
            this.txtXml.Size = new System.Drawing.Size(687, 20);
            this.txtXml.TabIndex = 1;
            // 
            // label1
            // 
            this.label1.AutoSize = true;
            this.label1.Location = new System.Drawing.Point(9, 25);
            this.label1.Name = "label1";
            this.label1.Size = new System.Drawing.Size(107, 13);
            this.label1.TabIndex = 2;
            this.label1.Text = "Update XML filepath:";
            // 
            // label2
            // 
            this.label2.AutoSize = true;
            this.label2.Location = new System.Drawing.Point(9, 53);
            this.label2.Name = "label2";
            this.label2.Size = new System.Drawing.Size(102, 13);
            this.label2.TabIndex = 4;
            this.label2.Text = "Update exe filepath:";
            // 
            // txtExe
            // 
            this.txtExe.Anchor = ((System.Windows.Forms.AnchorStyles)(((System.Windows.Forms.AnchorStyles.Top | System.Windows.Forms.AnchorStyles.Left) 
            | System.Windows.Forms.AnchorStyles.Right)));
            this.txtExe.Location = new System.Drawing.Point(129, 50);
            this.txtExe.Name = "txtExe";
            this.txtExe.Size = new System.Drawing.Size(687, 20);
            this.txtExe.TabIndex = 3;
            // 
            // chkboxService
            // 
            this.chkboxService.AutoSize = true;
            this.chkboxService.Checked = true;
            this.chkboxService.CheckState = System.Windows.Forms.CheckState.Checked;
            this.chkboxService.Location = new System.Drawing.Point(129, 110);
            this.chkboxService.Name = "chkboxService";
            this.chkboxService.Size = new System.Drawing.Size(200, 17);
            this.chkboxService.TabIndex = 6;
            this.chkboxService.Text = "Updates if will be applied immediately";
            this.chkboxService.UseVisualStyleBackColor = true;
            // 
            // numMachineIDsLabel
            // 
            this.numMachineIDsLabel.AutoSize = true;
            this.numMachineIDsLabel.Location = new System.Drawing.Point(211, 187);
            this.numMachineIDsLabel.Name = "numMachineIDsLabel";
            this.numMachineIDsLabel.Size = new System.Drawing.Size(106, 13);
            this.numMachineIDsLabel.TabIndex = 8;
            this.numMachineIDsLabel.Text = "numMachineIDsLabel";
            // 
            // txtVersion
            // 
            this.txtVersion.Anchor = ((System.Windows.Forms.AnchorStyles)((System.Windows.Forms.AnchorStyles.Top | System.Windows.Forms.AnchorStyles.Right)));
            this.txtVersion.Location = new System.Drawing.Point(716, 87);
            this.txtVersion.Name = "txtVersion";
            this.txtVersion.Size = new System.Drawing.Size(100, 20);
            this.txtVersion.TabIndex = 9;
            // 
            // label3
            // 
            this.label3.Anchor = ((System.Windows.Forms.AnchorStyles)((System.Windows.Forms.AnchorStyles.Top | System.Windows.Forms.AnchorStyles.Right)));
            this.label3.AutoSize = true;
            this.label3.Location = new System.Drawing.Point(613, 91);
            this.label3.Name = "label3";
            this.label3.Size = new System.Drawing.Size(70, 13);
            this.label3.TabIndex = 10;
            this.label3.Text = "Build version:";
            // 
            // txtNumUniqueClientUpdatesRemaining
            // 
            this.txtNumUniqueClientUpdatesRemaining.Location = new System.Drawing.Point(129, 133);
            this.txtNumUniqueClientUpdatesRemaining.Name = "txtNumUniqueClientUpdatesRemaining";
            this.txtNumUniqueClientUpdatesRemaining.Size = new System.Drawing.Size(55, 20);
            this.txtNumUniqueClientUpdatesRemaining.TabIndex = 11;
            this.txtNumUniqueClientUpdatesRemaining.Text = "-1";
            // 
            // label4
            // 
            this.label4.AutoSize = true;
            this.label4.Location = new System.Drawing.Point(190, 136);
            this.label4.Name = "label4";
            this.label4.Size = new System.Drawing.Size(222, 13);
            this.label4.TabIndex = 12;
            this.label4.Text = "unique client updates remaining (-1 for infinite)";
            // 
            // label5
            // 
            this.label5.AutoSize = true;
            this.label5.Location = new System.Drawing.Point(9, 187);
            this.label5.Name = "label5";
            this.label5.Size = new System.Drawing.Size(205, 13);
            this.label5.TabIndex = 13;
            this.label5.Text = "Unique clients served update commands: ";
            // 
            // chkboxClientApp
            // 
            this.chkboxClientApp.AutoSize = true;
            this.chkboxClientApp.Location = new System.Drawing.Point(129, 87);
            this.chkboxClientApp.Name = "chkboxClientApp";
            this.chkboxClientApp.Size = new System.Drawing.Size(154, 17);
            this.chkboxClientApp.TabIndex = 5;
            this.chkboxClientApp.Text = "Updates if won\'t be applied";
            this.chkboxClientApp.UseVisualStyleBackColor = true;
            // 
            // Form1
            // 
            this.AutoScaleDimensions = new System.Drawing.SizeF(6F, 13F);
            this.AutoScaleMode = System.Windows.Forms.AutoScaleMode.Font;
            this.ClientSize = new System.Drawing.Size(828, 631);
            this.Controls.Add(this.label5);
            this.Controls.Add(this.label4);
            this.Controls.Add(this.txtNumUniqueClientUpdatesRemaining);
            this.Controls.Add(this.label3);
            this.Controls.Add(this.txtVersion);
            this.Controls.Add(this.numMachineIDsLabel);
            this.Controls.Add(this.chkboxService);
            this.Controls.Add(this.chkboxClientApp);
            this.Controls.Add(this.label2);
            this.Controls.Add(this.txtExe);
            this.Controls.Add(this.label1);
            this.Controls.Add(this.txtXml);
            this.Controls.Add(this.log);
            this.Name = "Form1";
            this.Text = "Form1";
            this.Load += new System.EventHandler(this.Form1_Load);
            this.ResumeLayout(false);
            this.PerformLayout();

        }

        #endregion

        private System.Windows.Forms.ListBox log;
        private System.Windows.Forms.TextBox txtXml;
        private System.Windows.Forms.Label label1;
        private System.Windows.Forms.Label label2;
        private System.Windows.Forms.TextBox txtExe;
        private System.Windows.Forms.CheckBox chkboxService;
        private System.Windows.Forms.Label numMachineIDsLabel;
        private System.Windows.Forms.TextBox txtVersion;
        private System.Windows.Forms.Label label3;
        private System.Windows.Forms.TextBox txtNumUniqueClientUpdatesRemaining;
        private System.Windows.Forms.Label label4;
        private System.Windows.Forms.Label label5;
        private System.Windows.Forms.CheckBox chkboxClientApp;
    }
}

