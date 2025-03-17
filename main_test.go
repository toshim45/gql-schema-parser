package main

import (
	"testing"
)

func TestParseGraphQLObject(t *testing.T) {
	// t.SkipNow()
	raw := []string{
		"job_job(",
		"where: $where",
		"limit: $limit",
		"offset: $offset",
		"order_by: $order_by",
		") {",
		"id",
		"company_id",
		"class",
		"is_cacheable",
		"input_hash",
		"output_file_url",
		"output_expires_at",
		"status",
		"}",
	}
	r, c := parseGraphQLObject(raw)

	if r.Name != "job_job" {
		t.Error("Failed: name should be job_job but got", r.Name)
	}

	if c != 14 {
		t.Error("Faild: processed count should be 14 but got", c)
	}

	lFields := len(r.Fields)
	if lFields != 8 {
		t.Error("Failed: fields should be 8 but got", lFields)
	}

}

func TestParseGraphQLObjectWithChildrenNormal(t *testing.T) {
	// t.SkipNow()
	raw := []string{
		"job_job(",
		"where: $where",
		"limit: $limit",
		"offset: $offset",
		"order_by: $order_by",
		") {",
		"id",
		"company_id",
		"class",
		"is_cacheable",
		"input_hash",
		"output_file_url",
		"output_expires_at",
		"status",
		"parameters {",
		"id",
		"key",
		"value",
		"created_at",
		"created_by",
		"updated_at",
		"updated_by",
		"}",
		"error_message",
		"created_at1",
		"created_by1",
		"updated_at1",
		"updated_by1",
		"}",
	}
	r, c := parseGraphQLObject(raw)

	if r.Name != "job_job" {
		t.Error("Failed: name should be job_job but got", r.Name)
	}

	lFields := len(r.Fields)
	if lFields != 14 {
		t.Error("Failed: fields should be 14 but got", lFields)
	}

	if c != 28 {
		t.Error("Failed: processed count should be 18 but got", c)
	}

	lChildren := len(r.Children)
	if lChildren == 0 {
		t.Error("Failed: should have 1 child but got", lChildren)
	}

	lChildrenFields := len(r.Children[0].Fields)
	if lChildrenFields != 7 {
		t.Error("Failed: should have children fields 7 but got", lChildrenFields)
	}
}

func TestParseGraphQLObjectWithChildrenWithNoCloseCurlyBracket(t *testing.T) {
	// t.SkipNow()
	raw := []string{
		"job_job(",
		"where: $where",
		"limit: $limit",
		"offset: $offset",
		"order_by: $order_by",
		") {",
		"id",
		"company_id",
		"class",
		"is_cacheable",
		"input_hash",
		"output_file_url",
		"output_expires_at",
		"status",
		"parameters {",
		"id",
		"key",
		"value",
		"created_at",
		"created_by",
		"updated_at",
		"updated_by",
	}
	r, c := parseGraphQLObject(raw)

	if r.Name != "job_job" {
		t.Error("Failed: name should be job_job but got", r.Name)
	}

	lFields := len(r.Fields)
	if lFields != 9 {
		t.Error("Failed: fields should be 8 but got", lFields)
		t.Logf("%v", r.Fields)
	}

	if c != 23 {
		t.Error("Failed: processed count should be 23 but got", c)
	}

	lChildren := len(r.Children)
	if lChildren == 0 {
		t.Error("Failed: should have 1 child but got", lChildren)
	}

	lChildrenFields := len(r.Children[0].Fields)
	if lChildrenFields != 7 {
		t.Error("Failed: should have children fields 7 but got", lChildrenFields)
	}
}

func TestParseGraphQLObjectWithChildrenWithNoParentCloseCurlyBracket(t *testing.T) {
	// t.SkipNow()
	raw := []string{
		"job_job(",
		"where: $where",
		"limit: $limit",
		"offset: $offset",
		"order_by: $order_by",
		") {",
		"id",
		"company_id",
		"class",
		"is_cacheable",
		"input_hash",
		"output_file_url",
		"output_expires_at",
		"status",
		"parameters {",
		"id",
		"key",
		"value",
		"created_at",
		"created_by",
		"updated_at",
		"updated_by",
		"}",
	}
	r, c := parseGraphQLObject(raw)

	if r.Name != "job_job" {
		t.Error("Failed: name should be job_job but got", r.Name)
	}

	lFields := len(r.Fields)
	if lFields != 9 {
		t.Error("Failed: fields should be 8 but got", lFields)
		t.Logf("%v", r.Fields)
	}

	if c != 23 {
		t.Error("Faild: processed count should be 23 but got", c)
	}

	lChildren := len(r.Children)
	if lChildren == 0 {
		t.Error("Failed: should have 1 child but got", lChildren)
	}

	lChildrenFields := len(r.Children[0].Fields)
	if lChildrenFields != 7 {
		t.Error("Failed: should have children fields 7 but got", lChildrenFields)
	}
}
