# Context Passing Example

This repository contains an Upbound project, tailored for users of [Upbound](https://cloud.upbound.io). This configuration demonstrates the use of several features of Compositions to pass information to users and between components.

## Overview

`function-vpclookup` is a golang-based function that uses the AWS go SDK to lookup vpcs by `tag:Name`. It then sets the vpc id of the first matching vpc to the function pipeline context, as well as the composite resource status. The context is the primary means of communication between functions in a given pipeline, and the resource status is the primary means of communication between the pipeline and the end user.

`function-deploysubnet` reads the vpc id set in the pipeline context to deploy a subnet to that vpc.

## Testing

The configuration can be tested using 
```bash
up composition render --xrd=apis/xexamples/definition.yaml apis/xexamples/composition.yaml examples/xexample/select-by-name.yaml
``` 
to render the composition

> Note: the `vpclookup` function expects a secret containing AWS credentials in the format:
>
> credentials.yaml
> ```
> apiVersion: v1
> data:
>   aws_access_key_id: <base64 encoded access key id>
>   aws_secret_access_key: <base64 encoded secret access key>
>   aws_session_token: <base64 encoded session token (optional)>
> kind: Secret
> metadata:
>   name: aws-credentials
>   namespace: crossplane-system
> ``` 
> This can be passed into the render command with `--function-credentials=credendials.yaml`