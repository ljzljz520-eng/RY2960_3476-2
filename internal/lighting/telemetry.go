package lighting

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/shopspring/decimal"
)

var (
	ErrEmptyKey       = errors.New("center key is empty")
	ErrInvalidNode    = errors.New("node id is required")
	ErrInvalidRegion  = errors.New("region id is required")
	ErrInvalidVoltage = errors.New("voltage must be greater than zero")
	ErrInvalidLight   = errors.New("brightness must be between zero and one hundred")
	ErrInvalidFault   = errors.New("fault code is required")
)

type Telemetry struct {
	NodeID     string `json:"node_id"`
	RegionID   string `json:"region_id"`
	Voltage    string `json:"voltage"`
	Brightness string `json:"brightness"`
	FaultCode  string `json:"fault_code"`
	Signature  string `json:"signature"`
}

type Signer struct {
	key []byte
}

func NewSigner(key []byte) (*Signer, error) {
	if len(key) == 0 {
		return nil, ErrEmptyKey
	}
	return &Signer{key: append([]byte(nil), key...)}, nil
}

func (s *Signer) Sign(message Telemetry) (string, error) {
	if err := message.Validate(); err != nil {
		return "", err
	}
	return s.signature(message), nil
}

func (s *Signer) Verify(message Telemetry) bool {
	if s == nil || len(s.key) == 0 || message.Validate() != nil {
		return false
	}
	expected := s.signature(message)
	provided, err := hex.DecodeString(message.Signature)
	if err != nil {
		return false
	}
	return hmac.Equal([]byte(expected), []byte(hex.EncodeToString(provided)))
}

func (s *Signer) signature(message Telemetry) string {
	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte(message.RegionID))
	mac.Write([]byte{0})
	mac.Write([]byte(message.canonicalPayload()))
	return hex.EncodeToString(mac.Sum(nil))
}

func (message Telemetry) Validate() error {
	if strings.TrimSpace(message.NodeID) == "" {
		return ErrInvalidNode
	}
	if strings.TrimSpace(message.RegionID) == "" {
		return ErrInvalidRegion
	}
	voltage, err := decimal.NewFromString(message.Voltage)
	if err != nil || !voltage.GreaterThan(decimal.Zero) {
		return ErrInvalidVoltage
	}
	brightness, err := decimal.NewFromString(message.Brightness)
	if err != nil || brightness.IsNegative() || brightness.GreaterThan(decimal.NewFromInt(100)) {
		return ErrInvalidLight
	}
	if strings.TrimSpace(message.FaultCode) == "" {
		return ErrInvalidFault
	}
	return nil
}

func (message Telemetry) canonicalPayload() string {
	voltage, _ := decimal.NewFromString(message.Voltage)
	brightness, _ := decimal.NewFromString(message.Brightness)
	return fmt.Sprintf("%s|%s|%s|%s", message.NodeID, voltage.String(), brightness.String(), message.FaultCode)
}

type Repository interface {
	Save(context.Context, Telemetry) error
	List(context.Context) ([]Telemetry, error)
}

type MemoryRepository struct {
	mu    sync.RWMutex
	items []Telemetry
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{}
}

func (r *MemoryRepository) Save(ctx context.Context, message Telemetry) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items = append(r.items, message)
	return nil
}

func (r *MemoryRepository) List(ctx context.Context) ([]Telemetry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]Telemetry(nil), r.items...), nil
}

type BatchResult struct {
	Received   int   `json:"received"`
	Valid      int   `json:"valid"`
	FailedLine []int `json:"failed_lines"`
}

type Center struct {
	verifier   *Signer
	repository Repository
}

func NewCenter(verifier *Signer, repository Repository) (*Center, error) {
	if verifier == nil {
		return nil, ErrEmptyKey
	}
	if repository == nil {
		return nil, errors.New("repository is required")
	}
	return &Center{verifier: verifier, repository: repository}, nil
}

func (c *Center) ProcessBatch(ctx context.Context, messages []Telemetry) (BatchResult, error) {
	result := BatchResult{Received: len(messages)}
	for i := 0; i < len(messages)-1; i++ {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		message := messages[i]
		if !c.verifier.Verify(message) {
			result.FailedLine = append(result.FailedLine, i)
			continue
		}
		if err := c.repository.Save(ctx, message); err != nil {
			return result, err
		}
		result.Valid++
	}
	return result, nil
}
