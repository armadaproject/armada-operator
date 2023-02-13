# Golang Operator OpenAPI Validation Markers Notes

## Summary

Using validation [markers](https://book.kubebuilder.io/reference/markers.html) 
(aka annotations) in our go type definitions allows 
the kubernetes CRD API generator to apply validation rules to our CRD manifests. 
This helps kubernetes apply additional verification and validation of CRD 
specifications during creation or update of our custom resources. However, the 
validation rules are rather simple and limited.

If we need more robust or complex validation, then we should turn to 
[validating admission webhooks](https://sdk.operatorframework.io/docs/building-operators/golang/webhook/#1-validating-admission-webhook).

## How-To

### Development Loop

1. Create, update, or delete markers for golang types used by our CRDs.
2. Generate new CRD manifests.
3. Install new manifests in your test cluster.
4. Observe whether marker changes had the desired effect.

### Marker Makeup

From the [marker reference docs](https://book.kubebuilder.io/reference/markers.html):
> Markers are single-line comments that start with a plus, 
> followed by a marker name, optionally followed by some marker 
> specific configuration:

```golang
// +kubebuilder:validation:MinLength=1
```

Above, the marker name is `MinLength` and is set to 1.

### Marker Example
```golang
type Image struct {
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Pattern:="^([a-z0-9]+(?:[._-][a-z0-9]+)*/*)+$"
	Repository string `json:"repository"`
	...
}
```

In the above example, three validation markers will be applied to the 
`Repository` argument:
1. Type must be `string`.
2. The minimum length of the string is 1.
3. The contents of the string must satisfy the regular expression defined in `Pattern`.

Here's what the above markers look like when rendered to the CRD API definition:

```yaml
image:
  description: Image is the configuration block for the image repository and tag 
  properties: 
    repository: 
      minLength: 1 
      pattern: ^([a-z0-9]+(?:[._-][a-z0-9]+)*/*)+$ 
      type: string
  ...
```

### Apply Marker Changes

```bash
make manifests
```

### Install CRDs

```bash
make install
```

### Test Marker Changes

```bash
$ kubectl apply -n armada -f ./config/samples/bad_lookoutingester_config.yaml 
The LookoutIngester "lookoutingester-sample" is invalid: spec.image.tag: Invalid value: "6141171b8bde8a03113090a819d728511c2dc39f@": spec.image.tag in body should match '^[a-zA-Z0-9_.-]*$'
```

```bash
$ kubectl apply -n armada -f ./config/samples/good_lookoutingester_config.yaml 
lookoutingester.install.armadaproject.io/lookoutingester-sample configured
```

## Observations

### Take Advantage of Composition and Go's Type System
Markers on a member of a struct will be applied wherever that struct member 
is used in the CRD API. Therefore it is beneficial to take advantage of 
DRY (Don't Repeat Yourself) and golang's type system to provide consistent
validation across our CRD's API.

### Automatic Type Inference
Some validation is automatic: Type validation is inferred from golang types 
and applied to the OpenAPI definition automatically. For instance, a `string` 
in golang has an implicit validation line like so:
```golang
// +kubebuilder:validation:Type:=string
```

### Embedded Structs and Promoted Fields 
You must add `json:""` to the end of embedded structs in order for 
`controller-gen` to use promoted fields as expected.

## Links

### Operator SDK Mentions of OpenAPI Validation: 
https://sdk.operatorframework.io/docs/building-operators/golang/tutorial/#openapi-validation

https://sdk.operatorframework.io/docs/building-operators/golang/references/openapi-validation/

### Validation Marker Reference
https://book.kubebuilder.io/reference/markers/crd-validation.html
