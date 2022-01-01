export AWS_ACCESS_KEY_ID=`aws configure get default.aws_access_key_id`
export AWS_SECRET_ACCESS_KEY_ID=`aws configure get default.aws_secret_access_key`
export AWS_REGION=`aws configure get region`

export COMMUNITY_DB_URI=`aws --region=us-west-1 --output text --query 'SecretString' secretsmanager get-secret-value --secret-id COMMUNITY_DB_URI`
export COMMUNITY_DB_DRIVER=postgres

export STRIPE_SECRET=`aws --region=us-west-1 --output text --query 'SecretString' secretsmanager get-secret-value --secret-id STRIPE_SECRET`
export STRIPE_WEBHOOK_SECRET=`aws --region=us-west-1 --output text --query 'SecretString' secretsmanager get-secret-value --secret-id STRIPE_WEBHOOK_SECRET`
export OCTOBAT_SECRET=`aws --region=us-west-1 --output text --query 'SecretString' secretsmanager get-secret-value --secret-id OCTOBAT_SECRET`
export OCTOBAT_PUBLISHABLE=`aws --region=us-west-1 --output text --query 'SecretString' secretsmanager get-secret-value --secret-id OCTOBAT_PUBLISHABLE`

export MIXPANEL_SECRET=`aws --region=us-west-1 --output text --query 'SecretString' secretsmanager get-secret-value --secret-id MIXPANEL_SECRET`
export DELIGHTED_SECRET=`aws --region=us-west-1 --output text --query 'SecretString' secretsmanager get-secret-value --secret-id DELIGHTED_SECRET`

export QUICK_EMAIL_TOKEN=`aws --region=us-west-1 --output text --query 'SecretString' secretsmanager get-secret-value --secret-id QUICK_EMAIL_TOKEN`

export LICENSE_RSA_KEY=`aws --region=us-west-1 --output text --query 'SecretString' secretsmanager get-secret-value --secret-id LICENSE_RSA_KEY`
