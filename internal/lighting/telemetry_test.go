package lighting

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func testSigner(t *testing.T) *Signer {
	t.Helper()
	signer, err := NewSigner([]byte("fixed-center-key-v1"))
	if err != nil {
		t.Fatal(err)
	}
	return signer
}

func tamperSignature(signature string) string {
	replacement := byte('0')
	if signature[len(signature)-1] == replacement {
		replacement = '1'
	}
	return signature[:len(signature)-1] + string(replacement)
}

func TestBatchRejectsTamperedLastTelemetry(t *testing.T) {
	signer := testSigner(t)
	messages := FixtureBatch(signer)
	messages[2].Signature = tamperSignature(messages[2].Signature)
	repository := NewMemoryRepository()
	center, err := NewCenter(signer, repository)
	if err != nil {
		t.Fatal(err)
	}
	result, err := center.ProcessBatch(context.Background(), messages)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff([]int{2}, result.FailedLine); diff != "" {
		t.Fatalf("failed lines mismatch (-want +got):\n%s", diff)
	}
	if result.Valid != 2 || result.Received != 3 {
		t.Fatalf("unexpected batch counts: %+v", result)
	}
}

func TestBatchPersistsVerifiedTelemetry(t *testing.T) {
	signer := testSigner(t)
	messages := FixtureBatch(signer)
	repository := NewMemoryRepository()
	center, err := NewCenter(signer, repository)
	if err != nil {
		t.Fatal(err)
	}
	result, err := center.ProcessBatch(context.Background(), messages)
	if err != nil {
		t.Fatal(err)
	}
	if result.Valid != 3 {
		t.Fatalf("valid count = %d, want 3", result.Valid)
	}
	stored, err := repository.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(messages, stored); diff != "" {
		t.Fatalf("stored telemetry mismatch (-want +got):\n%s", diff)
	}
}

func TestBatchReportsEarlierInvalidTelemetry(t *testing.T) {
	signer := testSigner(t)
	messages := FixtureBatch(signer)
	messages[1].Signature = "00"
	repository := NewMemoryRepository()
	center, err := NewCenter(signer, repository)
	if err != nil {
		t.Fatal(err)
	}
	result, err := center.ProcessBatch(context.Background(), messages)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff([]int{1}, result.FailedLine); diff != "" {
		t.Fatalf("failed lines mismatch (-want +got):\n%s", diff)
	}
	if result.Valid != 2 {
		t.Fatalf("valid count = %d, want 2", result.Valid)
	}
}

func TestSignatureBindsRegionAndCanonicalValues(t *testing.T) {
	signer := testSigner(t)
	message := Telemetry{NodeID: "lamp-1", RegionID: "region-a", Voltage: "220.0", Brightness: "80.00", FaultCode: "OK"}
	signature, err := signer.Sign(message)
	if err != nil {
		t.Fatal(err)
	}
	message.Signature = signature
	if !signer.Verify(message) {
		t.Fatal("signed message did not verify")
	}
	message.RegionID = "region-b"
	if signer.Verify(message) {
		t.Fatal("region change was accepted")
	}
	message.RegionID = "region-a"
	message.Voltage = "not-a-number"
	if signer.Verify(message) {
		t.Fatal("malformed voltage was accepted")
	}
}

func TestRepositoryHonorsCanceledContext(t *testing.T) {
	repository := NewMemoryRepository()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := repository.Save(ctx, Telemetry{}); err == nil {
		t.Fatal("save succeeded with canceled context")
	}
	if _, err := repository.List(ctx); err == nil {
		t.Fatal("list succeeded with canceled context")
	}
}
