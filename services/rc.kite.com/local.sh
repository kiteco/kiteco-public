export RC_KITE_COM_AWS_ACCESS_KEY_ID=`aws configure get default.aws_access_key_id`
export RC_KITE_COM_AWS_SECRET_ACCESS_KEY_ID=`aws configure get default.aws_secret_access_key`
export RC_KITE_COM_CUSTOMER_IO_API_KEY=`aws --region=us-west-1 --output text --query 'SecretString' secretsmanager get-secret-value --secret-id CUSTOMER_IO_API_KEY`
export RC_KITE_COM_ELASTIC_CLOUD_ID=`aws --region=us-west-1 --output text --query 'SecretString' secretsmanager get-secret-value --secret-id beats_elastic_cloud_id`
export RC_KITE_COM_ELASTIC_CLOUD_AUTH=`aws --region=us-west-1 --output text --query 'SecretString' secretsmanager get-secret-value --secret-id beats_elastic_auth_str`
