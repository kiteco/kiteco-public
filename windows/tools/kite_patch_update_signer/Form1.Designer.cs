namespace KiteUpdateSigner {
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
            this.label1 = new System.Windows.Forms.Label();
            this.txtUpdateExecutableFilePath = new System.Windows.Forms.TextBox();
            this.txtPrivateKeyFilePath = new System.Windows.Forms.TextBox();
            this.label2 = new System.Windows.Forms.Label();
            this.txtPassword = new System.Windows.Forms.TextBox();
            this.label3 = new System.Windows.Forms.Label();
            this.txtOutFilePath = new System.Windows.Forms.TextBox();
            this.label4 = new System.Windows.Forms.Label();
            this.butGo = new System.Windows.Forms.Button();
            this.txtDownloadUrl = new System.Windows.Forms.TextBox();
            this.label5 = new System.Windows.Forms.Label();
            this.btnUseTestKey = new System.Windows.Forms.Button();
            this.SuspendLayout();
            // 
            // label1
            // 
            this.label1.AutoSize = true;
            this.label1.Location = new System.Drawing.Point(13, 31);
            this.label1.Name = "label1";
            this.label1.Size = new System.Drawing.Size(104, 13);
            this.label1.TabIndex = 0;
            this.label1.Text = "Update Executable: ";
            // 
            // txtUpdateExecutableFilePath
            // 
            this.txtUpdateExecutableFilePath.Anchor = ((System.Windows.Forms.AnchorStyles)(((System.Windows.Forms.AnchorStyles.Top | System.Windows.Forms.AnchorStyles.Left) 
            | System.Windows.Forms.AnchorStyles.Right)));
            this.txtUpdateExecutableFilePath.Location = new System.Drawing.Point(124, 31);
            this.txtUpdateExecutableFilePath.Name = "txtUpdateExecutableFilePath";
            this.txtUpdateExecutableFilePath.Size = new System.Drawing.Size(726, 20);
            this.txtUpdateExecutableFilePath.TabIndex = 0;
            // 
            // txtPrivateKeyFilePath
            // 
            this.txtPrivateKeyFilePath.Anchor = ((System.Windows.Forms.AnchorStyles)(((System.Windows.Forms.AnchorStyles.Top | System.Windows.Forms.AnchorStyles.Left) 
            | System.Windows.Forms.AnchorStyles.Right)));
            this.txtPrivateKeyFilePath.Location = new System.Drawing.Point(124, 109);
            this.txtPrivateKeyFilePath.Name = "txtPrivateKeyFilePath";
            this.txtPrivateKeyFilePath.Size = new System.Drawing.Size(726, 20);
            this.txtPrivateKeyFilePath.TabIndex = 3;
            // 
            // label2
            // 
            this.label2.AutoSize = true;
            this.label2.Location = new System.Drawing.Point(13, 109);
            this.label2.Name = "label2";
            this.label2.Size = new System.Drawing.Size(86, 13);
            this.label2.TabIndex = 2;
            this.label2.Text = "Cert Private Key:";
            // 
            // txtPassword
            // 
            this.txtPassword.Anchor = ((System.Windows.Forms.AnchorStyles)(((System.Windows.Forms.AnchorStyles.Top | System.Windows.Forms.AnchorStyles.Left) 
            | System.Windows.Forms.AnchorStyles.Right)));
            this.txtPassword.Font = new System.Drawing.Font("Wingdings", 12F, System.Drawing.FontStyle.Regular, System.Drawing.GraphicsUnit.Point, ((byte)(0)));
            this.txtPassword.Location = new System.Drawing.Point(124, 135);
            this.txtPassword.Name = "txtPassword";
            this.txtPassword.PasswordChar = 'N';
            this.txtPassword.Size = new System.Drawing.Size(726, 25);
            this.txtPassword.TabIndex = 4;
            // 
            // label3
            // 
            this.label3.AutoSize = true;
            this.label3.Location = new System.Drawing.Point(13, 141);
            this.label3.Name = "label3";
            this.label3.Size = new System.Drawing.Size(56, 13);
            this.label3.TabIndex = 4;
            this.label3.Text = "Password:";
            // 
            // txtOutFilePath
            // 
            this.txtOutFilePath.Anchor = ((System.Windows.Forms.AnchorStyles)(((System.Windows.Forms.AnchorStyles.Top | System.Windows.Forms.AnchorStyles.Left) 
            | System.Windows.Forms.AnchorStyles.Right)));
            this.txtOutFilePath.Location = new System.Drawing.Point(124, 83);
            this.txtOutFilePath.Name = "txtOutFilePath";
            this.txtOutFilePath.Size = new System.Drawing.Size(726, 20);
            this.txtOutFilePath.TabIndex = 2;
            // 
            // label4
            // 
            this.label4.AutoSize = true;
            this.label4.Location = new System.Drawing.Point(13, 83);
            this.label4.Name = "label4";
            this.label4.Size = new System.Drawing.Size(61, 13);
            this.label4.TabIndex = 6;
            this.label4.Text = "Output File:";
            // 
            // butGo
            // 
            this.butGo.Anchor = ((System.Windows.Forms.AnchorStyles)((System.Windows.Forms.AnchorStyles.Top | System.Windows.Forms.AnchorStyles.Right)));
            this.butGo.Location = new System.Drawing.Point(774, 185);
            this.butGo.Name = "butGo";
            this.butGo.Size = new System.Drawing.Size(75, 23);
            this.butGo.TabIndex = 5;
            this.butGo.Text = "Go";
            this.butGo.UseVisualStyleBackColor = true;
            this.butGo.Click += new System.EventHandler(this.butGo_Click);
            // 
            // txtDownloadUrl
            // 
            this.txtDownloadUrl.Anchor = ((System.Windows.Forms.AnchorStyles)(((System.Windows.Forms.AnchorStyles.Top | System.Windows.Forms.AnchorStyles.Left) 
            | System.Windows.Forms.AnchorStyles.Right)));
            this.txtDownloadUrl.Location = new System.Drawing.Point(124, 57);
            this.txtDownloadUrl.Name = "txtDownloadUrl";
            this.txtDownloadUrl.Size = new System.Drawing.Size(726, 20);
            this.txtDownloadUrl.TabIndex = 1;
            // 
            // label5
            // 
            this.label5.AutoSize = true;
            this.label5.Location = new System.Drawing.Point(13, 57);
            this.label5.Name = "label5";
            this.label5.Size = new System.Drawing.Size(83, 13);
            this.label5.TabIndex = 9;
            this.label5.Text = "Download URL:";
            // 
            // btnUseTestKey
            // 
            this.btnUseTestKey.Anchor = ((System.Windows.Forms.AnchorStyles)((System.Windows.Forms.AnchorStyles.Top | System.Windows.Forms.AnchorStyles.Right)));
            this.btnUseTestKey.Location = new System.Drawing.Point(652, 185);
            this.btnUseTestKey.Name = "btnUseTestKey";
            this.btnUseTestKey.Size = new System.Drawing.Size(116, 23);
            this.btnUseTestKey.TabIndex = 10;
            this.btnUseTestKey.Text = "-> Use TEST key";
            this.btnUseTestKey.UseVisualStyleBackColor = true;
            this.btnUseTestKey.Click += new System.EventHandler(this.btnUseTestKey_Click);
            // 
            // Form1
            // 
            this.AutoScaleDimensions = new System.Drawing.SizeF(6F, 13F);
            this.AutoScaleMode = System.Windows.Forms.AutoScaleMode.Font;
            this.ClientSize = new System.Drawing.Size(862, 243);
            this.Controls.Add(this.btnUseTestKey);
            this.Controls.Add(this.txtDownloadUrl);
            this.Controls.Add(this.label5);
            this.Controls.Add(this.butGo);
            this.Controls.Add(this.txtOutFilePath);
            this.Controls.Add(this.label4);
            this.Controls.Add(this.txtPassword);
            this.Controls.Add(this.label3);
            this.Controls.Add(this.txtPrivateKeyFilePath);
            this.Controls.Add(this.label2);
            this.Controls.Add(this.txtUpdateExecutableFilePath);
            this.Controls.Add(this.label1);
            this.Name = "Form1";
            this.Text = "Form1";
            this.Load += new System.EventHandler(this.Form1_Load);
            this.ResumeLayout(false);
            this.PerformLayout();

        }

        #endregion

        private System.Windows.Forms.Label label1;
        private System.Windows.Forms.TextBox txtUpdateExecutableFilePath;
        private System.Windows.Forms.TextBox txtPrivateKeyFilePath;
        private System.Windows.Forms.Label label2;
        private System.Windows.Forms.TextBox txtPassword;
        private System.Windows.Forms.Label label3;
        private System.Windows.Forms.TextBox txtOutFilePath;
        private System.Windows.Forms.Label label4;
        private System.Windows.Forms.Button butGo;
        private System.Windows.Forms.TextBox txtDownloadUrl;
        private System.Windows.Forms.Label label5;
        private System.Windows.Forms.Button btnUseTestKey;
    }
}

