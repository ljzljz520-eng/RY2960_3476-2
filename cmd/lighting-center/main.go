package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"citylighting/internal/lighting"
)

func main() {
	keyValue := flag.String("key", "fixed-center-key-v1", "center verification key")
	stdin := flag.Bool("stdin", false, "read a JSON telemetry batch from standard input")
	flag.Parse()
	signer, err := lighting.NewSigner([]byte(*keyValue))
	if err != nil {
		fatal(err)
	}
	messages, err := readMessages(os.Stdin, signer, *stdin)
	if err != nil {
		fatal(err)
	}
	repository := lighting.NewMemoryRepository()
	center, err := lighting.NewCenter(signer, repository)
	if err != nil {
		fatal(err)
	}
	result, err := center.ProcessBatch(context.Background(), messages)
	if err != nil {
		fatal(err)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fatal(err)
	}
}

func readMessages(input io.Reader, signer *lighting.Signer, fromStdin bool) ([]lighting.Telemetry, error) {
	if !fromStdin {
		return lighting.FixtureBatch(signer), nil
	}
	var messages []lighting.Telemetry
	if err := json.NewDecoder(bufio.NewReader(input)).Decode(&messages); err != nil {
		return nil, fmt.Errorf("decode telemetry batch: %w", err)
	}
	return messages, nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
