package main

import (
	"context"
	"encoding/json"

	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/logging"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/request"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/resource/composite"
	"github.com/crossplane/function-sdk-go/response"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	v1 "dev.upbound.io/models/io/upbound/v1"
)

// Function is your composition function.
type Function struct {
	fnv1.UnimplementedFunctionRunnerServiceServer

	log logging.Logger
}

// RunFunction runs the Function.
func (f *Function) RunFunction(_ context.Context, req *fnv1.RunFunctionRequest) (*fnv1.RunFunctionResponse, error) {
	f.log.Info("Running function", "tag", req.GetMeta().GetTag())

	// Build a default response object
	rsp := response.To(req, response.DefaultTTL)

	// Retrieve the observed composite resource
	observedComposite, err := request.GetObservedCompositeResource(req)
	if err != nil {
		response.Fatal(rsp, errors.Wrap(err, "cannot get observed composite"))
		return rsp, nil
	}

	// Convert to the concrete type
	var xr v1.XExample
	if err := convertViaJSON(&xr, observedComposite.Resource); err != nil {
		response.Fatal(rsp, errors.Wrap(err, "cannot convert observed composite"))
		return rsp, nil
	}

	// Set up the AWS client library
	// First, pull back the credentials
	creds, err := request.GetCredentials(req, "aws")
	if err != nil {
		response.Fatal(rsp, errors.Wrap(err, "unable to load AWS SDK credentials"))
		return rsp, nil
	}

	// Configure the AWS SDK
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("eu-west-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(string(creds.Data["aws_access_key_id"]), string(creds.Data["aws_secret_access_key"]), string(creds.Data["aws_session_token"]))),
	)
	if err != nil {
		response.Fatal(rsp, errors.Wrap(err, "unable to load AWS SDK config"))
		return rsp, nil
	}

	// Using the Config value, create the EC2 client
	svc := ec2.NewFromConfig(cfg)

	// Pull the VPCs back by tag
	describeOutput, err := svc.DescribeVpcs(context.TODO(), &ec2.DescribeVpcsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("tag:Name"),
				Values: []string{*xr.Spec.Selector.Name},
			},
		},
	})
	if err != nil {
		response.Fatal(rsp, errors.Wrap(err, "unable to query vpcs"))
		return rsp, nil
	}

	// if there are any VPCs returned, set the ID of the first match to the pipeline context, as well as to the Composition Status
	if len(describeOutput.Vpcs) > 0 {
		// set to context
		response.SetContextKey(rsp, "vpcid", structpb.NewStringValue(*describeOutput.Vpcs[0].VpcId))

		// set to XR Status
		xr.Status = &v1.XExampleStatus{
			VpcID: describeOutput.Vpcs[0].VpcId,
		}

		// convert xr to a desired state
		desiredComposite := composite.New()
		if err := convertViaJSON(&desiredComposite.Unstructured, &xr); err != nil {
			response.Fatal(rsp, errors.Wrap(err, "cannot convert desired composite"))
			return rsp, nil
		}

		// set the desired state to the composite
		if err := response.SetDesiredCompositeResource(rsp, &resource.Composite{Resource: desiredComposite}); err != nil {
			response.Fatal(rsp, errors.Wrap(err, "cannot set desired composite"))
			return rsp, nil
		}
	}

	response.ConditionTrue(rsp, "FunctionSuccess", "Success").
		TargetCompositeAndClaim()

	return rsp, nil
}

// Helper function to convert from one type to another via JSON
func convertViaJSON(to, from any) error {
	bs, err := json.Marshal(from)
	if err != nil {
		return err
	}
	return json.Unmarshal(bs, to)
}
