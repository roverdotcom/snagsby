import json

import boto3

"""
# A bucket policy requiring AES256 server side encryption and https
# communication with s3 (I would imagine all the SDKs do this).

{
  "Version": "2012-10-17",
  "Id": "PutObjPolicy",
  "Statement": [
    {
      "Sid": "DenyIncorrectEncryptionHeader",
      "Effect": "Deny",
      "Principal": "*",
      "Action": "s3:PutObject",
      "Resource": "arn:aws:s3:::rover-bryan/*",
      "Condition": {
        "StringNotEquals": {
          "s3:x-amz-server-side-encryption": "AES256"
        }
      }
    },
    {
      "Sid": "DenyUnEncryptedObjectUploads",
      "Effect": "Deny",
      "Principal": "*",
      "Action": "s3:PutObject",
      "Resource": "arn:aws:s3:::rover-bryan/*",
      "Condition": {
        "Null": {
          "s3:x-amz-server-side-encryption": "true"
        }
      }
    },
    {
        "Action": "s3:*",
        "Effect":"Deny",
        "Principal": "*",
        "Resource":"arn:aws:s3:::rover-bryan/*",
        "Condition":{
            "Bool":
            { "aws:SecureTransport": false }
        }
    }

  ]
}
"""

s3 = boto3.resource('s3')
bucket = s3.Bucket('rover-bryan')

multiline_secret = """123
456
789
10"""

secrets = {
    "SECRET": "tulkinghorn",
    "upcase": "Keys will be upcased.",
    "MULTILINE": multiline_secret,
    "DOUBLE_QUOTES": 'My name is "Charles"',
    "SINGLE_QUOTE": "My surname is 'Dickens'",
    "NUMBER": 1,
    "FLOAT_LIKE": 7.777,
    "yes": True,
    "no": False,
    "will fail": "because of the space in the key",
}

bucket.Object('snagsby-config.json').put(
    Body=json.dumps(secrets),
    ServerSideEncryption="AES256",
)

# Write to a second source for testing
bucket.Object('snagsby-config2.json').put(
    Body=json.dumps({'another_source': '12345'}),
    ServerSideEncryption="AES256",
)

bucket.Object('snagsby-config3.json').put(
    Body=json.dumps({'third_source': 'third_source_data'}),
    ServerSideEncryption="AES256",
)
