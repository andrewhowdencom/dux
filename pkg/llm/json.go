package llm

import (
	"encoding/json"
	"fmt"
)

// messageAlias is used to avoid infinite recursion during json marshaling.
type messageAlias Message

type diskMessage struct {
	messageAlias
	Parts []diskPart `json:"Parts"`
}

type diskPart struct {
	Type    PartType        `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// MarshalJSON implements custom JSON serialization to handle the polymorphic Part interface.
func (m Message) MarshalJSON() ([]byte, error) {
	dp := make([]diskPart, len(m.Parts))
	for i, part := range m.Parts {
		payloadBytes, err := json.Marshal(part)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal part: %w", err)
		}
		dp[i] = diskPart{
			Type:    part.Type(),
			Payload: payloadBytes,
		}
	}

	dm := diskMessage{
		messageAlias: (messageAlias)(m),
		Parts:        dp,
	}
	return json.Marshal(dm)
}

// UnmarshalJSON for ToolRequestPart counter-acts its custom JSON Marshaler mapping 'Tool' to 'Name'.
func (t *ToolRequestPart) UnmarshalJSON(data []byte) error {
	var aux struct {
		ToolID string                 `json:"tool_id"`
		Tool   string                 `json:"tool"`
		Args   map[string]interface{} `json:"args"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	t.ToolID = aux.ToolID // Not strictly marshaled in original, adding support
	t.Name = aux.Tool
	t.Args = aux.Args
	return nil
}

// UnmarshalJSON implements custom JSON deserialization to correctly instantiate the polymorphic Part interface.
func (m *Message) UnmarshalJSON(data []byte) error {
	var dm diskMessage
	if err := json.Unmarshal(data, &dm); err != nil {
		return err
	}

	parts := make([]Part, len(dm.Parts))
	for i, dp := range dm.Parts {
		var p Part
		switch dp.Type {
		case TypeText:
			var txt string
			if err := json.Unmarshal(dp.Payload, &txt); err != nil {
				return err
			}
			p = TextPart(txt)
		case TypeReasoning:
			var str string
			if err := json.Unmarshal(dp.Payload, &str); err != nil {
				return err
			}
			p = ReasoningPart(str)
		case TypeImage:
			var img ImagePart
			if err := json.Unmarshal(dp.Payload, &img); err != nil {
				return err
			}
			p = img
		case TypeToolCall:
			var tr ToolRequestPart
			if err := json.Unmarshal(dp.Payload, &tr); err != nil {
				return err
			}
			p = tr
		case TypeToolDefinition:
			var td ToolDefinitionPart
			if err := json.Unmarshal(dp.Payload, &td); err != nil {
				return err
			}
			p = td
		case TypeToolResult:
			var tr ToolResultPart
			if err := json.Unmarshal(dp.Payload, &tr); err != nil {
				return err
			}
			p = tr
		case TypeTransitionSignal:
			var ts TransitionSignalPart
			if err := json.Unmarshal(dp.Payload, &ts); err != nil {
				return err
			}
			p = ts
		case TypeTelemetry:
			var tp TelemetryPart
			if err := json.Unmarshal(dp.Payload, &tp); err != nil {
				return err
			}
			p = tp
		default:
			return fmt.Errorf("unknown part type: %s", dp.Type)
		}
		parts[i] = p
	}

	*m = Message(dm.messageAlias)
	m.Parts = parts
	return nil
}
