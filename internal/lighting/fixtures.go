package lighting

func FixtureBatch(signer *Signer) []Telemetry {
	messages := []Telemetry{
		{NodeID: "lamp-ny-01", RegionID: "region-7", Voltage: "220.00", Brightness: "80", FaultCode: "OK"},
		{NodeID: "lamp-ny-02", RegionID: "region-7", Voltage: "219.50", Brightness: "65.5", FaultCode: "OK"},
		{NodeID: "lamp-ny-03", RegionID: "region-8", Voltage: "221.25", Brightness: "0", FaultCode: "LAMP_OFF"},
	}
	for i := range messages {
		messages[i].Signature, _ = signer.Sign(messages[i])
	}
	return messages
}
