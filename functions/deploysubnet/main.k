
import models.io.upbound.aws.ec2.v1beta1 as ec2v1beta1

ctx = option("params").ctx

_metadata = lambda name: str -> any {
    { annotations = { "krm.kcl.dev/composition-resource-name" = name }}
}

_items = [
    ec2v1beta1.Subnet{
        spec.forProvider: {
            region: "eu-west-1"
            availabilityZone: "eu-west-1a"
            cidrBlock: "192.168.0.0/24"
            vpcId: ctx["vpcid"]
        }
    }
]
items = _items
