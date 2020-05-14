Route53 for `libdns`
=======================

[![godoc reference](https://img.shields.io/badge/godoc-reference-blue.svg)](https://pkg.go.dev/github.com/libdns/route53)

This package implements the [libdns interfaces](https://github.com/libdns/libdns) for AWS [Route53](https://aws.amazon.com/route53/).

## Authenticating

This package supports all the credential configuration methods described in the [AWS Developer Guide](https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html), such as `Environment Variables`, `EC2 Instance Profile` and the `AWS Credentials file` located in `.aws/credentials`

The following IAM policy is a minimal working example to give `libdns` permissions to manage DNS records:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "",
            "Effect": "Allow",
            "Action": [
                "route53:ListResourceRecordSets",
                "route53:GetChange",
                "route53:ChangeResourceRecordSets"
            ],
            "Resource": [
                "arn:aws:route53:::hostedzone/ZABCD1EFGHIL",
                "arn:aws:route53:::change/*"
            ]
        },
        {
            "Sid": "",
            "Effect": "Allow",
            "Action": [
                "route53:ListHostedZonesByName",
                "route53:ListHostedZones"
            ],
            "Resource": "*"
        }
    ]
}
```
