## Kiteco devops Good Practices

### Terraform images

#### Storing Terraform tfstate file on S3

We store all tfstate file on s3 to avoid git conflict on them. To do so, add : 
```
terraform {
  backend "s3" {
    bucket = "kite-terraform-state"
    key    = "tf_serving/terraform.tfstate"
    region = "us-west-1"
  }
}
```

In your main terraform files.

Note that you need to have aws credentials configured locally in your env to be able to run a terraform script. 

#### Configuring GCP and AWS credentials

We use the module `instance-role` defined in `terraform/cloud/modules/instance_role` to automatically create a GCP and AWS service accounts for the instance. 

To do so, add in your main tf file : 
```
module "instance_role" {
  source  = "../terraform/cloud/modules/instance_role" # Replace by correct relative path to instance_role folder
  name    = "tfserving"
  secrets = []

  policy_statements = [{
    sid = "100"
    actions = ["s3:GetObject"]
    resources = ["arn:aws:s3:::kite-data/*"] # List all the bucket you need here
  }, { # This second block is only required if you do folder download
    sid = "101" 
    actions = ["s3:ListBucket"]
    resources = ["arn:aws:s3:::kite-data"] # Note the absence of /* at the end of the path, that should only be a bucket name
  }]
}
``` 
