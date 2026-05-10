package main

import "testing"

func TestEvaluateConditionSupportsNumericScalarFailValue(t *testing.T) {
	if !evaluateCondition("1", ">", 0) {
		t.Fatalf("expected numeric scalar fail_value to trigger failure for output 1 > 0")
	}

	if evaluateCondition("0", ">", 0) {
		t.Fatalf("did not expect output 0 to fail against numeric scalar fail_value 0")
	}
}

func TestParseFailValuesSupportsNumericScalar(t *testing.T) {
	values := parseFailValues(0)
	if len(values) != 1 || values[0] != "0" {
		t.Fatalf("expected numeric scalar fail_value to resolve to [\"0\"], got %#v", values)
	}
}
