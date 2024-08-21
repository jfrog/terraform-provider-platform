resource "platform_aws_iam_role" "myuser-aws-iam-role" {
  username = "myuser"
  iam_role = "arn:aws:iam::000000000000:role/example"
}